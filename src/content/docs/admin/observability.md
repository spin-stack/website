---
title: Observability
description: The logs, traces and metrics an installation keeps about itself and its workspaces, where they are kept, how long, and how administrators read them.
---

An installation keeps its own logs, traces and metrics. Nothing is sent outside it, and there is
no third-party service or token to hold.

## Where they are kept

Every component sends OpenTelemetry to one collector, Grafana Alloy, on the control plane's
machine. Alloy keeps:

- **Logs and traces** in [Quickwit](https://quickwit.io),
- **Metrics** in [VictoriaMetrics](https://victoriametrics.com),

both on the control plane's machine, listening only on loopback, with their data in the logs
bucket. The metric store is backed up to the bucket every fifteen minutes and when the machine
stops, so a machine replaced by an update takes nothing with it.

| What | Kept for |
|---|---|
| The installation's logs | 30 days |
| Workspaces' console logs | 7 days, when collected |
| Traces | 7 days |
| Metrics | 30 days |

Retention and whether workspaces' console output is collected are
[installation settings](/admin/configuration/).

## Reading them

| Where | What |
|---|---|
| **Admin → Logs** | Search the control plane's, the proxy's and the runners' logs |
| **Admin → Traces** | Find a trace, such as one workspace's start from the request to the guest (`span_name:workspace.start`) |
| **Admin → Metrics** | Grafana, over the metric store, inside the dashboard |
| A workspace's **Logs** tab | What its machine printed |
| `spin admin logs` | The same search as Admin → Logs, from the CLI |
| `spin workspace logs <name>` | A workspace's console, from the CLI |

Grafana is served to administrators alone, through the control plane, which signs each request.
Grafana has no accounts and no password of its own.

```bash
spin admin logs 'severity_text:error' --since 24h
spin admin logs 'service_name:runner AND body.message:"no host"' --limit 20
```

The query language is Quickwit's.

## What is kept

**Admin → Settings → Telemetry** decides which traces and metrics are kept:

- every trace that failed anywhere,
- every trace slower than a threshold,
- a percentage of the rest,
- and metrics dropped by name.

Alloy reads the policy every thirty seconds and decides at the end of each trace. The same page
shows what the collector kept, and failed to, over its last minute. Metrics are pushed every
minute.

## What is not there

- **No alerts.** Nothing pages anyone.
- The collector accepts OpenTelemetry from whatever its security group lets in: the proxy and the
  runners.
- AWS keeps its own records outside the installation: the VPC flow log and every machine's boot
  log, in CloudWatch.
