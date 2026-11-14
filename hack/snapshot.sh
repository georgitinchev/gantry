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

jq -n \
  --argjson xrs "$xrs" \
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
  '{
    generatedAt: $generatedAt, commit: $commit, runUrl: (if $runUrl == "" then null else $runUrl end),
    bootstrapSeconds: $bootstrap,
    versions: {kubernetes: $k8s, crossplane: $crossplane, cloudnativepg: $cnpg, kyverno: $kyverno},
    checks: $checks,
    drills: $drills,
    resources: [$xrs[] | . + {composed: ($composed["\(.kind)/\(.namespace // "")/\(.name)"] // [])}]
  }'
