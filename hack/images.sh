#!/usr/bin/env bash
# Builds gantry-cloud and provider-gantry-cloud and loads them into the kind cluster.
set -euo pipefail
cd "$(dirname "$0")/.."
tag=${IMAGE_TAG:-dev}
docker build -q -t "gantry/cloud:$tag" cloud
docker build -q -t "gantry/provider-gantry-cloud:$tag" provider
docker build -q -t "gantry/git-server:$tag" gitops/git-server
kind load docker-image --name "${KIND_CLUSTER:-gantry}" "gantry/cloud:$tag" "gantry/provider-gantry-cloud:$tag" "gantry/git-server:$tag"
