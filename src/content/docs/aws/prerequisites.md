---
title: Before you start
description: The tools, AWS account, region, domain and spot quota you need before installing Spinbox on AWS.
---

## Tools

- [OpenTofu](https://opentofu.org/docs/intro/install/) 1.11 or newer.
- The [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html),
  signed in to the account: `aws sts get-caller-identity` answers.
- The [Session Manager plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html),
  for a shell on a machine.

## An AWS account

Use an account, or at least a region, that holds nothing else. The module creates IAM roles, a
VPC and Route 53 zones, and you need credentials that can create them. `us-west-2` is the
default region.

## A domain

A domain, or a subdomain, that belongs to the installation alone. The module creates its public
zone in Route 53, and you point the domain's nameservers at it. The installation answers on:

| Name | What |
|---|---|
| `app.<domain>` | The dashboard and the API |
| `tunnel.app.<domain>` | The relay runners connect to |
| `term.<domain>` | The browser terminal: an SSH client for a workspace, in the browser, that signs in with a passkey |
| `*.ws.<domain>` | Workspaces' ports |

Workspaces are under a name of their own so that a workspace is never a subdomain of the
dashboard: nothing a workspace serves can set a cookie the dashboard reads.

### Optional modules

One module of spin-terraform-aws is not part of an installation unless you add it beside it. It
serves one more name, on its own CloudFront origin, and takes a provider in `us-east-1` for its
certificate:

| Name | Module | What |
|---|---|---|
| `id.<domain>` | [`modules/identity-issuer`](/reference/terraform/identity-issuer/) | The identity issuer: the documents a system outside Spinbox verifies a workspace's identity token with |

Its page says what it takes, and its README what it needs and how to add it.

## Room for runners

Each runner is a large spot machine, of one of the types
[`instance_types`](/reference/terraform/runners/#instance_types) lists: one CPU generation, with
local NVMe, and nested virtualization.

**Quota.** A new account's spot vCPU quota can be lower than one runner (Service Quotas → EC2 →
*All Standard Spot Instance Requests*). The module reads the quota when it plans and warns when
it is too low. With `request_quota` (on by default) the apply also asks AWS for the increase.
Until AWS grants it, a workspace waits for a runner that cannot start. This shows how the
request is going:

```bash
aws service-quotas list-requested-service-quota-change-history --service-code ec2
```

**Spot capacity.** How likely spot is to give you one of those types in your region, from 1 to
10:

```bash
aws ec2 get-spot-placement-scores --target-capacity 1 --region-names <region> \
  --instance-types <the types>
```

Below 7, name more types in `runner_instance_types`, or use on-demand with `runner_spot = false`.

In a region where none of these types has local NVMe, give runners an EBS data volume with
`runner_data_volume_gb`.

## A region with the release's images

Every machine boots Spin OS, an image built for one release with that release already inside.
The module boots the images published for the release you name, which exist in the regions
[Spin OS images](/reference/terraform/images/) lists. Install in one of them. To boot images of
your own instead, name them with [`image_ids`](/reference/terraform/root/#image_ids).

## Pick a release

Releases are named by date: `v<YYYYMMDD>.<N>`, the Nth release of that day. This site documents
`v20261025.01`. See [github.com/spin-stack/spin/releases](https://github.com/spin-stack/spin/releases).
