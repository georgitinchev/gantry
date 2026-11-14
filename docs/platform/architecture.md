---
title: Architecture
description: How a request travels from the portal to a reconciled, verified environment.
---

gantry is a small internal developer platform. Teams ask for infrastructure by
opening a pull request; the platform validates it in layers, proves it on a real
cluster, and reports the result back to the portal.

## System context

```mermaid
flowchart TB
  dev([Developer]) -->|1 · picks an offering| portal["Portal + docs<br/>Next.js on Vercel"]
  portal -->|2 · opens a pull request| repo[("gantry repository<br/>requests/team/*.yaml")]
  repo -->|3 · pull_request| ci
  subgraph ci ["GitHub Actions · ephemeral runner"]
    direction LR
    lint["gantry-lint"] --> kyv["kyverno test"] --> kind["kind cluster<br/>Crossplane · CNPG · Kyverno"]
  end
  ci -->|4 · nightly snapshot| status[("status branch<br/>snapshot.json")]
  status -->|5 · rendered| portal
```

There is no long-running cluster. Every pull request, every night and every
live demo builds the platform from scratch on a GitHub-hosted runner and throws it
away afterwards. See [ADR 0001](/docs/adr/0001-ci-only-runtime).

## Request lifecycle

```mermaid
sequenceDiagram
  autonumber
  actor Dev as Developer
  participant P as Portal
  participant GH as GitHub
  participant CI as CI runner
  participant XP as Crossplane
  participant K as Kubernetes

  Dev->>P: Request "PostgresInstance orders"
  P->>GH: Create branch + requests/checkout/postgres-orders.yaml + PR
  GH->>CI: pull_request event
  CI->>CI: gantry-lint (schema, layout, ownership, budget)
  CI->>CI: kyverno test (policy unit tests)
  CI->>K: kind create cluster, install platform
  CI->>K: kubectl apply request
  K->>XP: AppEnvironment / PostgresInstance created
  XP->>XP: function-go-templating renders resources
  XP->>K: Namespace, quota, NetworkPolicies, CNPG Cluster
  K-->>XP: CNPG reports Ready
  XP-->>CI: XR condition Ready=True
  CI->>K: isolation + admission assertions
  CI-->>GH: checks green
  GH-->>P: PR status
```

## Inside the cluster

```mermaid
flowchart TB
  subgraph platform ["Platform namespaces"]
    direction LR
    xp["crossplane-system"]
    cnpg["cnpg-system<br/>role: operators"]
    mon["monitoring<br/>role: monitoring"]
    gw["gateway<br/>role: gateway"]
  end
  subgraph t1 ["checkout-dev · exposure: public"]
    app1["app pods"] --> pg[("orders · CNPG")]
  end
  subgraph t2 ["search-dev · exposure: internal"]
    app2["app pods"]
  end
  xp -. composes .-> t1 & t2
  cnpg -->|":8000 status"| pg
  mon -->|scrape| t1 & t2
  gw -->|public only| t1
  app2 -- "dropped by NetworkPolicy" --x pg
```

Every tenant namespace starts with `default-deny` for ingress and egress. Allowed
paths are added explicitly: same-namespace traffic, DNS to `kube-system`,
scraping from `monitoring`, and — only for `exposure: public` — traffic from
the `gateway` namespace. The e2e suite proves the crossed-out edge is actually
dropped, not just declared.

## Resource state

```mermaid
stateDiagram-v2
  [*] --> Validating: PR opened
  Validating --> Rejected: lint / policy failure
  Validating --> Creating: merged / applied
  Creating --> Ready: all composed resources ready
  Creating --> Degraded: composed resource unhealthy
  Degraded --> Ready: self-heals or fix merged
  Ready --> Degraded: drift / failure
  Ready --> Deleting: request file removed
  Deleting --> [*]
  Rejected --> [*]
```
