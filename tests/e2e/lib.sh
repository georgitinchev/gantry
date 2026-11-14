# Shared helpers for e2e checks and the live demo. Source, do not execute.

# RESULTS_FILE (optional) collects one JSON object per check for the status snapshot.
record() { [[ -n "${RESULTS_FILE:-}" ]] && printf '{"name":"%s","status":"%s"}\n' "$2" "$1" >> "$RESULTS_FILE"; return 0; }
pass() { printf '\033[1;32m PASS\033[0m %s\n' "$*"; record pass "$*"; }
fail() { printf '\033[1;31m FAIL\033[0m %s\n' "$*"; record fail "$*"; exit 1; }

# A throwaway pod that satisfies Pod Security "restricted" and probes Postgres.
probe() { # probe <namespace> <host>
  kubectl -n "$1" run "probe-$RANDOM" --rm -i --restart=Never --quiet \
    --image=ghcr.io/cloudnative-pg/postgresql:17 \
    --overrides='{"spec":{"securityContext":{"runAsNonRoot":true,"runAsUser":26,"seccompProfile":{"type":"RuntimeDefault"}},"containers":[{"name":"p","image":"ghcr.io/cloudnative-pg/postgresql:17","command":["pg_isready","-t","5","-h","'"$2"'"],"securityContext":{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}}]}}' \
    2>&1 || true
}

# Server-side dry run of a pod that passes Pod Security, so only Kyverno can reject it.
admit_pod() { # admit_pod <namespace> <image>
  kubectl -n "$1" run "kyv-$RANDOM" --restart=Never --dry-run=server -o name --image="$2" \
    --overrides='{"spec":{"securityContext":{"runAsNonRoot":true,"runAsUser":1000,"seccompProfile":{"type":"RuntimeDefault"}},"containers":[{"name":"c","image":"'"$2"'","securityContext":{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}}]}}' 2>&1 || true
}
dry_apply() { kubectl apply --dry-run=server -f - 2>&1 || true; }

