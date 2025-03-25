# Testing the DuckDNS Webhook for cert-manager

This document describes how to test the DuckDNS webhook for cert-manager.

## Prerequisites

- Docker
- A DuckDNS account and API token
- A DuckDNS domain for testing

## Setting Up Test Configuration

1. Copy the example test configuration:

```bash
cp testdata/duckdns/env.testconfig.example testdata/duckdns/env.testconfig
```

2. Edit the configuration file to include your DuckDNS token and domain:

```bash
# testdata/duckdns/env.testconfig
export DUCKDNS_TOKEN=<your DuckDNS token>
export TEST_ZONE_NAME=<your-domain>.duckdns.org.  # Note the trailing dot
export DNS_NAME=<your-domain>.duckdns.org  # No trailing dot
```

3. Build and run Docker unit test, it has items we don't use for production, and only run the test and quits.

```
make docker-unittest
```

The unit test uses the go modules from : https://pkg.go.dev/github.com/ebrianne/duckdns-go#section-readme

They mock a kube-api call using etcd.
They expect binaries in hardcoded locations. 

The line that indicates success is this one, it indicates it successfully created a challenge entry:

```
# Key line to look for:
**I0318 21:15:21.313985   12025 solver.go:168] Got txt record: 123d==**


Example output of running unit test via Docker

Replacing DuckDNS token in api-key.yml

=== RUN   TestRunsSuite
    fixture.go:127: started apiserver on "http://127.0.0.1:36203"

=== RUN   TestRunsSuite/Conformance
=== RUN   TestRunsSuite/Conformance/Basic
=== RUN   TestRunsSuite/Conformance/Basic/PresentRecord

    util.go:68: created fixture "basic-present-record"
    suite.go:37: Calling Present with ChallengeRequest: &v1alpha1.ChallengeRequest{
        UID: "",
        Action: "",
        Type: "",
        DNSName: "<REDACTED>.duckdns.org",
        Key: "123d==",
        ResourceNamespace: "basic-present-record",
        ResolvedFQDN: "cert-manager-dns01-tests.<REDACTED>.duckdns.org.",
        ResolvedZone: "<REDACTED>.duckdns.org.",
        AllowAmbientCredentials: false,
        Config: (*v1beta1.JSON)(0x400048a3f0),
    }

I0318 21:13:46.721003   12025 solver.go:127] Presenting txt record: cert-manager-dns01-tests.<REDACTED>.duckdns.org. <REDACTED>.duckdns.org.
<REDACTED>
I0318 21:13:46.729000   12025 solver.go:85] DNSName from ChallengeRequest is <REDACTED>.duckdns.org
I0318 21:13:46.729105   12025 solver.go:97] Got dns domain from challenge <REDACTED>
I0318 21:13:46.729113   12025 solver.go:135] Present txt record for domain <REDACTED>
I0318 21:13:46.729117   12025 client.go:105] Sending request to https://www.duckdns.org/update?domains=<REDACTED>&token=*********&txt=123d==
I0318 21:13:47.092179   12025 solver.go:142] Presented txt record cert-manager-dns01-tests.<REDACTED>.duckdns.org.

[... DNS lookup and timeout messages omitted ...]

I0318 21:15:20.361755   12025 solver.go:85] DNSName from ChallengeRequest is <REDACTED>.duckdns.org
I0318 21:15:20.361771   12025 solver.go:97] Got dns domain from challenge <REDACTED>
I0318 21:15:20.361787   12025 solver.go:161] Cleaning up txt record for domain <REDACTED>

# Key line to look for:
**I0318 21:15:21.313985   12025 solver.go:168] Got txt record: 123d==**

I0318 21:15:21.314040   12025 client.go:105] Sending request to https://www.duckdns.org/update?domains=<REDACTED>&token=*********&txt=123d==&clear=true
I0318 21:15:21.588405   12025 solver.go:180] Cleaned up txt record: cert-manager-dns01-tests.<REDACTED>.duckdns.org. <REDACTED>.duckdns.org.

[... more cleanup messages omitted ...]

=== RUN   TestRunsSuite/Conformance/Extended
=== RUN   TestRunsSuite/Conformance/Extended/DeletingOneRecordRetainsOthers

    suite.go:73: skipping test as strict mode is disabled, see: https://github.com/jetstack/cert-manager/pull/1354

--- PASS: TestRunsSuite (121.47s)
    --- PASS: TestRunsSuite/Conformance (99.79s)
        --- PASS: TestRunsSuite/Conformance/Basic (99.79s)
            --- PASS: TestRunsSuite/Conformance/Basic/PresentRecord (99.79s)
        --- PASS: TestRunsSuite/Conformance/Extended (0.00s)
            --- SKIP: TestRunsSuite/Conformance/Extended/DeletingOneRecordRetainsOthers (0.00s)

PASS
ok  	github.com/ebrianne/cert-manager-webhook-duckdns	121.498s


##Additional Notes
You cannot test on macos as it depends on linux specific network code, you will get this eror on macos:
```
go test -v .
# github.com/ebrianne/cert-manager-webhook-duckdns.test
link: golang.org/x/net/internal/socket: invalid reference to syscall.recvmsg
FAIL	github.com/ebrianne/cert-manager-webhook-duckdns [build failed]
FAIL
```
