---
title: Editors and SSH
description: Connect to a workspace with spin ssh, plain ssh, scp and rsync, VS Code Remote-SSH, or the terminal in the dashboard.
---

Every way into a workspace goes through the installation's relay. You never need network access
to the runner or to the workspace, only to `https://app.<domain>`.

## `spin ssh`

```bash
spin ssh api
spin ssh api --start                 # start it first if it is stopped
spin ssh api -- -L 8080:localhost:80 # anything after -- goes to ssh
```

You are `root`. The CLI carries the connection over a WebSocket to the relay, and SSH itself
stays end to end between your client and the workspace's `sshd`.

## Plain `ssh`, `scp` and `rsync`

```bash
spin ssh-config
ssh spin.api
scp ./notes.txt spin.api:/root/
rsync -a ./src/ spin.api:/root/src/
```

`spin ssh-config` writes `~/.spin/ssh/config` and includes it from `~/.ssh/config`. From then on,
any program that runs `ssh` reaches a workspace as the host `spin.<workspace>`. Each connection
gets a new certificate for your key that is valid for five minutes, so there is no long-lived
credential to leak.

It uses the key you name with `-i`, or the first of `~/.ssh/id_ed25519`, `id_rsa` and
`id_ecdsa`. If you have none, the CLI makes a key of its own the first time. The key is never
uploaded: the control plane signs a short certificate for its public half, per connection.

## VS Code

```bash
spin code api
spin code api /srv/app --start
```

This opens VS Code with Remote-SSH at the path you name (`/root` by default). It needs the
`code` command and the Remote-SSH extension.

An installation can publish the VS Code server for recent stable releases of VS Code, so that
it is already in the workspace and nothing is downloaded when you connect. If your VS Code is
a version the installation has not published, it downloads its own server into the workspace,
about 220 MB, and the workspace needs a network policy that lets it.

Any editor with a remote-over-SSH mode works the same way through `spin ssh-config`.

## The dashboard's terminal

A workspace's page in the dashboard has **Open Terminal**, a shell in the browser, over the
same relay.

## Running one command

```bash
spin exec api -- make test
spin exec api -w /srv/app -- git status
spin exec api -- cat build.log > local.log
```

`spin exec` runs a command in the workspace, connects its standard input, output and error to
yours, and exits with its exit status. It allocates a terminal only when both ends are one, so
its output is safe to pipe.

## SSH keys

```bash
spin ssh-key add laptop ~/.ssh/id_ed25519.pub
spin ssh-key list
```

None of the commands above need this. Keys you add are authorized in your workspaces, for a
client that connects without the CLI's certificate. Each workspace also has a key pair of its own;
`spin ssh-key get api` downloads its private key, for tools that cannot use the CLI.
