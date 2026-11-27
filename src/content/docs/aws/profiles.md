---
title: Profiles
metaTitle: "Profiles: the module set for one person to a hundred"
description: "How to set the module for one person, one person with agents, a small team, a team of twenty and a hundred people, and what each choice trades."
reads:
  - "spin-terraform-aws:variables.tf@v20261123.01"
  - "spin-terraform-aws:modules/runners/variables.tf@v20261123.01"
  - "spin-terraform-aws:modules/runners/main.tf@v20261123.01"
  - "spin-terraform-aws:modules/controlplane/variables.tf@v20261123.01"
  - "spin-terraform-aws:examples/complete/main.tf@v20261123.01"
  - "spin:internal/controlplane/settings/registry.go@v20261121.03"
  - "spin:internal/controlplane/domain/autoscale/autoscale.go@v20261121.03"
---

The module's defaults are a small installation: at most two runners
([`max_runners`](/reference/terraform/root/#max_runners)), of 32 to 48 vCPUs
([`runner_vcpus`](/reference/terraform/root/#runner_vcpus)), on spot, given back once nothing has
run for [`runner_idle_minutes`](/reference/terraform/root/#runner_idle_minutes). The control plane
asks at once for every runner the waiting workspaces need, up to `max_runners`
([Scaling from zero](/concepts/runners/#scaling-from-zero)), so the number is a ceiling. It trades
the wait once every runner is full against the spot vCPUs `request_quota` asks AWS for, two of
`runner_vcpus`' maximum, 96 with the defaults. While nobody works there are no runners, whatever
it is. Two also leaves room for a spot rebalance to start a runner's replacement while one runner
is up. Each profile below
says who it is for, what to change from those defaults and why, what that trades, and what has not
been tried. [Cost](/aws/cost/) says what each part is billed at; this page does not repeat it.

The settings are arguments of `module "spin"`, in your copy of `examples/complete` (its `main.tf`,
which already sets `runner_idle_minutes` and `max_runners`). Its
`terraform.tfvars` takes only the example's own variables, such as `spin_version`, `domain` and
`runner_spot`. Every argument, with its default, is in the [module reference](/reference/terraform/root/).

Some arguments seed the installation when it is made: `runner_instance_types`, `runner_vcpus` and
`runner_spot` its default host class (Admin → Runners), and `max_runners` and
`runner_idle_minutes` its settings.
On an installation already running, a change to one of them in the module is
[a proposal an administrator applies or dismisses](/admin/configuration/#the-config-file-seeds),
not a change by itself.

Two things no profile changes yet. How many workspaces fit on one runner has not been measured, so
every `max_runners` below is a starting point to correct against what you see. And the control
plane and the proxy run all the time: there is no standby that stops them while nobody works, nor a
smaller installation for one person than the one the module makes.

## What each might cost

A model, not a bill: us-east-1, AWS's prices of 2026-11-19, spot runners at their 30-day median
price ([Spot](/aws/cost/#spot)), the part that runs all the time at the one measurement so far
([With nobody working](/aws/cost/#with-nobody-working)), and, the largest unknown, eight
workspaces on a runner. Each line is in dollars a month.

| Profile | What the model assumes | About | Each person |
|---|---|---|---|
| One person | One or two workspaces through the working day | $165 | $165 |
| One person with agents | The person, and four agents' workspaces through the working day | $230 | $230 |
| Agents around the clock | The person, and three agents' workspaces never stopping | $560 | $560 |
| Small team | Five people | $210 | $42 |
| Team | 22 people in two time zones, a few agents, light CI | $950 | $43 |
| Large | 100 people in three time zones, agents, CI and prebuilds, a larger control plane | $4,100 | $41 |

The share of a person moves most with density: at four workspaces a runner rather than eight, the
team of 22 is about $1,560 a month, and at sixteen about $650. With every runner on demand, each
line is more than double. What a workspace writes to the bucket, and what it sends out
of the region, are guesses too; they matter most for agents.

## One person

**For** a developer on their own, with a workspace or two open while they work.

The defaults fit. What is worth setting:

```hcl
module "spin" {
  # ...
  monthly_budget_usd = 200
  budget_emails      = ["you@example.com"]
}
```

- `monthly_budget_usd` and `budget_emails`: a mail when the month passes 80% or 100% of it, or is
  forecast to. It stops nothing.

**Trades.** The runner goes once nothing has run for `runner_idle_minutes`, at night as by day:
no setting gives it back sooner at certain hours, so each evening it stays up that long after the
last workspace stops. The first workspace after the runner has gone waits while a new one starts
(*Waiting for a host*). A lower `runner_idle_minutes` saves runner hours and makes that wait more
frequent. A buffer on the host class (Admin → Runners) would take the wait away and keep a runner
up all night; the default class keeps none.

**A smaller runner.** Most of one person's bill is a runner of 32 vCPUs or more kept up for a few
cores of work, and the parts that run all the time. A runner of 8 vCPUs would cut the first:

```hcl
  runner_vcpus = { min = 8, max = 8 }
```

The module accepts it, and the default class's group then asks for the 2xlarge sizes of the default families with
local NVMe. In the model above, it takes one person from about $165 to about $100 a month. **No installation has run runners that small.** A preset as large as the runner leaves
nothing for the host itself, so keep presets well under it
([Installation settings](/admin/configuration/)); with `max_runners` at its default, a workspace
that does not fit on the first runner starts another, and once both are full it waits for room.

## One person with agents

**For** a person who runs agents in workspaces of their own beside the one they work in, including
workspaces that run for days.

```hcl
module "spin" {
  # ...
  max_runners        = 2
  runner_rollout     = "rolling"
  monthly_budget_usd = 600
  budget_emails      = ["you@example.com"]
}
```

- `max_runners = 2`, the default, written out: if the agents' workspaces do not fit on one
  runner, the control plane asks for another as soon as one waits, instead of it waiting for room
  on the first ([Scaling from zero](/concepts/runners/#scaling-from-zero)).
- `runner_rollout = "rolling"`: with the default, `idle`, a new release reaches the runners only
  once the fleet empties, which a fleet with a workspace that never stops does not. Rolling
  replaces them at an update, each workspace suspended and resumed on the new host
  ([Update](/aws/update/)).
- A budget: a workspace that runs for days keeps a runner up for days, and the bill shows it.

Not Terraform, and the setting a workspace that runs for days meets first: the installation
[suspends a workspace that has run as long as it allows](<feature:A workspace that ran as long as the installation allows is suspended>),
used or not, 12 hours by default. An administrator sets it, from no cap to a week:

```bash
spin admin settings get workspace_max_running_minutes
spin admin settings set workspace_max_running_minutes 0
```

Zero caps none. The owner of one workspace can instead keep it awake for a while
([Workspaces](/concepts/workspaces/)).

**Trades.** Without the cap, a forgotten workspace keeps its runner up until somebody stops it. On
spot, AWS can take a runner back at any time: [its workspaces are suspended to the bucket and
resumed on another](<feature:A cloud's reclaim warning empties the host before the machine goes>),
which an agent in the middle of a task sees as a pause. `runner_spot = false` avoids that at the
on-demand price ([Spot](/aws/cost/#spot)). A workspace that runs for a long time without stopping
also meets the [disk chain's limit](/overview/limitations/#workspaces).

**Not tried.** The agent features are built and tested, and the least used day to day. Agents
running for days on one installation, and how often spot interrupts a runner over a week, have not
been measured.

## Small team

**For** about five people in one time zone.

```hcl
module "spin" {
  # ...
  max_runners        = 2
  monthly_budget_usd = 400
  budget_emails      = ["ops@example.com"]
}
```

- `max_runners = 2`: at the density assumed above, five people fit on one runner; another covers
  being wrong about it, and busy days.
- The runners go once nothing has run for `runner_idle_minutes` after the last person stops.

**Trades.** On a busy morning the second runner is asked for as soon as a workspace does not fit
on the first, and that workspace waits while it starts. Once every runner is up, a workspace that
does not fit waits for room. A buffer of a workspace or two on the host class (Admin → Runners)
would take away the first wait, and keep a runner up through the night.

**Not tried.** How many workspaces fit on a runner. [The bill](/admin/operations/#the-bill) shows
what each workspace held, and the runners' capacity nobody held, which is the measure to correct
`max_runners` by.

## Team

**For** twenty to twenty-five people, in one or two time zones.

```hcl
module "spin" {
  # ...
  max_runners        = 6
  monthly_budget_usd = 1500
  budget_emails      = ["ops@example.com"]
}
```

- `max_runners = 6`: what the model needs at its busiest hour at eight workspaces a runner. At four
  it would need eleven, at sixteen three. `request_quota` (on by default) asks AWS for the spot
  vCPUs that many runners of `runner_vcpus`' maximum take ([Before you start](/aws/prerequisites/)).
- Runners go once nothing has run for `runner_idle_minutes`, at any hour: a gap between the two
  time zones shorter than that keeps them up, and a longer one empties the fleet, so the next
  person waits for a runner.
- Keep `runner_instance_types` broad: each type and zone it matches is one more spot pool to start
  from ([Spot](/aws/cost/#spot)).

**Trades.** When the morning's workspaces wait, the control plane asks for every runner they need
at once, up to six. Runners are given back one at a time, once a runner has held nothing only it
holds for `runner_idle_minutes`; a lower value gives runners back sooner and starts them more
often.

**Not tried.** Density, as above, and twenty people on one installation.

## Large

**For** about a hundred people across time zones, with agents, CI jobs in workspaces, and
[prebuilds](/concepts/projects/#prebuilds).

```hcl
module "spin" {
  # ...
  max_runners            = 20
  runner_rollout         = "rolling"
  instance_type          = "m8i.xlarge"
  controlplane_volume_gb = 200
  monthly_budget_usd     = 6000
  budget_emails          = ["ops@example.com"]
}
```

- `max_runners = 20`: the model's busiest hour at eight workspaces a runner; between ten and
  forty over the densities it was run at.
- `runner_rollout = "rolling"`: a fleet working across time zones may never empty.
- `instance_type` and `controlplane_volume_gb`: the control plane also runs the database and the
  stores of logs, traces and metrics, and nothing in the module sizes it by how many people use
  it. These two values are a guess, not a measurement: watch its machine
  ([Observability](/admin/observability/)) and size it from that. A change replaces the machine,
  as an [update](/aws/update/) does.

**Trades.** Spot capacity for many runners at once is less sure than for one: AWS's placement
score for the default types fell at 20 runners in us-east-1 and held in us-west-2 on the day it was
read ([Spot](/aws/cost/#spot)). A host class can fall back to on demand when spot is refused
(Admin → Runners). The control plane and the proxy are still one machine each
([Limitations](/overview/limitations/#where-it-runs)). Prebuilds keep their disks in the bucket
until retired, which an administrator can make automatic for unused ones in the installation's
settings.

**Not tried.** A hundred people, the control plane at that load, the proxy
([`proxy_instance_type`](/reference/terraform/controlplane/#proxy_instance_type)) at that
traffic, and twenty runners on spot at once.

## For every profile

- After the installation has run for a while, switch on its cost allocation tags in bootstrap, or
  the Costs page stays empty ([Reading the bill](/aws/cost/#reading-the-bill)).
- [`runner_spot`](/reference/terraform/root/#runner_spot),
  [`runner_instance_types`](/reference/terraform/root/#runner_instance_types) and
  [`runner_vcpus`](/reference/terraform/root/#runner_vcpus) change what a runner is; keep the types
  to one CPU generation, so that a suspended workspace can resume on any of them rather than boot
  cold.
- How long a workspace may sit unused before it is suspended is an installation setting, like the
  running cap above: [Installation settings](/admin/configuration/).
