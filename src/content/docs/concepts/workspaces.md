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
spin workspace create --name api --cpu 2 --memory 4096 --disk 20 --wait
spin workspace create --name retry --from-snapshot <snapshot-id> --wait
```

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

| | Default | Bounds |
|---|---|---|
| CPU | 1 core | up to 128 |
| Memory | 512 MB | 256 MB to 512 GB |
| Free disk | 10 GB | 1 GB to 2 TB |

These are the defaults of an installation with no presets. An administrator can define
**presets** (for example `small` and `large`) and mark one as the default; a preset can also be
a ceiling. See [Installation settings](/admin/configuration/).

A running workspace can **grow**: while it runs, the runner can give it more CPU and memory
than it started with, up to its maximum, which by default is as much as the runner has to
lend. The disk is the image's size plus the free space you asked for.

## States

| State | Meaning |
|---|---|
| `CREATING` | The workspace exists and has not run yet. |
| `WAITING` | No runner has room for it yet. On AWS, one is being started. |
| `STARTING` | A runner is preparing its disk and starting the machine. |
| `RUNNING` | The machine is up. |
| `STOPPING` | The machine is shutting down. |
| `STOPPED` | Not running. Its disk is kept. |
| `FAILED` | A start or a stop failed. The state says which part failed. |
| `DELETED` | Gone. |

A runner tries a start three times. After that, the workspace is `FAILED`, and it stays that way
until you start it again. `spin workspace status` says why a workspace is not behaving, and
`spin workspace console` shows what its machine printed.

## Stop, suspend, delete

- **Stop** shuts the machine down and keeps the disk. The next start boots from it. Stopping
  does not take a snapshot.
- **Suspend** keeps the running processes: the workspace's memory and disk are saved, and the
  next start resumes from that instant, on any runner that carries the same machine.
- **Delete** removes the workspace. Its snapshots are kept: delete them on their own with
  `spin snapshot delete`.

A workspace runs until someone stops it: there is no idle timeout.

## What is inside

The machine boots the kernel Spin provides and a small init, `vminitd`, which hands over to the
image's own systemd. Nothing else of Spin's lives on the disk:

- **The supervisor**, a service that Spin mounts read-only at each start. It clones your
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
