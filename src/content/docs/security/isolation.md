---
title: Isolation
description: What keeps a workspace from reaching the runner, other workspaces and the cloud — the virtual machine, how its QEMU is confined, and how untrusted disk bytes are parsed.
---

## A virtual machine per workspace

Every workspace is a QEMU/KVM virtual machine with a kernel of its own. Workspaces share no kernel
with each other or with the runner. Code that takes root in a workspace, or its kernel, is still
inside a virtual machine.

The QEMU Spin ships is built for KVM only: it refuses to emulate.

## QEMU itself is confined

If a workspace escapes into its QEMU process, that process reaches almost nothing:

- It runs as a **user of its own**, not as the runner's user.
- It starts in a **network namespace with nothing in it**. It cannot open a connection to the
  runner, another workspace or the network.
- Its vsock is in `local` mode, so it cannot reach another machine's vsock. The guest answers
  only on a reserved vsock port, which only the runner binds.
- It is born in a **cgroup** with `memory.max` and `pids.max`, and refused if one cannot be made.
- It has **a core-scheduling cookie of its own** for every thread, so its vCPUs never share a
  physical core's siblings with another machine's, where one could read what the other leaves in
  the core's caches.
- **Same-page merging is off** on the runner, so one machine cannot learn another's memory by
  timing page merges.
- Kernel modules for socket families nothing uses are never loaded.
- It gets **no environment**, and every message it sends on its control socket is bounded before
  it is decoded.

## Untrusted bytes are parsed confined

A workspace's disk is bytes its tenant wrote as root. The runner never mounts or parses a guest's
filesystem:

- `qemu-img`, the one program on the runner that reads a workspace's disk images, runs with no
  capabilities, under **Landlock** limited to the files of that one operation, under **seccomp**
  with no sockets, in namespaces of its own.
- A **diff** between two snapshots runs in a separate virtual machine with no network, whose
  kernel is the only one that parses the tenant's filesystem. Its answer is checked and bounded
  before it is believed.
- Text a guest sends to the host — console output, names, paths — is bounded in size, count and
  rate, and made valid, where it crosses.

## The network boundary

The workspace's network is enforced in nftables on the runner, in a namespace of its own, outside
the virtual machine. It drops every packet to private, loopback, link-local (including the
instance metadata service) and reserved addresses, and to the runner's own addresses, before any
policy is read. The workspace's resolver removes those addresses from its answers.

A workspace is never told the control plane's address or its CA. The programs Spin runs inside it
reach the control plane over vsock, through the runner.

## Where it depends on the kernel

Two of these protections need the runner's kernel to support them: vsock namespaces and core
scheduling. On a kernel without them, the runner says so once in its log and starts machines
anyway. See [Known gaps](/security/known-gaps/).
