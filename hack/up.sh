#!/usr/bin/env bash
# Installs the gantry platform onto the current kube-context.
set -euo pipefail
cd "$(dirname "$0")/.."
source hack/versions.env

log() { printf '\033[1;32m==>\033[0m %s\n' "$*"; }

log "helm repos"
helm repo add --force-update crossplane-stable https://charts.crossplane.io/stable >/dev/null
helm repo add --force-update cnpg https://cloudnative-pg.github.io/charts >/dev/null
helm repo update crossplane-stable cnpg >/dev/null

log "platform namespaces"
kubectl apply -f platform/namespaces.yaml

log "crossplane ${CROSSPLANE_CHART}"
helm upgrade --install crossplane crossplane-stable/crossplane \
  --version "${CROSSPLANE_CHART}" -n crossplane-system --wait

log "cloudnative-pg ${CNPG_CHART}"
helm upgrade --install cnpg cnpg/cloudnative-pg \
  --version "${CNPG_CHART}" -n cnpg-system --wait

log "rbac + functions"
kubectl apply -f platform/rbac/
kubectl apply -f platform/crossplane/functions.yaml
kubectl wait function --all --for=condition=Healthy --timeout=300s

log "platform APIs"
kubectl apply -R -f platform/apis/
kubectl wait xrd --all --for=condition=Established --timeout=120s

log "platform ready"
