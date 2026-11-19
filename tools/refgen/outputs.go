package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

// outputs are what spin's CLI prints that the site shows as its output: goldens a test in spin
// holds the command to (cmd/cli/cmd's TestWhatSnapshotDiffPrints), copied as they are. The site
// shows output it did not write, and a change to what the command prints reaches it.
var outputs = []struct{ from, to string }{
	{"cmd/cli/cmd/testdata/diff.text.golden", "diff.txt"},
	{"cmd/cli/cmd/testdata/diff.json.golden", "diff.json"},
}

// outputFiles copies each into dir and says which differ from what is on disk.
func outputFiles(spin, dir string, check bool) ([]string, error) {
	var stale []string
	for _, o := range outputs {
		raw, err := os.ReadFile(filepath.Join(spin, o.from))
		if err != nil {
			return nil, err
		}
		path := filepath.Join(dir, o.to)
		differs, err := syncFile(path, raw, check)
		if err != nil {
			return nil, err
		}
		if differs {
			stale = append(stale, path)
		}
	}
	return stale, nil
}

var (
	// outputTag is a page's <Output name="..." caption="..." .../>; a caption may hold a >.
	outputTag  = regexp.MustCompile(`<Output((?:\s+\w+="[^"]*")*)\s*/>`)
	outputAttr = regexp.MustCompile(`(\w+)="([^"]*)"`)
)

// checkOutputMarks holds what a page marks in a command's output (mark=, ins=, del= of
// <Output>) to the golden it shows: each must be text the output has. Expressive Code marks
// nothing, and says nothing, when the text is not there, so a golden that changes would take the
// mark away and leave the page pointing at nothing.
func checkOutputMarks(docs, dir string) ([]string, error) {
	var bad []string
	err := handPages(docs, func(path string, raw []byte) error {
		for i, line := range bytes.Split(raw, []byte("\n")) {
			for _, tag := range outputTag.FindAllSubmatch(line, -1) {
				attrs := map[string]string{}
				var marks [][2]string
				for _, a := range outputAttr.FindAllSubmatch(tag[1], -1) {
					k, v := string(a[1]), string(a[2])
					attrs[k] = v
					if k == "mark" || k == "ins" || k == "del" {
						marks = append(marks, [2]string{k, v})
					}
				}
				if len(marks) == 0 {
					continue
				}
				out, err := os.ReadFile(filepath.Join(dir, attrs["name"]))
				if errors.Is(err, fs.ErrNotExist) {
					bad = append(bad, fmt.Sprintf("%s:%d: no output %q in %s", path, i+1, attrs["name"], dir))
					continue
				} else if err != nil {
					return err
				}
				for _, m := range marks {
					if m[1] == "" || !bytes.Contains(out, []byte(m[1])) {
						bad = append(bad, fmt.Sprintf("%s:%d: %s=%q is not in %s", path, i+1, m[0], m[1], attrs["name"]))
					}
				}
			}
		}
		return nil
	})
	return bad, err
}
