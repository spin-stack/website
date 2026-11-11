---
title: The agent in a workspace
description: Open T3 Code in a workspace, sign Claude Code and Codex in once, and see what the agent reaches, where its chats live and how it drives other workspaces.
reads:
  - "spin:cmd/cli/cmd/agent.go@v20261108.01"
  - "spin:cmd/cli/cmd/agent_home.go@v20261108.01"
  - "spin:cmd/cli/cmd/workspace_threads.go@v20261108.01"
  - "spin:cmd/cli/cmd/workload.go@v20261108.01"
---

A workspace can run an agent UI of its own. [T3 Code runs in the workspace as its agent
user](<feature:An agent UI: T3 Code in a workspace as its agent user, over the owner's SSH session - from `spin agent` on the person's computer, or in the workspace's Agent tab in the dashboard, which frames the client and carries it over a passkey's session>),
`spin-agent-t3`, and drives Claude Code and Codex from a browser: threads, checkpoints, and
switching between them.

## Open it

```bash
spin agent <workspace>
spin agent <workspace> <path> --start
```

The server listens on the workspace's loopback alone. Your own SSH session forwards it to your
computer, and the command prints a link that pairs your browser with it. `--port` picks the
local port. Ctrl-C ends the forwarding, not the server. `path` defaults to the project's
checkout, or to `/workspace` in a workspace with no project.

The workspace's Agent tab in the dashboard opens the same server. It frames the client and
carries it over your passkey's session.

You do not start or stop the server yourself. [The supervisor starts it when you ask, and stops
it once nobody uses it and no chat is at
work](<feature:The supervisor runs the agent for its owner while it is used: started, paired and listed over the owner's own session on a socket only root opens, and stopped once nobody has used it for half an hour and no chat is at work>).
Every flag is in [`spin agent`](/reference/cli/agent/).

## Sign in once

[You sign Claude Code and Codex in from the agent's
page](<feature:Claude Code and Codex signed in from the agent's page: each CLI's own login, run by the supervisor as the agent's user, its credentials kept in the agent's home and never passing through the page>).
Each is the CLI's own login, run by the supervisor as the agent's user. The credentials stay in
the agent's home and never pass through the page.

[One login signs in every workspace of
yours](<feature:A person signs Claude Code and Codex in once: every workspace of theirs, plain or a project's, is handed the same credentials and hands back each refresh and logout, and a workspace that names a guest holds none>),
plain or a project's. Each workspace hands back every refresh and logout. A workspace that names
a guest holds none.

This is a known gap: whoever runs code in one of your workspaces holds the provider token of
all of them, until you log out or revoke it at the provider. See [Known
gaps](/reference/security/known-gaps/).

## What it reaches

The agent's user has no capability and is given only `path`. [Its traffic is marked, and it
reaches only brokered
names](<feature:An agent the supervisor starts is marked, and reaches only brokered names>).
An administrator decides which names those are: see [An agent without keys](/use/agent-keys/).

## Chats and the agent's home

```bash
spin workspace threads <workspace>
```

[The workspace lists the agent's
chats](<feature:The agent's chats listed on the workspace - titles, paths and times, never a message - reported by its supervisor>)
as its supervisor last reported them: a title, the directory, the model and a time. Never a
message. You read a chat in T3 Code.

[The agent's home is a volume of its
own](<feature:The agent's home is a volume of its own: the chats and sessions outlive the workspace, kept in the installation's store as every disk is, and a new workspace mounts them at its creation>),
mounted at `/home/spin-agent-t3`. Erasing a workspace leaves its home yours, unmounted. A new
workspace mounts it when it is created and finds the chats there.

```bash
spin agent home list
spin workspace create --agent-home <id> --wait
spin agent home delete <id>
```

`delete` erases a home no workspace mounts: its chats, its key and its bytes.

## Driving other workspaces

`spin` in a workspace needs no login of its own. [An agent there drives other workspaces on a
key it gets for the workspace's
identity](<feature:An agent in a workspace drives others with spin, on a key it gets for the workspace's own SPIFFE identity - a child of the key that made the workspace, through its host, whatever the workspace's network>).
The key is a child of the key that made the workspace. It holds no more than that key, and goes
when the workspace goes to the trash.

```bash
spin -o json workspace list
```

The call goes through the workspace's host, whatever the workspace's network allows: an agent
in an `offline` workspace drives others too. For keys, scopes and review, see [Agents and
automation](/use/agents/).
