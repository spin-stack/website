package main

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// diagrams are the pictures the site shows, each drawn - and rendered - in the repository whose
// code it describes, beside its source (.dot or .d2). The site copies the rendered SVG and never
// draws one of its own: a picture drawn here went stale with the database it showed.
//
// scale is the size a page shows it at, of the size it was drawn at: graphviz's at a little under
// their own, where their words read as body text does and a wide one scrolls; d2's installation
// at the column's width, a map whose details open full size on a click.
var diagrams = []struct {
	name, repo, path string
	scale            float64
}{
	{"installation", "spin-terraform-aws", "docs/diagrams/installation.svg", 0},
	{"traffic", "spin", "docs/security/traffic.svg", 0.75},
	{"datapath", "spin", "docs/network/datapath.svg", 0.75},
	{"egress-policy", "spin", "docs/network/egress-policy.svg", 0.75},
	{"isolation", "spin", "docs/network/isolation.svg", 0.75},
}

// figure is what src/data/figures.json says of a diagram: where it is drawn, and its words in
// the order the picture has them, which a Markdown twin carries in its place.
type figure struct {
	Source string `json:"source"`
	// Width is the size a page shows it at, in CSS pixels; 0 is the column's width.
	Width int      `json:"width"`
	Words []string `json:"words"`
}

// viewBox is an SVG's own box: its width is the size the picture was drawn at.
var viewBox = regexp.MustCompile(`viewBox="\s*[-\d.]+[\s,]+[-\d.]+[\s,]+([\d.]+)`)

var svgText = regexp.MustCompile(`(?s)<text\b[^>]*>(.*?)</text>`)
var svgTag = regexp.MustCompile(`<[^>]+>`)

// diagramFiles copies each diagram into src/assets/diagrams and writes src/data/figures.json, and
// says which of them differ from what is on disk. A diagram with no words fails: its twin would
// say nothing.
func diagramFiles(spin, tf, spinCommit, tfCommit, assets, data string, check bool) ([]string, error) {
	figures := map[string]figure{}
	var stale []string
	for _, d := range diagrams {
		root, commit := spin, spinCommit
		if d.repo == "spin-terraform-aws" {
			root, commit = tf, tfCommit
		}
		raw, err := os.ReadFile(filepath.Join(root, d.path))
		if err != nil {
			return nil, err
		}
		var words []string
		for _, m := range svgText.FindAllSubmatch(raw, -1) {
			w := strings.Join(strings.Fields(html.UnescapeString(svgTag.ReplaceAllString(string(m[1]), " "))), " ")
			if w != "" {
				words = append(words, w)
			}
		}
		if len(words) == 0 {
			return nil, fmt.Errorf("%s/%s has no words for a reader with no picture", d.repo, d.path)
		}
		vb := viewBox.FindSubmatch(raw)
		if vb == nil {
			return nil, fmt.Errorf("%s/%s has no viewBox", d.repo, d.path)
		}
		var width float64
		if _, err := fmt.Sscan(string(vb[1]), &width); err != nil {
			return nil, err
		}
		figures[d.name] = figure{
			Width:  int(width * d.scale),
			Source: fmt.Sprintf("https://github.com/spin-stack/%s/blob/%s/%s", d.repo, commit, strings.TrimSuffix(d.path, ".svg")),
			Words:  words,
		}
		path := filepath.Join(assets, d.name+".svg")
		if differs, err := syncFile(path, raw, check); err != nil {
			return nil, err
		} else if differs {
			stale = append(stale, path)
		}
	}
	b, err := json.MarshalIndent(figures, "", "\t")
	if err != nil {
		return nil, err
	}
	if differs, err := syncFile(data, append(b, '\n'), check); err != nil {
		return nil, err
	} else if differs {
		stale = append(stale, data)
	}
	return stale, nil
}
