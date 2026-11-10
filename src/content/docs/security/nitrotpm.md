---
title: NitroTPM and runner authorization
description: How OpenTofu, NitroTPM, AWS STS and KMS make a runner prove both its AWS identity and the image it booted before it is trusted.
reads:
  - "spin:internal/installation/provider/aws/machine.go@v20261108.01"
  - "spin:internal/installation/provider/aws/awscp/join.go@v20261108.01"
  - "spin:internal/installation/provider/aws/attestedkey/attestedkey.go@v20261108.01"
  - "spin-terraform-aws:modules/controlplane/config.tf@v20261027.02"
  - "spin-terraform-aws:modules/controlplane/attested_key.tf@v20261027.02"
  - "spin-terraform-aws:modules/controlplane/identity.tf@v20261027.02"
---

import NitroTPMDiagram from '../../../components/NitroTPMDiagram.astro';

A runner needs to prove two independent facts before the control plane trusts it:

1. **Which AWS instance is asking.** The instance role signs an STS `GetCallerIdentity` request
   for this installation, its audience and its host id.
2. **What that instance booted.** Its NitroTPM returns an AWS-signed attestation document with
   the measured boot values (PCR4, PCR7 and PCR12), the request's hash and a public key made for
   this boot.

Neither proof is enough by itself. STS does not say what the instance booted; a TPM document does
not say that the machine belongs to this installation. The control plane checks that both describe
the same EC2 instance.

<NitroTPMDiagram />

## What OpenTofu deploys

OpenTofu does not enable NitroTPM or distribute a TPM token. NitroTPM is part of the EC2
environment. The module instead declares the evidence it will accept:

| Terraform output | Runtime use |
|---|---|
| The trusted PCR4, PCR7 and PCR12 values for each Spin OS build | The control plane writes the runner values into `host_attestation.runner` in its bootstrap document. |
| The runner's instance role and the installation audience | STS identifies the caller and binds its signed request to this installation and host. |
| KMS key policies conditioned on `kms:RecipientAttestation:NitroTPMPCR*` | KMS releases an answer only to a machine whose attestation document carries an allowed complete PCR set. |

The image measurements come from the release's `measurements.json`, or from `image_measurements`
when an installation uses its own images. The module accepts non-revoked builds so an already
running prior build can remain available during an update, but a build removed from that set can
no longer join as a runner.

## How a runner is admitted

At first registration, the runner creates a boot key and signs the STS request with its instance
role. It asks NitroTPM to attest to the hash of that exact request and to the boot public key.
The control plane then verifies all of the following before issuing runner credentials:

- AWS signed the attestation document and it is current.
- Its PCR triplet is one of the trusted runner builds declared by the installation.
- Its nonce is the hash of the signed STS request, so it cannot be reused for another request.
- Its public key is this boot's key, so later runner credentials are tied to the attested boot.
- The STS role is allowed by `host_join`, and the attested module id is the same EC2 instance that
  STS identified.

The control plane records the attestation with the runner's boot key before it returns credentials.
That record lets storage readers verify that a key which wrote an object belongs to an admitted,
measured runner image.

## Why KMS checks the same evidence

The admission check decides whether a runner can participate in the control plane. KMS independently
enforces whether it can open sensitive keys. For a volume or workload-identity key, the runner
creates a short-lived recipient key, has NitroTPM attest to its public half, and sends that document
to KMS. KMS encrypts its answer to that recipient only if the document's PCRs match its key policy.

The same pattern protects the control plane's encryption key: at each start it derives the key from
KMS using an attested recipient. The key is not stored in the database, in OpenTofu state or in a
machine image. A control-plane IAM role on a machine that booted an untrusted image is not enough
to obtain it.

## What this protects against

- **Copied IAM credentials on another machine.** They can sign the STS request, but cannot produce
  a matching document from an allowed runner's NitroTPM, nor make its module id match that other
  machine.
- **A runner booted from a changed disk or image.** Changes affecting the measured kernel, command
  line or Secure Boot chain produce a PCR set the installation and KMS policy do not accept.
- **A captured registration proof.** The nonce binds the proof to the signed request, and the boot
  key binds it to one boot; the control plane rejects a repeat registration from that boot.
- **An untrusted control-plane image reading installation secrets.** KMS will not return the
  installation encryption key to its role without an allowed control-plane attestation.
- **A runner role alone opening volume or identity keys.** A KMS decrypt requires an allowed
  recipient attestation as well as the narrowly scoped role/session that requested it.

## What it does not protect against

NitroTPM attestation is a boot-time authorization boundary, not a cure for every compromise. In
particular, it does not prevent malicious code that has already compromised a legitimate, running
runner from using that runner's current authority. It also does not prevent an AWS account
administrator from changing IAM or KMS policy, and it relies on AWS's Nitro attestation root.

The remaining boundaries are therefore still necessary: a verified, read-only Spin OS image;
least-privilege IAM roles and permissions boundaries; per-workspace VM isolation; short-lived
credentials; and KMS policies that name exact PCR combinations. See [Supply chain](/security/supply-chain/),
[Data and encryption](/security/data/) and [Known gaps](/reference/security/known-gaps/) for the
adjacent controls and their limits.
