---
title: Troubleshooting
description: Where to look when an installation does not come up, a workspace waits for a host, a start fails, or a workspace cannot reach something.
reads:
  - "spin-terraform-aws:outputs.tf@v20261027.02"
  - "spin-terraform-aws:modules/runners/main.tf@v20261027.02"
  - "spin:cmd/cli/cmd/status.go@v20261108.01"
  - "spin:docs/runbook.md@v20261108.01"
---

## A machine does not come up

Each machine's group says `InService` when the machine is serving. What its boot said is kept in
CloudWatch, even for machines that are gone:

```bash
tofu output boot_log   # by role, the command that tails its machines' boot logs
aws autoscaling describe-scaling-activities --auto-scaling-group-name <group> --max-items 3
```

A machine refuses to start on a document that names another release than its image's. That
means the image and `spin_version` disagree: the images the module names are those of
`spin_version` unless `image_ids` names others.

## A shell on a machine

No machine runs SSH. The control plane's and the proxy's machines take a shell with Session
Manager, on the one in service. A runner takes none: its image carries no agent and its role
grants none, so its serial console, its boot log and its metrics say what it does, and one that
fails is replaced.

```bash
aws autoscaling describe-auto-scaling-groups \
  --query "AutoScalingGroups[].[AutoScalingGroupName,join(',',Instances[?LifecycleState=='InService'].InstanceId)]" \
  --output text
aws ssm start-session --target <instance-id>
```

Spin OS has the tools to look with — `tcpdump`, `strace`, `ss`, `nft`, `conntrack`, `dig`,
`bpftool`, `htop`, `iostat` — because nothing can be installed on a machine.

## The dashboard does not answer

- `dig +short app.<domain>` should answer with the proxy's address. If not, the domain's
  nameservers are not the zone's yet.
- The proxy gets its certificate a few minutes after DNS resolves. Port 80 must be reachable for
  Let's Encrypt.
- If you set `proxy_allowed_cidrs`, you must be on one of those networks.

## A workspace waits for a host

A workspace waits in *Waiting for a host* while no runner has room for it, and says why.

- **No runner is up.** The control plane has asked for one. The first runner takes a few
  minutes to boot and register.
- **The runner cannot start.** The account's spot vCPU quota is too low, or spot has no capacity
  for the types named. `describe-scaling-activities` on `<name>-runners` says which. See
  [Before you start](/aws/prerequisites/#room-for-runners).
- **Every runner takes no new workspace.** A runner that was drained stays drained until it is
  activated:

```bash
spin admin runner list
spin admin runner activate <runner-id>
```

- **`max_runners` is reached**, and the runners are full.

## A workspace does not start, or misbehaves

```bash
spin workspace status <name>     # which party disagrees: control plane, supervisor, host, volume
spin workspace console <name>    # what the machine printed while it booted
spin workspace history <name>    # every change of state, with the reason
```

A workspace that failed three times is `FAILED` until it is started again. The state names the
part that failed.

An administrator can follow one start from the request to the guest in **Admin → Traces**,
searching for `span_name:workspace.start`.

## A workspace cannot reach something

```bash
spin workspace egress <name>
```

It lists what the workspace reached, what was refused, and the lookups that failed, with why. A destination refused by policy is fixed with a policy (`spin workspace retag`), not with the
network. The record lives only while the workspace runs: start it and try again before reading
it.

## A runner is wedged

Replace its machine: terminate it without lowering the group's size, and the group starts
another, which joins as a new host. Drain it first if it holds workspaces you want kept.

```bash
aws autoscaling terminate-instance-in-auto-scaling-group \
  --instance-id <i-...> --no-should-decrement-desired-capacity
```
