IMAGE_NAME ?= "lucas-albers-lz4/cert-manager-webhook-duckdns"
IMAGE_TAG := "latest"
PLATFORMS := "linux/amd64,linux/arm64"
BUILDKIT_COMPRESSION := "zstd"
BUILDKIT_COMPRESSION_LEVEL := "9"

OUT := $(shell pwd)/_out

$(shell mkdir -p "$(OUT)")

verify:
	go test -v .

buildx:
	docker buildx build --platform $(PLATFORMS) --tag $(IMAGE_NAME):$(IMAGE_TAG) --provenance=mode=max . 

release: buildx
	docker buildx build --platform $(PLATFORMS) --tag $(IMAGE_NAME):$(IMAGE_TAG) --provenance=mode=max . --push

#don't change existing build commands
build:
	docker build -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

.PHONY: rendered-manifest.yaml
rendered-manifest.yaml:
	helm template \
	    --name cert-manager-webhook-duckdns \
        --set image.repository=$(IMAGE_NAME) \
        --set image.tag=$(IMAGE_TAG) \
        deploy/cert-manager-webhook-duckdns > "$(OUT)/rendered-manifest.yaml"

test:
	TEST_ZONE_NAME="duckdns.org." DNS_NAME="test.duckdns.org" go test -v .

# Check for required environment variables in .env file
.PHONY: check-env-file
check-env-file:
	@if [ ! -f testdata/duckdns/env.testconfig ]; then \
		echo "Error: testdata/duckdns/env.testconfigfile not found"; \
		exit 1; \
	fi

docker-build-unittest:
	docker build -f Dockerfile.unittest -t cert-manager-webhook-duckdns:test .

docker-run-unittest:
	source testdata/duckdns/env.testconfig && docker run --rm -e TEST_ZONE_NAME=$${TEST_ZONE_NAME} -e DNS_NAME=$${DNS_NAME} -e DUCKDNS_TOKEN=$${DUCKDNS_TOKEN} cert-manager-webhook-duckdns:test

docker-unittest: check-env-file docker-build-unittest docker-run-unittest

# quick compile works on macos
compile:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o webhook-arm64 -ldflags '-w -extldflags "-static"'

github-listbuild:
	gh run list --workflow=docker-build.yml  --repo github.com/$(IMAGE_NAME)

github-build:
	gh workflow run "Build and Push Docker Images" --repo github.com/$(IMAGE_NAME) --ref master
