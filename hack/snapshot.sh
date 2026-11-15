#!/usr/bin/env bash
# Writes a JSON snapshot of the platform for the portal's status page.
# usage: hack/snapshot.sh <e2e-results.jsonl> <bootstrap-seconds> [drills.jsonl] > snapshot.json
set -euo pipefail
cd "$(dirname "$0")/.."
results=${1:-/dev/null}
bootstrap=${2:-0}
drills=${3:-/dev/null}

xrs=$(kubectl get appenvironment,postgresinstance,virtualmachine -A -o json | jq '[.items[] | {
  kind, name: .metadata.name, namespace: (.metadata.namespace // null),
  ready:  ((.status.conditions // []) | map(select(.type=="Ready"))  | .[0].status == "True"),
  synced: ((.status.conditions // []) | map(select(.type=="Synced")) | .[0].status == "True"),
  created: .metadata.creationTimestamp,
  spec, status: (.status | del(.conditions))
}]')

composed=$(kubectl get appenvironment,postgresinstance,virtualmachine -A -o json | jq '[.items[] | {
  key: "\(.kind)/\(.metadata.namespace // "")/\(.metadata.name)",
  value: [(.spec.crossplane.resourceRefs // [])[] | {kind, name}]
}] | from_entries')

gitops='[]'
if kubectl get namespace argocd >/dev/null 2>&1; then
  gitops=$(kubectl -n argocd get applications -o json | jq '[.items[] | {name: .metadata.name, sync: .status.sync.status, health: .status.health.status, revision: ((.status.sync.revision // "")[0:7])}] | sort_by(.name)')
fi

jq -n \
  --argjson xrs "$xrs" \
  --argjson gitops "$gitops" \
  --argjson composed "$composed" \
  --slurpfile checks <(cat "$results" 2>/dev/null || true) \
  --slurpfile drills <(cat "$drills" 2>/dev/null || true) \
  --arg generatedAt "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg commit "${GITHUB_SHA:-$(git rev-parse HEAD)}" \
  --arg runUrl "${GITHUB_SERVER_URL:-}${GITHUB_RUN_ID:+/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}}" \
  --argjson bootstrap "$bootstrap" \
  --arg k8s "$(kubectl version -o json | jq -r .serverVersion.gitVersion)" \
  --arg crossplane "$(helm -n crossplane-system list -o json | jq -r '.[0].app_version')" \
  --arg cnpg "$(helm -n cnpg-system list -o json | jq -r '.[0].app_version')" \
  --arg kyverno "$(helm -n kyverno list -o json | jq -r '.[0].app_version')" \
  --arg argocd "$(helm -n argocd list -o json 2>/dev/null | jq -r '.[0].app_version // empty')" \
  '{
    generatedAt: $generatedAt, commit: $commit, runUrl: (if $runUrl == "" then null else $runUrl end),
    bootstrapSeconds: $bootstrap,
    versions: ({kubernetes: $k8s, crossplane: $crossplane, cloudnativepg: $cnpg, kyverno: $kyverno} + (if $argocd == "" then {} else {argocd: $argocd} end)),
    checks: $checks,
    drills: $drills,
    gitops: $gitops,
    resources: [$xrs[] | . + {composed: ($composed["\(.kind)/\(.namespace // "")/\(.name)"] // [])}]
  }'
