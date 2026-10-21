---
title: Backups and recovery
description: What an installation's durable state is, how each part is backed up, and what can be recovered after losing a runner, a machine, or the database.
---

An installation's durable state is in three places:

| State | Where | How it is protected |
|---|---|---|
| Users, teams, workspaces, snapshots, settings, secrets | PostgreSQL on the control plane's machine, archived to the database's bucket | A base backup every hour and each WAL segment within a minute, sealed under the installation's key; the bucket under Object Lock for [`database_lock_days`](/reference/terraform/controlplane/#database_lock_days) |
| Every workspace's disk and snapshots | The volumes bucket | Versioning, and Object Lock for [`object_lock_days`](/reference/terraform/controlplane/#object_lock_days) |
| The key that opens the installation's secrets | SSM Parameter Store | Written once by the install, never in the Terraform state |

The control plane's and the proxy's machines hold nothing that cannot be rebuilt. The proxy's
certificates are kept in a bucket of their own, and the metric store is backed up to the logs
bucket.

**The encryption key is the installation.** Without it, nothing in the database's archive or
its secrets can be opened, and neither can any volume's key. Nothing in the module makes a new one, and the parameter
refuses to be destroyed by a plan.

## Losing a runner

What the runner had not uploaded is lost. Every workspace can be moved to another runner by its
owner (`spin workspace recover`), or automatically if the installation allows it. See
[Runners and hosts](/concepts/runners/#when-a-runner-is-lost).

## Losing the control plane's or the proxy's machine

Its autoscaling group starts another. The new one starts from the same release and the same
document, and takes over. Workspaces keep running.

The control plane's machine carries the database, so the new machine restores it before it
serves: the newest base backup in the archive, then every WAL segment after it. What the old
machine wrote in its last minute and had not archived is lost with it. An archive that does not
open under the installation's key is refused, and the control plane does not start on it.

A release replaces the machine the same way, without the loss: the old machine stops writing,
archives its last segment and lets go, and the new one restores to that point.

## Losing the archive

A delete or an overwrite in the database's bucket is a version that Object Lock holds for
`database_lock_days`. Older base backups are pruned as new ones are taken, so a restore starts
from a recent one.

## Going back to an older release

A release older than the one that last started the database refuses to start on it. Going back
past a schema change is not something an update or a command does today.

## What is not backed up

- The runners' local disks: they are a cache, and whatever was not uploaded yet.
