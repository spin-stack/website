---
title: Cost
description: What an installation costs on AWS — the small part that runs all the time and the runners that run only while workspaces do — and the settings that change it.
---

An installation has two kinds of cost: a small one that runs all the time, and runners, which
run only while workspaces do. Prices depend on your region and change; use the
[AWS Pricing Calculator](https://calculator.aws/) with the sizes below.

## All the time

| Resource | Default size |
|---|---|
| Control plane | `t8i.small` |
| Proxy | `t8i.micro`, and an elastic IP |
| RDS PostgreSQL | `db.t4g.micro`, 7 days of backups |
| Route 53 | Two hosted zones, $0.50 a month each |
| S3 | What the volumes, logs and metric backups hold |
| CloudWatch | The flow log and the machines' boot logs, 30 days |

There is no NAT gateway, no load balancer and no interface endpoint.

## While workspaces run

Runners are spot machines of 32 or 48 vCPUs with local NVMe. They are the largest cost, and they
cost only while they are up:

- The control plane starts the first runner when a workspace needs one.
- It adds another only when a workspace has waited longer than a runner takes to start.
- It removes them all when nothing has run for `runner_idle_minutes` (60 by default), and at
  once during `quiet_hours` when nothing is running.

A runner holds many workspaces, so one runner is often enough for a team. `max_runners` caps
how many can be up.

A workspace that nobody stops keeps a runner up: Spin does not stop idle workspaces.

## Settings that change the cost

| Setting | Effect |
|---|---|
| `runner_idle_minutes` | How long runners stay up after the last workspace stops |
| `quiet_hours`, `time_zone` | A window, such as nights, when runners go as soon as nothing runs |
| `max_runners` | The most runners at once |
| `runner_spot = false` | On-demand runners: more expensive, never reclaimed |
| `runner_instance_types` | Smaller or larger runners |
| `instance_type`, `proxy_instance_type` | The size of the control plane and the proxy |
| `database` | The RDS instance class, storage, Multi-AZ and backup retention |
| `flow_logs`, `log_retention_days` | What is kept in CloudWatch |

## Storage

The volumes bucket keeps every object under Object Lock for `object_lock_days` (30 by default),
so a deleted workspace's bytes are paid for until the retention runs out. Clones share their
image's layers, so a hundred workspaces from one image store the image once.
