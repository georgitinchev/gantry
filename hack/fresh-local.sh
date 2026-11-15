#!/usr/bin/env bash
# Recreates the local kind cluster and runs a full verification.
set -euo pipefail
cd "$(dirname "$0")/.."
source hack/versions.env
kind delete cluster --name gantry >/dev/null 2>&1 || true
kind create cluster --config hack/kind.yaml --image "$KIND_NODE_IMAGE" --wait 120s
./hack/images.sh
GITOPS=${GITOPS:-1} ./hack/verify.sh "${1:-portal/data/snapshot.sample.json}"
