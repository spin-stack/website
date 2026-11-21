---
title: Jobs, workspaces, the bill and the store
description: What an administrator runs and reads beyond runners, from the control plane's jobs and the month's bill to the store, disk keys and a root shell.
reads:
  - "spin:cmd/cli/cmd/admin_jobs.go@v20261114.01"
  - "spin:cmd/cli/cmd/admin_costs.go@v20261114.01"
  - "spin:cmd/cli/cmd/admin_storage.go@v20261114.01"
  - "spin:cmd/cli/cmd/admin_volume.go@v20261114.01"
---

These are the `spin admin` commands that are not about a runner's state. For drain, activate and
revoke, see [Operating runners](/admin/hosts/).

## Jobs

The control plane does periodic work as jobs. You can
[list the jobs and run one now](<feature:The control plane's jobs, and running one now>).

```bash
spin admin jobs list
spin admin jobs run <job>
```

The list says how often each job runs and how its last run went. `run` does not wait for the job:
read the list again to see how it went. A job that is already running is refused.

## The bill

`spin admin costs` shows
[the month's bill, its forecast and each workspace's share](<feature:The month's bill, its forecast, and what each workspace cost>).

```bash
spin admin costs
spin admin costs --month <yyyy-mm>
spin admin costs by workspace
spin admin costs by team
spin admin costs by owner
```

The first form starts with the standby: the installation's daily cost with nobody working,
estimated from the last billed day. Then the cloud's bill by service, what the month is forecast
to end at, and its budget. It is the bill as the control plane last read it, never the cloud
itself. The control plane reads only the costs that `bill_scope` names in the
[configuration file](/reference/configuration/file/): without a scope it reads none, and says so.
The AWS module sets it; AWS also shows a tag in the bill only once the account has made it a cost
allocation tag ([Reading the bill](/aws/cost/#reading-the-bill)).

`by` shares the month out. The runners are shared by the cores and memory each workspace held,
and the volumes' bucket by what its volumes kept. What nobody caused is said apart and never
spread: runner capacity nobody held, bytes no workspace keeps, and the platform's own cost by
role and service.

The same estimate reaches the people it concerns:
[a workspace's owner sees its cost this month and its rate on its page, a team's admins their
workspaces summed, and a project's admins its workspaces with its prebuilds' storage](<feature:A workspace's cost this month and its rate, as an estimate with its day, for its owner; a team's workspaces summed for its admins, and a project's with its prebuilds' storage for its admins>).

## The store

`spin admin storage` shows
[every volume in the store and whose it is, and removes what nothing needs](<feature:The store: every volume, whose it is, and taking out what nothing needs>).

```bash
spin admin storage list
spin admin storage explain <key>
spin admin storage untag-image <ref>
spin admin storage remove <volume-id>
```

`list` says whether each volume may be removed, as the `storage-gc` job last read the store.
`explain` says what a key copied from the bucket's console is, and whose.

`remove` takes out an image, an extension release, an unclaimed volume or an orphan, and destroys
its key. An image its ref still names is untagged first. A workspace's disk, the newest release
of an extension, and a release a workspace runs are refused.

## Re-keying a disk

When a key is assumed taken by a host that held it, you can
[re-key the workspace's disk](<feature:Re-key a workspace's disk>).

```bash
spin admin volume rekey <workspace-id>
spin admin volume rekeys
```

`rekey` copies the disk whole under a key of its own, points the workspace at the copy and deletes
the old disk. That destroys the old key, and every snapshot of the disk with it. The workspace
is anybody's, named by its id, and must be stopped or quarantined. A disk something was cloned from is refused. `rekeys` lists the
disks being re-keyed, and the ones a revoked host's keys marked for it.

## A workspace that is not yours

An administrator can
[stop a workspace that is not theirs, move it off a host that is gone, read where it has sent, and
list the disks hosts refuse](<feature:An administrator stops a workspace that is not theirs, moves it off a lost host with a reason its owner reads on its history, reads where it sends, and lists the disks hosts refuse: each audited>).
Each is audited.

```bash
spin admin workspace stop <workspace-id> --reason "<why>"
spin admin workspace recover <workspace-id> --reason "<why>"
spin admin workspace egress <workspace-id> --reason "<why>"
spin admin volume refused
```

`stop` does what its owner's stop does; `--force` gives up what only its host has rather than
publish it first. `recover` needs a reason, which its owner reads, with who asked, on the
workspace's history; run without `--accept-data-loss` it shows the work the old host had not
published, and changes nothing. `refused` lists every disk its host refuses to serve, why, and
whose workspace cannot run for it.

## What happened

[The timeline](<feature:A timeline: what happened to a workspace, a runner, a team, a project or the installation, by whom, in one read>),
in **Admin → Timeline**, reads what happened to a workspace, a runner, a team, a project or the
installation, and who did it, in one place.

## A workspace's bandwidth

An administrator
[changes a running workspace's bandwidth, and its open connections stay open](<feature:An administrator changes a running workspace's bandwidth, its connections open>).

```bash
spin workspace bandwidth <id> <mbps>    # megabits a second, each way; 0 lifts the limit
```

A stopped or suspended workspace takes the value at its next start. The limit bounds what a
workspace can cost in egress, so its owner may not raise it.
