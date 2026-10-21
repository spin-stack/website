---
title: Agents and automation
description: Give an agent or a CI job its own API key, let it create workspaces it cannot escape, and use snapshots, diffs and clones to review and undo its work.
---

Spin is built for code you do not fully trust, and an agent is the most common author of such
code. This page shows how to give an agent a workspace, how to limit what it can reach, and how
to review and undo what it did.

## In four calls

```bash
spin workspace create --name task-1 --wait -o json
spin exec task-1 -- make test
spin snapshot diff task-1 <after-id> --from <before-id> -o json
spin workspace delete task-1
```

Every command prints JSON with `-o json`. A model reading this site starts at
[`/llms.txt`](/llms.txt), and every page has a Markdown twin. The control plane's API is in
[the API reference](/reference/api/).

## A key for the agent

Create an API key with only the scopes the agent needs:

```bash
spin api-key create "review-bot" --scope workspace:read --scope workspace:write --expires 30d
```

Every scope, and what it allows, is in [`spin api-key`](/reference/cli/api-key/).

A key can never create another key, approve a login, or administer the installation. It lasts
at most a year. The secret is shown once.

Give it to the agent through the environment:

```bash
export SPIN_ENDPOINT=https://app.<domain>
export SPIN_TOKEN=<api-key>
spin -o json workspace list
```

The CLI needs no config file with these set. `-o json` makes every answer one JSON document.

## What the agent can reach

An agent that creates its own workspace usually names no network policy, so the workspace runs
under the installation's `default_network_tags`, and under `offline` if there are none. This is
where an administrator decides what an agent may reach: for example a policy that allows your
Git host and your package registry, and nothing else. See [Network and
egress](/concepts/network/).

Afterwards, `spin workspace egress <workspace>` shows what it reached and what it was refused.

The key the agent talks to its provider with need not be on the disk at all: see [An agent
without keys](/use/agent-keys/).

## Review, then keep or undo

The pattern that makes an agent's work safe to accept:

```bash
spin workspace create --name task-142 --from-snapshot <base-snapshot> --wait
spin snapshot create task-142 --description before --wait

# ... the agent works in task-142, with spin exec or over SSH ...

spin snapshot create task-142 --description after --wait
spin snapshot diff task-142 <after> --from <before> --under /srv/app
```

- Like the diff: keep the workspace, or take what it produced.
- Do not like it: stop the workspace and `spin workspace restore task-142 <before>`.
- Want to try again from the same point: create another workspace `--from-snapshot <before>`.

## Many attempts at once

Every workspace created from one snapshot is a clone of the same layers. Starting ten attempts
from the same state costs the disk of one, plus what each attempt writes. Each is its own
machine, with its own kernel, so no attempt sees another.

## Running commands

```bash
spin exec task-142 -w /srv/app -- make test
echo $?
```

`spin exec` streams output, forwards standard input, and exits with the command's exit status,
so it composes with the agent's own tools. It needs no network path to the workspace.

## The API

Everything the CLI does is a ConnectRPC call to the control plane, over HTTPS with the same key
(`Authorization: Bearer <api-key>`). ConnectRPC speaks plain JSON over HTTP POST, so any language
can call it. The services are `spin.v1.WorkspaceService`, `ShareService`, `SSHKeyService`,
`TeamService`, `SettingsService` and `LogService`.
