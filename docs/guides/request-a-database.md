---
title: Request a database
description: A PostgreSQL instance inside your environment, managed by CloudNativePG.
---

Create `requests/<team>/postgres-<name>.yaml`. The file name must match
`metadata.name`.

```yaml
apiVersion: platform.gantry.internal/v1alpha1
kind: PostgresInstance
metadata:
  name: orders
  namespace: checkout-dev   # one of your team's AppEnvironments
spec:
  size: small               # small = 1 instance, medium = primary + replica
  version: "17"             # 16 | 17 | 18
  storageGB: 1              # 1-20
  database: app             # optional, default "app"
```

## Connecting

Once `READY` is `True`:

```bash
kubectl -n checkout-dev get pg orders
```

| Field | Value |
|---|---|
| `status.endpoint` | `orders-rw.checkout-dev.svc:5432` (read-write service) |
| `status.credentialsSecret` | `orders-app` (keys: `username`, `password`, `uri`) |

Only pods in the same namespace can reach the database. Traffic from other
tenants is dropped by NetworkPolicy, and the e2e suite checks that on every PR.
