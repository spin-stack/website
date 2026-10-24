package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// reference is what the site says it documents, written to src/data/reference.json for the pages
// to read: the spin release an installation runs, the commits the reference pages were generated
// from, and how many rows of the feature board are in each state.
type reference struct {
	Spin      spinRef   `json:"spin"`
	Terraform moduleRef `json:"terraform"`
	Brand     sourceRef `json:"brand"`
	Counts    counts    `json:"counts"`
	// OpenAPI is the sha256 of the control plane's OpenAPI document the site serves at
	// /openapi.json, so a reader can check a copy against it.
	OpenAPI string `json:"openapiSha256"`
}

// spinRef keeps apart two things one label once said: the release an installation runs, which
// the install pages name and images.json must have images of, and the tree the reference is
// generated from, which can be main ahead of every release.
type spinRef struct {
	// Installs is the newest release images.json has images of: what `spin_version` names.
	Installs string `json:"installs"`
	// Commit is the commit the reference is generated from.
	Commit string `json:"commit"`
	// Tag is Commit's release tag, empty when no release is cut from it.
	Tag string `json:"tag"`
	// Machine is the spin-machine release Installs pins: what its workspaces' machines are.
	Machine string `json:"machine"`
}

type sourceRef struct {
	Commit string `json:"commit"`
}

// moduleRef is spin-terraform-aws as the site documents it: the commit, and the newest release
// that commit holds, which every page pins the module at.
type moduleRef struct {
	Commit  string `json:"commit"`
	Release string `json:"release"`
}

type counts struct {
	Built    int `json:"built"`
	Partial  int `json:"partial"`
	NotBuilt int `json:"notBuilt"`
	Limits   int `json:"limits"`
	// EndToEnd is the built rows that name a test driving the real stack; Gap, those that say
	// what no test checks yet. Every built row is one or the other.
	EndToEnd int `json:"endToEnd"`
	Gap      int `json:"gap"`
}

// countRows counts the board by state, and refuses a built row that both names end-to-end tests
// and says none checks it, or does neither: the landing page's number is the first kind, and a
// row that is neither would fall out of it without a word.
func countRows(rows []feature) (counts, error) {
	var c counts
	for _, f := range rows {
		switch f.State {
		case "built":
			c.Built++
			switch e2e, gap := len(f.EndToEnd) > 0, f.EndToEndGap != ""; {
			case e2e && gap:
				return c, fmt.Errorf("spin features: built row %q names end-to-end tests and says none checks it", f.Name)
			case e2e:
				c.EndToEnd++
			case gap:
				c.Gap++
			default:
				return c, fmt.Errorf("spin features: built row %q names no end-to-end test and no gap", f.Name)
			}
			continue
		case "partial":
			c.Partial++
		case "not":
			c.NotBuilt++
		case "limit":
			c.Limits++
		default:
			return c, fmt.Errorf("spin features: row %q has state %q, which the site does not know", f.Name, f.State)
		}
		if len(f.EndToEnd) > 0 || f.EndToEndGap != "" {
			return c, fmt.Errorf("spin features: %s row %q says what checks it end to end, which only a built row does", f.State, f.Name)
		}
	}
	return c, nil
}

// spinData is the spin half of reference.json: the commit, its release tag if it has one, the
// board's counts and the OpenAPI document's digest.
func spinData(r *reference, spin string, rows []feature) error {
	var err error
	if r.Spin.Commit, err = git(spin, "rev-parse", "HEAD"); err != nil {
		return err
	}
	tags, err := git(spin, "tag", "--points-at", "HEAD", "--list", "v[0-9]*")
	if err != nil {
		return err
	}
	if r.Spin.Tag, err = newest(strings.Fields(tags)); err != nil {
		return err
	}
	doc, err := os.ReadFile(filepath.Join(spin, openAPIPath))
	if err != nil {
		return err
	}
	r.OpenAPI = fmt.Sprintf("%x", sha256.Sum256(doc))
	r.Counts, err = countRows(rows)
	return err
}

// publicData is the half of reference.json read from the public repositories, which -only
// terraform,brand writes with no spin checkout: the release an installation runs, the commit of
// spin-terraform-aws and its newest release, and brand's commit.
func publicData(r *reference, tf, brand string) error {
	raw, err := os.ReadFile(filepath.Join(tf, "images.json"))
	if err != nil {
		return err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(raw, &all); err != nil {
		return fmt.Errorf("images.json: %w", err)
	}
	releases := make([]string, 0, len(all))
	for k := range all {
		releases = append(releases, k)
	}
	if r.Spin.Installs, err = newest(releases); err != nil {
		return fmt.Errorf("images.json: %w", err)
	}
	if r.Spin.Installs == "" {
		return fmt.Errorf("images.json has no release")
	}
	if r.Terraform.Commit, err = git(tf, "rev-parse", "HEAD"); err != nil {
		return err
	}
	if r.Terraform.Release, err = moduleRelease(tf); err != nil {
		return err
	}
	r.Brand.Commit, err = git(brand, "rev-parse", "HEAD")
	return err
}

// readReference is reference.json as the site has it, for -only terraform,brand to keep its
// spin half: a field this does not know is one that mode would drop.
func readReference(path string) (reference, error) {
	var r reference
	raw, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return r, fmt.Errorf("%s: %w", path, err)
	}
	return r, nil
}

func (r reference) encode() ([]byte, error) {
	b, err := json.MarshalIndent(r, "", "\t")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// newest is the latest of releases, by date and then by number: v20261019.10 is after
// v20261019.9, which comparing the strings would not say. None is "".
func newest(releases []string) (string, error) {
	var best string
	var bestDate, bestN int
	for _, r := range releases {
		m := releaseParts.FindStringSubmatch(r)
		if m == nil {
			return "", fmt.Errorf("%q is not a release, v<YYYYMMDD>.<N>", r)
		}
		date, _ := strconv.Atoi(m[1])
		n, _ := strconv.Atoi(m[2])
		if best == "" || date > bestDate || date == bestDate && n > bestN {
			best, bestDate, bestN = r, date, n
		}
	}
	return best, nil
}

var releaseParts = regexp.MustCompile(`^v(20\d{6})\.(\d+)$`)

func git(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s in %s: %w", strings.Join(args, " "), dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// pinned is a release a page names: a module ref, `?ref=<ref>`, which must be the newest release
// of spin-terraform-aws, or a spin release, v<YYYYMMDD>.<N>, which must be the one installations
// run. A ref that goes on as a placeholder is the form of one a page explains,
// `?ref=v<YYYYMMDD>.<NN>`; a placeholder alone, `?ref=<tag>`, gives a reader nothing to pin.
var pinned = regexp.MustCompile("\\?ref=([A-Za-z0-9._/-]*)(<[^>\\s`\"]*>?)?|\\bv20\\d{6}\\.\\d+\\b")

// pinReleases holds every release a hand-written page names to the one it must be: a page that
// tells people to install an older one is wrong the day a release ships, one that names a spin
// release images.json has no images of tells them to install what cannot boot, and a module ref
// that is no tag of spin-terraform-aws fails at init. Written, it rewrites them; checked, each is
// a line to fix with `task reference`.
func pinReleases(docs, release, module string, check bool) ([]string, error) {
	var bad []string
	err := handPages(docs, func(path string, raw []byte) error {
		// The dashboard's page names the release its screenshots are of: the one the reference
		// documents, which installations may not run yet.
		if rel, _ := filepath.Rel(docs, path); rel == shotsPageDoc {
			return nil
		}
		lines := bytes.Split(raw, []byte("\n"))
		for i, line := range lines {
			lines[i] = pinned.ReplaceAllFunc(line, func(m []byte) []byte {
				sub := pinned.FindSubmatch(m)
				switch {
				case len(sub[1]) > 0 && len(sub[2]) > 0:
					return m
				case bytes.HasPrefix(m, []byte("?ref=")):
					switch {
					case len(sub[1]) == 0:
						bad = append(bad, fmt.Sprintf("%s:%d: %s pins the module at no release; its newest is %s", path, i+1, m, module))
					case string(sub[1]) != module:
						bad = append(bad, fmt.Sprintf("%s:%d: pins the module at %s; its newest release is %s", path, i+1, sub[1], module))
					}
					return []byte("?ref=" + module)
				case string(m) != release:
					bad = append(bad, fmt.Sprintf("%s:%d: names release %s; installations run %s", path, i+1, m, release))
				}
				return []byte(release)
			})
		}
		if check {
			return nil
		}
		if pinned := bytes.Join(lines, []byte("\n")); !bytes.Equal(pinned, raw) {
			return os.WriteFile(path, pinned, 0o644)
		}
		return nil
	})
	if !check {
		return nil, err
	}
	return bad, err
}

// moduleRelease is the newest release of spin-terraform-aws that the checkout tf's HEAD holds:
// what a page pins the module at. A checkout fetched at one commit has no tags; reference:public
// fetches them.
func moduleRelease(tf string) (string, error) {
	tags, err := git(tf, "tag", "--list", "--merged", "HEAD", "v[0-9]*")
	if err != nil {
		return "", err
	}
	r, err := newest(strings.Fields(tags))
	if err != nil {
		return "", fmt.Errorf("spin-terraform-aws's tags: %w", err)
	}
	if r == "" {
		return "", fmt.Errorf("%s holds no release of spin-terraform-aws: fetch its tags (git fetch --tags)", tf)
	}
	return r, nil
}
