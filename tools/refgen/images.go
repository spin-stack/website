package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spin-stack/go-tools/catalog"
	"github.com/spin-stack/go-tools/sigcheck"
)

// catalogRelease is release's file in the Spin OS image catalog, as its signatures say: the file
// the module's spin provider reads for an installation's spin_version. A release's file, not a
// channel's, so the pages change only when the release's own images do.
func catalogRelease(ctx context.Context, root, release string) (catalog.Verified, error) {
	check, err := catalog.Signers()
	if err != nil {
		return catalog.Verified{}, err
	}
	url := strings.TrimSuffix(root, "/") + "/" + catalog.ReleasePath(release)
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return catalog.Verified{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return catalog.Verified{}, fmt.Errorf("the image catalog: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // a read body; nothing to say of its close
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return catalog.Verified{}, fmt.Errorf("the image catalog has no images of %s, the release the site documents: %s", release, url)
	case resp.StatusCode != http.StatusOK:
		return catalog.Verified{}, fmt.Errorf("the image catalog: %s answered %s", url, resp.Status)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return catalog.Verified{}, fmt.Errorf("the image catalog: %s: %w", url, err)
	}
	return verifyRelease(raw, release, check)
}

// verifyRelease is raw, release's file, as check believes it, refused when the build an
// installation of it would boot is revoked: a page that lists it tells the reader to boot it.
func verifyRelease(raw []byte, release string, check sigcheck.Checker) (catalog.Verified, error) {
	v, err := catalog.Verify(raw, release, check)
	if err != nil {
		return v, err
	}
	var errs []error
	for _, role := range catalog.Roles {
		if why := v.Roles[role].Revoked; why != "" {
			errs = append(errs, fmt.Errorf("the image catalog revokes %s of the %s of %s, the build it boots: %s", v.Roles[role].Newest.Revision, role, release, why))
		}
	}
	return v, errors.Join(errs...)
}

// imagesPage writes the Spin OS images the module boots for the release the site documents, from
// the image catalog the module itself reads.
func imagesPage(v catalog.Verified) (map[string][]byte, error) {
	table, err := imagesTable(v)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString(frontmatter("spin-terraform-aws:catalog", "Spin OS images", "Spin OS images by region for this release", "The Spin OS images the module boots for the release this site documents, by region, from the image catalog.", len(tfModules)+1))
	fmt.Fprintf(&b, "Every machine of an installation boots Spin OS, built for one release of spin. The module "+
		"reads the image of each machine from the image catalog on spin-terraform-aws's `catalog` branch, through the spin provider, "+
		"by `spin_version` (a release, or a channel such as `edge`) and region, and boots each role's newest build "+
		"whose signature it checks. An installation of `%s` can be made in a region listed here and nowhere else. "+
		"To boot images of your own, name them with `image_ids` and their measurements with "+
		"`image_measurements` ([the module's variables](/reference/terraform/root/#image_ids)).\n\n", v.Release)
	b.WriteString(table)
	return map[string][]byte{filepath.Join("terraform", "images.md"): []byte(b.String())}, nil
}

// imagesTable is the images of v by region, as a Markdown table: a region is one an installation
// of that release can be made in only when every role's newest build has an image there.
func imagesTable(v catalog.Verified) (string, error) {
	regions := map[string]int{}
	for _, role := range catalog.Roles {
		for region := range v.Roles[role].Newest.Images {
			regions[region]++
		}
	}
	var complete []string
	for region, n := range regions {
		if n == len(catalog.Roles) {
			complete = append(complete, region)
		}
	}
	if len(complete) == 0 {
		return "", fmt.Errorf("the image catalog has no region with every image of %s", v.Release)
	}
	sort.Strings(complete)
	image := func(role, region string) string { return v.Roles[role].Newest.Images[region] }
	var b strings.Builder
	b.WriteString("| Region | Control plane | Proxy | Runner |\n|---|---|---|---|\n")
	for _, region := range complete {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | `%s` |\n", region,
			image("control-plane", region), image("proxy", region), image("runner", region))
	}
	return b.String(), nil
}
