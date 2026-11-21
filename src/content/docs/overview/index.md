---
title: What is Spinbox
description: What Spinbox is, who it is for, what it lets you do, how to evaluate it, and the projects an installation is made of.
reads:
  - "spin:internal/features/features.go@v20261114.01"
  - "spin-terraform-aws:README.md@v20261027.02"
---

Spinbox runs **workspaces**: virtual machines that a person or a program uses to run code. You
install it in your own AWS account. It was built as a place to work, and what makes it a good one
(a whole machine, snapshots, a network closed until you open it) is what makes it a place to run code
you don't fully trust, an agent's included, and to see what that code changed and undo it.

Each workspace is its own QEMU/KVM machine. It has a kernel of its own, root access, a
persistent disk and SSH. Its disk is cloned from an image, so creating a workspace does not
download anything. Taking a snapshot seals the disk's current layer, so it is fast.

## Who it is for

- **Developers who want a remote machine per task.** Open it in VS Code or over SSH. Stop it
  at the end of the day and its disk is still there the next morning.
- **Teams that run AI agents on real code.** An agent gets the same whole machine a person
  does: it can install packages, run services, and break things. You take a snapshot before it
  starts, look at the diff when it finishes, and roll back if you do not like the result. The
  pieces made only for agents are built and tested, and the least used of all so far.
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

Each is a built feature, with what proves it on the status page:
[workspaces](/reference/status/#workspaces), [snapshots](/reference/status/#snapshots) and
[network](/reference/status/#network). See [Workspaces](/concepts/workspaces/) and
[Snapshots and rollback](/concepts/snapshots/) for the details.

## Evaluate it

In this order:

1. **What it costs.** A small part runs all the time: the control plane and the proxy, two
   machines, and two Route 53 zones. Runners, the large spot machines workspaces run on, run while
   workspaces do and stop a while after, and are the largest cost. [Cost](/aws/cost/) lists each
   part with the setting that changes it.
2. **Try it on one Linux machine.** [Try it on one machine](/overview/one-machine/) brings the
   development stack up on a single Linux host with KVM, with a real runner booting real virtual
   machines. It is not an installation: nothing in it is backed up, and it answers only on
   `localhost`.
3. **What it does not do.** Spinbox is experimental: APIs change and data may be lost.
   [Limitations](/overview/limitations/) is what it does not do by design or by where it runs, and
   [Status](/reference/status/) every feature with what proves it, what is partly there, and what
   is not built. [Security](/security/) is the model, its mechanisms and its known gaps.
4. **Install it.** One OpenTofu module creates everything in your account and your region:
   [Install on AWS](/aws/overview/) says what it creates, [Before you start](/aws/prerequisites/)
   what you need, and [Install](/aws/setup/) the commands. Then the [Quickstart](/quickstart/):
   sign in, create a workspace, snapshot, change something, see the diff, go back.

## Questions people ask first

### Is there a hosted version?

No. Spinbox runs in your AWS account, installed by one OpenTofu module.

### Does it need Kubernetes?

No. An installation is a handful of EC2 machines, buckets and roles that the module creates.

### How do I report a security problem?

Privately, through a security advisory: see [Reporting a vulnerability](/security/reporting/).

## Why it is built this way

- **Parts that already exist.** The disk is QEMU's chain of qcow2 files, the network is Linux
  namespaces and nftables, the boot is a signed kernel and a verified root. Spinbox's own code is
  mostly what joins them, which leaves less of it to be wrong.
- **A virtual machine, not a container.** A container shares the host's kernel with every other
  container. A workspace does not. Code that escapes a process still has to get out of a
  virtual machine, whose QEMU runs under its own user with almost nothing to reach. See
  [Isolation](/security/isolation/).
- **Snapshots are layers, not copies**, so a snapshot, a clone and a diff are cheap: see
  [Disks and storage](/concepts/volumes/).
- **Your account, your data.** The control plane, the database, the bucket that holds every
  disk, the logs, the metrics and the traces all stay in your AWS account. There is no hosted
  part of Spinbox for any of it to be sent to.
- **Machines you do not patch.** Every machine boots Spin OS, a read-only image with a
  verified root filesystem. An update replaces the machine with one built from the new
  release. See [Update](/aws/update/).
- **Pay for runners only while they work.** The control plane starts one when a workspace needs
  it and removes it when nothing has run for a while. See [Cost](/aws/cost/).

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
