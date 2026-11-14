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

# Expect a connection to succeed. Postgres can report Ready a moment before its
# service accepts connections, so the allowed path gets a few attempts; the
# denied path never retries (a retry could only hide a leak).
expect_reachable() { # expect_reachable <namespace> <host> ; prints the last probe output on failure
  local out
  for _ in 1 2 3 4 5 6; do
    out=$(probe "$1" "$2")
    grep -q "accepting connections" <<<"$out" && return 0
    sleep 5
  done
  echo "    last probe output: $(tr '\n' ' ' <<<"$out" | cut -c1-200)" >&2
  return 1
}

# Server-side dry run of a pod that passes Pod Security, so only Kyverno can reject it.
admit_pod() { # admit_pod <namespace> <image>
  kubectl -n "$1" run "kyv-$RANDOM" --restart=Never --dry-run=server -o name --image="$2" \
    --overrides='{"spec":{"securityContext":{"runAsNonRoot":true,"runAsUser":1000,"seccompProfile":{"type":"RuntimeDefault"}},"containers":[{"name":"c","image":"'"$2"'","securityContext":{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}}]}}' 2>&1 || true
}
dry_apply() { kubectl apply --dry-run=server -f - 2>&1 || true; }

