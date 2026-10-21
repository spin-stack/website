---
title: Remove
description: Delete an installation on purpose: what refuses to be destroyed, why, and the steps to remove it anyway.
---

Some parts of an installation refuse to be destroyed, because a plan that replaced them by
mistake would leave an installation nothing can open:

- **The encryption key.** With the database, it is the installation: a new key is a database that
  cannot be opened.
- **The CA.** Its certificate is in every machine's configuration.
- **The database**, which has deletion protection.
- **The volumes bucket**, whose every object is under Object Lock.

To remove an installation on purpose:

```bash
tofu apply -var 'database={deletion_protection=false}'
tofu state rm module.spin.module.controlplane.aws_ssm_parameter.encryption_key \
  module.spin.module.controlplane.tls_private_key.ca \
  module.spin.module.controlplane.tls_self_signed_cert.ca
tofu destroy
aws ssm delete-parameter --name /<name>/controlplane-encryption-key
```

The destroy stops at the volumes bucket. Its objects are kept until the Object Lock retention
(`object_lock_days`, 30 by default) runs out, and images carry a legal hold. To remove it now,
lift the holds and delete every version bypassing the governance retention, with credentials
allowed `s3:PutObjectLegalHold` and `s3:BypassGovernanceRetention`, then destroy again:

```bash
B=<name>-volumes-<account>-<region>
aws s3api list-object-versions --bucket $B --query 'Versions[].[Key,VersionId]' --output text |
  while IFS=$'\t' read -r k v; do
    aws s3api put-object-legal-hold --bucket $B --key "$k" --version-id "$v" --legal-hold Status=OFF
  done
aws s3api list-object-versions --bucket $B \
  --query '{Objects: [Versions, DeleteMarkers][][].{Key: Key, VersionId: VersionId}}' --output json > v.json
aws s3api delete-objects --bucket $B --bypass-governance-retention --delete file://v.json
tofu destroy
```

What is left is the database's final snapshot. The next installation in the account should use
another `name`.

The state bucket and its key, from `bootstrap`, are shared by every installation in the account
and are not removed.
