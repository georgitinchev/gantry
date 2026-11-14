---
title: Runbooks
description: One page per alert. Every alert links here through its runbook_url, and every alert is drilled nightly.
---

| Alert | Fires when | Drilled by |
|---|---|---|
| [GantryCompositeNotReady](/docs/runbooks/composite-not-ready) | a requested resource is not Ready for 2 min | requesting a database in an environment with no pod quota left |
| [GantryTenantQuotaNearLimit](/docs/runbooks/tenant-quota-near-limit) | a tenant uses more than 90% of a quota | filling an environment to its pod quota |
| [GantryTenantPodCrashLooping](/docs/runbooks/tenant-pod-crashlooping) | a tenant container is in CrashLoopBackOff for 1 min | a container that exits on start |

## How the drills work

The nightly run builds the platform, applies every request, then injects
these faults on purpose and measures how long each alert takes to fire. It
also deletes a composed NetworkPolicy to time Crossplane's drift correction,
and frees the quota again to time recovery. Results appear on the
[status page](/status). A runbook whose alert does not fire breaks the nightly
build, so the runbooks cannot quietly rot.

Alert rules have offline unit tests too (`promtool test rules`), run on every
pull request.
