---
title: Supply chain
description: How a release is built, signed and verified, how Spin OS makes every machine run exactly what the release signed, and what each image is made of.
---

## Releases are signed

A release of Spin is built from a tag, in one workflow, from a cold cache:

- **Every file is signed** with Sigstore, by a short-lived certificate that names the release
  workflow at that tag. There is no signing key to hold or leak.
- **Every signature is verified** in the same run that made it.
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
- **A kernel built for it**, with every driver built in and module loading off once it is up.
- **No cloud-init and no SSH.** A machine's user data is two values, its role and where its
  configuration is, which systemd reads. Nothing in the user data runs.
- **Pinned packages.** Every package is a [Wolfi](https://wolfi.dev) package, pinned by version
  in a lock file, and the image's manifest records every input by content: the commit, each file of
  the release with its digest, the kernel, the package lock and the disk.

Publishing an image refuses a disk built from a tree with uncommitted changes, or one its
manifest does not describe byte for byte.

## The machine a workspace runs

[spin-machine](https://github.com/spin-stack/spin-machine) builds QEMU, the guest kernel and the
base image. A release of Spin pins the machine's release by version and digest. Each machine
release lists every upstream source with its version, URL and SHA-256, and every package in the
base image with its exact version.

## Disks and images

A volume's history is a chain of manifests, each naming its parent by digest, anchored in digests
only the control plane writes to the database. A runner refuses a history that does not reach the
anchor. An image's layers are also measured with fs-verity on the runner's disk, and the kernel
verifies every block read from them.

## Boot, verified

Each role's image is signed for Secure Boot by a key of that role held in KMS, and an instance
boots only what its role's key signed. Every build records the measurements an instance of it
takes in its NitroTPM. A runner joins an installation only by proving those measurements in an
attestation document, and the keys that open volumes are released only to a runner that proves
them. What is still open is on [Known gaps](/reference/security/known-gaps/).

A commit built again comes out with the same root hash, kernel image and boot measurement. The
disk's bytes differ only in the times on its EFI partition, which nothing measures.
