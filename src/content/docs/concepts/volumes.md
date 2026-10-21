---
title: Disks and storage
description: How a workspace's disk is stored: layers in the bucket, a cache on the runner, encryption, when work is uploaded, and what deleting removes.
---

A workspace's disk is a **volume**. You never manage one directly: a volume is created with its
workspace and goes with it. This page explains where its bytes are, because that decides what
survives what.

## Layers

A volume is a chain of qcow2 **layers**. The bottom layers belong to the image or snapshot it
was cloned from, and are shared with every other workspace cloned from the same place. The top
layer is the only one written to.

From time to time the runner **seals** the top layer, uploads it to the bucket, and starts a new
one above it. It does this:

- once about 32 MiB has been written to the top layer,
- when a snapshot is taken,
- when the workspace stops.

A snapshot is a sealed layer with a name. A clone is a new volume whose chain points at the
snapshot's layers. A diff compares two chains. None of these copies a disk.

## Where the bytes are

| | Where | Survives the runner |
|---|---|---|
| Sealed layers | The bucket, compressed and encrypted | Yes |
| The top layer | The runner's local disk | No |
| A copy of the sealed layers | The runner's local disk, as a cache | Rebuilt from the bucket |

The bucket is the only place a volume is durable. What a runner holds can be rebuilt from it:
a workspace that starts on another runner downloads its layers there. A runner that already
holds an image's layers shares them between every workspace cloned from it.

## Encryption

Every volume has a **data key**. Each layer is compressed, then encrypted with AES-256-GCM
under that key, before it leaves the runner. The data key is itself encrypted under the
installation's key-encryption key, which the control plane keeps in the database, sealed under
a key that the database does not hold.

A runner holds no key-encryption key. It asks the control plane for the data key of each volume
placed on it, and keeps it in memory only.

A workspace cloned from a snapshot shares the data key of the snapshot's volume, because it
reads the same layers. See [Data and encryption](/security/data/) for what this means for
deletion.

## Deleting

Deleting a workspace deletes its volume. The bucket keeps every object under Object Lock, so
deleted objects become delete markers, and their bytes go when the retention runs out (30 days
by default). A layer that another workspace still reads through a clone is kept.

A daily job in the control plane removes what nothing refers to any more.

## Capacity on a runner

A runner keeps volumes on its local NVMe disks. The control plane counts the full size of each
volume placed on a runner, and does not place a workspace on a runner without room for it. A
runner whose disk fills past 70% stops taking new workspaces until it is back under 65%.
