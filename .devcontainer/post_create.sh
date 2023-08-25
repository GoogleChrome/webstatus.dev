#!/bin/bash

# Install special version until https://github.com/GoogleContainerTools/skaffold/pull/9008 is released
curl -Lo skaffold https://github.com/jcscottiii/skaffold/releases/download/9006-workaround/skaffold && sudo install skaffold /usr/local/bin/ && rm skaffold

# For AMD64 / x86_64
[ $(uname -m) = x86_64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
# For ARM64
[ $(uname -m) = aarch64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-arm64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

kind delete cluster; kind create cluster