---
title: GantryTenantPodCrashLooping
description: A container in a team environment keeps crashing.
---

**Severity:** warning · **For:** 1 minute ·
**Expression:** `kube_pod_container_status_waiting_reason{reason="CrashLoopBackOff"} == 1` (tenant namespaces only)

## Meaning

A container in a tenant namespace keeps exiting, and Kubernetes is backing off
between restarts.

## Impact

Whatever the pod serves is degraded or down. It also burns quota and can hide
other problems in the logs.

## Diagnose

```bash
kubectl -n <namespace> describe pod <pod>
kubectl -n <namespace> logs <pod> -c <container> --previous
kubectl -n <namespace> get events --field-selector involvedObject.name=<pod>
```

Look at the exit code in `describe`:
- `1` means an application error
- `137` means OOMKilled; compare against the LimitRange defaults
- `126`/`127` means a bad command or entrypoint

Remember that tenant namespaces are default-deny. A crash on startup is often
an outbound connection that the NetworkPolicies don't allow.

## Mitigate

Roll back the last change to the workload, or fix configuration and redeploy.
If it's a network dependency, request the access through the platform rather
than editing NetworkPolicies by hand. Crossplane reverts manual edits (see the
drift drill).

## Drill

Every night a pod that exits immediately is started in `checkout-dev`. The
alert must fire for that pod.
