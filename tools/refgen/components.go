package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// componentFiles are the site's code outside its pages that shows a reader commands and claims:
// the landing page's story and cards are written there, not in Markdown.
func componentFiles(src string) ([]string, error) {
	var files []string
	for _, glob := range []string{"lib/*.ts", "components/*.astro"} {
		m, err := filepath.Glob(filepath.Join(src, glob))
		if err != nil {
			return nil, err
		}
		files = append(files, m...)
	}
	sort.Strings(files)
	return files, nil
}

// commandLiteral is a string literal of a component that is a spin command: 'spin ...' or
// "spin ...". A template literal is code that builds a sentence, not a command a reader types.
var commandLiteral = regexp.MustCompile(`'(spin [^'\n]*)'|"(spin [^"\n]*)"`)

// componentCommands is every spin command a component's string literals hold, by file, for
// checkCommands to hold to the CLI as it does a page's shell blocks.
func componentCommands(src string) (map[string][]shellLine, error) {
	files, err := componentFiles(src)
	if err != nil {
		return nil, err
	}
	out := map[string][]shellLine{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		for n, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			for _, m := range commandLiteral.FindAllStringSubmatch(line, -1) {
				// Each literal is a command of its own: no cd in one reaches the next.
				out[f] = append(out[f], shellLine{n: n + 1, block: len(out[f]), text: m[1] + m[2]})
			}
		}
	}
	return out, nil
}

var (
	// featureProp is a component's claim on the landing page: <BentoCard feature="...">.
	featureProp = regexp.MustCompile(`\bfeature="([^"]+)"`)
	// storyFeature is a step of the landing page's story: feature: '...' in src/lib/story.ts.
	storyFeature = regexp.MustCompile(`\bfeature:\s*(?:'((?:[^'\\]|\\.)*)'|"((?:[^"\\]|\\.)*)")`)
)

// checkCardFeatures holds the landing page's claims to what is tested end to end: every
// feature= prop of a page and every step of the story must name a built row of the board that
// names an end-to-end test. A claim the landing page makes is one a reader takes as proved; a
// row that only exists, or says what no test checks yet, is not.
func checkCardFeatures(docs, src string, rows []feature) ([]string, error) {
	byName := map[string]feature{}
	for _, r := range rows {
		byName[r.Name] = r
	}
	var bad []string
	claim := func(path string, line int, name string) {
		r, ok := byName[name]
		switch {
		case !ok:
			bad = append(bad, fmt.Sprintf("%s:%d: no row of spin's feature board is %q", path, line, name))
		case r.State != "built":
			bad = append(bad, fmt.Sprintf("%s:%d: %q is %s on spin's feature board, not built", path, line, name, r.State))
		case len(r.EndToEnd) == 0:
			bad = append(bad, fmt.Sprintf("%s:%d: %q names no end-to-end test (%s)", path, line, name, r.EndToEndGap))
		}
	}
	err := handPages(docs, func(path string, raw []byte) error {
		for i, line := range bytes.Split(raw, []byte("\n")) {
			for _, m := range featureProp.FindAllSubmatch(line, -1) {
				claim(path, i+1, string(m[1]))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	story := filepath.Join(src, "lib", "story.ts")
	raw, err := os.ReadFile(story)
	if err != nil {
		return nil, err
	}
	for i, line := range strings.Split(string(raw), "\n") {
		for _, m := range storyFeature.FindAllStringSubmatch(line, -1) {
			claim(story, i+1, unescape(m[1]+m[2]))
		}
	}
	return bad, nil
}

// unescape is a JavaScript string literal's text: \' is ', \\ is \.
func unescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
