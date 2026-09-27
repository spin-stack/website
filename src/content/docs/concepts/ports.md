---
title: Ports and sharing
description: How to reach a service running in a workspace over HTTPS, who can reach it, and how to share it with other users or make it public.
---

A service listening on a TCP port in a workspace can be reached through the installation's
proxy, at its own HTTPS address:

```
https://<port>-<workspace>.ws.<domain>
```

The proxy holds the certificate. Nothing on the network connects to the workspace directly:
the runner carries each connection over the machine's vsock to the relay.

## Exposing a port

```bash
spin port expose api 3000
spin port expose api 3000 --description "dev server"
spin port list api
spin port unexpose api 3000
```

Exposing a port is not publishing it. An exposed port is **private**: only you can reach it,
after signing in to the dashboard in the same browser.

If the service itself speaks TLS, expose it with `--https`. The proxy then speaks TLS to it,
without checking its certificate, and the port is reached as `<port>s-<workspace>` instead.

Port 22 is always reachable by the workspace's owner, for SSH. It is not listed and cannot be
unexposed.

## Who can reach it

| Access | Who |
|---|---|
| `private` | You. The default. |
| `whitelist` | You, and the users you grant it to. |
| `public` | Anyone, with no sign-in. |

```bash
spin port grant api 3000 <user-id>
spin port access api 3000 whitelist
spin port access api 3000 public
spin port access api 3000 private
```

A grant does nothing until the port's access is `whitelist`, so sharing a port with someone
never quietly publishes it. Going back to `private` keeps the grants for next time.

A `public` port goes back to `private` when the workspace stops or fails. Whether ports may be
made public at all is the administrator's decision; it is off by default.

For a workspace that belongs to a team, everyone let in must still be a member of the team.
Removing someone from the team closes the connections they already have.

## From your own machine

To reach a port on `localhost` instead, forward it over SSH:

```bash
spin ssh api -- -L 3000:localhost:3000
```
