---
title: Status and limitations
description: What Spin does not do yet, where it runs, and the limits you should know before you rely on it.
---

Spin is **experimental**. APIs will change, and data may be lost. This page lists what it does
not do today, so that each one is something you know about before you meet it.

## Where it runs

- **AWS only.** The module in [spin-terraform-aws](/aws/overview/) is the supported way to
  install Spin. The platform itself is not tied to AWS, but nothing else is packaged or tested.
- **x86-64 only.** Runners are x86-64 machines with nested virtualization. There are no Arm
  runners.
- **The CLI** is built for Linux (x86-64) and macOS (Apple silicon).
- **One region, one control plane.** An installation lives in one AWS region. The control plane
  and the proxy are each a single machine; an update replaces them without stopping workspaces,
  but a machine that fails is replaced in minutes, not seconds.

## Workspaces

- **No idle stop.** A workspace runs until someone stops it. Runners scale to zero, but only
  when no workspace is running.
- **No GPUs.** There is no way to ask for a runner of a particular kind.
- **Linux guests only**, booted with the kernel Spin provides.
- **No IPv6** inside a workspace. The resolver a workspace uses drops AAAA answers.
- **TCP only** for published ports.
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

- **Work since the last upload can be lost.** A runner uploads a workspace's writes in layers,
  and on snapshot and stop. If the runner is lost, what it had not uploaded is lost with it. See
  [How it works](/overview/how-it-works/#where-your-work-is-stored).
- **Nothing tells you a workspace is stranded.** When a runner disappears, its workspaces wait
  for their owner to run `spin workspace recover`. Nothing notifies the owner.
- **The database's backups are RDS's**: point-in-time recovery for seven days, and a final
  snapshot. The control plane does not copy the database anywhere else.

## Identity and access

- **No multi-factor authentication** of Spin's own. Sign in through GitHub or an OIDC provider
  that enforces it.
- **No invitations.** A person must sign in once before they can be added to a team.
- **Team roles are fixed**: owner, admin and member.

## Security

The open items of the security model are listed in [Known gaps](/security/known-gaps/).

## Operations

- **Nothing pages anyone.** Metrics, logs and traces are kept inside the installation, and
  there are no alerts.
- **An update is one way.** A release older than the one that last started the database
  refuses to start on it. Going back past a schema change is a point-in-time restore of the
  database.
