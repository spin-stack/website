---
title: Security model
description: What Spin protects, from whom, and how: isolation, network, data, identity and supply chain, and where to read about each.
---

Spin runs code its operators do not fully trust, on machines in their own account. Its security
model starts from that: **a workspace's tenant is root inside it, and may be hostile.** Everything
else is built so that being root in a workspace gives nothing outside it.

## What it protects

| Asset | From |
|---|---|
| Other workspaces, and the runner | Code inside a workspace, including root and the kernel |
| The installation's network and the cloud account | A workspace's traffic |
| Workspaces' disks and memory | Anyone who reads the bucket, or a runner's disk |
| The installation's secrets | Anyone who reads the database alone |
| Each user's workspaces, snapshots and ports | Other users of the installation |
| What a machine runs | Anyone who changes an image, a package or a release after it was built |

## The layers

- **[Isolation](/security/isolation/).** Every workspace is a virtual machine with its own kernel.
  Its QEMU runs as its own user, in an empty network namespace and a bounded cgroup, and reaches
  nothing on the host but what it is handed.
- **Network.** A workspace reaches nothing until a policy allows it, never reaches the runner's
  own networks or the cloud's metadata service, and cannot be reached from the network at all.
  See [Network and egress](/concepts/network/) and [Network on AWS](/aws/network/).
- **[Data and encryption](/security/data/).** Every layer of every disk is encrypted with its
  volume's key before it leaves the runner. Keys are wrapped under a key the database does not
  hold. Memory snapshots have keys of their own, destroyed with them.
- **[Identity and access](/security/identity/).** Every request is authorized against the
  resource it names, at the time it is made. "Not yours" is answered as "not found".
- **[Supply chain](/security/supply-chain/).** Every release is signed and checked before it is
  unpacked. Every machine boots a read-only root the kernel verifies block by block.

## Rules with gates

Spin keeps its security rules in a table in its repository, `docs/security/README.md`, numbered
I1 to I16. Each rule names the code that keeps it and the test that fails the build when it is
broken. A rule no test keeps says so, and is listed in [Known gaps](/security/known-gaps/).
