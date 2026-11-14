---
title: Policies
description: Every rule a request or workload has to pass, and where it is enforced.
---

## gantry-lint (pull request)

| Rule | What it checks |
|---|---|
| `layout` | `requests/<team>/<file>.yaml`, lowercase kebab-case |
| `single-document` | one object per file |
| `known-api` | `apiVersion`/`kind` is a served platform API (read from the XRDs) |
| `file-name` | `<prefix>[-<suffix>].yaml`; namespaced kinds use `<prefix>-<name>.yaml` |
| `scope` | namespaced kinds set a namespace, cluster-scoped kinds do not |
| `schema` | the XRD's OpenAPI schema: enums, patterns, ranges, required fields |
| `unknown-field` | fields Kubernetes would silently prune |
| `ownership` | `spec.team` matches the directory; databases live in your own environments |
| `unique` | no two requests define the same object |
| `budget` | team total ≤ 6 points (small 1, medium 2, large 4) |

## Admission (cluster)

| Policy | Engine | Applies to | Rule |
|---|---|---|---|
| `require-approval-for-large` | Kyverno `ValidatingPolicy` | `AppEnvironment` | `large` needs `approved-by` |
| `tenant-image-hygiene` | Kyverno `ValidatingPolicy` | pods in tenant namespaces | trusted registries; pinned tag or digest |
| Pod Security `restricted` | built-in | pods in tenant namespaces | non-root, no privilege escalation, seccomp, drop ALL |

Policies have their own unit tests in `policy/kyverno/tests`, with fixtures
that must **pass** and fixtures that must **fail**, so a policy that silently
stops matching breaks CI.
