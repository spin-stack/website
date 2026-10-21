package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// reference is what the site says it documents, written to src/data/reference.json for the pages
// to read: the spin release an installation runs, the commits the reference pages were generated
// from, and how many rows of the feature board are in each state.
type reference struct {
	Spin      sourceRef `json:"spin"`
	Terraform sourceRef `json:"terraform"`
	Counts    counts    `json:"counts"`
	// OpenAPI is the sha256 of the control plane's OpenAPI document the site serves at
	// /openapi.json, so a reader can check a copy against it.
	OpenAPI string `json:"openapiSha256"`
}

type sourceRef struct {
	// Release is the newest release tag the commit reaches: what `spin_version` names.
	Release string `json:"release,omitempty"`
	Commit  string `json:"commit"`
}

type counts struct {
	Built    int `json:"built"`
	Partial  int `json:"partial"`
	NotBuilt int `json:"notBuilt"`
	Limits   int `json:"limits"`
}

func referenceData(spin, tf string, rows []feature) ([]byte, error) {
	var r reference
	var err error
	if r.Spin.Commit, err = git(spin, "rev-parse", "HEAD"); err != nil {
		return nil, err
	}
	if r.Spin.Release, err = git(spin, "describe", "--tags", "--abbrev=0", "--match", "v[0-9]*"); err != nil {
		return nil, err
	}
	if r.Terraform.Commit, err = git(tf, "rev-parse", "HEAD"); err != nil {
		return nil, err
	}
	doc, err := os.ReadFile(filepath.Join(spin, openAPIPath))
	if err != nil {
		return nil, err
	}
	r.OpenAPI = fmt.Sprintf("%x", sha256.Sum256(doc))
	for _, f := range rows {
		switch f.State {
		case "built":
			r.Counts.Built++
		case "partial":
			r.Counts.Partial++
		case "not":
			r.Counts.NotBuilt++
		case "limit":
			r.Counts.Limits++
		default:
			return nil, fmt.Errorf("spin features: row %q has state %q, which the site does not know", f.Name, f.State)
		}
	}
	b, err := json.MarshalIndent(r, "", "\t")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func git(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s in %s: %w", strings.Join(args, " "), dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// releaseName is a spin release as a page writes it: v<YYYYMMDD>.<N>.
var releaseName = regexp.MustCompile(`\bv20\d{6}\.\d+\b`)

// pinReleases holds every release a hand-written page names to the one the site documents: a
// page that tells people to install an older one is wrong the day a release ships. Written, it
// rewrites them; checked, each is a line to fix with `task reference`.
func pinReleases(docs, release string, check bool) ([]string, error) {
	var bad []string
	err := filepath.WalkDir(docs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "reference" {
			return fs.SkipDir
		}
		if d.IsDir() || !page(path) {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range bytes.Split(raw, []byte("\n")) {
			for _, m := range releaseName.FindAll(line, -1) {
				if string(m) != release {
					bad = append(bad, fmt.Sprintf("%s:%d: names release %s; the site documents %s", path, i+1, m, release))
				}
			}
		}
		if check {
			return nil
		}
		pinned := releaseName.ReplaceAll(raw, []byte(release))
		if bytes.Equal(pinned, raw) {
			return nil
		}
		return os.WriteFile(path, pinned, 0o644)
	})
	if !check {
		return nil, err
	}
	return bad, err
}
