#!/usr/bin/env bash
# End-to-end: apply every request, wait for Ready, then prove tenant isolation.
set -euo pipefail
cd "$(dirname "$0")/../.."

pass() { printf '\033[1;32m PASS\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m FAIL\033[0m %s\n' "$*"; exit 1; }

# A throwaway pod that satisfies Pod Security "restricted" and probes Postgres.
probe() { # probe <namespace> <host>
  kubectl -n "$1" run "probe-$RANDOM" --rm -i --restart=Never --quiet \
    --image=ghcr.io/cloudnative-pg/postgresql:17 \
    --overrides='{"spec":{"securityContext":{"runAsNonRoot":true,"runAsUser":26,"seccompProfile":{"type":"RuntimeDefault"}},"containers":[{"name":"p","image":"ghcr.io/cloudnative-pg/postgresql:17","command":["pg_isready","-t","5","-h","'"$2"'"],"securityContext":{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}}]}}' \
    2>&1 || true
}

echo "--- apply requests (environments first, then services)"
find requests -name 'appenvironment.yaml' -exec kubectl apply -f {} \;
kubectl wait appenvironment --all --for=condition=Ready --timeout=180s
find requests -name '*.yaml' ! -name 'appenvironment.yaml' -exec kubectl apply -f {} \;
kubectl wait postgresinstance -A --all --for=condition=Ready --timeout=480s
pass "all requests Ready"

echo "--- tenant guarantees"
[[ $(kubectl get ns checkout-dev -o jsonpath='{.metadata.labels.pod-security\.kubernetes\.io/enforce}') == restricted ]] \
  && pass "pod security 'restricted' enforced" || fail "pod security label missing"

kubectl -n checkout-dev get resourcequota quota >/dev/null && pass "quota present" || fail "quota missing"

if kubectl -n checkout-dev run root-test --image=busybox:1.37 --restart=Never --dry-run=server -- id >/dev/null 2>&1; then
  fail "privileged/root pod admitted"
else
  pass "non-compliant pod rejected by admission"
fi

echo "--- network isolation"
host=orders-rw.checkout-dev.svc
# pg_isready prints "accepting connections" or, when packets are dropped, "no response".
probe checkout-dev "$host" | grep -q "accepting connections" \
  && pass "same-namespace -> postgres allowed" || fail "same-namespace blocked"
probe search-dev "$host" | grep -q "no response" \
  && pass "cross-tenant -> postgres dropped by NetworkPolicy" || fail "cross-tenant traffic not dropped"

echo "--- e2e ok"
