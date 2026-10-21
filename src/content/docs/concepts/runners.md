---
title: Runners and hosts
description: What a runner is, how it joins, how workspaces are placed on it, how runners scale from zero, and what happens when one is lost.
---

A **runner** is a machine that runs workspaces. In the dashboard and the API a runner is also
called a **host**. On AWS, runners are an autoscaling group of large EC2 machines with local
NVMe disks and nested virtualization, spot by default.

## What a runner does

- Keeps a connection open to the control plane, and runs what the control plane says should
  run on it.
- Starts each workspace as a QEMU/KVM virtual machine, confined to its own user, cgroup and
  empty network namespace.
- Keeps each workspace's disk on its local NVMe, and uploads its sealed layers to the bucket.
- Enforces each workspace's network policies, in nftables, outside the machine.
- Opens sessions to the relay when someone connects to a workspace.

A runner opens every connection it has. It has no inbound rule at all.

## Joining

A runner in the installation's autoscaling group joins **by who it is**. It signs an AWS
`GetCallerIdentity` request with its instance role, bound to this installation and to its own
id, and sends it to the control plane. The control plane passes it to AWS STS and checks that
the answer names the runners' role. There is no token to create, store or rotate.

A machine outside the group can join with a one-time registration token from Admin → Runners.

A runner must be of the same release as the control plane or older, and no more than 90 days
older. A newer one is refused: it may speak of things the control plane does not know.

## Placement

The control plane places a workspace on an **active** runner with room for its disk and
memory. It prefers the runner holding fewer of the same team's workspaces, so one team does not
pile onto one machine. A runner whose local disk is more than 70% full stops taking new
workspaces until it is back under 65%.

There is no way yet to ask for a particular kind of runner, such as one with a GPU.

## Scaling from zero

On AWS the control plane sizes the runners' group itself. It knows what nothing else does: that
a workspace is waiting, and when the last one stopped.

- With no runner up, the first workspace to start waits in *Waiting for a host* while the control
  plane asks for one. It is placed on the runner as soon as it registers.
- If a workspace has waited longer than a runner takes to start, the control plane adds one.
- When nothing has run for [`runner_idle_minutes`](/reference/terraform/root/#runner_idle_minutes), it scales the group back to
  none. During the installation's `quiet_hours`, it does so as soon as nothing is running.
- It never asks for more than `max_runners`.

It never chooses one runner of several to remove: it only goes from none to one, adds one, and
goes back to none.

## When a runner goes away

A runner that is going away empties itself first. It suspends each of its workspaces, with
memory, to the bucket, and each one resumes on another runner from where it was. This happens
when:

- AWS reclaims a spot runner (about two minutes' notice),
- the autoscaling group removes a runner (the group holds it for `drain_seconds`),
- a release is rolled out and runners are replaced.

A runner restarting its own process is not a drain: the virtual machines keep running, and the
new process adopts them.

## Operating runners

| Action | The runner | Its workspaces | Undo |
|---|---|---|---|
| **Drain** | Stays connected, takes no new workspace | Keep running; with `--suspend`, are moved off | Activate |
| **Activate** | Takes workspaces again | — | — |
| **Revoke** | Its credentials are deleted and it is disconnected for good | Refused while it holds any, unless you choose to leave them | None: replace the machine |

See [Operating hosts](/admin/hosts/).

## When a runner is lost

A runner can vanish without draining: the machine loses power, or its network. The control plane
does not guess. Its workspaces stay where they are, and each one has lost only what the runner
had not yet uploaded.

- The owner moves one with `spin workspace recover <workspace>`, which says how much would be
  lost and waits for `--accept-data-loss`.
- An administrator can let the control plane move workspaces on its own, after a runner has
  been silent for `rehome_after_seconds`, when no more than `rehome_max_unpublished_bytes` would
  be lost. This is off by default.

Nothing tells an owner that a workspace is stranded yet.
