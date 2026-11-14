#!/usr/bin/env bash
# Prints {"resources":[...]} for the demo's composite resources and everything they compose.
set -euo pipefail
env=$1
xrs=$( { kubectl get appenvironment "$env" -o json 2>/dev/null || echo '{}'; kubectl -n "$env" get postgresinstance -o json 2>/dev/null | jq '.items[]' || true; } | jq -s '[.[] | select(.kind)]')
rows=$(jq -c '.[] | {kind, name: .metadata.name, ns: (.metadata.namespace // ""), ready: ((.status.conditions // []) | any(.type=="Ready" and .status=="True")), refs: (.spec.crossplane.resourceRefs // [])}' <<<"$xrs")
out='[]'
while IFS= read -r xr; do
  [[ -z "$xr" ]] && continue
  out=$(jq -c --argjson xr "$xr" '. + [{kind: $xr.kind, name: $xr.name, ready: $xr.ready}]' <<<"$out")
  while IFS= read -r ref; do
    [[ -z "$ref" ]] && continue
    kind=$(jq -r .kind <<<"$ref"); name=$(jq -r .name <<<"$ref"); ns=$(jq -r '.namespace // ""' <<<"$ref")
    [[ "$kind" != Namespace && -z "$ns" ]] && ns=$(jq -r 'if .ns == "" then .name else .ns end' <<<"$xr")
    api=$(jq -r .apiVersion <<<"$ref"); group=${api%/*}; [[ "$group" == "$api" ]] && group=""
    res="$(tr "[:upper:]" "[:lower:]" <<<"$kind")${group:+.$group}"
    obj=$(kubectl get "$res" "$name" ${ns:+-n "$ns"} -o json 2>/dev/null || echo null)
    ready=$(jq '. != null and ((.status.conditions // []) | (length == 0 or any(.type=="Ready" and .status=="True")))' <<<"$obj")
    out=$(jq -c --arg k "$kind" --arg n "$name" --argjson r "$ready" '. + [{kind: $k, name: $n, ready: $r}]' <<<"$out")
  done < <(jq -c '.refs[]' <<<"$xr")
done <<<"$rows"
jq -c '{resources: .}' <<<"$out"
