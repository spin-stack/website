---
title: Projects and repositories
description: How a project ties a repository to the shape of its workspace, and how you open one from a URL, prebuild its refs and connect your GitHub account.
reads:
  - "spin:cmd/cli/cmd/project.go@v20261110.01"
  - "spin:cmd/cli/cmd/project_prebuild.go@v20261110.01"
  - "spin:cmd/cli/cmd/open.go@v20261110.01"
  - "spin:cmd/cli/cmd/git.go@v20261110.01"
---

A project is [a repository and the shape of the workspace that opens it](<feature:Projects of the installation: a repository and the shape of the workspace that opens it, shared with everyone or some>).
The shape is the image, the preset and the network policies. The project also names the ref a
workspace opens at.

The guest of a workspace made from a project clones the repository into
`/workspace/<repository name>` and runs the repository's own `.spin.yaml` tasks there.

## Making and sharing a project

Projects belong to the installation. Whoever makes one is its admin. An installation can keep
making projects to its administrators, with `project_creation` in the
[configuration file](/reference/configuration/file/).

```bash
spin project create --name <name> --repository <https-url> --ref <branch> --preset <preset>
spin project list
```

Everybody signed in sees a project until its admins share it with only some people and teams.
A share is saved whole: what you do not name is no longer shared. Admins always see the project.

```bash
spin project share <slug> --team <team> --admin <user-id>
spin project share <slug> --everyone --admin <user-id>
```

Whoever sees a project opens workspaces from it, in a team of theirs. See
[spin project](/reference/cli/project/).

## Opening from a URL

You can [open a repository, a branch, a commit or a pull request from its URL](<feature:Open a repository, branch, commit or pull request from its URL, in the dashboard (/launch) and from the terminal>):
on the dashboard's `/launch` page, or from the terminal with [spin open](/reference/cli/open/).

```bash
spin open https://github.com/<owner>/<repository>
spin open https://github.com/<owner>/<repository>/tree/<branch>
spin open https://github.com/<owner>/<repository>/commit/<sha>
spin open https://github.com/<owner>/<repository>/pull/<n>
```

The project is found by its repository among the projects you see. When more than one has it,
`--project` says which. A repository no project has is one to make a project of first.

A plain repository URL opens at the project's ref. A workspace can also be
[opened at a commit or at a pull request, which is checked out as `pr-<n>`](<feature:A workspace opened at a commit or a pull request, checked out as pr-%3Cn%3E>).

The workspace is made in `--team`, or in your personal team, and sized by the project's preset
unless `--preset` names another. `spin open` ends in `spin ssh`,
unless you pass `--no-connect`.

To open a project by its slug instead of a URL:

```bash
spin workspace create --project <slug> --ref <branch-or-tag>
```

## Prebuilds

A project can name refs to prebuild. Each one is built as it moves, and the result is a
snapshot. [A workspace that opens the project at that ref is cloned from its prebuild](<feature:Prebuilds of a project's refs, as the floor of the workspaces that open them>),
so it starts from what the build already made. The builds run as the person who named the refs.

```bash
spin project create --name <name> --repository <https-url> --ref <branch> --prebuild-ref <branch>
spin project prebuilds <slug>
spin project prebuild <slug> <ref>
```

`spin project prebuilds` lists each build of a ref at a commit and what became of it.
`spin project prebuild` builds a ref now, at the commit the forge says it is at. A commit whose
build failed is built again. Only the project's admins can run it.

An installation can retire a ready prebuild that no workspace has opened for a while. The
project's next open then starts from the image. See [Snapshots](/concepts/snapshots/).

## Your GitHub account

A private repository on GitHub is cloned with the workspace owner's own account. You
[connect your GitHub account once](<feature:Your GitHub account, connected once: private repositories cloned and pushed with nothing on the disk>),
and it clones and pushes your workspaces' private repositories. The control plane keeps the
token. The host your workspace runs on sets it on the requests a network policy brokers to
GitHub, so no workspace's disk holds it.

Connect in the dashboard, under Settings → Git, or
[from the CLI, with a code you enter on GitHub's page](<feature:Your GitHub account connected from the CLI, with a code entered on GitHub's page>):

```bash
spin git connect
spin git list
```

Either way, GitHub asks you to authorize the installation's app. The token never reaches your
machine. `spin git connect` needs "Enable Device Flow" checked on the installation's GitHub App.

To choose a repository for a project,
[list the repositories your account reaches](<feature:A repository picked from the ones your account reaches>),
or only those whose name has a query in it:

```bash
spin git repos
spin git repos <query>
```

`spin git disconnect github` forgets the account and has GitHub revoke the token. See
[spin git](/reference/cli/git/).
