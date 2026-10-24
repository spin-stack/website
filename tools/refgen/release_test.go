package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The release an installation runs is the newest images.json has, by date and then by number,
// not by the strings' order.
func TestNewestIsByDateThenNumber(t *testing.T) {
	for _, c := range []struct {
		in   []string
		want string
	}{
		{[]string{"v20261019.9", "v20261019.10", "v20261018.11"}, "v20261019.10"},
		{[]string{"v20261019.02", "v20260927.04"}, "v20261019.02"},
		{nil, ""},
	} {
		got, err := newest(c.in)
		if err != nil || got != c.want {
			t.Errorf("newest(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	if _, err := newest([]string{"v20261019.02", "latest"}); err == nil {
		t.Error("newest took a key that is no release")
	}
}

// -only leaves spin out or takes all three: the public half is never left out.
func TestOnlyLeavesOutSpinAndNothingElse(t *testing.T) {
	for only, want := range map[string]bool{"spin,terraform,brand": true, "terraform,brand": false, "brand, terraform": false} {
		got, err := parseOnly(only)
		if err != nil || got != want {
			t.Errorf("parseOnly(%q) = %v, %v; want %v", only, got, err, want)
		}
	}
	for _, only := range []string{"spin", "spin,terraform", "terraform", "terraform,brand,ami"} {
		if _, err := parseOnly(only); err == nil {
			t.Errorf("parseOnly took %q", only)
		}
	}
}

// With no spin checkout, spin's diagrams keep what figures.json has of them, and the others are
// drawn again from their own repository.
func TestDiagramsOfARepositoryNotReadKeepTheirFigures(t *testing.T) {
	dir := t.TempDir()
	tf := filepath.Join(dir, "tf")
	svg := filepath.Join(tf, "docs", "diagrams", "installation.svg")
	if err := os.MkdirAll(filepath.Dir(svg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(svg, []byte(`<svg viewBox="0 0 800 400"><text>control plane</text></svg>`), 0o644); err != nil {
		t.Fatal(err)
	}
	figuresPath := filepath.Join(dir, "figures.json")
	old := map[string]figure{}
	for _, d := range diagrams {
		old[d.name] = figure{Source: "https://github.com/spin-stack/" + d.repo + "/blob/old/" + d.path, Width: 1, Words: []string{"old"}}
	}
	raw, _ := json.Marshal(old)
	if err := os.WriteFile(figuresPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := diagramFiles(map[string]checkout{"spin-terraform-aws": {tf, "new"}}, filepath.Join(dir, "assets"), figuresPath, false); err != nil {
		t.Fatal(err)
	}
	var got map[string]figure
	raw, _ = os.ReadFile(figuresPath)
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if f := got["installation"]; f.Source != "https://github.com/spin-stack/spin-terraform-aws/blob/new/docs/diagrams/installation" || len(f.Words) != 1 || f.Words[0] != "control plane" {
		t.Errorf("installation = %+v, want it drawn again from the checkout", f)
	}
	if f := got["traffic"]; f.Source != old["traffic"].Source || f.Words[0] != "old" {
		t.Errorf("traffic = %+v, want what figures.json had", f)
	}
	if len(got) != len(diagrams) {
		t.Errorf("figures.json has %d diagrams, want %d", len(got), len(diagrams))
	}
}
