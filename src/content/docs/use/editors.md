---
title: Editors and SSH
metaTitle: "Editors and SSH: VS Code, ssh, scp and rsync"
description: Connect to a workspace with spin ssh, plain ssh, scp and rsync, VS Code Remote-SSH, or the terminal in the dashboard.
reads:
  - "spin:cmd/cli/cmd/ssh.go@v20261121.03"
  - "spin:cmd/cli/cmd/sshconfig.go@v20261121.03"
  - "spin:cmd/cli/cmd/code.go@v20261121.03"
  - "spin:cmd/cli/cmd/exec.go@v20261121.03"
  - "spin:cmd/cli/remote/openssh.go@v20261121.03"
---

Every way into a workspace goes through the installation's relay. You never need network access
to the runner or to the workspace, only to `https://app.<domain>`.

## A key that logs in as you

Everything below - `spin ssh`, plain `ssh`, `spin exec`, VS Code - [logs in with an SSH key you
added once, with your passkey](<feature:A key logs in only once its person adds it with a passkey: their own for good, a browser's for twelve hours>).
[A computer's key is added as it signs in, and as it first connects](<feature:A computer's key is added as it signs in, with the passkey that authorizes it, and as it first connects>):
`spin login` offers it, and the page that confirms the login adds it with your passkey. Any other
key is added the same way from a terminal:

```bash
spin ssh-key add laptop ~/.ssh/id_ed25519.pub
```

[Who gets into your workspaces](/use/access/) says what a passkey is for, which keys log in,
and how a guest is let in.

## `spin ssh`

```bash
spin ssh api
spin ssh api --start                 # start it first if it is stopped
spin ssh api -- -L 8080:localhost:80 # anything after -- goes to ssh
```

You are `root`. The CLI carries the connection over a WebSocket to the relay, and SSH itself
stays end to end between your client and the workspace's `sshd`.

It runs your own `ssh`, which has to be OpenSSH 9.9 or newer: a workspace accepts only the
post-quantum key exchange `mlkem768x25519-sha256`, so `spin ssh`, `spin ssh-config` and
`spin code` check your `ssh` first and refuse an older one before asking anything of the
workspace.

## Plain `ssh`, `scp` and `rsync`

```bash
spin ssh-config
ssh spin.api
scp ./notes.txt spin.api:/root/
rsync -a ./src/ spin.api:/root/src/
```

`spin ssh-config` writes `~/.spin/ssh/config`, with that key exchange in its block, and includes
it from `~/.ssh/config`; for an older `ssh` it writes nothing, since one would refuse the whole
file over that line. From then on,
[any program that runs `ssh` reaches a workspace as the host `spin.<workspace>`](<feature:SSH keys, and plain ssh to spin.%3Cworkspace%3E>). Each connection
gets a new certificate for your key that is valid for five minutes, so a certificate that leaks
is useless minutes later; what lasts is the key itself, which stays on your machine.

It uses the key you name with `-i`, or the first key this machine holds that you have added:
`~/.ssh`'s, a FIDO2 key's among them, then `ssh-agent`'s. If the machine has none, the CLI makes
one of its own. `spin ssh`, `spin exec` and `spin code` add a key they settle on that you have not added before
they connect, opening the page where your passkey confirms it; a key named with `-i`, or one on a
session signed in with an API key, is not added, and the workspace refuses it. The private half never leaves your machine: the control
plane signs a short certificate for the public half, per connection.

## VS Code

```bash
spin code api
spin code api /srv/app --start
```

[This opens VS Code with Remote-SSH](<feature:VS Code over SSH, with the server the installation publishes and nothing downloaded>) at the path you name: by default the project's checkout,
`/workspace/<repository>`, or `/workspace` in a workspace with no project. It needs the `code`
command and the Remote-SSH extension; `--editor` names another editor's command.

VS Code runs a server in the workspace, built for its own commit. An installation's servers
reach a workspace on a read-only disk its host attaches, and `~/.vscode-server` on the
workspace's disk holds only links into it, so nothing is downloaded when you connect. No image
carries a server: an administrator publishes them from **Admin → Storage**.

Before VS Code connects, `spin code` asks the workspace whether it has the server for your
VS Code. If it has not, and the installation has published that version, the running workspace
is given it. If the installation has not, it prepares that version first, once for the whole
installation, where an administrator has accepted the server's terms (**Admin → Extensions**).
Otherwise VS Code downloads its own server into the workspace, about 220 MB, and `spin code`
says so before connecting; where the workspace has no room for that download, it refuses and
says what would free it.

The server is Microsoft's, and its [licence](https://code.visualstudio.com/license/server)
allows it for an organization's own internal use only. An installation that offers workspaces
to others must not carry it: it leaves VS Code to download its own, under a licence each user
accepts.

Any editor with a remote-over-SSH mode works the same way through `spin ssh-config`.

## The dashboard's terminal

A workspace's page in the dashboard has **Open terminal**, a shell in the browser, over the
same relay.

## Running one command

```bash
spin exec api -- make test
spin exec api -w /srv/app -- git status
spin exec api -- cat build.log > local.log
```

[`spin exec` runs a command in the workspace](<feature:Run a command in a workspace, copy files to and from it>), connects its standard input, output and error to
yours, and exits with its exit status. It allocates a terminal only when both ends are one, so
its output is safe to pipe.

## Letting someone else in

`spin workspace guests add` [lets a person into a workspace of yours, with a key of their own](<feature:Guests of a workspace, let in by its owner with a passkey>):
see [Sharing a workspace](/use/access/#sharing-a-workspace).
