---
title: Data and encryption
description: How workspace disks, memory snapshots and the installation's secrets are encrypted, where each key lives, who can open what, and what each kind of delete destroys.
---

## Disks

Every workspace's disk is a chain of layers, and every layer is encrypted before it leaves the
runner:

- Each layer is compressed, then sealed with **AES-256-GCM** under its volume's **data key**
  (DEK), in 64 KiB frames, each with a nonce derived from the volume, the layer and the frame's
  position.
- The DEK is wrapped under the installation's **key-encryption key** (KEK). The wrapping is bound
  to the volume's id and the key's version, so a wrapped key copied onto another volume opens
  nothing.
- The bucket also encrypts every object at rest (SSE-S3).

A runner never holds the KEK. It asks the control plane for the DEK of a volume placed on it, or
of the volumes that one was cloned from, and keeps it in memory only.

## Memory snapshots

A snapshot with memory holds the machine's whole RAM. It is sealed under **a key of its own**,
made for that one snapshot and kept only in the database row that describes it. Deleting the
memory deletes the row, and with it the only copy of the key: from that moment nothing can read
the memory, including older versions of the object that Object Lock still keeps.

Memory is never shared. A shared or public snapshot hands over the disk only, and a memory
restore is only allowed on a snapshot of the workspace asking for it.

## Clones share a key

A clone reads its parent's layers, so it must be able to decrypt them: a workspace created from a
snapshot or an image carries **the same DEK** as the volume it was cloned from, wrapped under its
own id. Every workspace descended from one image shares that image's key.

This decides what deleting can destroy:

| Deleting | The key | The bytes |
|---|---|---|
| A snapshot's memory | Destroyed at once | Unreadable at once; the object expires after the retention |
| A snapshot | Kept: it belongs to the lineage | Kept while a clone still reads them |
| A workspace | Kept while the volume it was cloned from exists | Delete markers now; the versions expire after the retention |
| The last volume of a lineage | Destroyed | Unreadable from then on |

Because every workspace descends from a base image, and an image is never the last member of its
lineage, **deleting a workspace does not destroy its data key**. Its bytes are gone when Object
Lock's retention (`object_lock_days`) runs out. If you need a workspace's contents to become
unreadable at once, today only memory snapshots offer that.

## The installation's secrets

| Secret | Where it is kept | Sealed with |
|---|---|---|
| The KEK | The database | The installation's encryption key, with its own name as context |
| The first administrator's password, the object store credential | The database | The same |
| SSH keys, identity providers' client secrets | The database | AES-256-GCM, bound to the row's id, in a versioned key ring |
| Sessions, API keys, runner tokens | The database | Stored only as a hash |
| The encryption key itself | SSM Parameter Store | The account's key; readable by the control plane's role alone |

The database alone opens none of them. The encryption key and the first password are written by
the install through write-only attributes, and never reach OpenTofu's state. The CA's key is in the
state on purpose, which is why the state is encrypted with KMS.

## In transit

- Browsers and the CLI reach the proxy over **TLS** with Let's Encrypt certificates.
- The proxy and the runners reach the control plane over TLS under **the installation's own CA**.
- The control plane reaches its database on its own machine's loopback, with a password derived
  from the installation's key, so nothing stores it.
- SSH is end to end between your client and the workspace: the relay carries bytes it cannot
  read.
- A port exposed with `--https` is the one exception: the proxy speaks TLS to the workspace without
  checking its certificate.

## Integrity

A volume's history hangs from digests only the control plane writes, in the database. A runner
checks every layer it reads against them, so a layer replaced in the bucket is refused. Image
layers are also protected by fs-verity on the runner's disk. See
[Supply chain](/security/supply-chain/) for what this does not cover.

## Who can read the bucket

- The control plane, through its role, which cannot lift Object Lock, bypass a retention or
  rewrite the bucket's policy.
- Each runner, with credentials the control plane mints for one hour and narrows to the volumes
  placed on it and what they descend from.
- Object requests are refused from anywhere but the VPC's gateway endpoint, so a runner's
  credentials copied off the machine open nothing.
