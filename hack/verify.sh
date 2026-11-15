#!/usr/bin/env bash
# Full verification run: bootstrap the platform, run e2e, write a status snapshot.
# Used by CI (nightly + live demo) and locally. Assumes an empty cluster in the current context.
# usage: hack/verify.sh [snapshot-out]
# env: GITOPS=1 hands APIs, policies and requests to Argo CD.
#      DRILLS=1 also installs observability and runs the failure drills (nightly).
set -euo pipefail
cd "$(dirname "$0")/.."
out=${1:-snapshot.json}
results=$(mktemp)

start=$(date +%s)
./hack/up.sh
[[ "${GITOPS:-0}" == 1 ]] && ./hack/gitops.sh
bootstrap=$(( $(date +%s) - start ))
echo "platform bootstrap: ${bootstrap}s"

status=0
RESULTS_FILE="$results" ./tests/e2e/run.sh || status=$?

drills=$(mktemp)
if [[ "${DRILLS:-0}" == 1 && $status == 0 ]]; then
  ./hack/observability.sh
  ./hack/drills.sh "$drills" || status=$?
fi
./hack/snapshot.sh "$results" "$bootstrap" "$drills" > "$out"
echo "snapshot written to $out"
exit $status
