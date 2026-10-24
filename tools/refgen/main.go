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
	"strings"
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
	brand := flag.String("brand", "../brand", "a checkout of spin-stack/brand")
	shotsFrom := flag.String("shots", ".sources/spin-site-shots", "where task sources unpacked spin's release's site-shots.tar.gz")
	only := flag.String("only", "spin,terraform,brand", "the sources to generate from: all three, or terraform,brand, which needs no spin checkout and keeps what the site has of spin")
	out := flag.String("out", "src/content/docs/reference", "where the pages go")
	docs := flag.String("docs", "src/content/docs", "the hand-written pages, whose commands are checked")
	src := flag.String("src", "src", "the site's source, whose components' commands and claims are checked")
	constantsData := flag.String("constants", "src/data/constants.json", "the figures the code decides, for <Value>")
	data := flag.String("data", "src/data/reference.json", "what the site documents, for the pages to read")
	featuresData := flag.String("features", "src/data/features.json", "the feature board, for the pages' feature: links")
	openapi := flag.String("openapi", "public/openapi.json", "where the site serves the control plane's OpenAPI document")
	outputsDir := flag.String("outputs", "src/data/output", "where what the CLI prints, as spin's goldens hold it, is copied")
	assets := flag.String("diagrams", "src/assets/diagrams", "where the diagrams the source repositories draw are copied")
	figuresData := flag.String("figures", "src/data/figures.json", "each diagram's source and words, for its Markdown twin")
	pinned := flag.String("versions", "versions.yaml", "what the sources are pinned at, which reference.json must record")
	check := flag.Bool("check", false, "write nothing; fail if a page on disk differs from what the sources say")
	flag.Parse()

	withSpin, err := parseOnly(*only)
	if err != nil {
		return err
	}
	tfCommit, err := git(*tf, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	from := map[string]checkout{"spin-terraform-aws": {*tf, tfCommit}}
	pages := map[string][]byte{}
	dirs := []string{"terraform"}
	var r reference
	var c constants
	var stale, bad []string
	if withSpin {
		s, err := spinHalf(*spin, *docs, *src, pages, *outputsDir, *featuresData, *openapi, *check)
		if err != nil {
			return err
		}
		r, stale, bad = s.ref, s.stale, s.bad
		from["spin"] = checkout{*spin, r.Spin.Commit}
		shot, err := releaseShots(*spin, *shotsFrom, r.Spin, *check)
		if err != nil {
			return err
		}
		stale = append(stale, shot...)
		dirs = append(dirs, "status", "cli", "configuration", "api", "security")
		if err := spinConstants(&c, *spin); err != nil {
			return err
		}
	} else {
		// The spin half stays as the site has it: what the reference is generated from, and the
		// board's counts, are spin's to say, and only a run with spin's checkout says them.
		if r, err = readReference(*data); err != nil {
			return err
		}
		if c, err = readConstants(*constantsData); err != nil {
			return err
		}
		*spin = ""
	}
	installs := r.Spin.Installs
	if err := publicData(&r, *tf, *brand); err != nil {
		return err
	}
	if withSpin {
		if r.Spin.Machine, err = machineOf(*spin, r.Spin.Installs); err != nil {
			return err
		}
	} else if r.Spin.Installs != installs {
		// What the release is made of is read from spin at its tag.
		bad = append(bad, fmt.Sprintf("images.json's newest release is %s, and %s says %s: run task reference with a spin checkout", r.Spin.Installs, *data, installs))
	}
	if err := terraformConstants(&c, *tf); err != nil {
		return err
	}
	ref, err := r.encode()
	if err != nil {
		return err
	}
	for _, gen := range []func() (map[string][]byte, error){
		func() (map[string][]byte, error) { return terraformPages(*tf, r.Terraform.Release) },
		func() (map[string][]byte, error) { return imagesPage(*tf, r.Spin.Installs) },
		func() (map[string][]byte, error) { return releasePage(r, *tf) },
	} {
		p, err := gen()
		if err != nil {
			return err
		}
		for k, v := range p {
			pages[k] = v
		}
	}
	written, err := sync(*out, dirs, pages, *check)
	if err != nil {
		return err
	}
	stale = append(stale, written...)
	procedures, err := includedPages(*tf, tfCommit)
	if err != nil {
		return err
	}
	figures, err := c.encode()
	if err != nil {
		return err
	}
	files := map[string][]byte{*data: ref, *constantsData: figures}
	for page, raw := range procedures {
		files[filepath.Join(*docs, page)] = raw
	}
	for _, path := range sortedKeys(files) {
		if differs, err := syncFile(path, files[path], *check); err != nil {
			return err
		} else if differs {
			stale = append(stale, path)
		}
	}
	drawn, err := diagramFiles(from, *assets, *figuresData, *check)
	if err != nil {
		return err
	}
	stale = append(stale, drawn...)
	branded, err := brandCopies(*brand, *check)
	if err != nil {
		return err
	}
	stale = append(stale, branded...)
	shotBad, shotStale, err := checkShots(r.Spin, *check)
	if err != nil {
		return err
	}
	stale = append(stale, shotStale...)
	releases, err := pinReleases(*docs, r.Spin.Installs, r.Terraform.Release, *check)
	if err != nil {
		return err
	}
	bad = append(bad, releases...)
	bad = append(bad, shotBad...)
	for _, c := range []func() ([]string, error){
		func() ([]string, error) { return checkDescriptions(*docs) },
		func() ([]string, error) { return checkRepos(*docs) },
		func() ([]string, error) { return checkPins(*pinned, r) },
		func() ([]string, error) { return checkDomains(*docs, *tf) },
		func() ([]string, error) { return checkTofu(*docs, *tf) },
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
		return fmt.Errorf("%d lines on the pages that the sources do not have", len(bad))
	}
	return nil
}

// parseOnly is whether -only takes spin: the half that needs spin's checkout, and builds its
// CLI, is all or nothing, and the public half - spin-terraform-aws and brand - always runs.
func parseOnly(only string) (bool, error) {
	set := map[string]bool{}
	for _, s := range strings.Split(only, ",") {
		switch s = strings.TrimSpace(s); s {
		case "spin", "terraform", "brand":
			set[s] = true
		default:
			return false, fmt.Errorf("-only %s: %q is not spin, terraform or brand", only, s)
		}
	}
	if !set["terraform"] || !set["brand"] {
		return false, fmt.Errorf("-only %s: terraform and brand are always read; spin is the one to leave out", only)
	}
	return set["spin"], nil
}

// spinWritten is what the spin half wrote and found: its half of reference.json, the files
// that differ, and the lines of hand-written pages spin does not have.
type spinWritten struct {
	ref        reference
	stale, bad []string
}

// spinHalf builds spin's CLI and writes, into pages, everything read from spin - the status,
// CLI, configuration, API and security pages - and the files beside them, then checks the
// hand-written pages against spin.
func spinHalf(spin, docs, src string, pages map[string][]byte, outputsDir, featuresData, openapi string, check bool) (spinWritten, error) {
	var w spinWritten
	// The CLI is built once, here, and runs with this empty directory as its HOME.
	home, err := os.MkdirTemp("", "refgen")
	if err != nil {
		return w, err
	}
	defer os.RemoveAll(home)
	bin, err := buildCLI(spin, home)
	if err != nil {
		return w, err
	}
	rows, err := board(bin, home)
	if err != nil {
		return w, err
	}
	if err := spinData(&w.ref, spin, rows); err != nil {
		return w, err
	}
	commit := w.ref.Spin.Commit
	tests, err := testFiles(spin)
	if err != nil {
		return w, err
	}
	for _, gen := range []func() (map[string][]byte, error){
		func() (map[string][]byte, error) { return statusPage(rows, tests, commit) },
		func() (map[string][]byte, error) { return cliPages(bin, home) },
		func() (map[string][]byte, error) { return configPages(spin) },
		func() (map[string][]byte, error) { return apiPages(spin) },
		func() (map[string][]byte, error) { return securityPages(spin, commit) },
	} {
		p, err := gen()
		if err != nil {
			return w, err
		}
		for k, v := range p {
			pages[k] = v
		}
	}
	printed, err := outputFiles(spin, outputsDir, check)
	if err != nil {
		return w, err
	}
	w.stale = append(w.stale, printed...)
	board, err := json.MarshalIndent(rows, "", "\t")
	if err != nil {
		return w, err
	}
	doc, err := os.ReadFile(filepath.Join(spin, openAPIPath))
	if err != nil {
		return w, err
	}
	for path, want := range map[string][]byte{featuresData: append(board, '\n'), openapi: doc} {
		if differs, err := syncFile(path, want, check); err != nil {
			return w, err
		} else if differs {
			w.stale = append(w.stale, path)
		}
	}
	bad, err := checkCommands(docs, src, spin, bin, home)
	if err != nil {
		return w, err
	}
	w.bad = bad
	for _, c := range []func() ([]string, error){
		func() ([]string, error) { return checkLimits(docs, rows) },
		func() ([]string, error) { return checkEnv(docs, spin) },
		func() ([]string, error) { return checkStates(docs, pages) },
		func() ([]string, error) { return checkFeatureLinks(docs, rows) },
		func() ([]string, error) { return checkCardFeatures(docs, src, rows) },
	} {
		b, err := c()
		if err != nil {
			return w, err
		}
		w.bad = append(w.bad, b...)
	}
	return w, nil
}

// sync makes the generated directories dirs under out hold exactly pages: a page the sources no
// longer produce is removed, since a command that is gone must not keep a page.
func sync(out string, dirs []string, pages map[string][]byte, check bool) ([]string, error) {
	var stale []string
	for _, dir := range dirs {
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

// frontmatter is a generated page's head. Every page on the site carries a description: it is
// the page's line in llms.txt. source is what the page is written from, as repository:path; the
// footer links it at the commit the site documents, in place of an edit link, since editing the
// page would be undone by the next `task reference`.
func frontmatter(source, title, description string, order int) string {
	return fmt.Sprintf("---\ntitle: %q\ndescription: %q\nsource: %q\neditUrl: false\nsidebar:\n  order: %d\n---\n\n"+
		"<!-- Generated by tools/refgen from %s. Edit the source, not this file. -->\n\n",
		title, description, source, order, source)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
