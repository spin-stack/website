---
title: Quickstart
metaTitle: "Spinbox quickstart: workspace, snapshot, diff, rollback"
description: Sign in to an installation, create a workspace, take a snapshot, change something, see the diff, and go back.
reads:
  - "spin:cmd/cli/cmd/auth.go@v20261114.01"
  - "spin:cmd/cli/cmd/sshkey.go@v20261114.01"
  - "spin:cmd/cli/cmd/snapshot.go@v20261114.01"
  - "spin:cmd/cli/cmd/port.go@v20261114.01"
---

This page uses an installation that already exists. To make one, see
[Install on AWS](/aws/overview/); to try Spinbox on one Linux machine without one, see
[Try it on one machine](/overview/one-machine/).

You need the address of the installation's dashboard, `https://app.<domain>`, and an account
on it. Ask whoever runs it, or sign in with the first administrator's account if you installed
it.

## 1. Get the CLI

The CLI is a single binary of your installation's release, for Linux and macOS:
[Install](/use/cli/#install) says where to get the one for your machine. Put it on your `PATH` as
`spin`:

```bash
chmod +x spin-linux-amd64
sudo mv spin-linux-amd64 /usr/local/bin/spin
spin version
```

## 2. Sign in

```bash
spin login --endpoint https://app.<domain>
```

The CLI prints a page's address and a code, and opens the page in your browser where it can. Sign
in there, from this machine or any other, enter the code, and the CLI is signed in. `spin whoami` shows who you are, and [`spin smoke` checks the installation does](<feature:A smoke test of what an installation's API promises, with a workspace of its own>)
what this page expects, with a workspace of its own that it erases.

## 3. Create a workspace

```bash
spin workspace create --name demo --wait
```

The workspace starts from the installation's default image, with its default CPU, memory and
disk. If no runner is up, it is `WAITING` (*Waiting for a host* in the dashboard) for the
few minutes one takes to start; `--wait` waits for it.

A workspace lets you in with an SSH key you added with your passkey. Add one once, from this
machine: the command opens the page where your passkey confirms it, and waits. See
[Who gets into your workspaces](/use/access/).

```bash
spin ssh-key add laptop ~/.ssh/id_ed25519.pub
```

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
`--under` narrows it to one directory; installing a package alone changes tens of thousands of files.

## 6. Go back

A disk restore needs the workspace stopped:

```bash
spin workspace stop demo --wait
spin workspace restore demo <clean-id>
spin workspace start demo --wait
```

The disk is back to the snapshot. Anything written after the snapshot is given up, unless a
snapshot was taken after it: the `after` snapshot is still there, so
`spin workspace restore demo <after-id>` undoes this restore. What the disk wrote after its last
snapshot is lost.

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

A workspace is stopped before it is deleted:

```bash
spin workspace stop demo --wait
spin workspace delete demo
```

Do the same for `online` if you created it. Where the installation keeps a trash, the workspace goes there with its disk, and
`spin workspace undelete demo` brings it back; `--permanently` erases it now.

## Next

- [Workspaces](/concepts/workspaces/) and [Snapshots and rollback](/concepts/snapshots/)
- [Editors and SSH](/use/editors/): VS Code, `ssh`, `scp` and `rsync`
- [Automation and API keys](/use/agents/): API keys and JSON output
