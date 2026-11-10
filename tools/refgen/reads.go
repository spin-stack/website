package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// A hand-written page says, in its frontmatter, what it was written from and when it was last
// read against it:
//
//	reads:
//	  - "spin:docs/storage/README.md@v20261108.01"
//
// A command, a variable and a feature a page names are each held to the sources by a check of
// their own; what a page says in prose - how long, who may, what happens when - is held by
// nothing, and went stale release by release. This does not read prose either. It says when the
// thing a page describes has changed since somebody read the page against it, which page, and
// the diff to read: the page is corrected, or found still true, and its line stamped again.
//
// A path is a file, not a package: the narrower it is, the less often a page is asked about a
// change that does not touch what it says.

// readLine is one line of a page's reads: the repository, the path in it, and the tag or commit
// the page was last read against.
var readLine = regexp.MustCompile(`^  - "([\w-]+):([^@"\s]+)@([\w.-]+)"$`)

// unsealed are the hand-written pages that read nothing a line could name, each with why. It
// may shrink.
var unsealed = map[string]string{
	"index.mdx":               "the landing page: every claim on it is a row of the board, held by checkCardFeatures",
	"overview/limitations.md": "its bullets are the board's limit rows, both ways, held by checkLimits",
}

type pageRead struct {
	line             int
	repo, path, read string
}

// pageReads is a page's reads, from its frontmatter, and whether refgen wrote the page.
func pageReads(raw []byte) (reads []pageRead, generated bool, err error) {
	lines := strings.Split(string(raw), "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return nil, false, fmt.Errorf("no frontmatter")
	}
	in := false
	for i, l := range lines[1:] {
		switch {
		case l == "---":
			return reads, generated, nil
		case strings.HasPrefix(l, "source:"):
			generated = true
		case l == "reads:":
			in = true
		case in && strings.HasPrefix(l, "  - "):
			m := readLine.FindStringSubmatch(l)
			if m == nil {
				return nil, false, fmt.Errorf("line %d: a read is \"<repository>:<path>@<tag or commit>\"", i+2)
			}
			reads = append(reads, pageRead{i + 2, m[1], m[2], m[3]})
		default:
			in = false
		}
	}
	return nil, false, fmt.Errorf("the frontmatter is not closed")
}

// checkReads fails a hand-written page that names nothing it reads, and one whose source has
// changed since the page was read against it. from is the checkouts at hand: a read of a
// repository that is not among them - spin's, in a run without spin - is left to the run that
// has it.
func checkReads(docs string, from map[string]checkout) ([]string, error) {
	var bad []string
	err := handPages(docs, func(path string, raw []byte) error {
		reads, generated, err := pageReads(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		rel, _ := filepath.Rel(docs, path)
		_, exempt := unsealed[filepath.ToSlash(rel)]
		switch {
		case generated:
			return nil
		case exempt && len(reads) > 0:
			bad = append(bad, fmt.Sprintf("%s: names what it reads, so it leaves refgen's unsealed", path))
		case !exempt && len(reads) == 0:
			bad = append(bad, fmt.Sprintf("%s: names nothing it reads: give it reads, each \"<repository>:<path>@<tag>\", for the files whose change would make it wrong", path))
		}
		for _, r := range reads {
			c, ok := from[r.repo]
			switch {
			case r.repo != "spin" && r.repo != "spin-terraform-aws":
				bad = append(bad, fmt.Sprintf("%s:%d: reads %s, which is not spin or spin-terraform-aws", path, r.line, r.repo))
				continue
			case !ok:
				continue
			}
			why, err := changed(c, r)
			if err != nil {
				return err
			}
			if why != "" {
				bad = append(bad, fmt.Sprintf("%s:%d: %s", path, r.line, why))
			}
		}
		return nil
	})
	return bad, err
}

// changed says what became of a read's file since the page was read against it, or "".
func changed(c checkout, r pageRead) (string, error) {
	if _, err := os.Stat(filepath.Join(c.root, r.path)); err != nil {
		return fmt.Sprintf("%s no longer has %s: read the page against what took its place, and name that", r.repo, r.path), nil
	}
	if _, err := git(c.root, "rev-parse", "--verify", "--quiet", r.read+"^{commit}"); err != nil {
		return fmt.Sprintf("%s has no %s to have read %s at", r.repo, r.read, r.path), nil
	}
	cmd := exec.Command("git", "diff", "--quiet", r.read, "HEAD", "--", r.path)
	cmd.Dir = c.root
	err := cmd.Run()
	if err == nil {
		return "", nil
	}
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
		return "", fmt.Errorf("git diff %s HEAD -- %s in %s: %w", r.read, r.path, c.root, err)
	}
	now, err := stamp(c.root)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s's %s changed since the page was read against it: read `git -C %s diff %s HEAD -- %s`, correct the page, and stamp the line @%s (task reviewed PAGE=<page>)",
		r.repo, r.path, c.root, r.read, r.path, now), nil
}

// stamp is what a page read against the checkout now is stamped with: its release's tag, or its
// commit where no release is cut from it.
func stamp(dir string) (string, error) {
	if tag, err := git(dir, "describe", "--tags", "--exact-match", "HEAD"); err == nil {
		return tag, nil
	}
	return git(dir, "rev-parse", "--short=10", "HEAD")
}

// stampPage rewrites every read of one page to the checkouts as they are: what a person runs
// once they have read the page against what changed.
func stampPage(page string, from map[string]checkout) error {
	raw, err := os.ReadFile(page)
	if err != nil {
		return err
	}
	reads, _, err := pageReads(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", page, err)
	}
	if len(reads) == 0 {
		return fmt.Errorf("%s names nothing it reads", page)
	}
	lines := bytes.Split(raw, []byte("\n"))
	for _, r := range reads {
		c, ok := from[r.repo]
		if !ok {
			return fmt.Errorf("%s:%d: no checkout of %s to stamp it against", page, r.line, r.repo)
		}
		now, err := stamp(c.root)
		if err != nil {
			return err
		}
		lines[r.line-1] = []byte(fmt.Sprintf("  - \"%s:%s@%s\"", r.repo, r.path, now))
	}
	return os.WriteFile(page, bytes.Join(lines, []byte("\n")), 0o644)
}
