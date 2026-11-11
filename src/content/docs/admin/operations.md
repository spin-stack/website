---
title: Jobs, the bill and the store
description: What an administrator runs and reads beyond runners, from the control plane's jobs and the month's bill to the store, disk keys and a root shell.
reads:
  - "spin:cmd/cli/cmd/admin_jobs.go@v20261110.01"
  - "spin:cmd/cli/cmd/admin_costs.go@v20261110.01"
  - "spin:cmd/cli/cmd/admin_storage.go@v20261110.01"
  - "spin:cmd/cli/cmd/admin_volume.go@v20261110.01"
  - "spin:cmd/cli/cmd/admin_shell.go@v20261110.01"
---

These are the `spin admin` commands that are not about a runner's state. For drain, activate and
revoke, see [Operating hosts](/admin/hosts/).

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

The first form is the cloud's bill by service, what the month is forecast to end at, and its
budget. It is the bill as the control plane last read it, never the cloud itself. The control
plane reads only the costs that `bill_scope` names in the
[configuration file](/reference/configuration/file/): without a scope it reads none.

`by` shares the month out. The runners are shared by the cores and memory each workspace held,
and the volumes' bucket by what its volumes kept. What nobody caused is said apart and never
spread: runner capacity nobody held, bytes no workspace keeps, and the platform's own cost by
service.

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

## A root shell on a runner

An administrator can open
[a root shell on a runner, for a while and with a reason](<feature:A root shell on a runner, for a while and with a reason: the runner emptied first, every session recorded under Object Lock, the runner retired after>).

```bash
spin admin shell open <runner-id> --reason "<why>" --minutes <minutes>
spin admin shell open <runner-id> --reason "<why>" --incident
spin admin shell list
spin admin shell approve <shell-id>
spin admin shell close <shell-id>
spin admin shell recording <shell-id>
```

The runner is emptied first: each workspace is suspended and resumed on another runner. With
`--incident` the shell opens on the runner as it is. Where the installation sets
`host_shell_needs_approval`, another administrator approves the shell before the runner is
touched. Every session is recorded by the control plane and kept in the logs bucket under Object
Lock.

Once the shell has ended, the runner is retired: it is revoked, every disk it held the key of is
marked to be re-keyed, and its machine is replaced by its group.

## A workspace's bandwidth

An administrator
[changes a running workspace's bandwidth, and its open connections stay open](<feature:An administrator changes a running workspace's bandwidth, its connections open>).

```bash
spin workspace bandwidth <id> <mbps>    # megabits a second, each way; 0 lifts the limit
```

A stopped or suspended workspace takes the value at its next start. The limit bounds what a
workspace can cost in egress, so its owner may not raise it.

## Writeback on a host

A runner's policy can carry
[a writeback bound of its own, measured on that host first](<feature:A writeback bound per host, measured on that host before it is taken>).

```bash
spin admin runner policy <runner-id>
spin admin runner policy <runner-id> --writeback-limit-bytes <bytes> --writeback-background-bytes <bytes>
```

With no flags the command shows the policy. Any flag writes the whole policy: a setting you do not
pass goes back to the runner's own default. Thresholds that no preset names are refused until
that host has been tested with them. The test is
[TestHostWriteback](/reference/api/hostservice/#testhostwriteback), and it loads the host's disk
while it runs.
