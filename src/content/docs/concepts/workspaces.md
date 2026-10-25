---
title: Workspaces
description: What a workspace is, its states, its resources and disk, what runs inside it, and how it is started, stopped, suspended and recovered.
---

A workspace is a virtual machine that belongs to a person, or to a team. It has its own
kernel, root access, a persistent disk and SSH. It runs on a runner, but it is not tied to one:
its disk lives in the installation's bucket, and it can start again on any runner.

## Creating one

A workspace starts from an **image** or from a **snapshot**. Both are the same thing
underneath: a published, read-only disk. Creating a workspace clones it, which copies nothing.

```bash
spin workspace create --name api --preset large --wait
spin workspace create --name retry --from-snapshot <snapshot-id> --wait
```

- **Size.** `--preset` names one of the installation's presets; without it, the default one.
  See [Resources](#resources).
- **Name.** Lower-case letters, digits and hyphens, starting with a letter, at most 55
  characters. Names are unique in the installation, because a workspace's name is its host
  name. Every command takes the name or the id.
- **Image.** Without `--image`, the installation's default: the base image of the machine the
  installation runs, which is Ubuntu. An administrator can limit which images may be used. See
  [Images](/concepts/images/).
- **Team.** With `--team`, the workspace belongs to the team, and the team's members can use
  it. Without it, it is yours.
- **Network.** `--network-tag` names the policies it runs under. See
  [Network and egress](/concepts/network/).

## Resources

A workspace is sized by a **preset**: its cores, its memory and its free disk. An administrator
defines the presets (for example `small` and `large`) and marks one as the default. See
[Installation settings](/admin/configuration/). An installation with no default preset gives
1 core, 512 MB and 10 GB of free disk.

A preset is fixed, starting at its size and staying there, or it bursts: it starts at a floor
and may grow up to its size, or up to whatever its runner has room for. A running workspace
can **grow**: the runner lends it more CPU and memory, up to that ceiling, and takes them back.
`spin workspace grant` gives it cores and memory of its own, kept until
`spin workspace release` gives them back. The disk is the image's size plus the preset's free
space.

## States

<!-- The table is written by tools/refgen from the API's WorkspaceState, each row from its value's comment in spin's api/proto/spin/v1/common.proto: edit that, not the table. -->

| State | Meaning |
|---|---|
| `CREATING` | The workspace exists and has not run yet. |
| `STARTING` | A runner is preparing its disk and starting its machine. |
| `RUNNING` | Its machine is up. |
| `STOPPING` | Its machine is shutting down, and then what its disk wrote since its last commit is uploaded to the bucket. A stop stays here until the upload is done, and says why it is not; `spin workspace stop --force` does not wait for it. |
| `STOPPED` | Not running. Its disk is kept, and the next start boots from it. |
| `FAILED` | A start or a stop failed: `spin workspace status` says which part, and why. It stays so until it is started again. |
| `DELETED` | In the trash, with its disk, for the days the installation keeps it there (`workspace_trash_days`): `spin workspace undelete` brings it back, stopped. An installation that keeps no trash erases it at once. |
| `WAITING` | Wanted running, and no runner has room for it yet. On AWS, where runners scale to none, one is being started, and the workspace starts on it once it is up. |
| `QUARANTINED` | Held by an administrator with `spin workspace quarantine`: it cannot start until `spin workspace unquarantine` releases it, and its disk is kept as it was. |

A runner tries a start three times before the workspace is `FAILED`. `spin workspace status`
says why a workspace is not behaving, and `spin workspace console` shows what its machine printed.

## Stop, suspend, delete

- **Stop** shuts the machine down, uploads what the disk wrote since its last commit, and keeps
  the disk. The next start boots from it. Stopping does not take a snapshot.
  `spin workspace stop --force` stops the machine without waiting for the guest or the upload,
  and is also what ends a stop that did not finish.
- **Suspend** [keeps the running processes](<feature:Suspend with the processes where they are; a start resumes them>): the workspace's memory and disk are saved, and the
  next start resumes from that instant, on any runner that carries the same machine.
- **Delete** puts the workspace in [the trash](<feature:A trash: a deleted workspace stays whole, for its owner to undelete, until its days are up>), with its disk. `spin workspace deleted` lists the
  trash, and `spin workspace undelete` takes one out, stopped. Its snapshots are kept: delete them
  on their own with `spin snapshot delete`.

A workspace nobody uses is [suspended with its processes](<feature:Idle suspend, with a question before it that a keep answers>), and so is one that [has run longer than the installation allows](<feature:A workspace that ran as long as the installation allows is suspended>), used or not. An administrator sets both times in the installation's
settings. Ten minutes before a suspension its owner is asked, on the workspace's page and in its
terminals; `spin workspace keep <workspace> 2h` answers. A start resumes a suspended workspace.

## What is inside

The machine boots the kernel Spinbox provides and a small init, `vminitd`, which hands over to the
image's own systemd. Nothing else of Spinbox's lives on the disk:

- **The supervisor**, a service that Spinbox mounts read-only at each start. It clones your
  dotfiles and talks to the runner over vsock, never over the network. It is the release's own
  and is updated at every start.
- **The VS Code server**, when the installation publishes one, mounted the same way. See
  [Editors and SSH](/use/editors/).

`/usr` is the image's, so `apt install` works as it does anywhere. You are `root`.

When a workspace is created from a snapshot someone else shared, its first boot removes that
person's identity from the disk: SSH keys, shell history and credentials.

## When a runner is lost

If the runner a workspace was on disappears, the workspace keeps what reached the bucket and
loses what did not. Nothing moves it without asking:

```bash
spin workspace recover api                      # says what moving it would lose
spin workspace recover api --accept-data-loss   # moves it
```

An administrator can let the control plane move workspaces on its own after a runner has been
silent for a while, when little would be lost. See [Runners and hosts](/concepts/runners/).
