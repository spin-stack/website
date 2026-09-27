---
title: The dashboard
description: What the web dashboard shows a user and an administrator, page by page.
---

The dashboard is at `https://app.<domain>`. Everything in it is also in the CLI and the API.

## For everyone

- **Home**: your workspaces and teams.
- **Workspaces**: every workspace you can use, and **Create**, which offers the images and
  presets the installation allows.
- **A workspace's page**: its state and resources, and tabs for **History** (every change of
  state), **Snapshots**, **Ports**, **Network** (policies, and what it reached or was refused),
  and **Logs** (what its machine printed). **Open Terminal** gives a shell in the browser.
- **Shared**: snapshots others shared with you.
- **Teams**: your teams, their members and their settings.
- **Settings**: your profile, your SSH keys, your API keys and your dotfiles repository.

Two pages the CLI sends you to: **Device** approves a `spin login`, and **Workspace sign-in**
lets your browser open a private port.

## For administrators

The **Admin** section is shown to the installation's administrators only.

| Page | What it is for |
|---|---|
| Users | Every user, and their API keys |
| Teams | Every team |
| Security | Sign-in providers (GitHub, OIDC) and who may register |
| Settings | Limits, workspace presets, telemetry sampling |
| Network | Network policies, and whether every runner enforces the current version |
| Runners | Every runner: its state, capacity and workspaces; drain, activate, revoke |
| Storage | What the bucket holds, read-only |
| Extensions | Images, and the releases of the supervisor and the VS Code server |
| Logs, Traces | The installation's own logs and traces |
| Metrics | Grafana over the installation's metrics |
| Jobs | Background jobs, such as the daily storage cleanup |
| Audit | Who did what: sign-ins, changes to workspaces, teams, settings and runners |

When the installation's config file changes, the dashboard shows each change as a proposal to
apply or dismiss. See [Installation settings](/admin/configuration/).
