---
title: GitOps
description: How requests travel from Git to the cluster through Argo CD.
---

Requests are never applied by hand. Argo CD pulls them from Git and keeps the
cluster equal to it. See [ADR 0006](/docs/adr/0006-gitops-in-the-throwaway-cluster)
for why the Git server lives inside the cluster.

```mermaid
flowchart LR
  ci["CI run"] -->|"git push (commit under test)"| git[("in-cluster Git server")]
  git --> argo["Argo CD"]
  subgraph apps ["App of apps: gantry"]
    direction TB
    pa["platform-apis<br/>XRDs + Compositions"]
    po["policies<br/>Kyverno"]
    ts["tenants ApplicationSet"]
    ts --> t1["tenant-checkout"]
    ts --> t2["tenant-search"]
  end
  argo --> apps
  t1 & t2 --> xp["Crossplane"] --> k8s["namespaces, databases, VMs"]
```

## Applications

| Application | Source | Notes |
|---|---|---|
| `gantry` | `gitops/apps` | app of apps |
| `platform-apis` | `platform/apis` | sync wave 0 |
| `policies` | `policy/kyverno` | sync wave 0, test fixtures excluded |
| `tenant-<team>` | `requests/<team>` | generated per directory by the `tenants` ApplicationSet |

Every Application syncs automatically, prunes, and self-heals. Tenant
Applications retry with backoff, because a database can only be created once its
environment's namespace exists.

## Health

Argo CD doesn't know gantry's resource kinds, so `gitops/argocd-values.yaml`
adds a health check: an `AppEnvironment`, `PostgresInstance` or `VirtualMachine`
is Healthy once Crossplane sets `Ready=True`, and Progressing with Crossplane's
message until then. A tenant Application turning Healthy therefore means
the team's infrastructure actually works, not just that YAML was applied.

## Drills

| Drill | Expected outcome |
|---|---|
| change a request directly in the cluster | Argo CD reverts it to what Git says |
| push a commit that changes a request | Argo CD syncs it, and Crossplane updates the composed NetworkPolicy |
