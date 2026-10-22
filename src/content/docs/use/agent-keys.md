---
title: An agent without keys
description: "Run a coding agent that needs a provider key inside a workspace, with the key never on its disk: a brokered policy, a team secret and a tag."
---

A coding agent needs a key to talk to its provider, and a key on a workspace's disk is in every
snapshot of it and in every clone. Spinbox has what it takes for the key to never be there: the
host's egress proxy can stand in for a host name and set a header from a secret the team holds.
Nothing here knows what Anthropic or OpenAI is; it is four things Spinbox already does, in order.

## 1. The installation brokers the name

An administrator declares a policy in the installation's config file (or in the dashboard under
Network). A brokered rule has the host's proxy stand in for the names it allows, on HTTPS, and
set a header from a team secret:

```yaml
network_policies:
  - name: anthropic
    rules:
      - allowNames: [api.anthropic.com]
        broker: true
        inject: [{header: x-api-key, secret: anthropic-api-key}]
  - name: openai
    rules:
      - allowNames: [api.openai.com]
        broker: true
        inject: [{header: authorization, secret: openai-api-key, prefix: "Bearer "}]
```

A team cannot declare a brokered rule; the installation decides which names its proxy stands in
for. The header replaces whatever the workspace sent under that name, never sits beside it.

## 2. The team sets the secret

A team admin sets the secret the rule names, read from standard input and never an argument,
and says exactly which host it may be sent to. A rule that names any other host is refused it.

```bash
printf %s "$ANTHROPIC_API_KEY" | spin team secret set <team> anthropic-api-key --host api.anthropic.com
```

A team is named by its id, its path or `me`: your personal team, which every workspace you
create without naming a team is in. A key for your own workspaces alone is `spin team secret
set me ...`, and nobody else holds it.

The value is sealed in the control plane's database under the installation's key, or, with
`--ref`, kept in the team's own AWS Secrets Manager or Parameter Store and read by the control
plane as the team whenever a host asks. It is never answered back to a client, and a host is given it only for a
workspace of the team that is running under a rule that injects it.

## 3. The workspace runs under the policy

```bash
spin workspace create --team <team> --name agent-1 --network-tag anthropic --wait
```

A project's tags do the same for every workspace that opens it.

## 4. The tool wants to see a key

Claude Code and Codex refuse to start without a key in their environment. Give them one that
means nothing; the proxy replaces the header on the wire:

```bash
spin settings set env.ANTHROPIC_API_KEY spin
spin settings set env.OPENAI_API_KEY spin
```

That is the whole recipe. The CLI talks to `api.anthropic.com` by its real name, the guest's
trust store holds the installation's egress root once the owner's word names it, and the
request leaves the host with the team's key on it. HTTP/1.1, HTTP/2 and WebSockets all go
through: a WebSocket's handshake is a request like any other, and what follows its 101 is
carried as bytes.

## Your own services: an identity, not a key

A key is for a provider that takes nothing else. For a service the team runs itself - a model
gateway, an internal API, a Vault - there is nothing to set and nothing to rotate: the rule names
an `identity`, the audience the token is for, and the proxy sets a five-minute token of the
workspace's identity in the header instead. No secret exists, in the database or anywhere.

```yaml
network_policies:
  - name: gateway
    rules:
      - allowNames: [gateway.internal.example]
        broker: true
        inject: [{header: authorization, identity: https://gateway.internal.example, prefix: "Bearer "}]
```

Two things name who may receive it. The installation lists the audiences a workspace's identity
may be given for (`identity_audiences` in the config file), and the workspace's owner names the
ones theirs may go to:

```bash
spin workspace audiences agent-1 --add https://gateway.internal.example
```

The service verifies the token against the installation's issuer, `https://id.<domain>`, whose
discovery document and key set the installation publishes. The token's subject is
`spiffe://<domain>/t/<team>/w/<workspace>/p` - the `/p` says the proxy presented it on the
workspace's behalf, so the service admits what it chooses to: this workspace, every workspace of
the team, or any of the installation. It is bound to no key and says so; the proxy holds none of
the workspace's.

## Seeing it work, and what went wrong

```bash
spin workspace egress agent-1
spin workspace status agent-1
```

`egress` lists each request the proxy read to a brokered name, newest first, with the real
host's status or, for one the proxy refused, why: `credentials` is a secret the team does not
hold, `scope` a method or path the rule does not allow. `status` says, per injection, whether
the team holds what it is set from.

## What the proxy cannot do

- An upgrade to anything but a WebSocket - h2c, for one - is refused with 501: what would follow
  it is nothing a rule can read.
- A request body is read whole only when a rule signs it as a cloud principal, up to 16 MiB.
- Nothing but HTTPS on 443 is brokered: `git` over SSH is not, and `git` over HTTPS with the
  team's forge app is the way to a private repository.
- A client that pins its provider's certificate does not accept the proxy's; it keeps its own
  key, and the rule is better left as a plain allow.
