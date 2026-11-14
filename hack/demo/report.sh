#!/usr/bin/env bash
# Posts a signed progress event to the portal.
# usage: report.sh <phase> <message> [extra-json]
# env: PORTAL_URL, DEMO_ID, DEMO_CALLBACK_SECRET (all optional locally: prints instead)
set -euo pipefail
phase=$1 message=$2 extra=${3:-'{}'}
body=$(jq -cn --arg phase "$phase" --arg message "$message" --argjson extra "$extra" '{phase: $phase, message: $message} + $extra')
echo "[$phase] $message"
if [[ -z "${PORTAL_URL:-}" || -z "${DEMO_CALLBACK_SECRET:-}" ]]; then
  echo "  (no PORTAL_URL/secret; not sending) $body"
  exit 0
fi
ts=$(date +%s)
sig="sha256=$(printf '%s.%s' "$ts" "$body" | openssl dgst -sha256 -hmac "$DEMO_CALLBACK_SECRET" -r | cut -d' ' -f1)"
curl -fsS --max-time 10 --retry 3 -X POST "$PORTAL_URL/api/demo/$DEMO_ID/events" \
  -H 'content-type: application/json' -H "x-gantry-timestamp: $ts" -H "x-gantry-signature: $sig" \
  --data "$body" >/dev/null || echo "  (report failed; continuing)"
