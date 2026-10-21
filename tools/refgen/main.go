// Command refgen writes the site's reference pages from what they describe: the CLI's own
// help, the Terraform modules' variables and outputs, the installation's example config file,
// the control plane's OpenAPI document and spin's feature board (`spin features`). A reference
// page written by hand is a list nothing checks; these are regenerated, and -check fails when one
// on disk is not what the sources say. The hand-written pages are held to the same sources: every
// spin and task command they quote must be one spin has, and every bullet of the limitations page
// is a limit row of the board.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "refgen:", err)
		os.Exit(1)
	}
}

func run() error {
	spin := flag.String("spin", "../spin", "a checkout of spin-stack/spin")
	tf := flag.String("terraform", "../spin-terraform-aws", "a checkout of spin-stack/spin-terraform-aws")
	out := flag.String("out", "src/content/docs/reference", "where the pages go")
	docs := flag.String("docs", "src/content/docs", "the hand-written pages, whose commands are checked")
	data := flag.String("data", "src/data/reference.json", "what the site documents, for the pages to read")
	check := flag.Bool("check", false, "write nothing; fail if a page on disk differs from what the sources say")
	flag.Parse()

	// The CLI is built once, here, and runs with this empty directory as its HOME.
	home, err := os.MkdirTemp("", "refgen")
	if err != nil {
		return err
	}
	defer os.RemoveAll(home)
	bin, err := buildCLI(*spin, home)
	if err != nil {
		return err
	}

	rows, err := board(bin, home)
	if err != nil {
		return err
	}
	commit, err := git(*spin, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	pages, err := statusPage(rows, *spin, commit)
	if err != nil {
		return err
	}
	for _, gen := range []func() (map[string][]byte, error){
		func() (map[string][]byte, error) { return cliPages(bin, home) },
		func() (map[string][]byte, error) { return terraformPages(*tf) },
		func() (map[string][]byte, error) { return configPages(*spin) },
		func() (map[string][]byte, error) { return apiPages(*spin) },
	} {
		p, err := gen()
		if err != nil {
			return err
		}
		for k, v := range p {
			pages[k] = v
		}
	}

	ref, err := referenceData(*spin, *tf, rows)
	if err != nil {
		return err
	}
	var r reference
	if err := json.Unmarshal(ref, &r); err != nil {
		return err
	}
	images, err := imagesPage(*tf, r.Spin.Release)
	if err != nil {
		return err
	}
	for k, v := range images {
		pages[k] = v
	}
	stale, err := sync(*out, pages, *check)
	if err != nil {
		return err
	}
	if differs, err := syncFile(*data, ref, *check); err != nil {
		return err
	} else if differs {
		stale = append(stale, *data)
	}
	releases, err := pinReleases(*docs, r.Spin.Release, *check)
	if err != nil {
		return err
	}
	bad, err := checkCommands(*docs, *spin, bin, home)
	if err != nil {
		return err
	}
	limits, err := checkLimits(*docs, rows)
	if err != nil {
		return err
	}
	bad = append(bad, limits...)
	bad = append(bad, releases...)
	for _, c := range []func() ([]string, error){
		func() ([]string, error) { return checkDescriptions(*docs) },
		func() ([]string, error) { return checkEnv(*docs, *spin) },
		func() ([]string, error) { return checkStates(*docs, pages) },
	} {
		b, err := c()
		if err != nil {
			return err
		}
		bad = append(bad, b...)
	}
	for _, b := range bad {
		fmt.Fprintln(os.Stderr, "refgen:", b)
	}
	switch {
	case *check && len(stale) > 0:
		return fmt.Errorf("out of date, run `task reference`: %v", stale)
	case len(bad) > 0:
		return fmt.Errorf("%d lines on the pages that spin does not have", len(bad))
	}
	return nil
}

// sync makes the generated directories under out hold exactly pages: a page the sources no
// longer produce is removed, since a command that is gone must not keep a page.
func sync(out string, pages map[string][]byte, check bool) ([]string, error) {
	var stale []string
	for _, dir := range []string{"status", "cli", "terraform", "configuration", "api"} {
		matches, err := filepath.Glob(filepath.Join(out, dir, "*.md"))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			rel, err := filepath.Rel(out, m)
			if err != nil {
				return nil, err
			}
			if _, ok := pages[rel]; ok {
				continue
			}
			stale = append(stale, rel)
			if !check {
				if err := os.Remove(m); err != nil {
					return nil, err
				}
			}
		}
	}
	names := make([]string, 0, len(pages))
	for k := range pages {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, rel := range names {
		p := filepath.Join(out, rel)
		have, err := os.ReadFile(p)
		if err == nil && bytes.Equal(have, pages[rel]) {
			continue
		}
		stale = append(stale, rel)
		if check {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(p, pages[rel], 0o644); err != nil {
			return nil, err
		}
	}
	return stale, nil
}

// syncFile is sync for one file outside the reference pages: it says whether the file differs
// from want, and writes it unless checking.
func syncFile(path string, want []byte, check bool) (bool, error) {
	have, err := os.ReadFile(path)
	if err == nil && bytes.Equal(have, want) {
		return false, nil
	}
	if check {
		return true, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return true, err
	}
	return true, os.WriteFile(path, want, 0o644)
}

// frontmatter is a page's head. Every page on the site carries a description: it is the
// page's line in llms.txt.
func frontmatter(title, description string, order int) string {
	return fmt.Sprintf("---\ntitle: %q\ndescription: %q\nsidebar:\n  order: %d\n---\n\n"+
		"<!-- Generated by tools/refgen. Edit the source it names, not this file. -->\n\n",
		title, description, order)
}
