---
title: "ADR 0002: Compose Kubernetes resources directly with Crossplane v2"
description: Why there is no provider-kubernetes in the platform core.
---

**Status:** accepted · 2026-11-14

## Context

Crossplane v1 could only compose managed resources, so creating a Namespace or
a NetworkPolicy meant wrapping it in a `provider-kubernetes` `Object`.
Crossplane v2 composes any Kubernetes resource directly and supports
namespaced composite resources.

## Decision

- `AppEnvironment` is **cluster-scoped** because it owns a Namespace.
- `PostgresInstance` is **namespaced** and lives inside the tenant's
  environment, so tenant RBAC (`gantry:tenant`) can grant it like any other
  namespaced resource.
- Compositions are pipelines: `function-go-templating` renders resources,
  `function-auto-ready` derives readiness.
- Crossplane's service account gets an explicit, aggregated ClusterRole for
  exactly the kinds it composes, including `bind` on `gantry:tenant` only.

## Consequences

- No extra provider pod, no extra hop, one less thing to version.
- Readiness for kinds without status (NetworkPolicy, Namespace) is declared in
  the template; CNPG's `Ready` condition is picked up automatically.
- Adding a new composed kind requires a matching RBAC rule. The e2e suite
  fails fast if one is missing.
