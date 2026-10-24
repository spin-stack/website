package main

import (
	"strings"
	"testing"
)

const versionsYAML = `- name: spin-machine
  kind: download
  source: https://github.com/spin-stack/spin-machine/releases/download/{version}/spin-machine-{version}-linux-x86_64.tar.gz
  version: v20261014.01
  pin: 10b1685e82b8b3e34785e0a7a17d618bac711aa9edee5f13856d1586e345a44e
  track: github-release https://github.com/spin-stack/spin-machine
`

// The machine of a release is the one spin's versions.yaml pins at that release's tag, and a
// checkout without the tag says to fetch it rather than read another tree.
func TestTheMachineIsTheReleasesPin(t *testing.T) {
	spin := gitRepo(t, map[string]string{"versions.yaml": versionsYAML}, "v20261019.02")
	got, err := machineOf(spin, "v20261019.02")
	if err != nil || got != "v20261014.01" {
		t.Errorf("machineOf = %q, %v", got, err)
	}
	if _, err := machineOf(spin, "v20261020.01"); err == nil || !strings.Contains(err.Error(), "fetch spin's tags") {
		t.Errorf("a release the checkout has no tag of gave %v", err)
	}
}

// What a release is made of names each part's release and where it was read, and the images of
// the regions that have every role's.
func TestTheReleasePageNamesEachPart(t *testing.T) {
	r := reference{Spin: spinRef{Installs: "v20261019.02", Machine: "v20261014.01"}, Terraform: moduleRef{Release: "v20261024.01"}}
	pages, err := releasePage(r, fixtureTF)
	if err != nil {
		t.Fatal(err)
	}
	page := string(pages["release.md"])
	for _, want := range []string{
		"An installation runs Spinbox `v20261019.02`",
		"[`v20261014.01`](https://github.com/spin-stack/spin-machine/releases/tag/v20261014.01) | spin's `versions.yaml` at `v20261019.02`",
		"[`v20261024.01`](https://github.com/spin-stack/spin-terraform-aws/releases/tag/v20261024.01)",
		"| `us-east-1` | `ami-1` | `ami-2` | `ami-3` |",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("release.md has no %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "eu-west-1") {
		t.Errorf("release.md lists a region without every role's image:\n%s", page)
	}
	r.Spin.Machine = ""
	if _, err := releasePage(r, fixtureTF); err == nil {
		t.Error("a release with no machine was written")
	}
}
