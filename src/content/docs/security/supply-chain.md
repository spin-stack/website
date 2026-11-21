---
title: Supply chain
description: How a release is built, signed and verified, how Spin OS makes every machine run exactly what the release signed, and what each image is made of.
reads:
  - "spin:docs/releases.md@v20261114.01"
  - "spin:docs/machine.md@v20261114.01"
---

## Releases are signed

A release of Spinbox is built from a tag, in one workflow, from a cold cache:

- **Every file is signed** with Sigstore, by a short-lived certificate that names the release
  workflow at that tag. There is no signing key to hold or leak.
- **Every signature is verified** in the same run that made it.
- **A release refuses a commit the tests on real KVM machines have not passed**: those run on
  `main`'s head every hour, and a commit merged since is released once they pass on it.
- **Tarballs are the same bytes for the same inputs**: names sorted, times and owners zeroed.
- Every third-party GitHub Action is pinned by commit, and a lint fails the build otherwise.

You can check a file yourself:

```bash
cosign verify-blob spin-controlplane-linux-amd64.tar.gz \
  --bundle spin-controlplane-linux-amd64.tar.gz.sigstore.json \
  --certificate-identity https://github.com/spin-stack/spin/.github/workflows/release.yml@refs/tags/<version> \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

`spin-boot`, which lays a release into an image, makes the same check in Go before it unpacks
anything.

## Spin OS

Every machine of an installation boots **Spin OS**, built for one release:

- **A read-only root under dm-verity.** The root filesystem is EROFS with its hash tree beside
  it. The kernel checks every block against a root hash in its own command line, and stops the
  machine on one that does not match. Only `/var` and `/run` are writable, and nothing runs from `/var`.
- **The release is inside the image.** It is laid in when the image is built, and each file is
  checked against its signature then. A machine downloads and installs nothing when it boots.
- **A kernel built for it**, with every driver built in and no support for loadable modules at all.
- **No cloud-init and no SSH.** A machine's user data is two values, its role and where its
  configuration is, which systemd reads. Nothing in the user data runs.
- **Pinned packages.** Every package is a [Wolfi](https://wolfi.dev) package, pinned by version
  in a lock file, and the image's manifest records every input by content: the commit, each file of
  the release with its digest, the kernel, the package lock and the disk.

Publishing an image refuses a disk built from a tree with uncommitted changes, or one its
manifest does not describe byte for byte.

## The machine a workspace runs

[spin-machine](https://github.com/spin-stack/spin-machine) builds QEMU, the guest kernel and the
base image. A release of Spinbox pins the machine's release by version and digest. Each machine
release lists every upstream source with its version, URL and SHA-256, and every package in the
base image with its exact version.

## Boot, verified

Each role's image is signed for Secure Boot by a key of that role held in KMS, and an instance
boots only what its role's key signed. Every build records the measurements an instance of it
takes in its NitroTPM, and [NitroTPM](/security/nitrotpm/) is what they decide: which runner
joins, and which machine the keys are released to. What is still open is on
[Known gaps](/reference/security/known-gaps/).

A commit built again comes out with the same root hash, kernel image and boot measurement. The
disk's bytes differ only in the times on its EFI partition, which nothing measures.
