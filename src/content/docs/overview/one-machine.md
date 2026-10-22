---
title: Try it on one machine
description: The development stack on a single Linux host with KVM - from git clone to a running workspace and a command in it. Not an installation.
---

This is the **development stack**: the control plane, the proxy and the stores in containers,
and a real runner on the same host, booting real virtual machines. It is what Spinbox's own e2e
lane brings up, and every command below is one that lane runs. It is not an installation:
nothing in it is backed up, its credentials are public test fixtures, and it answers only on
`localhost`. To install Spinbox, see [Install on AWS](/aws/overview/).

## What the machine needs

- Linux on x86_64, with `/dev/kvm`, `/dev/vhost-vsock` and `/dev/net/tun` usable by you
- `sudo` without a password: the runner and its machines run as root
- Docker, with the `compose` and `buildx` plugins
- Go (the version in `go.mod`), [Task](https://taskfile.dev), `pnpm` and `mkcert`
- 30 GB free on the disk the checkout is on

## 1. Check the machine

```bash
git clone https://github.com/spin-stack/spin
cd spin
task doctor
```

`task doctor` says what the machine has and fails on the first thing the stack cannot do
without. `task e2e:up` runs it before building anything.

## 2. Bring the stack up

```bash
task e2e:setup
task e2e:up
```

The first run downloads the pinned machine - QEMU, the guest kernel and its base image - and
builds the control plane, the dashboard and the runner; expect several minutes. When it ends,
the dashboard is at `https://app.localhost:10443`. If the host has more than one default
route, name the one the machines go out through: `E2E_VM_NETWORK_UPLINK=<interface> task e2e:up`.

## 3. Create a workspace and run a command in it

The stack signs in its test user, with an SSH key of their own, in
`_output/run/e2e/people/user`. Run the CLI as that user:

```bash
root=$PWD user=$PWD/_output/run/e2e/people/user
spin() { HOME="$user" SSL_CERT_DIR="/etc/ssl/certs:$user/.trust" "$root/bin/spin" "$@"; }
spin workspace create --name demo --wait --timeout 3m
spin exec demo --no-tty -- uname -a
```

The workspace is a virtual machine of its own on this host. From here the
[Quickstart](/quickstart/) applies, with `demo` on this stack.

## 4. Take it down

```bash
task e2e:down
```

This stops the runner and its machines and removes the containers with their volumes: the
database and the object store go with them, and the next `task e2e:up` starts empty.
