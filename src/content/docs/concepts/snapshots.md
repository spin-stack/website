---
title: Snapshots and rollback
description: Disk and memory snapshots, restoring and resetting a workspace, diffing two snapshots, cloning, and sharing a snapshot with others.
---

A snapshot is a point in a workspace's life that you can come back to. It is read-only and
lives in the installation's bucket, so it does not depend on any runner.

## Taking one

```bash
spin snapshot create api --description "before the migration" --wait
spin snapshot create api --with-memory --wait
```

Both are built features with the tests that prove them: [disk snapshots](<feature:Snapshots of a running workspace's disk>) and [snapshots with memory](<feature:Snapshots with memory, resumed with their processes>).

A snapshot is taken of a **running** workspace. The runner seals the disk at that moment and
uploads what changed since the last snapshot. It is usable once it is *published*; `--wait`
returns then.

- **Disk only** (the default): the files, as they were at that moment.
- **With memory** (`--with-memory`): the disk and the machine's memory together. The guest
  pauses for a few tens of milliseconds. Resuming it brings back every process as it was.

The memory of a snapshot is encrypted with a key of its own. Deleting the memory destroys that
key, so nothing can read it again, even from an older version of the object in the bucket.

## Going back

```bash
spin workspace stop api --wait
spin workspace restore api <snapshot-id>        # the disk only; the workspace must be stopped
spin workspace restore api <snapshot-id> --with-memory   # the whole machine; may be running
spin workspace reset api                        # back to what it was created from
```

- A **disk restore** puts the disk back to the snapshot. The workspace keeps its name, keys,
  ports and resources. What it wrote after the snapshot is set aside, not destroyed: the disk
  it had is kept among its snapshots, so you can restore to that too.
- A **memory restore** puts the whole machine back to the instant of the snapshot. It works on
  a stopped workspace, and on a running one, which is put back where it runs.
- A **reset** goes back to the image or snapshot the workspace was created from. Nothing is
  uploaded or downloaded, so it is the fastest way back.

## Seeing what changed

```bash
spin snapshot diff api <later-id> --from <earlier-id>
spin snapshot diff api <snapshot-id>                   # since the workspace was created
spin snapshot diff api <later-id> --from <earlier-id> --under /srv/app
```

The diff lists every file added (`A`), removed (`D`), modified (`M`) or that changed type (`T`).
It runs in a machine of its own with no network, so the workspace can be stopped, and nothing
it left on its disk runs.

Two things a diff does not report. It hashes a file's content only when its size, inode or
times changed, so root in the guest can hide a change by setting the times back, and the clock
with them. And it does not compare extended attributes.

`--under` starts the comparison at one directory. Use it: a diff covers the whole root
filesystem, and one package install changes tens of thousands of files.

## Starting from a snapshot

Any published snapshot you can see is a starting point for a new workspace:

```bash
spin workspace create --name retry --from-snapshot <snapshot-id> --wait
```

The new workspace is a clone: it reads through the snapshot's layers and writes only its own.
This is how you run the same starting state many times, for an agent's attempts or a test's.

## Sharing

A snapshot can be shared with another user, or made public:

```bash
spin share add <snapshot-id> alice@example.com
spin share public <snapshot-id>
spin share browse            # snapshots shared with you
```

Sharing hands over the **whole disk**, including anything left on it, such as
`/root/.ssh/authorized_keys` or a token in a dotfile. Before a snapshot is shared, Spinbox scans it
for credentials, and refuses the share until you acknowledge what it found
(`--acknowledge-findings`). **Memory is never shared**: a shared snapshot gives the disk alone.

When someone creates a workspace from a snapshot you shared, the first boot removes your
identity from it: SSH keys, shell history and credentials.

An administrator decides whether sharing is allowed at all, and whether snapshots may be made
public.

## Deleting

```bash
spin snapshot delete api <snapshot-id>
```

A snapshot's layers are kept while a workspace created from it still reads them. Deleting a
snapshot's memory destroys its key at once. See [Data and encryption](/security/data/) for what
each delete removes, and when.
