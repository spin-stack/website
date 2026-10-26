---
title: Limitations
description: Where Spinbox runs and the limits to know before you rely on it. What is built and what is not is on the status page.
---

Spinbox is **experimental**. APIs will change, and data may be lost. What is built, what is partly
there and what is not built yet is the [status page](/reference/status/), written from the tree
by a test. This page is the limits: what Spinbox does not do by design, or by where it runs today.
Each bullet here is a limit row of that same list, and the build fails when the two disagree.

## Where it runs

- **AWS only.** The module in [spin-terraform-aws](/aws/overview/) is the supported way to
  install Spinbox. The platform itself is not tied to AWS, but nothing else is packaged or tested.
- **x86-64 only.** Runners are x86-64 machines with nested virtualization. There are no Arm
  runners.
- **The CLI runs on Linux x86-64 and macOS Apple silicon.**
- **One region, one control plane.** An installation lives in one AWS region. The control plane
  and the proxy are each a single machine; an update replaces them without stopping workspaces,
  but a machine that fails is replaced in minutes, not seconds.

## Workspaces

- **No GPUs.** There is no way to ask for a runner of a particular kind.
- **Linux guests only.** Booted with the kernel Spinbox provides.
- **No IPv6 inside a workspace.** The resolver a workspace uses drops AAAA answers.
- **TCP only for published ports.**
- **A diff does not see everything.** It hashes a file only when its size, inode or times
  changed, so a change hidden by setting the times and the clock back is not reported. Extended
  attributes are not compared.
- **A memory snapshot resumes only on the same machine.** It needs a runner carrying the same
  QEMU, guest kernel and guest init as the one it was taken on. A disk restore has no such
  limit.
- **Disk chains have a limit.** A disk is a chain of layers. A workspace that runs for a very
  long time without stopping stops adding layers at the limit, and its top layer grows instead,
  so less of its recent work is in the bucket.

## Durability

- **Work since the last upload can be lost.** A runner uploads a workspace's writes in layers:
  on the installation's RPO, on snapshot and on stop. If the runner is lost, what it had not
  uploaded is lost with it. See [How it works](/overview/how-it-works/#where-your-work-is-stored).

## Identity and access

- **No multi-factor authentication of Spinbox's own.** Sign in through GitHub or an OIDC provider
  that enforces it.
- **No team invitations.** A person must sign in once before they can be added to a team. A
  workspace's guests are invited by email.
- **Team roles are fixed.** Owner, admin and member.

## Security

The open items of the security model are listed in [Known gaps](/reference/security/known-gaps/).

## Operations

- **Nothing pages anyone.** Metrics, logs and traces are kept inside the installation. Alert
  rules fire in its Grafana, and no contact point tells anyone.
- **An update is one way.** A release older than the one that last started the database
  refuses to start on it. Going back past a schema change is a point-in-time restore of the
  database.
