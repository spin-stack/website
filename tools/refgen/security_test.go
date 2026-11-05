package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The known gaps are the README's open items and, where its tag has the section, what the model does
// not protect: a limit moved between the two stays on the page.
func TestKnownGapsCarryWhatTheModelDoesNotProtect(t *testing.T) {
	const rules = "| I1 | a rule | keeps | gate |\n"
	for name, tc := range map[string]struct {
		readme string
		want   []string
	}{
		"with the section": {
			readme: rules + "\n## Open\n\n- an open item\n\n## What this does not protect\n\n- a limit\n\n## Next\n\nelse\n",
			want:   []string{"- an open item", "## What this does not protect", "- a limit"},
		},
		"without it": {
			readme: rules + "\n## Open\n\n- an open item\n",
			want:   []string{"- an open item"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			spin := t.TempDir()
			path := filepath.Join(spin, "docs", "security", "README.md")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.readme), 0o644); err != nil {
				t.Fatal(err)
			}
			pages, err := securityPages(spin, "0123456789abcdef")
			if err != nil {
				t.Fatal(err)
			}
			page := string(pages[filepath.Join("security", "known-gaps.md")])
			for _, want := range tc.want {
				if !strings.Contains(page, want) {
					t.Errorf("the page does not say %q:\n%s", want, page)
				}
			}
			if strings.Contains(page, "else") {
				t.Errorf("the page runs past its sections:\n%s", page)
			}
		})
	}
}
