---
title: Evaluate Spinbox
description: What Spinbox is, what it costs, how to try it on one Linux machine, what this release does not do, and how to install it - in that order.
---

Spinbox runs **workspaces**: virtual machines that a person or a program uses to run code,
installed in your own AWS account. It is built for work where the code that runs is not fully
trusted - an AI agent on a real repository, a dependency nobody has read - or where you need to
see what it changed and undo it. [What is Spinbox](/overview/) says what one is made of.

Each workspace is its own QEMU/KVM machine, with a kernel of its own, root, a persistent disk and
SSH. A snapshot keeps a point to come back to, with the running processes if you ask; a diff
between two snapshots lists every file added, modified or removed; a restore puts the disk, and
the processes, back. A workspace's network is closed until a policy opens it.

## What it costs

A small part runs all the time: the control plane and the proxy, two machines, and two Route 53
zones. Runners, the large spot machines workspaces run on, run only while workspaces do, and
are the largest cost. [Cost](/aws/cost/) lists each part with the setting that changes it, and
what to put into the AWS Pricing Calculator.

## Try it on one Linux machine

[Try it on one machine](/overview/one-machine/) brings the development stack up on a single
Linux host, with a real runner booting real virtual machines, from `git clone` to a command in a
workspace. It is not an instant demo, and it is not an installation: nothing in it is backed up,
and it answers only on `localhost`. The machine needs:

- Linux on x86-64 with KVM (`/dev/kvm`, `/dev/vhost-vsock` and `/dev/net/tun` usable by you), and
  `sudo` without a password
- Docker with the `compose` and `buildx` plugins
- Go, [Task](https://taskfile.dev), `pnpm` and `mkcert`
- 30 GB free on the disk the checkout is on, and several minutes for the first run

## What this release does not do

Spinbox is experimental: APIs change and data may be lost. Before relying on it, three limits
to know, each a row of [Limitations](/overview/limitations/):

- A runner that is lost takes with it [what it had not uploaded yet](/overview/limitations/#durability),
  up to the installation's RPO.
- [An update is one way](/overview/limitations/#operations): going back past a schema change is
  a restore of the database.
- [A diff does not see everything](/overview/limitations/#workspaces): a change that sets a
  file's times and the clock back is not reported, and extended attributes are not compared.

[Status](/reference/status/) lists every feature with what proves it, what is partly there, and
what is not built. [Security](/security/) is the model, its mechanisms and its known gaps.

## Install it

One OpenTofu module creates everything in your account and your region: [Install on
AWS](/aws/overview/) says what it creates and the choices it makes for you, [Before you
start](/aws/prerequisites/) what you need, and [Install](/aws/setup/) the commands, in order.
Then the [Quickstart](/quickstart/): sign in, create a workspace, snapshot, change something,
see the diff, go back.
