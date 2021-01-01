package duckdns

import (
	"fmt"
	"context"

	"github.com/jetstack/cert-manager/pkg/acme/webhook"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"

	"github.com/pkg/errors"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

func NewSolver() webhook.Solver {
	return &duckDNSProviderSolver{}
}

// Solver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/jetstack/cert-manager/pkg/acme/webhook.Solver`
// interface.
type duckDNSProviderSolver struct {
	client *kubernetes.Clientset
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (s *duckDNSProviderSolver) Name() string {
	return "duckdns"
}

func (s *duckDNSProviderSolver) validateConfig(cfg *Config) error {

	if cfg.APITokenSecretRef.LocalObjectReference.Name == "" {
		return errors.New("no api token secret provided in DuckDNS config")
	}

	return nil
}

func (s *duckDNSProviderSolver) newClientFromChallenge(ch *v1alpha1.ChallengeRequest) (*Client, error) {
	
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return nil, err
	}

	err = s.validateConfig(&cfg)
	if err != nil {
		return nil, err
	}

	klog.Infof("Decoded config: %v", cfg)

	apiToken, err := s.getApiToken(&cfg, ch.ResourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("get credential error: %v", err)
	}

	client, err := newClient(*apiToken)
	if err != nil {
		return nil, fmt.Errorf("new dns client error: %v", err)
	}

	return client, nil
}

func (s *duckDNSProviderSolver) getApiToken(cfg *Config, namespace string) (*string, error) {

	secretName := cfg.APITokenSecretRef.LocalObjectReference.Name

	secret, err := s.client.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load secret %q", namespace+"/"+secretName)
	}

	data, ok := secret.Data[cfg.APITokenSecretRef.Key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in secret \"%s/%s\"", cfg.APITokenSecretRef.Key,
			cfg.APITokenSecretRef.LocalObjectReference.Name, namespace)
	}

	apiKey := string(data)
	return &apiKey, nil
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (s *duckDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.Infof("Presenting txt record: %v %v", ch.ResolvedFQDN, ch.ResolvedZone)
	client, err := s.newClientFromChallenge(ch)
	if err != nil {
		klog.Errorf("New client from challenge error: %v", err)
		return err
	}

	domain, duckdnszone := client.getDomainZone(ch.ResolvedZone, ch.ResolvedFQDN)
	klog.Infof("Got domain %v and duckdns zone %v", domain, duckdnszone)

	if err := client.addTxtRecord(duckdnszone, ch.Key); err != nil {
		klog.Errorf("Add txt record %q error: %v", ch.ResolvedFQDN, err)
		return err
	}

	klog.Infof("Presented txt record %v", ch.ResolvedFQDN)
	return nil
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (s *duckDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	klog.Infof("Cleaning up txt record: %v %v", ch.ResolvedFQDN, ch.ResolvedZone)
	client, err := s.newClientFromChallenge(ch)
	if err != nil {
		klog.Errorf("New client from challenge error: %v", err)
		return err
	}

	domain, duckdnszone := client.getDomainZone(ch.ResolvedZone, ch.ResolvedFQDN)
	klog.Infof("Got domain %v and duckdns zone %v", domain, duckdnszone)

	record, err := client.getTxtRecord(domain); 
	if err != nil {
		klog.Errorf("Get text record %v error: %v", ch.ResolvedFQDN, err)
		return err
	}
	klog.Infof("Got txt record: %v", record)

	if record != ch.Key {
		klog.Errorf("Records value does not match: %v", ch.ResolvedFQDN)
		return errors.New("record value does not match")
	}

	if err := client.deleteTxtRecord(duckdnszone, ch.Key); err != nil {
		klog.Errorf("Delete domain record %v error: %v", ch.ResolvedFQDN, err)
		return err
	}

	klog.Infof("Cleaned up txt record: %v %v", ch.ResolvedFQDN, ch.ResolvedZone)
	return nil
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
//
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
//
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (s *duckDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	s.client = cl
	return nil
}