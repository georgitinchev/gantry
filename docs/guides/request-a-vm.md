---
title: Request a virtual machine
description: A VM in gantry-cloud, inside your environment's private network.
---

Every `AppEnvironment` comes with its own private network in gantry-cloud (a
`/24` from the cloud's address pool). A `VirtualMachine` is placed in that
network.

```yaml
# requests/<team>/vm-<name>.yaml
apiVersion: platform.gantry.internal/v1alpha1
kind: VirtualMachine
metadata:
  name: worker
  namespace: checkout-dev
spec:
  size: small   # small | medium | large; changing it resizes in place
```

| Environment size | VMs allowed |
|---|---|
| small | 1 |
| medium | 2 |
| large | 4 |

gantry-lint enforces the per-environment limit on the pull request. The
cloud's account quota is the backstop at runtime, and the provider reports a
quota error on the resource if it's hit.

```bash
kubectl -n checkout-dev get vm worker
# NAME     SIZE    IP           SYNCED   READY
# worker   small   10.40.1.10   True     True
```

> gantry-cloud is a simulation that ships with gantry. See
> [the mock cloud](/docs/platform/cloud).
