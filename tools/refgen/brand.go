package main

import (
	"os"
	"path/filepath"
)

// brandFiles are what the site takes from spin-stack/brand, which checks every color it writes
// (contrast, and perceptual distance on real panels and through colour-vision deficiencies): the
// palette as Starlight's variables and as Spinbox's own, and the organization's mark. A copy here is
// held to brand's, so a palette changed there reaches the site with `task reference`.
var brandFiles = []struct{ from, to string }{
	{"dist/tokens.css", "src/styles/brand-tokens.css"},
	{"dist/starlight.css", "src/styles/brand-starlight.css"},
	// The mark as brand draws it, a vector: sharp in the header and the tab at any size and on a 2x
	// screen. The PNGs are for what takes no SVG: an older browser's tab, iOS's home screen.
	{"logos/mark.svg", "src/assets/logo.svg"},
	{"logos/mark-clear.svg", "src/assets/logo-light.svg"},
	{"logos/mark.svg", "public/favicon.svg"},
	{"logos/favicon-64.png", "public/favicon.png"},
	{"logos/apple-touch-icon.png", "public/apple-touch-icon.png"},
	{"logos/mark-512.png", "public/mark.png"},
	{"shots/shots.json", "src/data/shots.json"},
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
	// The app's screenshots, every one brand keeps, and none it does not.
	pngs, err := filepath.Glob(filepath.Join(brand, "shots", "*.png"))
	if err != nil {
		return nil, err
	}
	keep := map[string]bool{}
	for _, p := range pngs {
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		to := filepath.Join(shotsDir, filepath.Base(p))
		keep[to] = true
		differs, err := syncFile(to, raw, check)
		if err != nil {
			return nil, err
		}
		if differs {
			stale = append(stale, to)
		}
	}
	have, _ := filepath.Glob(filepath.Join(shotsDir, "*.png"))
	for _, h := range have {
		if keep[h] {
			continue
		}
		stale = append(stale, h)
		if !check {
			if err := os.Remove(h); err != nil {
				return nil, err
			}
		}
	}
	return stale, nil
}
