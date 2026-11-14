---
title: Request an environment
description: Get an isolated, quota-bound namespace for your team.
---

An `AppEnvironment` is a namespace your team owns, with guard rails already
in place.

## 1. Add the request

Create `requests/<team>/appenvironment.yaml`:

```yaml
apiVersion: platform.gantry.internal/v1alpha1
kind: AppEnvironment
metadata:
  name: checkout-dev        # must start with "<team>-"
spec:
  team: checkout            # must match the directory name
  size: small               # small | medium | large
  exposure: public          # internal (default) | public
  owners: [georgitinchev]   # bound to the gantry:tenant role
```

Need a second environment? Name the file `appenvironment-<suffix>.yaml`.

## 2. Open a pull request

CI reports problems as annotations on the exact line. Run the same check
locally:

```bash
cd tools/lint && go run ./cmd/gantry-lint --root ../..
```

## What you get

| | small | medium | large |
|---|---|---|---|
| CPU (requests / limits) | 1 / 2 | 2 / 4 | 4 / 8 |
| Memory (requests / limits) | 2Gi / 4Gi | 4Gi / 8Gi | 8Gi / 16Gi |
| Pods | 10 | 20 | 40 |
| PVCs / storage | 2 / 10Gi | 4 / 25Gi | 8 / 50Gi |
| Budget points | 1 | 2 | 4 |

Every environment also gets:

- Pod Security **restricted** enforced
- default container requests and limits (LimitRange)
- **default-deny** networking with explicit allows for same-namespace
  traffic, DNS, monitoring, and — when `public` — the gateway
- image policy: pinned tags or digests from trusted registries only

`large` requires an approver. Add
`platform.gantry.internal/approved-by: <name>` to `metadata.annotations`.

## Budget

Each team may use **6 points** in total across all its requests. The linter
fails the PR if a change goes over budget and warns when a team is at it.
