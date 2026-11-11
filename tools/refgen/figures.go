package main

import (
	"bytes"
	"fmt"
	"go/constant"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// figuresList is every duration, size and count a hand-written page says in its prose, each with
// what decides it: a line is
//
//	<page> | <the words on the page> | spin:<file>#<Constant> = <its value>
//
// for a figure a constant of spin's decides, which refgen reads and compares - 15m, 12h, 300, 3 -
// or, in the third place, why nothing holds it: a default of a file refgen cannot evaluate, or
// words that are no figure of the code's. A page that showed a figure with <Value> has no line:
// the component is the code's own number.
//
// Before it a page said "an hour" of a credential the code had made fifteen minutes, and "about
// 32 MiB" of a commit the code makes on a clock, and nothing read either.
const figuresList = "tools/refgen/figures.txt"

// saidFigure is a duration, a size or a count of tries, in digits or in words.
var saidFigure = regexp.MustCompile(`(?i)\b(?:(?:\d+(?:[.,]\d+)?|an?|one|two|three|four|five|six|seven|eight|nine|ten|twelve|fourteen|fifteen|twenty|thirty|sixty|ninety|half an?|a quarter of an|a few|few)[ -](?:(?:second|minute|hour|day|week|month|year)s?|times|attempts)|\d+(?:\.\d+)? ?[KMGT]i?B)\b`)

// held is a figure's third place when a constant decides it.
var held = regexp.MustCompile(`^spin:(\S+)#(\w+) = (\S+)$`)

type figureLine struct {
	n                 int
	page, words, what string
}

// checkFigures holds every figure in the prose of a hand-written page to a line of list, every
// line to words its page still says, and every line that names a constant to that constant's
// value in the checkout spin. spin is "" in a run without it, which leaves the constants to the
// run that has it.
func checkFigures(docs, list, spin string) ([]string, error) {
	raw, err := os.ReadFile(list)
	if err != nil {
		return nil, err
	}
	var lines []figureLine
	listed := map[string]bool{}
	var bad []string
	for i, l := range strings.Split(string(raw), "\n") {
		if l = strings.TrimSpace(l); l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		f := strings.SplitN(l, " | ", 3)
		if len(f) != 3 || strings.TrimSpace(f[2]) == "" {
			bad = append(bad, fmt.Sprintf("%s:%d: a line is <page> | <words> | <what decides it, or why nothing holds it>", list, i+1))
			continue
		}
		fl := figureLine{i + 1, strings.TrimSpace(f[0]), strings.ToLower(strings.TrimSpace(f[1])), strings.TrimSpace(f[2])}
		lines = append(lines, fl)
		listed[fl.page+"|"+fl.words] = true
	}
	said := map[string]bool{}
	err = handPages(docs, func(path string, raw []byte) error {
		if sourceOf.Match(raw) {
			return nil
		}
		rel, _ := filepath.Rel(docs, path)
		rel = filepath.ToSlash(rel)
		for _, f := range proseFigures(raw) {
			key := rel + "|" + strings.ToLower(f.words)
			if said[key] {
				continue
			}
			said[key] = true
			if !listed[key] {
				bad = append(bad, fmt.Sprintf("%s:%d: says %q, which nothing holds: show it with <Value>, or give it a line in %s naming what decides it", path, f.n, f.words, list))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, l := range lines {
		if !said[l.page+"|"+l.words] {
			bad = append(bad, fmt.Sprintf("%s:%d: %s does not say %q: take the line out", list, l.n, l.page, l.words))
			continue
		}
		m := held.FindStringSubmatch(l.what)
		if m == nil || spin == "" {
			continue
		}
		why, err := constantIs(spin, m[1], m[2], m[3])
		if err != nil {
			return nil, err
		}
		if why != "" {
			bad = append(bad, fmt.Sprintf("%s:%d: %s says %q, and %s: correct the page and the line", list, l.n, l.page, l.words, why))
		}
	}
	return bad, nil
}

// sourceOf is the frontmatter line of a page refgen wrote.
var sourceOf = regexp.MustCompile(`(?m)^source: `)

type proseFigure struct {
	n     int
	words string
}

// proseFigures is the figures a page says outside its frontmatter, its code blocks and its code
// spans: a command's own flag is the CLI's to hold.
func proseFigures(raw []byte) []proseFigure {
	var out []proseFigure
	front, fence := 0, false
	for i, line := range bytes.Split(raw, []byte("\n")) {
		switch {
		case i == 0 && string(line) == "---":
			front = 1
			continue
		case front == 1:
			if string(line) == "---" {
				front = 2
			}
			continue
		case bytes.HasPrefix(bytes.TrimSpace(line), []byte("```")):
			fence = !fence
			continue
		case fence:
			continue
		}
		// A link's target is a row's name or an address, which are not the page's words.
		line = linkTarget.ReplaceAll(codeSpan.ReplaceAll(line, nil), []byte("]"))
		for _, m := range saidFigure.FindAll(line, -1) {
			out = append(out, proseFigure{i + 1, string(m)})
		}
	}
	return out
}

var (
	codeSpan   = regexp.MustCompile("`[^`]*`")
	linkTarget = regexp.MustCompile(`\]\((?:<[^>]*>|[^)\s]*)\)`)
)

// constantIs says how the constant name of spin's file differs from want - a count, or a
// duration as time writes one - or "".
func constantIs(spin, file, name, want string) (string, error) {
	path := filepath.Join(spin, file)
	if _, err := os.Stat(path); err != nil {
		return fmt.Sprintf("spin no longer has %s", file), nil
	}
	v, err := goConstant(path, name)
	if err != nil {
		return fmt.Sprintf("%s no longer declares %s as a constant that can be read (%v)", file, name, err), nil
	}
	have, ok := constant.Int64Val(constant.ToInt(v))
	if !ok {
		return "", fmt.Errorf("%s: %s is no integer", file, name)
	}
	n, err := strconv.ParseInt(want, 10, 64)
	if err != nil {
		d, derr := time.ParseDuration(want)
		if derr != nil {
			return "", fmt.Errorf("%s#%s = %s: neither a count nor a duration", file, name, want)
		}
		if have != int64(d) {
			return fmt.Sprintf("%s's %s is now %s, not %s", file, name, time.Duration(have), want), nil
		}
		return "", nil
	}
	if have != n {
		return fmt.Sprintf("%s's %s is now %d, not %d", file, name, have, n), nil
	}
	return "", nil
}
