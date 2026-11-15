#!/usr/bin/env bash
# End-to-end: apply every request, wait for Ready, then prove tenant isolation.
set -euo pipefail
cd "$(dirname "$0")/../.."

source tests/e2e/lib.sh

if kubectl get namespace argocd >/dev/null 2>&1; then
  echo "--- requests are synced from Git by Argo CD"
  teams=$(find requests -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort)
  for t in $teams; do
    until kubectl -n argocd get application "tenant-$t" >/dev/null 2>&1; do sleep 3; done
  done
  kubectl -n argocd wait application -l platform.gantry.internal/team --for=jsonpath='{.status.health.status}'=Healthy --timeout=900s
  kubectl -n argocd wait application --all --for=jsonpath='{.status.sync.status}'=Synced --timeout=120s
  [[ $(kubectl -n argocd get application -l platform.gantry.internal/team -o name | wc -l) -eq $(wc -w <<<"$teams") ]] \
    && pass "Argo CD: one Application per team, all Synced and Healthy" || fail "tenant Applications do not match requests/"
else
  echo "--- apply requests (environments first, then services)"
  find requests -name 'appenvironment*.yaml' -exec kubectl apply -f {} \;
  kubectl wait appenvironment --all --for=condition=Ready --timeout=180s
  find requests -name '*.yaml' ! -name 'appenvironment*.yaml' -exec kubectl apply -f {} \;
fi
kubectl wait postgresinstance -A --all --for=condition=Ready --timeout=480s
kubectl wait virtualmachine -A --all --for=condition=Ready --timeout=240s
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

echo "--- cloud networking (gantry-cloud via provider-gantry-cloud)"
cidrs=$(kubectl get appenvironment -o jsonpath='{range .items[*]}{.status.network.cidr}{"\n"}{end}')
[[ $(sort <<<"$cidrs" | uniq | grep -c /) -eq $(kubectl get appenvironment -o name | wc -l) ]] \
  && pass "every environment has its own cloud network" || fail "environments share or lack networks: $cidrs"
ip=$(kubectl -n checkout-dev get virtualmachine worker -o jsonpath='{.status.ip}')
cidr=$(kubectl get appenvironment checkout-dev -o jsonpath='{.status.network.cidr}')
python3 -c "import ipaddress,sys; sys.exit(0 if ipaddress.ip_address('$ip') in ipaddress.ip_network('$cidr') else 1)" \
  && pass "VM ip $ip is inside its environment's network $cidr" || fail "VM ip $ip not in $cidr"

echo "--- network isolation"
host=orders-rw.checkout-dev.svc
# pg_isready prints "accepting connections" or, when packets are dropped, "no response".
expect_reachable checkout-dev "$host" \
  && pass "same-namespace -> postgres allowed" || fail "same-namespace blocked"
probe search-dev "$host" | grep -q "no response" \
  && pass "cross-tenant -> postgres dropped by NetworkPolicy" || fail "cross-tenant traffic not dropped"

echo "--- e2e ok"
