package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// securityPages writes the security model's rules and its open items from spin's own
// docs/security/README.md, where code cites each rule at the line that keeps it and a gate fails
// the build when one is broken. A page that restated them went stale within a week of being
// written; these are the README's words, at the commit the site documents.
func securityPages(spin, commit string) (map[string][]byte, error) {
	const readme = "docs/security/README.md"
	raw, err := os.ReadFile(filepath.Join(spin, readme))
	if err != nil {
		return nil, err
	}
	raw = absoluteLinks(raw, commit, path.Dir(readme))
	source := fmt.Sprintf("[`%s`](https://github.com/spin-stack/spin/blob/%s/%s)", readme, commit, readme)

	var rules strings.Builder
	rules.WriteString(frontmatter("spin:"+readme, "Rules and their gates", "The rules every entry point of Spinbox must keep, and the test or lint that fails the build when one is broken.", 1))
	fmt.Fprintf(&rules, "The rules every entry point must keep, from spin's %s at `%s`. Code cites a rule's "+
		"number at the line that keeps it. **Gate** is what fails a build when the rule is broken; "+
		"*none* marks a rule kept only by each place remembering it. What is still open is on "+
		"[Known gaps](/reference/security/known-gaps/).\n\n", source, commit[:9])
	rules.WriteString("| # | Rule | Gate |\n|---|---|---|\n")
	n := 0
	for _, line := range strings.Split(string(raw), "\n") {
		cells := tableCells(line)
		if len(cells) != 4 || !ruleID.MatchString(cells[0]) {
			continue
		}
		n++
		fmt.Fprintf(&rules, "| %s | %s | %s |\n", cells[0], cells[1], cells[3])
	}
	if n == 0 {
		return nil, fmt.Errorf("%s: no table of rules", readme)
	}

	open := section(raw, "## Open")
	if open == nil {
		return nil, fmt.Errorf("%s: no \"## Open\" section", readme)
	}
	var gaps strings.Builder
	gaps.WriteString(frontmatter("spin:"+readme, "Known gaps", "The parts of Spinbox's security model that are not enforced yet, each a design change rather than a patch.", 2))
	fmt.Fprintf(&gaps, "The open items of Spinbox's security model, from spin's %s at `%s`. The numbers are "+
		"[its rules](/reference/security/rules/).\n", source, commit[:9])
	gaps.Write(bytes.TrimRight(open, "\n"))
	gaps.WriteString("\n")

	return map[string][]byte{
		filepath.Join("security", "rules.md"):      []byte(rules.String()),
		filepath.Join("security", "known-gaps.md"): []byte(gaps.String()),
	}, nil
}

var (
	ruleID       = regexp.MustCompile(`^I\d+$`)
	relativeLink = regexp.MustCompile(`\]\(((?:\.\./|\./)?[\w./-]+\.(?:md|svg|dot))(#[\w-]+)?\)`)
)

// tableCells splits a Markdown table row into its trimmed cells, or nil for another line. A
// pipe inside backticks is part of its cell.
func tableCells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	var cells []string
	var cur strings.Builder
	code := false
	for _, r := range line[1 : len(line)-1] {
		switch {
		case r == '`':
			code = !code
		case r == '|' && !code:
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteRune(r)
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

// absoluteLinks points the README's relative links at the files they name in spin, at commit.
func absoluteLinks(raw []byte, commit, dir string) []byte {
	return relativeLink.ReplaceAllFunc(raw, func(m []byte) []byte {
		sub := relativeLink.FindSubmatch(m)
		return []byte(fmt.Sprintf("](https://github.com/spin-stack/spin/blob/%s/%s%s)", commit, path.Join(dir, string(sub[1])), sub[2]))
	})
}
