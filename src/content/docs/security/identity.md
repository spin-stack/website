---
title: Identity and access
description: How users, API keys, runners and machines prove who they are, how every request is authorized, and what limits each machine's cloud role.
---

## Users

- Users sign in with **GitHub or an OIDC provider**. The first administrator's password is
  hashed with bcrypt, and can be turned off once a provider works.
- A session lasts **seven days** by default, at most fourteen. The dashboard's session is a
  cookie, and a request that changes something or opens a socket must come from the dashboard's
  own origin.
- The CLI signs in with a **device code** the user approves in the dashboard.
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

## API keys

A key carries scopes, lasts at most a year, and can never create another key, approve a login, or
administer the installation. Administrators can list and revoke every user's keys.

## Runners

A runner in the installation's autoscaling group joins **by its AWS identity**. It signs an STS
`GetCallerIdentity` request with its instance role, bound to this installation and to its own host
id; the control plane sends it to AWS and checks the role in the answer. A signed request is good
for an hour at most. Nothing is minted, stored or rotated.

Once joined, a runner holds an access token that lasts an hour and is refreshed, over TLS under
the installation's CA. A runner is trusted only for what is placed on it: it can report on and ask
keys for its own workspaces, and nothing else.

A runner outside the group joins with a one-time registration token from an administrator.

## Machines and their roles

Every IAM role the module creates carries **one permissions boundary**. Whatever policy is
attached later, by mistake or from a machine that was taken, no role can:

- write IAM or pass a role,
- assume a role other than the runners' credentials,
- change the VPC's network or a security group,
- write a parameter in SSM,
- run a command or open a session on another machine,
- lift the bucket's Object Lock, bypass a retention, or rewrite the bucket's policy,
- delete logs or flow logs, or touch CloudTrail.

**Only the control plane's role reads the installation's secrets**: the encryption key, the CA's
key, the first password and the installation's configuration. Session Manager is granted by a
policy of the module's own, not by AWS's managed policy, which would let any machine read every
parameter in the account.

The machines' metadata service requires **IMDSv2**. No machine runs SSH: a shell is Session
Manager, decided by IAM and logged by AWS.

## What reaches the control plane

| From | Proven by |
|---|---|
| A browser or the CLI | A session or an API key, then scopes and per-resource checks |
| The proxy | TLS under the installation's CA; the request's own credential. The proxy has none of its own. |
| A runner | TLS under the installation's CA and its access token |
| The programs inside a workspace | A token for that workspace, carried over vsock by its runner |
