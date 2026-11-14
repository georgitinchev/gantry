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

# Server-side dry run of a pod that passes Pod Security, so only Kyverno can reject it.
admit_pod() { # admit_pod <namespace> <image>
  kubectl -n "$1" run "kyv-$RANDOM" --restart=Never --dry-run=server -o name --image="$2" \
    --overrides='{"spec":{"securityContext":{"runAsNonRoot":true,"runAsUser":1000,"seccompProfile":{"type":"RuntimeDefault"}},"containers":[{"name":"c","image":"'"$2"'","securityContext":{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}}]}}' 2>&1 || true
}
dry_apply() { kubectl apply --dry-run=server -f - 2>&1 || true; }

echo "--- admission policy (kyverno)"
admit_pod checkout-dev ghcr.io/cloudnative-pg/postgresql:17 | grep -q "pod/" \
  && pass "pinned image from trusted registry admitted" || fail "compliant pod rejected"
admit_pod checkout-dev ghcr.io/cloudnative-pg/postgresql:latest | grep -q "tenant-image-hygiene" \
  && pass ":latest image rejected" || fail ":latest image admitted"
admit_pod checkout-dev evil.example.com/miner:1.0 | grep -q "tenant-image-hygiene" \
  && pass "untrusted registry rejected" || fail "untrusted registry admitted"
printf 'apiVersion: platform.gantry.internal/v1alpha1\nkind: AppEnvironment\nmetadata: {name: search-big}\nspec: {team: search, size: large}\n' \
  | dry_apply | grep -q "require-approval-for-large" \
  && pass "unapproved large environment rejected" || fail "unapproved large environment admitted"

echo "--- network isolation"
host=orders-rw.checkout-dev.svc
# pg_isready prints "accepting connections" or, when packets are dropped, "no response".
probe checkout-dev "$host" | grep -q "accepting connections" \
  && pass "same-namespace -> postgres allowed" || fail "same-namespace blocked"
probe search-dev "$host" | grep -q "no response" \
  && pass "cross-tenant -> postgres dropped by NetworkPolicy" || fail "cross-tenant traffic not dropped"

echo "--- e2e ok"
