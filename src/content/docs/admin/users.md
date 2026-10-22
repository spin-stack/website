---
title: Users and teams
description: How people sign in, who may register, administrators, teams and their roles, API keys, and the audit log.
---

## Signing in

A user signs in to the dashboard with:

- **GitHub**, or
- **any OIDC provider**: your company's identity provider, Google, Microsoft Entra ID, Okta and
  others that speak OpenID Connect.

Providers are configured in **Admin → Security**. A provider can be the installation's, or a
team's own. Spinbox has no multi-factor authentication of its own: require it in your provider.

The first administrator signs in with a password, from the one-time password the install wrote.
Once a provider works, password sign-in can be turned off.

The CLI signs in through the dashboard, with a code the user approves (see [The CLI](/use/cli/)).
Sessions last seven days by default.

## Who may register

| Mode | Who can create an account |
|---|---|
| `closed` | Nobody. The default. |
| `whitelist` | The addresses and domains listed |
| `open` | Anyone who can sign in with a provider |

## Administrators

An **instance administrator** sees the Admin section of the dashboard and the `spin admin`
commands. An administrator makes another user one, or disables a user, in **Admin → Users**.

## Teams

A team owns workspaces together. A workspace created with `--team` belongs to the team, and its
members can use it.

| Role | Can |
|---|---|
| Owner | Everything, including deleting the team. One per team. |
| Admin | Manage members and the team's settings |
| Member | Use the team's workspaces |

```bash
spin team create platform --description "Platform team"
spin team create api --parent platform
spin team member add <team-id> ana@example.com --role member
spin team member list <team-id>
```

Teams can be nested under a parent team. A person must have signed in once before they can be
added: there are no invitations.

Membership is checked every time something is done, not only when a workspace was created. A
member removed from a team loses its workspaces at once, including terminals and port
connections they had open.

## API keys

Users create API keys for scripts and agents, with scopes and an expiry of at most a year. A key
can never create another key, approve a login, or administer the installation. See
[Agents and automation](/use/agents/). Administrators see and revoke every user's keys in
**Admin → Users**.

## The audit log

**Admin → Audit** lists who did what: sign-ins and failed sign-ins, sessions revoked, workspaces
created, started, stopped, recovered and deleted, changes to teams, to settings and to users, and
runners registered. Nothing in the API can change or delete an entry.
