package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A figure a page says in prose has a line, a line has its words on its page, and a line that
// names a constant has that constant's value.
func TestCheckFigures(t *testing.T) {
	const ttl = "package p\n\nimport \"time\"\n\nconst TTL = 15 * time.Minute\n\nconst Tries = 3\n"
	for name, c := range map[string]struct {
		page, list, source, want string
	}{
		"held by a duration":                {"Lasts fifteen minutes.", "p.md | fifteen minutes | spin:p/p.go#TTL = 15m\n", ttl, ""},
		"held by a count":                   {"Tried three times.", "p.md | three times | spin:p/p.go#Tries = 3\n", ttl, ""},
		"listed with why nothing holds it":  {"About 220 MB.", "# sizes\np.md | 220 MB | not held: the size of Microsoft's server\n", ttl, ""},
		"a link's target is not the page's": {"[It stops](<feature:Stopped after half an hour>) and [reads](/a/30-days/).", "", ttl, ""},
		"a flag's own value is the CLI's":   {"Pass `--expires 30 days`.\n\n```bash\nsleep 5 minutes\n```", "", ttl, ""},
		"a figure with no line":             {"Lasts an hour.", "", ttl, `says "an hour", which nothing holds`},
		"the constant moved":                {"Lasts fifteen minutes.", "p.md | fifteen minutes | spin:p/p.go#TTL = 15m\n", strings.Replace(ttl, "15", "10", 1), "TTL is now 10m0s, not 15m"},
		"the count moved":                   {"Tried three times.", "p.md | three times | spin:p/p.go#Tries = 3\n", strings.Replace(ttl, "= 3", "= 5", 1), "Tries is now 5, not 3"},
		"the constant is gone":              {"Lasts fifteen minutes.", "p.md | fifteen minutes | spin:p/p.go#Life = 15m\n", ttl, "no longer declares Life"},
		"the file is gone":                  {"Lasts fifteen minutes.", "p.md | fifteen minutes | spin:q/q.go#TTL = 15m\n", ttl, "spin no longer has q/q.go"},
		"a line the page no longer says":    {"Lasts.", "p.md | fifteen minutes | spin:p/p.go#TTL = 15m\n", ttl, `p.md does not say "fifteen minutes"`},
		"a line with no third place":        {"Lasts.", "p.md | fifteen minutes\n", ttl, "a line is <page>"},
	} {
		t.Run(name, func(t *testing.T) {
			docs, spin := t.TempDir(), t.TempDir()
			if err := os.WriteFile(filepath.Join(docs, "p.md"), []byte("---\ntitle: fifteen minutes\n---\n\n"+c.page+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(spin, "p"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(spin, "p", "p.go"), []byte(c.source), 0o644); err != nil {
				t.Fatal(err)
			}
			list := filepath.Join(spin, "figures.txt")
			if err := os.WriteFile(list, []byte(c.list), 0o644); err != nil {
				t.Fatal(err)
			}
			bad, err := checkFigures(docs, list, spin)
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Join(bad, "\n")
			if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
				t.Errorf("checkFigures said %q, want %q", got, c.want)
			}
		})
	}
}

// Without spin's checkout a line's constant waits for the run that has it; its words do not.
func TestCheckFiguresWithoutSpin(t *testing.T) {
	docs := t.TempDir()
	if err := os.WriteFile(filepath.Join(docs, "p.md"), []byte("---\ntitle: P\n---\n\nLasts fifteen minutes.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(t.TempDir(), "figures.txt")
	if err := os.WriteFile(list, []byte("p.md | fifteen minutes | spin:gone.go#TTL = 15m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if bad, err := checkFigures(docs, list, ""); err != nil || len(bad) > 0 {
		t.Errorf("checkFigures without spin: %v, %v", bad, err)
	}
}
