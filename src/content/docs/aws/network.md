---
title: Network
description: What the internet can reach in an installation, which machine talks to which on what port, and how to close the installation to your own networks.
---

## What the internet reaches

Only the **proxy**, on two ports:

| Port | From | For |
|---|---|---|
| 80 | Anywhere | Let's Encrypt's challenge, and a redirect to 443 |
| 443 | `proxy_allowed_cidrs` (anywhere by default) | The dashboard, the API, workspaces' ports, SSH and terminals |

Nothing else in the installation has an inbound rule for the internet. The control plane's
public address is only a way out. A runner has no inbound rule at all. No machine runs SSH.

## Inside the VPC

| From | To | Port | Who the other end is |
|---|---|---|---|
| Proxy | Control plane | 8080 | TLS under the installation's CA |
| Runners | Control plane | 8080 | TLS under the installation's CA, and the runner's token |
| Runners | Proxy (relay) | 443 | The relay's certificate, and a one-time token per session |
| Proxy, runners | Collector on the control plane | 4317 | Only the security group |
| Control plane | RDS | 5432 | TLS verified against RDS's CA, IAM authentication |
| Control plane, runners | S3 | 443 | Instance roles; the buckets refuse object requests from anywhere but the VPC's gateway endpoint |

Machines find each other by name in a private zone, `spin.internal`: `cp.spin.internal` and
`proxy.spin.internal`, with a ten-second TTL. Each machine writes its own record when it starts
serving, so a replaced machine is a record changed, not an address in every configuration.

Runners reach the relay inside the VPC, at `proxy.spin.internal`, so the proxy's port 443 does
not need to be open to wherever runners are.

## Closing the installation to your networks

Set `proxy_allowed_cidrs` to the networks your users are on:

```hcl
proxy_allowed_cidrs = ["203.0.113.0/24", "198.51.100.10/32"]
```

Port 80 stays open to everyone, for Let's Encrypt.

## Why there is no NAT gateway

Every machine has a public address in a public subnet, and reaches S3 through the free gateway
endpoint. A NAT gateway costs more per month than the machines that run all the time, plus a
charge for every gigabyte, before any workspace runs. A public address is not an open door: the
security groups decide what reaches each machine, and for all but the proxy that is nothing from
outside the VPC.

## What crossed the network is kept

The VPC's flow log goes to CloudWatch for 30 days (`log_retention_days`); `flow_logs = false`
turns it off. `dns_query_logs = true` turns on Route 53 Resolver's query log.

A workspace's own traffic is controlled separately, by its network policies, on the runner. See
[Network and egress](/concepts/network/).
