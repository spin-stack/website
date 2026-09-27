---
title: Known gaps
description: The parts of Spin's security model that are not enforced yet, each with what it means for an installation.
---

These are the open items of Spin's security model. Each is a design change, not a patch, which is
why it is listed rather than fixed quietly.

## The proxy has no credential of its own

The control plane answers three questions the proxy asks — which workspace a host name is, whether
a port may be reached, and opening a session to a workspace — for anyone who can reach it. On AWS,
only the proxy and the runners can reach the control plane's port, but a component that speaks
for another should prove it is that component.

## A runner holding many volumes can write every volume

A runner's bucket credentials are narrowed to the volumes placed on it. When there are too many to
fit in one session policy, the policy is widened to every volume's prefix. The fix is credentials
per volume.

## Some URLs an administrator sets are checked only in part

An identity provider's address is checked not to point into the installation's own networks when
it is set, but the endpoints its discovery document names are not checked. The storage endpoint is
deliberately exempt: an S3-compatible store inside the installation's network is normal.

## Kernel features the runner depends on

- **Without vsock namespaces**, a machine's QEMU can reach another machine's vsock address, where
  the guest turns it away for not holding a reserved port.
- **Without core scheduling**, a machine's vCPUs can run beside another machine's on one core's
  siblings, and read what the other leaves in the core's caches. Turning SMT off is the answer
  there, and it halves the runner.
- **With Landlock older than ABI 3**, a runner running as root refuses to run `qemu-img` at all.

In each case the runner says so in its log once.

## The stores on the control plane's machine

Quickwit, which holds every log and trace, and the metric store's write endpoint answer anything
on the control plane machine's loopback. What runs there is the installation's own processes, each
as a user of its own. The collector likewise accepts telemetry from whatever its security group
lets in.

## A CLI tunnel ends with its credential, not with the port

Closing a port ends the connections the proxy carries to it, but a tunnel the CLI opened into it
lasts until the credential that opened it expires.

## Deleting a workspace does not destroy its key

Clones share their lineage's data key, so deleting a workspace makes its bytes unreadable only
when Object Lock's retention expires. See [Data and encryption](/security/data/#clones-share-a-key).

## Boot is measured, not verified

Spin OS's kernel image is not signed, and nothing checks the boot measurements before a machine is
trusted. See [Supply chain](/security/supply-chain/#what-it-does-not-cover-yet).

## Rules no test keeps

Four of the rules in Spin's security table are kept by each place that must follow them, with no
test that fails when a new one does not:

- credentials are minted from a secure random source, stored hashed or sealed, compared in constant
  time, and revoked with what issued them;
- an effect outside the database happens before the row that names it is forgotten;
- a URL a non-administrator can set is refused when it points into the installation's networks;
- a component that speaks for another proves it is that component.
