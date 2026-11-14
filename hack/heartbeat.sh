#!/usr/bin/env bash
# Signed keepalive for the portal's Redis. env: PORTAL_URL, DEMO_CALLBACK_SECRET
set -euo pipefail
body='{"source":"nightly"}'
ts=$(date +%s)
sig="sha256=$(printf '%s.%s' "$ts" "$body" | openssl dgst -sha256 -hmac "$DEMO_CALLBACK_SECRET" -r | cut -d' ' -f1)"
curl -fsS --retry 3 -X POST "$PORTAL_URL/api/heartbeat" -H 'content-type: application/json' \
  -H "x-gantry-timestamp: $ts" -H "x-gantry-signature: $sig" --data "$body"
echo
