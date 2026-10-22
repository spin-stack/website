package main

import (
	"os"
	"path/filepath"
)

// brandFiles are what the site takes from spin-stack/brand, which checks every color it writes
// (contrast, and perceptual distance on real panels and through colour-vision deficiencies): the
// palette as Starlight's variables and as Spin's own, and the organization's mark. A copy here is
// held to brand's, so a palette changed there reaches the site with `task reference`.
var brandFiles = []struct{ from, to string }{
	{"dist/tokens.css", "src/styles/brand-tokens.css"},
	{"dist/starlight.css", "src/styles/brand-starlight.css"},
	{"logos/mark-256.png", "src/assets/logo.png"},
	{"logos/mark-clear-256.png", "src/assets/logo-light.png"},
	{"logos/mark-512.png", "public/mark.png"},
	{"logos/favicon-64.png", "public/favicon.png"},
}

func brandCopies(brand string, check bool) ([]string, error) {
	var stale []string
	for _, f := range brandFiles {
		raw, err := os.ReadFile(filepath.Join(brand, f.from))
		if err != nil {
			return nil, err
		}
		differs, err := syncFile(f.to, raw, check)
		if err != nil {
			return nil, err
		}
		if differs {
			stale = append(stale, f.to)
		}
	}
	return stale, nil
}
