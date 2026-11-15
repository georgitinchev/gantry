---
title: "ADR 0006: Argo CD syncs from an in-cluster Git server"
description: How gantry does pull-based GitOps on clusters that only live for one CI run.
---

**Status:** accepted · 2026-11-15

## Context

Platform teams deliver through GitOps: Argo CD pulls from Git and keeps the
cluster equal to it. gantry's clusters only exist for one CI run
([ADR 0001](/docs/adr/0001-ci-only-runtime)), and CI must test the exact
commit under review, including pull requests from forks, without any
credentials.

Pointing Argo CD at github.com would test whatever is on a branch rather than
the commit under test, would need a fork-aware revision for PRs, and could not
see local uncommitted changes.

## Decision

- Each run starts a minimal Git server inside the cluster (`gitops/git-server`,
  Alpine plus `git daemon`, about 10 MB).
- The run pushes the commit under test to it. Locally this is
  `git stash create`, so uncommitted changes to tracked files are included.
- Argo CD syncs from that server. A root Application (app of apps) manages:
  - `platform-apis`: the XRDs and Compositions
  - `policies`: the Kyverno admission policies
  - `tenants`: an ApplicationSet with one Application per directory in
    `requests/`
- Argo CD gets health checks for gantry's composite resources, so an
  Application is only Healthy once Crossplane reports every request Ready.
- Operators (Crossplane, CloudNativePG, Kyverno) and gantry-cloud stay in the
  imperative bootstrap (`hack/up.sh`). They are the cluster's foundation, and
  installing them through Argo CD would only add ordering waits to every run.

## Consequences

- Merging is deploying: the e2e suite proves a change syncs, reconciles and
  passes every check before it reaches `main`.
- Two layers correct drift: Crossplane restores composed resources, and
  Argo CD restores the requests themselves. The nightly drills exercise both.
- There is no long-lived Argo CD UI. Application sync and health states are
  published on the status page instead.
- The live demo still applies its one request directly. Syncing every team's
  requests would make it slower without showing anything new.
