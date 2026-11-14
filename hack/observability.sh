#!/usr/bin/env bash
# Installs the trimmed Prometheus stack and gantry's alert rules.
set -euo pipefail
cd "$(dirname "$0")/.."
source hack/versions.env
log() { printf '\033[1;32m==>\033[0m %s\n' "$*"; }

log "kube-prometheus-stack ${KPS_CHART}"
helm repo add --force-update prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null
helm repo update prometheus-community >/dev/null
helm upgrade --install kps prometheus-community/kube-prometheus-stack \
  --version "${KPS_CHART}" -n monitoring -f observability/values.yaml --wait --timeout 8m

log "alert rules"
kubectl apply -f observability/rules.yaml
kubectl -n monitoring rollout status statefulset/prometheus-kps-prometheus --timeout=180s
log "observability ready"
