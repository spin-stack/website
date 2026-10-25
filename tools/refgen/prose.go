package main

import (
	"bytes"
	"encoding/json"
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

// statesPage is the hand-written page whose table of states refgen writes.
var statesPage = filepath.Join("concepts", "workspaces.md")

// stateTable writes the workspaces page's table of states from the API's WorkspaceState: a row
// a value, in the API's order, saying what spin's proto says of it, which the document carries
// as x-enum-descriptions (spin's buf.gen.openapi.yaml asks for with-enum-value-descriptions).
// A state is explained once, where it is defined, and one the API adds is a row the next sync
// brings. The prose around the table is the page's own.
func stateTable(docs, spin string, check bool) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(spin, openAPIPath))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Enum         []string `json:"enum"`
				Descriptions []string `json:"x-enum-descriptions"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", openAPIPath, err)
	}
	state, ok := doc.Components.Schemas["spin.v1.WorkspaceState"]
	if !ok {
		return nil, fmt.Errorf("%s has no spin.v1.WorkspaceState", openAPIPath)
	}
	if len(state.Descriptions) != len(state.Enum) {
		return nil, fmt.Errorf("%s: WorkspaceState has %d values and %d x-enum-descriptions: spin's buf.gen.openapi.yaml asks for with-enum-value-descriptions",
			openAPIPath, len(state.Enum), len(state.Descriptions))
	}
	var table strings.Builder
	table.WriteString("| State | Meaning |\n|---|---|\n")
	for i, v := range state.Enum {
		name := strings.TrimPrefix(v, "WORKSPACE_STATE_")
		if name == "UNSPECIFIED" {
			continue // never a workspace's state
		}
		meaning := strings.Join(strings.Fields(state.Descriptions[i]), " ")
		if meaning == "" {
			return nil, fmt.Errorf("%s: WorkspaceState %s says nothing of what it means: its comment in spin's proto is the row", openAPIPath, v)
		}
		fmt.Fprintf(&table, "| `%s` | %s |\n", name, strings.ReplaceAll(meaning, "|", `\|`))
	}

	page := filepath.Join(docs, statesPage)
	have, err := os.ReadFile(page)
	if err != nil {
		return nil, err
	}
	// The table is the run of | lines that starts at the States section's header row.
	_, after, ok := bytes.Cut(have, []byte("\n## States\n"))
	header := bytes.Index(after, []byte("\n| State | Meaning |\n"))
	if !ok || header < 0 || bytes.Contains(after[:header], []byte("\n#")) {
		return nil, fmt.Errorf("%s: no table of states (| State | Meaning |) under \"## States\"", page)
	}
	begin := len(have) - len(after) + header + 1
	end := begin
	for end < len(have) && have[end] == '|' {
		next := bytes.IndexByte(have[end:], '\n')
		if next < 0 {
			end = len(have)
			break
		}
		end += next + 1
	}
	want := append(append(append([]byte{}, have[:begin]...), table.String()...), have[end:]...)
	differs, err := syncFile(page, want, check)
	if err != nil || !differs {
		return nil, err
	}
	return []string{page}, nil
}

// publicRepos are the spin-stack repositories a page may link to: each public, or made public
// when the site is. spin-stack/ami is not among them: an installation boots the images it
// publishes, and a reader has no use for its source.
var publicRepos = map[string]bool{
	"spin": true, "spin-terraform-aws": true, "spin-machine": true, "qboot": true, "website": true,
}

var repoLink = regexp.MustCompile(`github\.com/spin-stack/([A-Za-z0-9_.-]+)`)

// checkRepos fails a link to a spin-stack repository a reader cannot open.
func checkRepos(docs string) ([]string, error) {
	var bad []string
	err := handPages(docs, func(path string, raw []byte) error {
		for i, line := range bytes.Split(raw, []byte("\n")) {
			for _, m := range repoLink.FindAllSubmatch(line, -1) {
				if !publicRepos[string(m[1])] {
					bad = append(bad, fmt.Sprintf("%s:%d: links spin-stack/%s, which a reader cannot open", path, i+1, m[1]))
				}
			}
		}
		return nil
	})
	return bad, err
}
