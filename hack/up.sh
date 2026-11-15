#!/usr/bin/env bash
# Installs the gantry platform onto the current kube-context.
# GITOPS=1 leaves platform APIs and policies to Argo CD (see hack/gitops.sh).
set -euo pipefail
cd "$(dirname "$0")/.."
source hack/versions.env

log() { printf '\033[1;32m==>\033[0m %s\n' "$*"; }

log "helm repos"
helm repo add --force-update crossplane-stable https://charts.crossplane.io/stable >/dev/null
helm repo add --force-update cnpg https://cloudnative-pg.github.io/charts >/dev/null
helm repo add --force-update kyverno https://kyverno.github.io/kyverno >/dev/null
helm repo update crossplane-stable cnpg kyverno >/dev/null

log "platform namespaces"
kubectl apply -f platform/namespaces.yaml

log "crossplane ${CROSSPLANE_CHART}"
helm upgrade --install crossplane crossplane-stable/crossplane \
  --version "${CROSSPLANE_CHART}" -n crossplane-system --wait

log "cloudnative-pg ${CNPG_CHART}"
helm upgrade --install cnpg cnpg/cloudnative-pg \
  --version "${CNPG_CHART}" -n cnpg-system --wait

log "kyverno ${KYVERNO_CHART} (admission only)"
helm upgrade --install kyverno kyverno/kyverno \
  --version "${KYVERNO_CHART}" -n kyverno --create-namespace --wait --timeout 5m \
  --set admissionController.replicas=1 \
  --set backgroundController.enabled=false \
  --set cleanupController.enabled=false \
  --set reportsController.enabled=false

log "gantry-cloud + provider"
kubectl apply -f provider/package/crds/
kubectl apply -f platform/cloud/cloud.yaml
# The API token is generated per cluster and never committed.
kubectl -n gantry-system get secret gantry-cloud-token >/dev/null 2>&1 || \
  kubectl -n gantry-system create secret generic gantry-cloud-token --from-literal=token="$(openssl rand -hex 24)"
kubectl -n gantry-system rollout status deploy/gantry-cloud --timeout=120s
kubectl -n gantry-system rollout status deploy/provider-gantry-cloud --timeout=120s

log "rbac + functions"
kubectl apply -f platform/rbac/
kubectl apply -f platform/crossplane/functions.yaml
kubectl wait function --all --for=condition=Healthy --timeout=300s

if [[ "${GITOPS:-0}" != 1 ]]; then
  log "platform APIs"
  kubectl apply -R -f platform/apis/
  kubectl wait xrd --all --for=condition=Established --timeout=120s

  log "admission policies"
  kubectl apply -f policy/kyverno/
  kubectl wait validatingpolicies.policies.kyverno.io --all --for=jsonpath='{.status.conditionStatus.ready}'=true --timeout=60s
fi

log "platform ready"
