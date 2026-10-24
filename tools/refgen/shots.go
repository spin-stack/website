package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The dashboard's screenshots are taken by spin's own dashboard suite, of its e2e stack - the
// user's real workspaces on a real runner - by dashboard/e2e/site.spec.ts, each test checking
// the screen shows what its title says before it shoots. spin's site-shots.yml takes them of
// every release on the KVM fleet and puts them on it as site-shots.tar.gz, which task sources
// unpacks beside the spin checkout. refgen copies them from there with the release, the commit
// and each test's title (releaseShots), writes the dashboard's page from them, and fails when
// they are of another release than the site documents (checkShots). So a release's pictures
// arrive with the pin that moves to it, and nobody runs a stack to take them.
const (
	shotsSpec = "dashboard/e2e/site.spec.ts"
	shotsDir  = "src/assets/shots"
	shotsData = "src/data/shots.json"
	// shotsPageDoc is the dashboard's page under the docs, written by refgen: it names the
	// release its screenshots are of, which pinReleases leaves as it is.
	shotsPageDoc = "use/dashboard.md"
	shotsPage    = "src/content/docs/" + shotsPageDoc
)

type shots struct {
	Release string `json:"release"`
	Commit  string `json:"commit"`
	Shots   []shot `json:"shots"`
}

type shot struct {
	Name string `json:"name"`
	Test string `json:"test"`
}

// specShots reads, from the spec, each test's title and the name its shoot() call writes.
var specTest = regexp.MustCompile(`(?s)test\("([^"]+)",.*?shoot\(\w+, "([\w-]+)"\)`)

func specShots(spin string) ([]shot, error) {
	raw, err := os.ReadFile(filepath.Join(spin, shotsSpec))
	if err != nil {
		return nil, err
	}
	var out []shot
	for _, m := range specTest.FindAllSubmatch(raw, -1) {
		out = append(out, shot{Name: string(m[2]), Test: string(m[1])})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no test that shoots", shotsSpec)
	}
	return out, nil
}

// shotsStamp is the file task sources writes in the screenshots' directory once they are all
// there: "<tag> <pin>\n" of the release they were unpacked from.
const shotsStamp = ".release"

// releaseShots copies, from the directory task sources unpacked spin's release's
// site-shots.tar.gz into, the screenshot of every test of the spec that shoots one, and no
// other, and records them in shots.json with the release and commit spin is at. It fails when
// the directory is of another release, or spin is at none: a screenshot is the documented
// release's or the site has none.
func releaseShots(spin, from string, at spinRef, check bool) (stale []string, err error) {
	if at.Tag == "" {
		return nil, fmt.Errorf("spin %s is no release, and the dashboard's screenshots are a release's: pin spin at one", at.Commit)
	}
	raw, err := os.ReadFile(filepath.Join(from, shotsStamp))
	if err != nil {
		return nil, fmt.Errorf("%s has no screenshots of a release (%w): run task sources", from, err)
	}
	if got, want := string(raw), at.Tag+" "+at.Commit+"\n"; got != want {
		return nil, fmt.Errorf("%s holds the screenshots of %q, and spin is at %q: run task sources", from, strings.TrimSpace(got), strings.TrimSpace(want))
	}
	inSpec, err := specShots(spin)
	if err != nil {
		return nil, err
	}
	keep := map[string]bool{}
	for _, t := range inSpec {
		png, err := os.ReadFile(filepath.Join(from, t.Name+".png"))
		if err != nil {
			return nil, fmt.Errorf("spin %s's site-shots.tar.gz has no %s.png, which %q shoots: retake them with `gh workflow run site-shots.yml -R spin-stack/spin -f tag=%s`", at.Tag, t.Name, t.Test, at.Tag)
		}
		to := filepath.Join(shotsDir, t.Name+".png")
		keep[to] = true
		differs, err := syncFile(to, png, check)
		if err != nil {
			return nil, err
		}
		if differs {
			stale = append(stale, to)
		}
	}
	have, err := filepath.Glob(filepath.Join(shotsDir, "*.png"))
	if err != nil {
		return nil, err
	}
	for _, h := range have {
		if keep[h] {
			continue
		}
		stale = append(stale, h)
		if !check {
			if err := os.Remove(h); err != nil {
				return nil, err
			}
		}
	}
	b, err := json.MarshalIndent(shots{Release: at.Tag, Commit: at.Commit, Shots: inSpec}, "", "\t")
	if err != nil {
		return nil, err
	}
	differs, err := syncFile(shotsData, append(b, '\n'), check)
	if err != nil {
		return nil, err
	}
	if differs {
		stale = append(stale, shotsData)
	}
	return stale, nil
}

// checkShots holds the screenshots shots.json records to the spin release and commit the
// reference documents, and writes the dashboard's page from them. With a spin checkout,
// releaseShots has just written them from the release; without one, what holds the pictures
// to the release is that run's, in Gate with the token, and this holds the record to the
// reference all the same.
func checkShots(at spinRef, check bool) (bad, stale []string, err error) {
	raw, err := os.ReadFile(shotsData)
	if err != nil {
		return nil, nil, err
	}
	var s shots
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", shotsData, err)
	}
	if at.Tag == "" || s.Release != at.Tag || s.Commit != at.Commit {
		bad = append(bad, fmt.Sprintf("%s: the dashboard's screenshots are of spin %s at %s, and the site documents spin %q at %s: run `task reference`", shotsData, s.Release, s.Commit, at.Tag, at.Commit))
	}
	for _, t := range s.Shots {
		if _, err := os.Stat(filepath.Join(shotsDir, t.Name+".png")); err != nil {
			bad = append(bad, fmt.Sprintf("%s: no %s.png", shotsDir, t.Name))
		}
	}

	var b strings.Builder
	b.WriteString("---\ntitle: The dashboard\ndescription: What the dashboard shows, in screenshots spin's own dashboard suite takes of a real installation.\nsource: \"spin:" + shotsSpec + "\"\neditUrl: false\n---\n\n")
	b.WriteString("<!-- Generated by tools/refgen from spin's " + shotsSpec + ". Edit the source, not this file. -->\n\n")
	fmt.Fprintf(&b, "The dashboard is at `https://app.<domain>`. These screenshots are of spin %s, taken by "+
		"spin's dashboard suite on its end-to-end stack - real workspaces on a real runner - each by a "+
		"test that first checks the screen shows what its title says. Everything here is also in the "+
		"[CLI](/reference/cli/) and the [API](/reference/api/).\n\n", s.Release)
	for _, t := range s.Shots {
		title := strings.ToUpper(t.Test[:1]) + t.Test[1:]
		fmt.Fprintf(&b, "## %s\n\n![%s, in the dashboard](../../../assets/shots/%s.png)\n\n", title, title, t.Name)
	}
	differs, err := syncFile(shotsPage, []byte(b.String()), check)
	if err != nil {
		return nil, nil, err
	}
	if differs {
		stale = append(stale, shotsPage)
	}
	return bad, stale, nil
}
