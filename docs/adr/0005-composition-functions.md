---
title: "ADR 0005: Stay on function-go-templating in the cluster"
description: Why there is no custom composition function running in the platform yet.
---

**Status:** accepted · 2026-11-14

## Context

Crossplane installs composition functions as OCI packages pulled by its own
package manager. A custom function therefore needs a registry Crossplane can
pull from over TLS. kind has no such registry, and pushing to GHCR from CI
would make e2e on pull requests from forks fail, because they get no write token.

## Decision

Compositions use `function-go-templating` and `function-auto-ready` from the
Crossplane contrib registry. Logic that would otherwise justify a custom
function lives in better-suited layers:

- per-environment VM limits → `gantry-lint` (`vm-limit`), at PR time
- account quota → gantry-cloud, surfaced by the provider at runtime
- custom reconcile logic → provider-gantry-cloud, which is real Go code

## Consequences

- Every PR, including PRs from forks, runs the full e2e with no secrets.
- If a custom function becomes necessary, publish it to GHCR from `main` only,
  and pin it by digest in the composition.
