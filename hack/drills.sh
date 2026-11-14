#!/usr/bin/env bash
# Failure drills: inject real faults and measure how the platform responds.
# Requires the platform, the e2e requests and observability to be installed.
# usage: drills.sh [results.jsonl]
set -uo pipefail
cd "$(dirname "$0")/.."
out=${1:-/dev/stdout}
: > "${out/#\/dev\/stdout/\/dev\/null}"
log() { printf '\033[1;34m--\033[0m %s\n' "$*" >&2; }
emit() { jq -cn --arg name "$1" --arg alert "$2" --arg status "$3" --argjson seconds "$4" --arg note "$5" \
  '{name: $name, alert: (if $alert == "" then null else $alert end), status: $status, seconds: $seconds, note: $note}' >> "$out"; }

firing() { # firing <alertname> <jq label filter>
  ./hack/prom.sh alerts 2>/dev/null | jq -e --arg a "$1" "[.data.alerts[] | select(.labels.alertname == \$a and .state == \"firing\") | select($2)] | length > 0" >/dev/null
}
wait_since() { # wait_since <start-SECONDS> <timeout> <command...> ; prints seconds since start
  local start=$1 deadline=$(($1 + $2)); shift 2
  until "$@" >/dev/null 2>&1; do (( SECONDS >= deadline )) && { echo -1; return 1; }; sleep 5; done
  echo $((SECONDS - start))
}
wait_for() { wait_since "$SECONDS" "$@"; }

restricted='{"runAsNonRoot":true,"runAsUser":65532,"seccompProfile":{"type":"RuntimeDefault"}}'
container_sc='{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}'
tiny='{"requests":{"cpu":"10m","memory":"16Mi"},"limits":{"cpu":"10m","memory":"16Mi"}}'

# 1. Drift: delete a composed NetworkPolicy and time Crossplane putting it back.
log "drill: drift correction"
kubectl -n checkout-dev delete networkpolicy default-deny --wait=true >/dev/null
s=$(wait_for 120 kubectl -n checkout-dev get networkpolicy default-deny -o name) \
  && emit "Deleted default-deny NetworkPolicy is restored" "" pass "$s" "Crossplane re-created it" \
  || emit "Deleted default-deny NetworkPolicy is restored" "" fail -1 "not restored within 120s"

# 2-4. Start the remaining faults together, then wait for each alert.
log "fault: crash-looping container in checkout-dev"
t_crash=$SECONDS
kubectl -n checkout-dev apply -f - >/dev/null <<YAML
apiVersion: v1
kind: Pod
metadata: {name: drill-crash, labels: {gantry.drill: "true"}}
spec:
  securityContext: $restricted
  containers:
    - name: app
      image: ghcr.io/cloudnative-pg/postgresql:17
      command: ["sh", "-c", "echo simulated failure; exit 1"]
      securityContext: $container_sc
      resources: $tiny
YAML

hard=$(kubectl -n search-dev get resourcequota quota -o jsonpath='{.spec.hard.pods}')
log "fault: fill search-dev to its pod quota ($hard pods)"
t_quota=$SECONDS
kubectl -n search-dev apply -f - >/dev/null <<YAML
apiVersion: apps/v1
kind: Deployment
metadata: {name: drill-filler, labels: {gantry.drill: "true"}}
spec:
  replicas: $hard
  selector: {matchLabels: {app: drill-filler}}
  template:
    metadata: {labels: {app: drill-filler}}
    spec:
      securityContext: $restricted
      containers:
        - name: pause
          image: registry.k8s.io/pause:3.10
          securityContext: $container_sc
          resources: $tiny
YAML

log "fault: request a database in the full environment"
t_blocked=$SECONDS
kubectl apply -f - >/dev/null <<YAML
apiVersion: platform.gantry.internal/v1alpha1
kind: PostgresInstance
metadata: {name: drill-blocked, namespace: search-dev}
spec: {size: small}
YAML

s=$(wait_since $t_crash 420 firing GantryTenantPodCrashLooping '.labels.pod == "drill-crash"') \
  && emit "Crash-looping container raises an alert" GantryTenantPodCrashLooping pass "$s" "firing" \
  || emit "Crash-looping container raises an alert" GantryTenantPodCrashLooping fail -1 "not firing within 420s"

s=$(wait_since $t_quota 420 firing GantryTenantQuotaNearLimit '.labels.namespace == "search-dev" and .labels.resource == "pods"') \
  && emit "Exhausted pod quota raises an alert" GantryTenantQuotaNearLimit pass "$s" "firing" \
  || emit "Exhausted pod quota raises an alert" GantryTenantQuotaNearLimit fail -1 "not firing within 420s"

s=$(wait_since $t_blocked 480 firing GantryCompositeNotReady '.labels.name == "drill-blocked"') \
  && emit "Database blocked by quota raises an alert" GantryCompositeNotReady pass "$s" "firing" \
  || emit "Database blocked by quota raises an alert" GantryCompositeNotReady fail -1 "not firing within 420s"

# Recovery: free the quota and time the blocked database becoming Ready.
log "recovery: remove the filler and wait for the blocked database"
kubectl -n search-dev delete deployment drill-filler --wait=false >/dev/null
s=$(wait_for 420 sh -c '[ "$(kubectl -n search-dev get postgresinstance drill-blocked -o jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}")" = True ]') \
  && emit "Blocked database recovers once quota frees up" "" pass "$s" "became Ready without intervention" \
  || emit "Blocked database recovers once quota frees up" "" fail -1 "not Ready within 420s"

log "cleanup"
kubectl -n checkout-dev delete pod drill-crash --wait=false >/dev/null 2>&1
kubectl -n search-dev delete postgresinstance drill-blocked --wait=false >/dev/null 2>&1
[[ "$out" == /dev/stdout ]] || { cat "$out"; ! grep -q '"fail"' "$out"; }
