---
title: Reporting a vulnerability
description: How to report a security problem in Spin privately, and what to include.
---

Report a vulnerability through a private security advisory on spin's repository, never in a
public issue or a pull request:

[github.com/spin-stack/spin/security/advisories/new](https://github.com/spin-stack/spin/security/advisories/new)

An advisory is seen only by you and the maintainers until it is published. The conversation, the
fix and the release that carries it happen there.

## What it covers

- **spin**: the control plane, runner, proxy, CLI, dashboard, and what runs inside a workspace.
- **spin-terraform-aws**: what the module creates in an AWS account, and the permissions it grants.
- **spin-machine**: QEMU, the guest kernel and the base image a workspace boots.
- **Spin OS**: the image every machine of an installation boots.

A problem in one of these that you found through another goes in spin's advisory too.

## What to include

- The release you found it in: `spin version` prints it.
- What an attacker needs to start with: a workspace of their own, an API key, a place on the
  installation's network, or nothing.
- What they get, and the steps that show it.

The model these are measured against is on [Security model](/security/), and the problems Spin
already knows of are on [Known gaps](/security/known-gaps/): one of those is not a new report,
though a way to make it worse is.
