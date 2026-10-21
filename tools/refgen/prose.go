package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// handPages calls fn with every hand-written page under docs: the reference is refgen's own.
func handPages(docs string, fn func(path string, raw []byte) error) error {
	return filepath.WalkDir(docs, func(path string, d fs.DirEntry, err error) error {
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
		return fn(path, raw)
	})
}

// maxDescription is the longest a page's description may be: it is the page's one line in
// llms.txt, and the line a search result shows.
const maxDescription = 160

var description = regexp.MustCompile(`(?m)^description: (.*)$`)

// checkDescriptions holds every hand-written page's description to one plain line: no longer
// than maxDescription, and one sentence's worth, with no semicolon or dash joining two.
func checkDescriptions(docs string) ([]string, error) {
	var bad []string
	err := handPages(docs, func(path string, raw []byte) error {
		m := description.FindSubmatch(raw)
		if m == nil {
			bad = append(bad, path+": no description, which is its line in llms.txt")
			return nil
		}
		d := strings.Trim(string(m[1]), `"'`)
		if n := len([]rune(d)); n > maxDescription {
			bad = append(bad, fmt.Sprintf("%s: the description is %d characters, more than %d", path, n, maxDescription))
		}
		if strings.ContainsAny(d, ";—") {
			bad = append(bad, path+": the description joins two sentences with a semicolon or a dash")
		}
		return nil
	})
	return bad, err
}

var envName = regexp.MustCompile(`\bSPIN_[A-Z0-9_]+\b`)

// checkEnv holds every SPIN_ variable a page names to spin's own list of what it reads from the
// environment (hack/env-catalog.txt, which spin's lint holds to the code): a variable the page
// tells people to set must be one something reads.
func checkEnv(docs, spin string) ([]string, error) {
	catalog, err := os.ReadFile(filepath.Join(spin, "hack", "env-catalog.txt"))
	if err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, m := range envName.FindAll(catalog, -1) {
		known[string(m)] = true
	}
	var bad []string
	err = handPages(docs, func(path string, raw []byte) error {
		for i, line := range bytes.Split(raw, []byte("\n")) {
			for _, m := range envName.FindAll(line, -1) {
				if !known[string(m)] {
					bad = append(bad, fmt.Sprintf("%s:%d: %s is no variable spin reads (hack/env-catalog.txt)", path, i+1, m))
				}
			}
		}
		return nil
	})
	return bad, err
}

var (
	stateRow  = regexp.MustCompile("(?m)^\\| `([A-Z_]+)` \\|")
	enumValue = regexp.MustCompile("`WORKSPACE_STATE_([A-Z_]+)`")
)

// checkStates holds the workspaces page's table of states to the API's WorkspaceState, both
// ways: the table says what each state means, which the API does not, and a state the API adds
// is one the table must explain.
func checkStates(docs string, pages map[string][]byte) ([]string, error) {
	page := filepath.Join(docs, "concepts", "workspaces.md")
	raw, err := os.ReadFile(page)
	if err != nil {
		return nil, err
	}
	types := pages[filepath.Join("api", "types.md")]
	enum := section(types, "## WorkspaceState")
	if enum == nil {
		return []string{"the API reference has no WorkspaceState"}, nil
	}
	want := map[string]bool{}
	for _, m := range enumValue.FindAllSubmatch(enum, -1) {
		if s := string(m[1]); s != "UNSPECIFIED" {
			want[s] = true
		}
	}
	table := section(raw, "## States")
	if table == nil {
		return []string{page + `: no "## States" section`}, nil
	}
	have := map[string]bool{}
	for _, m := range stateRow.FindAllSubmatch(table, -1) {
		have[string(m[1])] = true
	}
	var bad []string
	for s := range have {
		if !want[s] {
			bad = append(bad, fmt.Sprintf("%s: the state %s is not one of the API's WorkspaceState", page, s))
		}
	}
	for s := range want {
		if !have[s] {
			bad = append(bad, fmt.Sprintf("%s: the API's WorkspaceState %s has no row in the table of states", page, s))
		}
	}
	return bad, nil
}
