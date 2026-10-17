---
title: Quickstart
description: Sign in to an installation, create a workspace, take a snapshot, change something, see the diff, and go back.
---

This page uses an installation that already exists. To make one, see
[Install on AWS](/aws/overview/); to try Spin on one Linux machine without one, see
[Try it on one machine](/overview/one-machine/).

You need the address of the installation's dashboard, `https://app.<domain>`, and an account
on it. Ask whoever runs it, or sign in with the first administrator's account if you installed
it.

## 1. Get the CLI

The CLI is a single binary, published with every release as `spin-linux-amd64` and
`spin-darwin-arm64`. Put it on your `PATH` as `spin`:

```bash
chmod +x spin-linux-amd64
sudo mv spin-linux-amd64 /usr/local/bin/spin
spin version
```

The Linux build is also in the release's public package:

```bash
oras pull ghcr.io/spin-stack/spin-release:<version>-spin-linux-amd64
```

## 2. Sign in

```bash
spin login --endpoint https://app.<domain>
```

The CLI opens the dashboard in your browser and shows a code. Approve the code there, and the
CLI is signed in. `spin whoami` shows who you are.

## 3. Create a workspace

```bash
spin workspace create --name demo --wait
```

The workspace starts from the installation's default image, with its default CPU, memory and
disk. If no runner is up, it waits in *Waiting for a host* for the few minutes one takes to
start; `--wait` waits for it.

```bash
spin workspace list
spin ssh demo
```

You are `root` inside a virtual machine of your own, running Ubuntu.

A workspace reaches only what its network policies allow, and by default that can be
nothing at all. To see which policies the installation has, and to create a workspace that can
reach what one of them allows:

```bash
spin workspace policies
spin workspace create --name online --network-tag <policy> --wait
```

See [Network and egress](/concepts/network/).

## 4. Take a snapshot

Back on your machine:

```bash
spin snapshot create demo --description "clean" --wait
spin snapshot list demo
```

A snapshot is a point to come back to. Add `--with-memory` to keep the running processes too.

## 5. Change something, and see what changed

```bash
spin exec demo -- sh -c 'echo hello > /etc/greeting; rm /etc/issue; echo "# edited" >> /etc/hosts'
spin snapshot create demo --description "after" --wait
spin snapshot diff demo <after-id> --from <clean-id> --under /etc
```

The diff lists every file added (`A`), removed (`D`), modified (`M`) or changed type (`T`)
between the two snapshots: here, `/etc/greeting` added, `/etc/issue` removed and `/etc/hosts`
modified. It runs in a separate machine, so the workspace does not need to be running.
`--under` narrows it to one directory; installing a package alone changes thousands of files.

## 6. Go back

A disk restore needs the workspace stopped:

```bash
spin workspace stop demo --wait
spin workspace restore demo <clean-id>
spin workspace start demo --wait
```

The disk is back to the snapshot. The disk you left is kept among the workspace's snapshots,
so you can restore to it too.

To go back with the processes as they were, take the snapshot with `--with-memory`, then:

```bash
spin workspace restore demo <snapshot-id> --with-memory
```

## 7. Open a port

Start a web server in the workspace and expose its port:

```bash
spin exec demo -- sh -c 'nohup sh -c "while true; do printf \"HTTP/1.0 200 OK\r\n\r\nhello\n\" | nc -l 8000; done" >/dev/null 2>&1 &'
spin port expose demo 8000
```

It is at `https://8000-demo.ws.<domain>`, and only you can open it. See
[Ports and sharing](/concepts/ports/) to let others in.

## 8. Clean up

```bash
spin workspace delete demo
```

## Next

- [Workspaces](/concepts/workspaces/) and [Snapshots and rollback](/concepts/snapshots/)
- [Editors and SSH](/use/editors/): VS Code, `ssh`, `scp` and `rsync`
- [Agents and automation](/use/agents/): API keys and JSON output
