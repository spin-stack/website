---
title: The CLI
description: Install the spin CLI, sign in, use profiles for several installations, and get JSON output for scripts.
---

`spin` is one binary. It does everything the dashboard does for a user, and the
administration of runners and storage for an administrator. Every command is listed, with its
own help, in the [CLI reference](/reference/cli/).

## Install

Each release publishes `spin-linux-amd64` and `spin-darwin-arm64`. Download the one for your
machine, make it executable and put it on your `PATH` as `spin`. The Linux build is also in the
release's public package:

```bash
oras pull ghcr.io/spin-stack/spin-release:v20261025.01-spin-linux-amd64
```

`spin version` prints the release it was built from, and the CLI says it on every request. An
installation refuses a CLI more than 90 days older than its own release, and says to install the
installation's. A CLI newer than the installation is served, and says once that what it asks for
and the installation does not know is ignored. Use a CLI of the installation's release.

## Sign in

```bash
spin login --endpoint https://app.<domain>
```

The CLI shows a code and opens your browser at the dashboard, where you approve it. It then
keeps a session token in `~/.spin/config.yaml`. `spin whoami` shows who you are, and
`spin logout` forgets the token.

For a machine with no browser, create an API key in the dashboard (Settings → API Keys) and use
it instead:

```bash
spin login --endpoint https://app.<domain> --token <api-key>
```

## Several installations

A **profile** is a named sign-in. Every command takes `--profile`, and `SPIN_PROFILE` sets it for
a shell:

```bash
spin login --profile work --endpoint https://app.work.example.com
spin --profile work workspace list
```

## Environment

| Variable | What it does |
|---|---|
| `SPIN_ENDPOINT` | The installation to talk to, instead of the profile's |
| `SPIN_TOKEN` | An API key to use instead of a login. No config file is needed. |
| `SPIN_PROFILE` | The profile to use when `--profile` is not given |
| `SPIN_CA_CERT` | A CA certificate to trust for the endpoint, for an installation behind a private CA |

## Output for scripts

`-o json` prints one JSON document of what the API answered, and no prose. Errors still go to
standard error, and the exit status says whether the command succeeded.

```bash
spin -o json workspace get api | jq .
```

## Names and ids

Every command that takes a workspace takes its name or its id.

## Waiting

`create`, `start` and `stop` take `--wait`, which returns once the workspace is where you asked
it to be, and `--timeout` for how long to wait. `spin ssh`, `spin exec` and `spin code` take
`--start` to start a stopped workspace first.
