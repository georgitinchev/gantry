---
title: "ADR 0003: Validate in layers"
description: Lint, policy tests, admission, and runtime assertions each catch something different.
---

**Status:** accepted · 2026-11-14

## Context

A bad request can fail in many places: a typo, a missing namespace, a team
asking for more than its share, an insecure pod, or a policy that silently
stops matching. No single tool catches all of them, and the cheapest place to
catch an error is the pull request.

## Decision

| Layer | Tool | Runs | Catches |
|---|---|---|---|
| 1 | `gantry-lint` (Go) | every PR, seconds | schema, unknown fields, layout, ownership, uniqueness, team budget |
| 2 | `kyverno test` | every PR, seconds | policy regressions, with pass *and* fail fixtures |
| 3 | Kyverno admission + Pod Security | in-cluster | anything that reaches the API server anyway |
| 4 | e2e assertions | every PR, minutes | isolation and admission actually enforced at runtime |

`gantry-lint` reads the XRDs directly, so a new platform API is validated the
moment its definition is merged. It also flags fields that Kubernetes would
silently prune (`exposur: public`), which schema validation alone misses.

## Consequences

- Error messages point at the file and line, as GitHub annotations on the PR.
- Some rules exist in two layers on purpose (e.g. size enums in the XRD and
  the linter) — the API server is the source of truth, the linter is the fast
  feedback loop.
