---
title: Before you start
description: The tools, AWS account, domain, spot quota and Spin OS image you need before installing Spin on AWS.
---

## Tools

- [OpenTofu](https://opentofu.org/docs/intro/install/) 1.11 or newer.
- The [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html),
  signed in to the account: `aws sts get-caller-identity` answers.
- The [Session Manager plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html),
  for a shell on a machine.
- To build the Spin OS image: Go, Docker and [Task](https://taskfile.dev).

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
| `*.ws.<domain>` | Workspaces' ports |

Workspaces are under a name of their own so that a workspace is never a subdomain of the
dashboard: nothing a workspace serves can set a cookie the dashboard reads.

## Room for runners

Each runner is a spot machine of 32 or 48 vCPUs, chosen from five types of one generation:
`m8id.8xlarge`, `c8id.8xlarge`, `r8id.8xlarge`, `m8id.12xlarge` and `c8id.12xlarge`. These have
local NVMe and support nested virtualization.

**Quota.** A new account's spot vCPU quota can be lower than one runner (Service Quotas → EC2 →
*All Standard Spot Instance Requests*). The module reads the quota when it plans and warns when
it is too low. With `request_quota` (on by default) the apply also asks AWS for the increase.
Until AWS grants it, a workspace waits for a runner that cannot start. This shows how the
request is going:

```bash
aws service-quotas list-requested-service-quota-change-history --service-code ec2
```

**Spot capacity.** How likely spot is to give you one of these in your region, from 1 to 10:

```bash
aws ec2 get-spot-placement-scores --target-capacity 1 --region-names <region> \
  --instance-types m8id.8xlarge c8id.8xlarge r8id.8xlarge m8id.12xlarge c8id.12xlarge
```

Below 7, name more types in `runner_instance_types`, or use on-demand with `runner_spot = false`.

In a region where none of these types has local NVMe, give runners an EBS data volume with
`runner_data_volume_gb`.

## A Spin OS image for the release

Every machine boots Spin OS of the release it runs. The image is built from
[spin-stack/ami](https://github.com/spin-stack/ami) and registered as an AMI in **your** account
and region, tagged `spin:version`:

```bash
git clone https://github.com/spin-stack/ami && cd ami
SPIN_VERSION=v20261019.02 task build publish
```

The build fetches the release, checks every file against its signature, and lays it into a
read-only root filesystem. `publish` refuses a disk built from a tree with uncommitted changes,
or one its manifest does not describe byte for byte.

Publish the image **before** the apply that names the release: the module looks it up by its
tag.

## Pick a release

Releases are named by date: `v<YYYYMMDD>.<N>`, the Nth release of that day. This site documents
`v20261019.02`. See [github.com/spin-stack/spin/releases](https://github.com/spin-stack/spin/releases).
