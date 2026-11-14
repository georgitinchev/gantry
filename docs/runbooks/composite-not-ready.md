---
title: GantryCompositeNotReady
description: A requested environment or database has not become Ready.
---

**Severity:** warning · **For:** 2 minutes ·
**Expression:** `gantry_composite_condition{type="Ready"} == 0`

## Meaning

Crossplane reports a composite resource (`AppEnvironment`, `PostgresInstance`)
as not Ready. At least one resource it composes is missing or unhealthy, so the
team's request isn't fulfilled.

## Impact

The team can't use what it asked for. For a database, the application can't
connect. Nothing already running is affected.

## Diagnose

```bash
# what does Crossplane see?
crossplane resource trace postgresinstance/<name> -n <namespace>

# conditions and events on the composite
kubectl -n <namespace> describe postgresinstance <name>

# for databases: is CloudNativePG stuck creating pods?
kubectl -n <namespace> get cluster.postgresql.cnpg.io,pods,events --sort-by=.lastTimestamp
```

Common causes, in order of likelihood:

1. **Quota exhausted.** Look for `exceeded quota` in the namespace events.
   Also check [GantryTenantQuotaNearLimit](/docs/runbooks/tenant-quota-near-limit).
2. **Admission rejected a composed pod.** Look for Kyverno or Pod Security
   messages in events.
3. **Crossplane can't compose a kind.** `crossplane resource trace` shows a
   `forbidden` error, which means a missing rule in `platform/rbac/`.
4. **Operator unhealthy.** `kubectl -n cnpg-system get pods`.

## Mitigate

- Quota: free capacity, or ask for a larger environment size in a PR.
- Admission: fix the composition so composed pods satisfy policy, then let
  CI prove it.
- RBAC: add the missing verb to `platform/rbac/crossplane-compose.yaml`.

## Drill

Every night `search-dev` is filled to its pod quota, then a
`PostgresInstance` is requested inside it. The alert must fire. Afterwards the
filler is removed and the database must become Ready without intervention.
