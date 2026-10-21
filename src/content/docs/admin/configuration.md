---
title: Installation settings
description: How an installation is configured: the file that seeds it, the dashboard that edits it, and how a changed file becomes a proposal.
---

An installation's settings live in its database, and an administrator edits all of them in the
dashboard. A **config file** is where they start.

## The config file seeds

On AWS, the file is `installation_config` in Terraform:

```hcl
module "spin" {
  # ...
  installation_config = file("installation.yaml")
}
```

When the control plane starts, it creates each item of the file that the installation does not
have yet: a setting, a preset, a network policy. From then on, the dashboard is where they change.

When the file later says something different about an item, the control plane does **not** write
over what an administrator decided. It records the difference as a **proposal**, and the dashboard
shows it with **Apply** and **Dismiss**. An item the file stops declaring becomes a proposal to
delete it; nothing is removed without an administrator deciding. A dismissed proposal stays
hidden until the file changes again.

This is how a new release's defaults reach an installation that already runs: as proposals, not
as surprises.

Check a file before you apply it:

```bash
controlplane config check installation.yaml
```

The whole file, with the comment on every field, is in the
[config file reference](/reference/configuration/file/).

## What it holds

The installation's settings, who may register, the sizes a workspace may be created with, and the
network policies it may run under. Each block, with every field and its default, is in the
[config file reference](/reference/configuration/file/), written from the example file the
control plane's own test reads.

On AWS the module adds what it knows itself: the domain, which machines join as runners, and the
autoscaling settings.

## Settings only the dashboard has

Some settings are not in the file. In **Admin → Security** and **Admin → Settings**:

- **Sign-in providers**: GitHub and OIDC. See [Users and teams](/admin/users/).
- **Session length**: seven days by default, at most fourteen.
- **Sharing**: whether snapshots may be shared (on by default), and whether ports and snapshots
  may be made public (off by default).
- **Rate limits** on sign-in and on the API.
- **Telemetry**: which traces and metrics are kept. See [Observability](/admin/observability/).
