---
title: Your settings
description: What you set once with spin settings and every workspace of yours applies, and when a workspace is given none of it.
reads:
  - "spin:cmd/cli/cmd/settings.go@v20261108.01"
  - "spin:docs/access.md@v20261108.01"
---

Your settings are yours, not a workspace's. You save them once, and
[every workspace you own applies them as you saved them](<feature:The owner's shell, environment, editor settings, Taskfile and dotfiles reach the guest as they save them>).
A workspace's own `.spin.yaml` goes over them.

## Seeing them

```bash
spin settings get
```

## Shell, editor and environment

```bash
spin settings set shell /bin/zsh
spin settings set editor vim
spin settings set env.GOPROXY direct
```

`shell` is your login shell. `editor` is your default editor. `env.KEY` sets the environment
variable `KEY`.

## Dotfiles

Name a repository, and the command that installs it. The repository is cloned into new workspaces.

```bash
spin settings dotfiles --repo https://github.com/<user>/dotfiles
spin settings dotfiles --repo https://github.com/<user>/dotfiles --branch main --install ./install.sh
spin settings dotfiles --clear
```

Without `--branch` the branch is `main`. Without `--install` the command is `./install.sh`.

## A Taskfile

A startup Taskfile runs when your workspaces start. Check a file before you save it:
`--validate` reads it and saves nothing.

```bash
spin settings taskfile --validate <taskfile.yml>
spin settings taskfile --file <taskfile.yml>
spin settings taskfile --clear
```

## A workspace's own environment

A workspace can hold variables of its own, set when you create it or later. They go over your
environment, in that workspace only: another workspace's environment never reaches it.

```bash
spin workspace create --name <name> --env <KEY>=<value>
spin workspace update <workspace> --env <KEY>=<value>
```

## When a workspace is not given them

A workspace with a guest in it, or cloned from somebody else's snapshot, is given none of your
dotfiles, environment or Taskfile. Letting a guest into a workspace that already applied them
changes what it holds:
[What you set, and what a workspace applies](/use/access/#what-you-set-and-what-a-workspace-applies)
says what is taken out and what stays.

## Reference

[spin settings](/reference/cli/settings/) lists every command and flag.
[SettingsService](/reference/api/settingsservice/) is the same over the API.
