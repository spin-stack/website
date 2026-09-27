---
title: Troubleshooting
description: Where to look when an installation does not come up, a workspace waits for a host, a start fails, or a workspace cannot reach something.
---

## A machine does not come up

Each machine's group says `InService` when the machine is serving. What its boot said is kept in
CloudWatch, even for machines that are gone:

```bash
tofu output -raw boot_log   # prints the command that tails every machine's boot log
aws autoscaling describe-scaling-activities --auto-scaling-group-name <group> --max-items 3
```

A machine refuses to start on a document that names another release than its image's. That
means the Spin OS image and `spin_version` disagree: publish the image of the release you named.

## A shell on a machine

No machine runs SSH. Open a shell with Session Manager, on the machine in service:

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

- **No runner is up.** The control plane has asked for one. The first runner takes a minute or
  two to boot and register.
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
spin workspace status <name>     # which party disagrees: control plane, runner, disk, supervisor, machine
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

It lists what was refused, with the policy that refused it, and the lookups that failed, with
why. A destination refused by policy is fixed with a policy (`spin workspace retag`), not with the
network. The record lives only while the workspace runs: start it and try again before reading
it.

## A runner is wedged

Replace its machine: terminate it without lowering the group's size, and the group starts
another, which joins as a new host. Drain it first if it holds workspaces you want kept.

```bash
aws autoscaling terminate-instance-in-auto-scaling-group \
  --instance-id <i-...> --no-should-decrement-desired-capacity
```
