package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// imageRoles are the machines an installation boots, each from an image of its own.
var imageRoles = []string{"control-plane", "proxy", "runner"}

// imagesPage writes the Spin OS images the module boots for the release the site documents,
// from spin-terraform-aws's images.json, which the module itself reads: a region is one an
// installation of that release can be made in only when every role has an image there.
func imagesPage(tf, release string) (map[string][]byte, error) {
	raw, err := os.ReadFile(filepath.Join(tf, "images.json"))
	if err != nil {
		return nil, err
	}
	var all map[string]map[string]map[string]string
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("images.json: %w", err)
	}
	byRole, ok := all[release]
	if !ok {
		return nil, fmt.Errorf("images.json has no images of %s, the release the site documents", release)
	}
	regions := map[string]int{}
	for _, role := range imageRoles {
		if len(byRole[role]) == 0 {
			return nil, fmt.Errorf("images.json has no %s image of %s", role, release)
		}
		for region := range byRole[role] {
			regions[region]++
		}
	}
	var complete []string
	for region, n := range regions {
		if n == len(imageRoles) {
			complete = append(complete, region)
		}
	}
	if len(complete) == 0 {
		return nil, fmt.Errorf("images.json has no region with every image of %s", release)
	}
	sort.Strings(complete)

	var b strings.Builder
	b.WriteString(frontmatter("spin-terraform-aws:images.json", "Spin OS images", "The Spin OS images the module boots for the release this site documents, by region, from images.json.", len(tfModules)+1))
	fmt.Fprintf(&b, "Every machine of an installation boots Spin OS, built for one release of spin. The module "+
		"reads the image of each machine from its `images.json`, by `spin_version` and region, so "+
		"an installation of `%s` can be made in a region listed here and nowhere else. "+
		"To boot images of your own, name them with `image_ids` and their measurements with "+
		"`image_measurements` ([the module's variables](/reference/terraform/root/#image_ids)).\n\n", release)
	b.WriteString("| Region | Control plane | Proxy | Runner |\n|---|---|---|---|\n")
	for _, region := range complete {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | `%s` |\n", region,
			byRole["control-plane"][region], byRole["proxy"][region], byRole["runner"][region])
	}
	return map[string][]byte{filepath.Join("terraform", "images.md"): []byte(b.String())}, nil
}
