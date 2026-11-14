#!/usr/bin/env bash
# Runtime checks for the demo environment; prints {"checks":[...]} and exits non-zero on failure.
set -uo pipefail
cd "$(dirname "$0")/../.."
env=$1 postgres=$2
export RESULTS_FILE; RESULTS_FILE=$(mktemp)
source tests/e2e/lib.sh
fail() { printf ' FAIL %s\n' "$*" >&2; record fail "$*"; }   # keep going; report everything
pass() { printf ' PASS %s\n' "$*" >&2; record pass "$*"; }

[[ $(kubectl get appenvironment "$env" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}') == True ]] \
  && pass "environment Ready" || fail "environment not Ready"
[[ $(kubectl get ns "$env" -o jsonpath='{.metadata.labels.pod-security\.kubernetes\.io/enforce}') == restricted ]] \
  && pass "Pod Security 'restricted' enforced" || fail "Pod Security not enforced"
kubectl -n "$env" get networkpolicy default-deny >/dev/null 2>&1 \
  && pass "default-deny NetworkPolicy in place" || fail "default-deny missing"
admit_pod "$env" ghcr.io/cloudnative-pg/postgresql:latest | grep -q tenant-image-hygiene \
  && pass "Kyverno rejects :latest images" || fail ":latest image admitted"

if [[ "$postgres" == true ]]; then
  [[ $(kubectl -n "$env" get postgresinstance db -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}') == True ]] \
    && pass "PostgreSQL Ready" || fail "PostgreSQL not Ready"
  expect_reachable "$env" "db-rw.$env.svc" \
    && pass "same-namespace -> postgres allowed" || fail "same-namespace blocked"
  kubectl create namespace demo-outsider --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  probe demo-outsider "db-rw.$env.svc" | grep -q "no response" \
    && pass "other namespace -> postgres dropped" || fail "cross-namespace traffic allowed"
fi

jq -cs '{checks: .}' "$RESULTS_FILE"
! grep -q '"fail"' "$RESULTS_FILE"
