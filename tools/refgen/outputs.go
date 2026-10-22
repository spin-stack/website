package main

import (
	"os"
	"path/filepath"
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
