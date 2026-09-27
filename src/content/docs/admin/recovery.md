---
title: Backups and recovery
description: What an installation's durable state is, how each part is backed up, and what can be recovered after losing a runner, a machine, or the database.
---

An installation's durable state is in three places:

| State | Where | How it is protected |
|---|---|---|
| Users, teams, workspaces, snapshots, settings, secrets | RDS PostgreSQL | Seven days of point-in-time recovery, a final snapshot on delete, deletion protection |
| Every workspace's disk and snapshots | The volumes bucket | Versioning, and Object Lock for `object_lock_days` (30 by default) |
| The key that opens the installation's secrets | SSM Parameter Store | Written once by the install, never in the Terraform state |

The control plane's and the proxy's machines hold nothing that cannot be rebuilt. The proxy's
certificates are kept in a bucket of their own, and the metric store is backed up to the logs
bucket.

**The encryption key is the installation.** Without it, nothing in the database's secrets can be
opened, and neither can any volume's key. Nothing in the module makes a new one, and the parameter
refuses to be destroyed by a plan.

## Losing a runner

What the runner had not uploaded is lost. Every workspace can be moved to another runner by its
owner (`spin workspace recover`), or automatically if the installation allows it. See
[Runners and hosts](/concepts/runners/#when-a-runner-is-lost).

## Losing the control plane's or the proxy's machine

Its autoscaling group starts another. The new one starts from the same release and the same
document, and takes over. Workspaces keep running.

## Losing the database

RDS point-in-time recovery restores it to a new instance, as of any moment in the last seven
days. Putting that instance in place of the installation's is done by hand today; the module does
not automate it. Anything the control plane recorded after that point is lost. Disks and snapshots published after
it are still in the bucket, but the database no longer knows them.

## Going back to an older release

A release older than the one that last started the database refuses to start on it. Going back
past a schema change is a point-in-time restore to before the update.

## What is not backed up

- The runners' local disks: they are a cache, and whatever was not uploaded yet.
- The database is not copied into the bucket or anywhere else by Spin. Its backups are RDS's.
