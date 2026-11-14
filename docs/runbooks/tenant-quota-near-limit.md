---
title: GantryTenantQuotaNearLimit
description: A team environment is about to run out of quota.
---

**Severity:** warning · **For:** 1 minute ·
**Expression:** `gantry:tenant_quota_usage:ratio > 0.9`

## Meaning

A tenant namespace is using more than 90% of one of its ResourceQuota
entries (pods, CPU, memory, PVCs or storage). The recording rule only covers
namespaces labelled with a gantry team, so platform namespaces never alert.

## Impact

Once the quota is reached, new pods are rejected at admission and new
databases stay pending (see
[GantryCompositeNotReady](/docs/runbooks/composite-not-ready)). Running
workloads keep running.

## Diagnose

```bash
kubectl -n <namespace> describe resourcequota quota
kubectl -n <namespace> top pods 2>/dev/null || kubectl -n <namespace> get pods
kubectl -n <namespace> get deploy,statefulset -o wide
```

Check for runaway replicas, leaked Jobs, or completed pods that a controller
keeps recreating.

## Mitigate

- Scale down or delete what's not needed.
- If the growth is real, raise the environment's `size` in a PR. The budget
  rule in gantry-lint shows the team's remaining points.

## Drill

Every night a Deployment of tiny `pause` pods fills `search-dev` to exactly
its pod quota. The alert must fire for `resource="pods"`.
