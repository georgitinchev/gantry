---
title: Introduction
description: gantry is a small, self-service internal developer platform you can read end to end.
---

gantry turns a YAML file in a pull request into a verified, isolated piece of
infrastructure. It is a deliberately small model of what platform teams
build: a **catalog** of offerings, a **platform API** behind it, a **policy gate**
in front of it, and **proof** that it works.

## What you can request

| Offering | Kind | Scope | Backed by |
|---|---|---|---|
| App environment | `AppEnvironment` | cluster | Namespace, quota, limits, NetworkPolicies, RBAC, a private cloud network |
| PostgreSQL | `PostgresInstance` | namespaced | CloudNativePG |
| Virtual machine | `VirtualMachine` | namespaced | gantry-cloud via provider-gantry-cloud |

## How it fits together

1. You add a file under `requests/<team>/`, by hand or from the portal.
2. CI lints it, unit-tests the policies, then builds the whole platform on a
   throwaway kind cluster, where Argo CD syncs every request from Git.
3. The e2e suite asserts readiness, admission rules and network isolation.
4. Green checks mean the request is proven, not just syntactically valid.

Start with [requesting an environment](/docs/guides/request-an-environment),
or read the [architecture](/docs/platform/architecture).

> gantry is a personal portfolio project. All infrastructure runs on ephemeral
> clusters in CI; nothing here is production.
