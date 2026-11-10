package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoAt is a repository with one file, tagged v1 as first written and then as each of then
// leaves it, the last being HEAD with no tag.
func repoAt(t *testing.T, file string, then ...string) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	for i, body := range append([]string{"first\n"}, then...) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, file)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		run("add", ".")
		run("commit", "-q", "-m", "c")
		if i == 0 {
			run("tag", "v1")
		}
	}
	return dir
}

func sealedPage(t *testing.T, docs, name, frontmatter string) string {
	t.Helper()
	path := filepath.Join(docs, name)
	if err := os.WriteFile(path, []byte("---\ntitle: A page\n"+frontmatter+"---\n\nWhat it says.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// A page is asked about exactly when what it reads has changed since it was read against it.
func TestCheckReads(t *testing.T) {
	same, moved := repoAt(t, "docs/a.md"), repoAt(t, "docs/a.md", "second\n")
	for name, c := range map[string]struct {
		spin, frontmatter, want string
	}{
		"read at what the source still is":       {same, "reads:\n  - \"spin:docs/a.md@v1\"\n", ""},
		"the source changed since":               {moved, "reads:\n  - \"spin:docs/a.md@v1\"\n", "docs/a.md changed since the page was read against it"},
		"the file is gone":                       {same, "reads:\n  - \"spin:docs/b.md@v1\"\n", "no longer has docs/b.md"},
		"read at a release the source never had": {same, "reads:\n  - \"spin:docs/a.md@v9\"\n", "has no v9"},
		"names nothing it reads":                 {same, "", "names nothing it reads"},
		"reads another repository":               {same, "reads:\n  - \"ami:docs/a.md@v1\"\n", "not spin or spin-terraform-aws"},
		"a generated page reads nothing":         {same, "source: \"spin:docs/a.md\"\n", ""},
		"a read of a checkout not at hand waits": {same, "reads:\n  - \"spin-terraform-aws:main.tf@v1\"\n", ""},
	} {
		t.Run(name, func(t *testing.T) {
			docs := t.TempDir()
			sealedPage(t, docs, "p.md", c.frontmatter)
			bad, err := checkReads(docs, map[string]checkout{"spin": {root: c.spin}})
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Join(bad, "\n")
			if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
				t.Errorf("checkReads said %q, want %q", got, c.want)
			}
		})
	}
}

// Stamping a page is what answers the check: its reads are of the source as it is.
func TestStampPageAnswersTheCheck(t *testing.T) {
	spin, docs := repoAt(t, "docs/a.md", "second\n"), t.TempDir()
	path := sealedPage(t, docs, "p.md", "reads:\n  - \"spin:docs/a.md@v1\"\n")
	from := map[string]checkout{"spin": {root: spin}}
	if err := stampPage(path, from); err != nil {
		t.Fatal(err)
	}
	if bad, err := checkReads(docs, from); err != nil || len(bad) > 0 {
		t.Errorf("a page just stamped is asked again: %v, %v", bad, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(raw), "---\n\nWhat it says.\n") || strings.Contains(string(raw), "@v1") {
		t.Errorf("stamped page:\n%s", raw)
	}
}

// A read's stamp is not a release the page names: pinReleases leaves it as it is.
func TestPinReleasesLeavesAReadsStamp(t *testing.T) {
	docs := t.TempDir()
	path := sealedPage(t, docs, "p.md", "reads:\n  - \"spin:docs/a.md@v20260101.01\"\n")
	if _, err := pinReleases(docs, "v20260202.02", "v20260303.03", false); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "@v20260101.01") {
		t.Errorf("the stamp was rewritten:\n%s", raw)
	}
}
