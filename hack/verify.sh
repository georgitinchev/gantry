#!/usr/bin/env bash
# Full verification run: bootstrap the platform, run e2e, write a status snapshot.
# Used by CI (nightly + live demo) and locally. Assumes an empty cluster in the current context.
# usage: hack/verify.sh [snapshot-out]
set -euo pipefail
cd "$(dirname "$0")/.."
out=${1:-snapshot.json}
results=$(mktemp)

start=$(date +%s)
./hack/up.sh
bootstrap=$(( $(date +%s) - start ))
echo "platform bootstrap: ${bootstrap}s"

status=0
RESULTS_FILE="$results" ./tests/e2e/run.sh || status=$?
./hack/snapshot.sh "$results" "$bootstrap" > "$out"
echo "snapshot written to $out"
exit $status
