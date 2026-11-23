package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// anySignature takes every bundle: what is tested here is what the site does with a catalog its
// signatures were already held to, which go-tools' catalog tests.
type anySignature struct{}

func (anySignature) Check(_, _ []byte) error { return nil }

// A release whose newest build of a role is revoked is one no installation boots, so the site
// refuses to list its images rather than send a reader to them.
func TestARevokedBuildIsNoImageToList(t *testing.T) {
	const release = "v20261121.03"
	pcr := strings.Repeat("a", 96)
	signed := func(v any) []byte {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	roles := map[string]any{}
	for _, role := range []string{"control-plane", "proxy", "runner"} {
		r := map[string]any{"builds": map[string]any{"r1": map[string]any{
			"pcr4": pcr, "pcr7": pcr, "pcr12": pcr, "bundle": []byte("b"),
			"statement": signed(map[string]any{"format_version": 1, "release": release, "role": role,
				"measurements": map[string]string{"pcr4": pcr, "pcr7": pcr, "pcr12": pcr}}),
			"images": map[string]string{"us-east-1": "ami-" + role},
		}}}
		if role == "runner" {
			r["revocations"] = map[string]any{"r1": map[string]any{"bundle": []byte("b"),
				"statement": signed(map[string]any{"format_version": 1, "release": release, "role": role, "revision": "r1", "reason": "leaks", "at": "2026-11-22T00:00:00Z"})}}
		}
		roles[role] = r
	}
	raw := signed(map[string]any{"format": 1, "release": release, "roles": roles})
	_, err := verifyRelease(raw, release, anySignature{})
	if err == nil || !strings.Contains(err.Error(), "revokes r1 of the runner") || !strings.Contains(err.Error(), "leaks") {
		t.Errorf("a release with its runner's build revoked gave %v", err)
	}
}
