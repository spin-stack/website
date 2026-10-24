package main

import (
	"fmt"
	"strings"

	"github.com/spin-stack/go-tools/versions"
)

// machineOf is the spin-machine release - QEMU, the guest kernel, the firmware and the base
// image, as one artefact - that spin's versions.yaml pins at the tag release: the one the images
// of that release were built with, which spin main may since have moved past.
func machineOf(spin, release string) (string, error) {
	raw, err := git(spin, "show", release+":"+versions.File)
	if err != nil {
		return "", fmt.Errorf("spin's %s at %s (fetch spin's tags): %w", versions.File, release, err)
	}
	v, err := versions.Parse([]byte(raw))
	if err != nil {
		return "", fmt.Errorf("spin's %s at %s: %w", versions.File, release, err)
	}
	e, err := v.Get("spin-machine")
	if err != nil {
		return "", fmt.Errorf("spin's %s at %s: %w", versions.File, release, err)
	}
	return e.Version, nil
}

// releasePage is what the release an installation runs is made of, each part with where the site
// read it: the reader who asks what QEMU or which images an installation has is answered from
// the records the build used, not from a list someone keeps.
func releasePage(r reference, tf string) (map[string][]byte, error) {
	images, err := imagesTable(tf, r.Spin.Installs)
	if err != nil {
		return nil, err
	}
	if r.Spin.Machine == "" {
		return nil, fmt.Errorf("reference.json names no spin-machine of %s: run task reference with a spin checkout", r.Spin.Installs)
	}
	var b strings.Builder
	b.WriteString(frontmatter("spin-terraform-aws:images.json", "What this release is made of",
		"The parts of the Spinbox release an installation runs: spin, its machine, the Spin OS images by region and the OpenTofu module.", 0))
	fmt.Fprintf(&b, "An installation runs Spinbox `%s`, the newest release spin-terraform-aws's `images.json` has images of. It is made of:\n\n", r.Spin.Installs)
	b.WriteString("| Part | Release | Read from |\n|---|---|---|\n")
	fmt.Fprintf(&b, "| spin: the control plane, the runner, the proxy and the CLI | `%s` | spin-terraform-aws's `images.json` |\n", r.Spin.Installs)
	fmt.Fprintf(&b, "| spin-machine: QEMU, the guest kernel, the firmware and the base image a workspace boots, as one artefact | [`%s`](https://github.com/spin-stack/spin-machine/releases/tag/%s) | spin's `versions.yaml` at `%s` |\n", r.Spin.Machine, r.Spin.Machine, r.Spin.Installs)
	fmt.Fprintf(&b, "| The OpenTofu module, spin-terraform-aws | [`%s`](https://github.com/spin-stack/spin-terraform-aws/releases/tag/%s) | its newest release tag |\n\n", r.Terraform.Release, r.Terraform.Release)
	fmt.Fprintf(&b, "The versions of QEMU and the guest kernel in spin-machine `%s` are its own [`versions.yaml`](https://github.com/spin-stack/spin-machine/blob/%s/versions.yaml) at that tag. "+
		"The host's kernel is the Spin OS image's: none of the sources this site is generated from records its version.\n\n", r.Spin.Machine, r.Spin.Machine)
	fmt.Fprintf(&b, "## Spin OS images\n\nEach machine boots the image of its role, in a region that has all three ([Spin OS images](/reference/terraform/images/)):\n\n%s", images)
	return map[string][]byte{"release.md": []byte(b.String())}, nil
}
