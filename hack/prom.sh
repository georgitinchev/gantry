#!/usr/bin/env bash
# Queries Prometheus through the API server proxy. usage: prom.sh query '<promql>' | prom.sh alerts
set -euo pipefail
base=/api/v1/namespaces/monitoring/services/kps-prometheus:http-web/proxy/api/v1
case "$1" in
  query)  kubectl get --raw "$base/query?query=$(jq -rn --arg q "$2" '$q|@uri')" ;;
  alerts) kubectl get --raw "$base/alerts" ;;
  rules)  kubectl get --raw "$base/rules" ;;
  *) echo "usage: $0 query <promql> | alerts | rules" >&2; exit 2 ;;
esac
