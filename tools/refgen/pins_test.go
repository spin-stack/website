package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pinsFile = `- name: spin
  kind: git
  source: https://github.com/spin-stack/spin
  version: main
  pin: 77380b13da641c24e3f79b9c7586c39888f4e7ba
  track: github-release
- name: spin-terraform-aws
  kind: git
  source: https://github.com/spin-stack/spin-terraform-aws
  version: v20261024.01
  pin: 3df623f0ca2b7f91650e0ac857ba0bde295e0481
  track: github-release
- name: brand
  kind: git
  source: https://github.com/spin-stack/brand
  version: main
  pin: c882b6f0f3901787525ef16ef29058c9c1418cbe
  track: branch
`

// reference.json records the commits versions.yaml pins, and the release a pin at one names.
func TestReferenceRecordsThePins(t *testing.T) {
	file := filepath.Join(t.TempDir(), "versions.yaml")
	if err := os.WriteFile(file, []byte(pinsFile), 0o644); err != nil {
		t.Fatal(err)
	}
	var r reference
	r.Spin.Commit = "77380b13da641c24e3f79b9c7586c39888f4e7ba"
	r.Terraform.Commit, r.Terraform.Release = "3df623f0ca2b7f91650e0ac857ba0bde295e0481", "v20261024.01"
	r.Brand.Commit = "c882b6f0f3901787525ef16ef29058c9c1418cbe"
	if bad, err := checkPins(file, r); err != nil || len(bad) != 0 {
		t.Fatalf("checkPins of the pins = %q, %v; want nothing", bad, err)
	}

	moved := r
	moved.Brand.Commit = strings.Repeat("ab", 20)
	moved.Terraform.Release = "v20261019.01"
	bad, err := checkPins(file, moved)
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) != 2 || !strings.Contains(bad[0], "terraform.release") || !strings.Contains(bad[1], "brand at "+moved.Brand.Commit) {
		t.Errorf("checkPins = %q; want terraform.release and brand's commit", bad)
	}
}
