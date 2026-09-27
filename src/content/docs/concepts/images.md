---
title: Images
description: What an image is in Spin, the default image, how an administrator publishes and retires images, and how to build your own starting point from a snapshot.
---

An image is a disk a workspace can start from. In Spin an image **is** a published snapshot
with a name, its **ref**, such as `python:3.12`. There is no registry and no image cache: creating a
workspace from an image clones the snapshot, which copies nothing.

```bash
spin workspace create --name web --image <ref> --wait
```

## The default image

Each release of Spin runs one machine, and that machine comes with a base image: Ubuntu, with
systemd, SSH, `git`, `curl`, `make`, `jq` and the usual tools to find out what is wrong. The
control plane publishes it by itself the first time it starts, as
`ghcr.io/spin-stack/spinbox/sandbox:<machine version>`. A workspace created without `--image`
starts from it.

## Your own starting point

The simplest way to have a workspace start with your tools already installed is a snapshot:

1. Create a workspace, and install and configure what you need.
2. Take a snapshot of it.
3. Create new workspaces from that snapshot with `--from-snapshot`, or share it with your team.

Every workspace created this way is a clone of the same layers, so a hundred of them cost the
disk space of one, plus what each writes.

## Images an administrator publishes

An administrator can publish any qcow2 disk as a named image from the control plane's machine:

```bash
controlplane storage publish-image ./python.qcow2 python:3.12 \
  --name "Python 3.12" --description "Python workloads"
```

A reference names one image forever. Publishing the same bytes again does nothing; publishing
different bytes under the same reference is refused, so a new version is a new reference.

Administrators can also:

- **Limit which images may be used**, with the `allowed_images` setting.
- **Retire an image**, so that no new workspace is given it. Workspaces already created from it
  keep reading it. Once nothing reads it, it can be removed.

The images a workspace can be created from are listed by the API (`ListImages`) and shown in the
dashboard's create dialog.

## Integrity

Every disk's history hangs from digests that only the control plane writes, in the database. A
runner checks each layer it downloads against them, so a layer changed in the bucket is
refused.

An image's layers get one more check, because every workspace cloned from the image reads the
same file on the runner. When an image is published, the control plane records the fs-verity
digest of its bytes. The runner turns fs-verity on for the file and refuses it unless the kernel
measures that digest. From then on the kernel will not open the file for writing, even for
root, and verifies every block QEMU reads from it. See [Supply chain](/security/supply-chain/).
