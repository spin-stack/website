package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const spec = `test("a person's workspaces", async ({ page }) => {
	await shoot(page, "workspaces");
});
test("a running workspace", async ({ page }) => {
	await shoot(page, "workspace");
});
`

// site is a working directory with a spin checkout holding the spec, and the screenshots task
// sources unpacked from the release at, with a picture the site has from before.
func site(t *testing.T, at spinRef, stamped string, pngs ...string) (spin, from string) {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	spin, from = filepath.Join(root, "spin"), filepath.Join(root, "shots")
	files := map[string]string{
		filepath.Join(spin, shotsSpec):           spec,
		filepath.Join(from, shotsStamp):          stamped,
		filepath.Join(shotsDir, "dashboard.png"): "a screenshot of an older release",
		shotsData:                                "{}",
	}
	for _, p := range pngs {
		files[filepath.Join(from, p+".png")] = p + " of " + at.Tag
	}
	for p, body := range files {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return spin, from
}

// The site's screenshots are the release's, one for each test of the spec that shoots, recorded
// with the release and commit spin is at; and -check then finds them the release's.
func TestTheScreenshotsAreTheDocumentedReleases(t *testing.T) {
	at := spinRef{Tag: "v20261024.01", Commit: strings.Repeat("b3", 20)}
	spin, from := site(t, at, at.Tag+" "+at.Commit+"\n", "workspaces", "workspace", "extra")
	if _, err := releaseShots(spin, from, at, false); err != nil {
		t.Fatal(err)
	}
	have, _ := filepath.Glob(filepath.Join(shotsDir, "*.png"))
	if want := []string{filepath.Join(shotsDir, "workspace.png"), filepath.Join(shotsDir, "workspaces.png")}; strings.Join(have, " ") != strings.Join(want, " ") {
		t.Errorf("the site has %q; want the spec's two, %q", have, want)
	}
	if got, _ := os.ReadFile(filepath.Join(shotsDir, "workspace.png")); string(got) != "workspace of v20261024.01" {
		t.Errorf("workspace.png is %q; want the release's", got)
	}
	raw, err := os.ReadFile(shotsData)
	if err != nil {
		t.Fatal(err)
	}
	var s shots
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if s.Release != at.Tag || s.Commit != at.Commit || len(s.Shots) != 2 || s.Shots[1] != (shot{Name: "workspace", Test: "a running workspace"}) {
		t.Errorf("shots.json = %+v; want the release, its commit and the spec's tests", s)
	}
	if stale, err := releaseShots(spin, from, at, true); err != nil || len(stale) > 0 {
		t.Errorf("-check after writing = %q, %v; want nothing stale", stale, err)
	}
	if bad, _, err := checkShots(at, false); err != nil || len(bad) > 0 {
		t.Errorf("checkShots = %q, %v; want nothing", bad, err)
	}
}

// Screenshots of another release than the site documents fail: unpacked from another, recorded
// of another, or taken of a spin that is no release.
func TestScreenshotsOfAnotherReleaseFail(t *testing.T) {
	at := spinRef{Tag: "v20261024.01", Commit: strings.Repeat("b3", 20)}
	spin, from := site(t, at, "v20261019.02 "+strings.Repeat("61", 20)+"\n", "workspaces", "workspace")
	if _, err := releaseShots(spin, from, at, true); err == nil || !strings.Contains(err.Error(), "run task sources") {
		t.Errorf("releaseShots of another release's directory = %v; want it refused", err)
	}
	if _, err := releaseShots(spin, from, spinRef{Commit: at.Commit}, true); err == nil || !strings.Contains(err.Error(), "no release") {
		t.Errorf("releaseShots of a spin at no release = %v; want it refused", err)
	}
	old := `{"release": "v20261019.02", "commit": "` + strings.Repeat("61", 20) + `", "shots": []}`
	if err := os.WriteFile(shotsData, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if bad, _, err := checkShots(at, true); err != nil || len(bad) != 1 || !strings.Contains(bad[0], "are of spin v20261019.02") {
		t.Errorf("checkShots of another release's record = %q, %v; want it named", bad, err)
	}
}
