---
title: Update
description: Move an installation to a new release — publish its Spin OS image, change spin_version, apply — and what happens to machines and workspaces while it rolls out.
---

A machine of an installation is never updated in place. Its root filesystem is read-only and
verified, so a new release is a new machine, started from the new release's image, beside the old
one.

## Steps

1. Build and publish the new release's Spin OS image into the account (see
   [Before you start](/aws/prerequisites/#a-spin-os-image-for-the-release)).
2. Change `spin_version` in `terraform.tfvars`.
3. Plan and apply:

```bash
tofu plan     # the image, the launch templates and the machines' documents change; nothing else
tofu apply
```

`tofu output update_status` prints, for the control plane's and the proxy's groups, the command
that says how the replacement went: `Successful`, `InProgress`, or `RollbackSuccessful` with the
reason.

## What happens

**The control plane and the proxy** are each an autoscaling group of one, and an update is an
instance refresh that keeps 100% healthy:

1. A new machine starts from the new image, while the old one keeps serving.
2. The new control plane points `cp.spin.internal` at itself and claims the term. The old one sees
   it has been superseded and stops.
3. Runners and the proxy reconnect within seconds. Workspaces keep running throughout.
4. A new proxy restores the certificates the last one had, starts, and takes the elastic IP once
   it answers.

What is lost is a few seconds of API calls and the connections open at that moment. No workspace
stops.

**If the new machine does not come up**, it hands the installation back: the old control plane
takes the term again, and the refresh is abandoned. The status then says `RollbackSuccessful`
with the reason, and the command `tofu output boot_log` prints shows why the new machine did not
come up.

**Runners** are replaced one at a time, the new one first. Each old runner leaves by draining: its
workspaces are suspended to the bucket and resume on another runner. A runner is never of a newer
release than its control plane, which is why the control plane goes first.

## Other changes

A change to anything else a machine starts from — `installation_config`, the autoscaling
settings, an instance type — rolls out the same way. A change to a machine's document alone
reaches it the next time it starts; start a refresh of its group to apply it now:

```bash
aws autoscaling start-instance-refresh --auto-scaling-group-name <group>
```

## Going back

Setting `spin_version` back rolls out the older release the same way, **unless the newer one
changed the database schema**. A release older than the one that last started the database
refuses to start on it. Going back past a schema change is not something an update or a command
does today.
