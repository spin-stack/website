package main

import (
	"bytes"
	"fmt"
	"regexp"
)

// featureLink is a page's claim that Spin does something, written as a link to the row of the
// feature board that says so: [idle suspend](<feature:Idle suspend, with a question before it that a keep answers>).
// The build turns it into a link to the row's area on the status page (src/lib/feature-links.mjs).
var featureLink = regexp.MustCompile(`\]\(<?feature:([^>)]+)>?\)`)

// checkFeatureLinks fails a claim whose row the board does not have, and one whose row is not
// built: a page may say Spin does what the board says it does, and nothing else.
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
