#!/usr/bin/env bash
# Extracts the PrometheusRule spec and runs promtool check + unit tests.
set -euo pipefail
cd "$(dirname "$0")/../observability"
python3 -c 'import sys,yaml,json; print(json.dumps(yaml.safe_load(open("rules.yaml"))["spec"]))' > tests/rules.extracted.yaml
PROMTOOL=${PROMTOOL:-promtool}
if ! command -v "$PROMTOOL" >/dev/null; then
  PROMTOOL="docker run --rm -v $PWD/tests:/t -w /t --entrypoint promtool quay.io/prometheus/prometheus:${PROMETHEUS_VERSION:-v3.16.0}"
  cd tests && $PROMTOOL check rules rules.extracted.yaml && $PROMTOOL test rules rules.test.yaml
else
  cd tests && $PROMTOOL check rules rules.extracted.yaml && $PROMTOOL test rules rules.test.yaml
fi
