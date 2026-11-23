---
title: Identity and access
metaTitle: "Identity and access: how every request is authorized"
description: How users, API keys, runners and machines prove who they are, how every request is authorized, and what limits each machine's cloud role.
reads:
  - "spin:docs/security/README.md@v20261121.03"
  - "spin:internal/controlplane/server/middleware/scopes.go@v20261121.03"
  - "spin:docs/security/traffic.md@v20261121.03"
  - "spin-terraform-aws:modules/controlplane/boundary.tf@v20261123.01"
---

## Users

- Users sign in with **GitHub or an OIDC provider**. The first administrator's password is
  hashed with bcrypt, and can be turned off once a provider works.
- A session lasts **seven days** by default, at most fourteen. The dashboard's session is a
  cookie, and a request that changes something or opens a socket must come from the dashboard's
  own origin.
- The CLI signs in with a **device code**: it prints a URL and a code, and the user signs in to
  the dashboard and enters it.
- Sessions, API keys and tokens are 32 random bytes, stored only as a hash.
- Sign-in and the API are **rate-limited**.

## Authorization

Every request is decided in the same way, and each rule has a test that fails the build when a
new endpoint forgets it:

1. **Every endpoint names its scope.** An API key reaches only the endpoints its scopes allow.
2. **Every id in a request is authorized for the caller** before anything is read or written
   with it.
3. **"Not yours" is answered as "not found"**, so a caller cannot learn what exists.
4. **Membership is checked at the time of the action**, not when the resource was made. A
   member removed from a team loses its workspaces at once, including streams already open.

A published port is decided the same way: the owner and the users granted it, each still a member
of the workspace's team, unless the port is public.

An administrator acts on a workspace that is not theirs only through the procedures that say so,
each from a session and audited with whose workspace it was: holding it in quarantine, bounding its
bandwidth, re-keying its disk, stopping it, moving it off a host that is gone (with a reason its
owner reads on the workspace's history), and reading where it sends. Listing a person's or a
runner's workspaces, to find the one to act on, shows each one's name, state and place, and
nothing in it. Being an administrator opens
no session or command in it, no snapshot of its disk and no say over who logs in.

## API keys

A key carries scopes, lasts at most a year, and can never approve a login or administer the
installation. Administrators can list and revoke every user's keys.

- **A key that runs commands** in its person's workspaces holds `workspace:exec`, and is made
  only with that person's passkey, in the dashboard. Every call it makes is in the audit log.
- **A key that makes keys** holds `keys:delegate`. Each key it makes holds no more than it does -
  its scopes, team, projects, end, and how many workspaces and cores all of them hold - and goes
  when it is revoked.

## Who logs in to a workspace

Only a workspace's owner and the guests the owner let in, each with a key they added with their
passkey: [Who gets into your workspaces](/use/access/) is the whole of it, in spin's own words,
and [Known gaps](/reference/security/known-gaps/#what-this-does-not-protect) what it does not
protect.

## Runners

A runner in the installation's autoscaling group joins **by its AWS identity and the build it
booted**: a request signed with its instance role, which AWS checks, and a NitroTPM document the
control plane holds to the builds it allows. Nothing is minted, stored or rotated.
[NitroTPM](/security/nitrotpm/#how-a-runner-is-admitted) is the whole of it.

Once joined, a runner holds an access token that lasts an hour and is refreshed, over TLS under
the installation's CA. A runner is trusted only for what is placed on it: it can report on and ask
keys for its own workspaces, and nothing else.

A runner outside the group joins with a one-time registration token from an administrator.

## Machines and their roles

Every IAM role the module creates carries **one permissions boundary**. Whatever policy is
attached later, by mistake or from a machine that was taken, no role can:

- write IAM, or pass a role but the runners', which only the control plane passes, to EC2,
- assume a role other than those the control plane works under: the runners' credentials,
  volumes' keys, the identity key and teams' secrets,
- change the VPC's network or a security group,
- write a parameter in SSM,
- run a command or open a session on another machine,
- lift the bucket's Object Lock, bypass a retention, or rewrite the bucket's policy,
- delete logs or flow logs, change CloudTrail, or touch the alarms that report a role doing what
  it should not.

**Only the control plane's role reads the installation's secrets**: the CA's key, the first
password and the installation's configuration. The encryption key is not among them: nothing
stores it ([Data and encryption](/security/data/#the-installations-secrets)). Session Manager is granted by a
policy of the module's own, not by AWS's managed policy, which would let any machine read every
parameter in the account.

The machines' metadata service requires **IMDSv2**. No machine runs SSH: a shell on the
control plane or the proxy is Session Manager, decided by IAM and logged by AWS. A runner takes no
Session Manager session; the one way onto it is
[a root shell the control plane opens, for a while and with a reason](<feature:A root shell on a runner, for a while and with a reason: the runner emptied first, every session recorded under Object Lock, the runner retired after>).
Those conditions are the control plane's own code, so they do not hold against a control plane
running other code: see
[What this does not protect](/reference/security/known-gaps/#what-this-does-not-protect).

## What reaches the control plane

| From | Proven by |
|---|---|
| A browser or the CLI | A session or an API key, then scopes and per-resource checks |
| The proxy | TLS under the installation's CA; the request's own credential, and for the calls only a proxy makes (who may open a port, whether it is published, a workspace's session) a token of its own, given for an hour to what its role and its boot prove |
| A runner | TLS under the installation's CA and its access token |
| The programs inside a workspace | A token for that workspace, carried over vsock by its runner |
