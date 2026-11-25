#!/usr/bin/env bash
# Dumps what you need to debug a failed reconcile.
set -uo pipefail
section() { printf '\n===== %s =====\n' "$*"; }
section "composite resources";  kubectl get appenvironment,postgresinstance -A -o wide
for xr in $(kubectl get appenvironment -o name); do
  section "trace $xr"; crossplane resource trace "$xr" 2>/dev/null || kubectl describe "$xr"
done
section "postgres instances";    kubectl describe postgresinstance -A
section "cnpg clusters";         kubectl get cluster.postgresql.cnpg.io -A -o wide
section "functions";             kubectl get functions -o wide
section "non-running pods";      kubectl get pods -A --field-selector=status.phase!=Running
section "warning events";        kubectl get events -A --field-selector type=Warning --sort-by=.lastTimestamp | tail -50
section "crossplane logs";       kubectl -n crossplane-system logs deploy/crossplane --tail=100
section "argo cd pods";          kubectl -n argocd describe pods
section "argo cd applications";  kubectl -n argocd get applications -o wide 2>/dev/null
for a in $(kubectl -n argocd get applications -o name 2>/dev/null); do
  kubectl -n argocd get "$a" -o jsonpath='{.metadata.name}: {.status.sync.status}/{.status.health.status} {.status.conditions[*].message}{"\n"}' 2>/dev/null
done
section "kyverno logs";          kubectl -n kyverno logs deploy/kyverno-admission-controller --tail=50
