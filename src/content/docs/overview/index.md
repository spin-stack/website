---
title: What is Spinbox
description: What Spinbox is, who it is for, what it lets you do, and the projects an installation is made of.
---

Spinbox runs **workspaces**: virtual machines that a person or a program uses to run code. You
install it in your own AWS account. It is built for work where the code that runs is not fully
trusted, or where you need to see what it changed and undo it.

Each workspace is its own QEMU/KVM machine. It has a kernel of its own, root access, a
persistent disk and SSH. Its disk is cloned from an image, so creating a workspace does not
download anything. Taking a snapshot seals the disk's current layer, so it is fast.

## Who it is for

- **Teams that run AI agents on real code.** An agent gets a whole machine it can install
  packages on, run services in, and break. You take a snapshot before it starts, look at the
  diff when it finishes, and roll back if you do not like the result.
- **Developers who want a remote machine per task.** Open it in VS Code or over SSH. Stop it
  at the end of the day and its disk is still there the next morning.
- **Platform and security teams** who need that work to happen in their own account, on their
  own network, with egress they decide and logs they keep.

## What you can do

| You want to | Spinbox gives you |
|---|---|
| Start from a known state | A workspace cloned from an image or from any snapshot, without copying a disk |
| Keep a point to come back to | [A snapshot of the disk](<feature:Snapshots of a running workspace's disk>), or [of the disk and memory together](<feature:Snapshots with memory, resumed with their processes>) |
| Undo | [Restore the disk to a snapshot](<feature:Restore a workspace to a snapshot>), or resume a memory snapshot with its processes |
| See what changed | [A file-by-file diff between two snapshots](<feature:A diff of the files that changed between two snapshots>), computed in a separate machine |
| Control what code reaches | [Network policies](<feature:Egress is closed by default; policies, by tag, name what a workspace may reach>): a workspace reaches nothing until a policy lets it |
| Reach a service inside | `https://<port>-<workspace>.ws.<your domain>`, private to you until you share it |
| Hand work to an agent | [API keys with scopes](<feature:API keys for scripts and agents>), and a CLI that prints JSON |

Each is a built feature with the test that proves it on the status page:
[workspaces](/reference/status/#workspaces), [snapshots](/reference/status/#snapshots) and
[network](/reference/status/#network). See [Workspaces](/concepts/workspaces/) and
[Snapshots and rollback](/concepts/snapshots/) for the details.

## Questions people ask first

### Is there a hosted version?

No. Spinbox runs in your AWS account, installed by one OpenTofu module. See
[Install on AWS](/aws/overview/).

### What does it cost?

A small part runs all the time, and runners run only while workspaces do. See [Cost](/aws/cost/).

### Can I try it without AWS?

Yes, on one Linux machine with KVM: [Try it on one machine](/overview/one-machine/).

### Does it need Kubernetes?

No. An installation is a handful of EC2 machines, buckets and roles that the module creates.

### What does it not do yet?

[Status](/reference/status/) lists every feature with the test that proves it, and what is not
built. [Limitations](/overview/limitations/) lists what Spinbox does not do by design.

### How do I report a security problem?

Privately, through a security advisory: see [Reporting a vulnerability](/security/reporting/).

## Why it is built this way

- **A virtual machine, not a container.** A container shares the host's kernel with every other
  container. A workspace does not. Code that escapes a process still has to get out of a
  virtual machine, whose QEMU runs under its own user with almost nothing to reach. See
  [Isolation](/security/isolation/).
- **Snapshots are layers, not copies.** A disk is a chain of qcow2 layers. A snapshot seals the
  top one, a clone points at the same layers, and a diff compares two of them. None of these
  copies a disk.
- **Your account, your data.** The control plane, the database, the bucket that holds every
  disk, the logs, the metrics and the traces all stay in your AWS account. Nothing is sent to a
  service outside it.
- **Machines you do not patch.** Every machine boots Spin OS, a read-only image with a
  verified root filesystem. An update replaces the machine with one built from the new
  release. See [Update](/aws/update/).
- **Pay for runners only while they work.** Runners are large spot machines. The control plane
  starts one when a workspace needs it and removes it when nothing has run for a while. See
  [Cost](/aws/cost/).

## The projects

An installation is made of four repositories under
[github.com/spin-stack](https://github.com/spin-stack):

| Project | What it is |
|---|---|
| **spin** | The platform: control plane, runner, proxy, CLI, dashboard, and the programs that run inside a workspace. |
| **spin-machine** | The virtual machine: QEMU, the guest kernel, the base image, and the definition of the machine they make. |
| **ami** (Spin OS) | The operating system every EC2 machine of an installation boots, with a release of spin already inside. |
| **spin-terraform-aws** | The OpenTofu module that creates an installation in an AWS account. |

A release of spin names the release of spin-machine it runs. A Spin OS image is built for one
release of spin. The OpenTofu module boots the published images of the release you choose.

## Next

- [How it works](/overview/how-it-works/): the pieces and how they talk.
- [Quickstart](/quickstart/): use an installation that already exists.
- [Install on AWS](/aws/overview/): make your own.
