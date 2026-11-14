---
title: "ADR 0004: A small mock cloud instead of a real one"
description: Why VMs, networks and buckets come from a cloud API that gantry ships itself.
---

**Status:** accepted · 2026-11-14

## Context

Real platform teams wrap a cloud provider's API. Using a real provider here
would cost money, need credentials in CI, and tie the project to one vendor's
terms. Self-hosted object storage options have also moved towards restrictive
licences or stopped publishing community images.

## Decision

gantry ships `gantry-cloud`, a small Go service with a REST API for networks,
subnets, IP allocations, security groups, instances and buckets. A custom
Crossplane provider (`provider-gantry-cloud`) manages those resources, and
platform APIs compose them like they would compose a real cloud's.

## Consequences

- The provider, its reconcile loop and its failure modes are real code,
  tested end-to-end, with no external dependency or cost.
- Behaviour that real clouds have — eventual consistency, quota errors, IP
  exhaustion, rate limits — can be injected on purpose for failure testing.
- It is a simulation; this is stated everywhere it appears.
