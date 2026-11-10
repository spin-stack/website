package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// featureLink is a page's claim that Spinbox does something, written as a link to the row of the
// feature board that says so: [idle suspend](<feature:Idle suspend, with a question before it that a keep answers>).
// The build turns it into a link to the row's area on the status page (src/lib/feature-links.mjs).
var featureLink = regexp.MustCompile(`\]\(<?feature:([^>)]+)>?\)`)

// checkFeatureLinks fails a claim whose row the board does not have, and one whose row is not
// built: a page may say Spinbox does what the board says it does, and nothing else.
func checkFeatureLinks(docs string, rows []feature) ([]string, error) {
	state := map[string]string{}
	for _, r := range rows {
		state[r.Name] = r.State
	}
	var bad []string
	err := handPages(docs, func(path string, raw []byte) error {
		for i, line := range bytes.Split(raw, []byte("\n")) {
			for _, m := range featureLink.FindAllSubmatch(line, -1) {
				name := string(m[1])
				switch s, ok := state[name]; {
				case !ok:
					bad = append(bad, fmt.Sprintf("%s:%d: no row of spin's feature board is %q", path, i+1, name))
				case s != "built":
					bad = append(bad, fmt.Sprintf("%s:%d: %q is %s on spin's feature board, not built", path, i+1, name, s))
				}
			}
		}
		return nil
	})
	return bad, err
}

// unsaidList is the built rows of the board that no hand-written page says yet, one name a
// line: what the site owes its readers. It may shrink. A row leaves it by being linked from the
// page that says it; a row that is neither linked nor here fails, so a feature spin ships
// reaches a page or is written down as owed.
const unsaidList = "tools/refgen/unsaid.txt"

// checkSaid holds the board to the pages the other way round from checkFeatureLinks: every built
// row is said somewhere - a feature: link of a page, a feature= prop, a step of the story - or
// is in list, and list names nothing else. Before it, three quarters of what the board called
// built was in the generated reference and on no page a reader starts from.
func checkSaid(docs, src, list string, rows []feature) ([]string, error) {
	said := map[string]bool{}
	err := handPages(docs, func(path string, raw []byte) error {
		for _, re := range []*regexp.Regexp{featureLink, featureProp} {
			for _, m := range re.FindAllSubmatch(raw, -1) {
				said[string(m[1])] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	story, err := os.ReadFile(filepath.Join(src, "lib", "story.ts"))
	if err != nil {
		return nil, err
	}
	for _, m := range storyFeature.FindAllStringSubmatch(string(story), -1) {
		said[unescape(m[1]+m[2])] = true
	}
	raw, err := os.ReadFile(list)
	if err != nil {
		return nil, err
	}
	built := map[string]bool{}
	for _, r := range rows {
		built[r.Name] = r.State == "built"
	}
	var bad []string
	owed := map[string]bool{}
	for i, line := range strings.Split(string(raw), "\n") {
		name := strings.TrimSpace(line)
		switch {
		case name == "" || strings.HasPrefix(name, "#"):
			continue
		case !built[name]:
			bad = append(bad, fmt.Sprintf("%s:%d: %q is no built row of spin's feature board: take the line out", list, i+1, name))
		case said[name]:
			bad = append(bad, fmt.Sprintf("%s:%d: %q is said by a page now: take the line out", list, i+1, name))
		}
		owed[name] = true
	}
	for _, r := range rows {
		if r.State == "built" && !said[r.Name] && !owed[r.Name] {
			bad = append(bad, fmt.Sprintf("%s: spin's feature board has built %q and no page says it: link it from the page that does, [...](<feature:%s>), or add the line", list, r.Name, r.Name))
		}
	}
	return bad, nil
}
