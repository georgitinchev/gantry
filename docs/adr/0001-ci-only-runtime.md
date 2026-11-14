---
title: "ADR 0001: CI-only runtime"
description: Why gantry has no long-running cluster.
---

**Status:** accepted · 2026-11-14

## Context

gantry has to be free to run indefinitely and demonstrable at any time.
There is no homelab hardware, and free managed Kubernetes tiers either need
a credit card, reclaim idle machines, or are too small for Crossplane plus an
operator plus Kyverno.

## Decision

The platform only ever runs on ephemeral [kind](https://kind.sigs.k8s.io)
clusters inside GitHub-hosted runners, which are free for public repositories.
A cluster is created for each pull request, for the nightly run and for each
live demo, then destroyed.

Persistent state is limited to Git (requests, policies, compositions) and JSON
snapshots that the nightly run publishes for the portal.

## Consequences

- Every change is proven on a real cluster before merge; there is no drift
  between "what is in Git" and "what runs", because nothing keeps running.
- There is no live UI to click around in at 3 a.m. The portal shows the last
  snapshot and offers a rate-limited live demo instead.
- The full bootstrap must stay fast (target: under 5 minutes) and lean
  (Kyverno runs admission-only; no HA replicas).
- GitHub's Actions terms limit usage to building and testing the repository's
  own software. The live demo only reconciles catalog requests against this
  repository's platform and is rate-limited per user.
