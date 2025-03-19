#!/usr/bin/env bash
set -e

#later versions are executables and not just tar.gz files so less useful
#this particular kubebulder version has these versions of  etcd=3.3.11, and kubectl/kube-apiserver v1.16.4
KUBEBUILDER_VERSION="v2.3.2"

echo "Setting up test binaries..."
# Determine OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Early check for MacOS ARM64 (M1/M2)
if [ "$OS" = "darwin" ] && [ "$ARCH" = "arm64" ]; then
    echo "ERROR: This script cannot run directly on MacOS ARM64 (Apple Silicon)."
    echo "Please use Docker to run the tests, which will provide the correct Linux environment."
    echo "no point to grab the test binaries on architectures we cannot test on."
    echo "go relies on network code that requires linux"
    echo "the version 2.3.2 does not support darwin/arm64, so we nope out on that one"
    exit 1
fi

# Determine the target architecture for binaries
case $ARCH in
    x86_64)
        KUBE_ARCH="amd64"
        ;;
    aarch64)
        KUBE_ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH, falling back to amd64"
        KUBE_ARCH="amd64"
        ;;
esac

echo "Detected: OS=$OS, ARCH=$ARCH (using KUBE_ARCH=$KUBE_ARCH)"

# Ensure we're targeting linux for the binaries regardless of host OS
if [ "$OS" != "linux" ]; then
    echo "Note: Running on $OS but downloading Linux binaries for use in Docker/containers"
    TARGET_OS="linux"
else
    TARGET_OS="linux"
fi

mkdir -p __main__/hack/bin

# Try architecture-specific version first, then fall back to amd64 if needed

ARCH_URL="https://github.com/kubernetes-sigs/kubebuilder/releases/download/${KUBEBUILDER_VERSION}/kubebuilder_${KUBEBUILDER_VERSION#v}_${TARGET_OS}_${KUBE_ARCH}.tar.gz"

# Check if architecture-specific version exists
if curl --output /dev/null --silent --head --fail "$ARCH_URL"; then
    echo "Found $KUBE_ARCH version of kubebuilder. Downloading from $ARCH_URL"
    curl -sfL "$ARCH_URL" | tar xvz --strip-components=1 -C __main__/hack
else
    echo "No $KUBE_ARCH version available. Falling back to amd64 version..."
    AMD64_URL="https://github.com/kubernetes-sigs/kubebuilder/releases/download/${KUBEBUILDER_VERSION}/kubebuilder_${KUBEBUILDER_VERSION#v}_${TARGET_OS}_amd64.tar.gz"
    echo "Downloading from $AMD64_URL"
    curl -sfL "$AMD64_URL" | tar xvz --strip-components=1 -C __main__/hack
fi

# Ensure binaries are in the expected location and have correct permissions
for binary in etcd kube-apiserver kubectl; do
    if [ -f "__main__/hack/$binary" ]; then
        echo "Installing $binary to __main__/hack/bin/"
        cp "__main__/hack/$binary" "__main__/hack/bin/" || ln -sf "../$binary" "__main__/hack/bin/$binary"
        chmod +x "__main__/hack/bin/$binary"
    else
        echo "Warning: $binary not found in extracted files"
    fi
done

echo "Test binaries ready:"
ls -la __main__/hack/bin/
