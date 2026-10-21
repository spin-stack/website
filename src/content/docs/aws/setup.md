---
title: Install
description: Install Spin in an AWS account step by step, from the state bucket to the first sign-in and the first workspace.
---

Read [Before you start](/aws/prerequisites/) first: the region must be one the release has
images in.

## 1. A home for the state

OpenTofu's state holds the installation's CA key, so it is kept encrypted, in a bucket, and never
as a file on your machine. Once per account and region, create that bucket and its KMS key:

```bash
git clone https://github.com/spin-stack/spin-terraform-aws
cd spin-terraform-aws/bootstrap
tofu init && tofu apply
```

The bucket and the key refuse to be destroyed: without the key, no installation's state can be
opened.

`bootstrap` can also create the roles a GitHub Actions workflow plans and applies with, through
GitHub's OIDC, if you name the repository in `github_repositories`.

## 2. The installation's configuration

`examples/complete` is an installation. Copy it into a repository of your own, or use it as it is:

```bash
cd ../examples/complete
$(tofu -chdir=../../bootstrap output -raw init)
```

That runs `tofu init` against the state bucket. Then write `terraform.tfvars`:

```hcl
spin_version = "v20261019.02"
domain       = "spin.example.com"
# name   = "spin"        # a prefix for every resource; one installation per name
# region = "us-west-2"   # the region bootstrap was applied in
```

The example exports `dashboard` and `next_steps`. Export the two you will want when you
[update](/aws/update/) as well:

```hcl
output "update_status" { value = module.spin.update_status }
output "boot_log"      { value = module.spin.boot_log }
```

The example also sets the runners' idle time, quiet hours and the most runners at once. Every
variable is in the [module reference](/reference/terraform/root/). The ones most installations
change:

| Variable | What it decides |
|---|---|
| `installation_config` | The installation's config file: policies, presets, limits. See [Installation settings](/admin/configuration/). |
| `admin_email` | Who the first administrator is. `admin@<domain>` if unset. |
| `proxy_allowed_cidrs` | Where users may reach the installation from. Anywhere if unset. |
| `runner_idle_minutes`, `quiet_hours`, `time_zone` | When runners go back to none. |
| `max_runners` | The most runners at once. |
| `runner_instance_types`, `runner_spot` | What runners are. |

## 3. Apply

```bash
tofu apply
tofu output -raw next_steps
```

`next_steps` says the rest, in order, with the exact commands for your installation.

## 4. Point the domain at the installation

Set the domain's nameservers, where it is registered or in its parent's zone, to the
`name_servers` output. When this answers with the proxy's address, they are there:

```bash
dig +short app.spin.example.com
```

The proxy gets its certificates from Let's Encrypt a few minutes later.

## 5. Wait for it to come up

About ten minutes the first time. Each machine's group says `InService` when it is ready;
`next_steps` has the command. If one does not come up, see
[Troubleshooting](/aws/troubleshooting/).

## 6. Sign in

Open `https://app.<domain>` and sign in as the first administrator, with the one-time password
the apply wrote to SSM. It never passed through the state. Step 3 of `next_steps` has the
command that reads it.

The first page asks for a password of your own.

## 7. Let people in

In **Admin → Security**:

- Add a sign-in provider: GitHub, or any OIDC provider.
- Choose who may register: nobody (the default), anyone, or a list of addresses and domains.

Then disable password sign-in for the first administrator, once a provider works.

## 8. Check it

Install the CLI and sign in as in the [Quickstart](/quickstart/), then run the installation's
own check:

```bash
spin smoke
```

It creates a workspace of its own, runs a command in it, checks the workspace cannot reach the
cloud's metadata service, suspends and resumes it, and erases it. It exits 0 when every step
passed; [`spin smoke`](/reference/cli/smoke/) lists the other exit codes. No runner is up yet,
so the first workspace waits a few minutes in *Waiting for a host* while the control plane
starts one.
