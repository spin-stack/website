// Command linkcheck fails when a page of the built site links to a path on the site that the
// build did not produce, or to an anchor the target page does not have. A broken link is found
// here rather than by a reader. It also fails a page that does not say Spinbox is experimental:
// every HTML page carries the banner once, and every Markdown twin and llms file the sentence.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	// An asset the page loads from another site: a script, a stylesheet, a font, an image or a
	// frame. The site serves every one of its own, so a reader's visit reaches nobody else.
	remoteAsset = regexp.MustCompile(`<(?:script|img|iframe|source)\b[^>]*\bsrc="(https?:)?//|<link\b[^>]*\brel="(?:stylesheet|preload|modulepreload|icon)"[^>]*\bhref="(https?:)?//|url\((?:'|")?(https?:)?//`)
	href        = regexp.MustCompile(`href="([^"]+)"`)
	id          = regexp.MustCompile(`\sid="([^"]+)"`)
)

// The experimental notice as the build writes it: the banner's attribute on an HTML page
// (src/components/Banner.astro), the sentence in a twin (src/lib/experimental.ts).
const (
	banner   = "data-experimental"
	sentence = "Spinbox is experimental: APIs change and data may be lost."
)

func main() {
	base := flag.String("base", "", "the path the site is served under, such as /website")
	ext := flag.Bool("external", false, "open every link out of the site and every ghcr.io reference instead, with no credential")
	site := flag.String("site", "https://spin-stack.github.io", "where the site is served, whose own links -external leaves alone")
	flag.Parse()
	dist := "dist"
	if flag.NArg() > 0 {
		dist = flag.Arg(0)
	}
	if *ext {
		failed, err := external(dist, *site)
		if err != nil {
			fmt.Fprintln(os.Stderr, "linkcheck:", err)
			os.Exit(1)
		}
		for _, f := range failed {
			fmt.Fprintln(os.Stderr, f)
		}
		if len(failed) > 0 {
			fmt.Fprintf(os.Stderr, "linkcheck: %d links out of the site do not open\n", len(failed))
			os.Exit(1)
		}
		fmt.Println("linkcheck: every link out of the site opens")
		return
	}
	prefix := strings.TrimSuffix(*base, "/")
	pages := map[string]string{} // the path a file is served at -> its HTML
	var unmarked []string
	err := filepath.WalkDir(dist, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dist, p)
		if err != nil {
			return err
		}
		served := prefix + "/" + filepath.ToSlash(rel)
		if strings.HasSuffix(served, "/index.html") {
			served = strings.TrimSuffix(served, "index.html")
		}
		body := ""
		switch ext := filepath.Ext(p); {
		case ext == ".html":
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			body = string(b)
			if m := remoteAsset.FindString(body); m != "" {
				unmarked = append(unmarked, fmt.Sprintf("%s: loads an asset from another site: %s", served, m))
			}
			if n := strings.Count(body, banner); n != 1 && !strings.HasSuffix(served, "/404.html") {
				unmarked = append(unmarked, fmt.Sprintf("%s: the experimental banner %d times, not once", served, n))
			}
		case ext == ".md" || strings.HasPrefix(path.Base(served), "llms"):
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			// It opens the file: within its first lines, before any of the page's own text.
			if !strings.Contains(string(b[:min(len(b), 400)]), sentence) {
				unmarked = append(unmarked, served+": does not open with "+strconv.Quote(sentence))
			}
		}
		pages[served] = body
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "linkcheck:", err)
		os.Exit(1)
	}

	var broken []string
	for from, body := range pages {
		if !strings.HasSuffix(from, "/") {
			continue
		}
		for _, m := range href.FindAllStringSubmatch(body, -1) {
			u, err := url.Parse(m[1])
			if err != nil || u.Scheme != "" || u.Host != "" || strings.HasPrefix(m[1], "//") {
				continue
			}
			target := u.Path
			if target == "" {
				target = from
			} else if !strings.HasPrefix(target, "/") {
				target = path.Join(from, target)
				if strings.HasSuffix(u.Path, "/") {
					target += "/"
				}
			}
			html, ok := pages[target]
			if !ok {
				broken = append(broken, fmt.Sprintf("%s -> %s", from, m[1]))
				continue
			}
			if u.Fragment != "" && !hasID(html, u.Fragment) {
				broken = append(broken, fmt.Sprintf("%s -> %s (no such anchor)", from, m[1]))
			}
		}
	}
	sort.Strings(unmarked)
	for _, u := range unmarked {
		fmt.Fprintln(os.Stderr, u)
	}
	sort.Strings(broken)
	for _, b := range broken {
		fmt.Fprintln(os.Stderr, b)
	}
	if len(broken) > 0 || len(unmarked) > 0 {
		fmt.Fprintf(os.Stderr, "linkcheck: %d broken links, %d pages that break a rule above\n", len(broken), len(unmarked))
		os.Exit(1)
	}
	fmt.Printf("linkcheck: %d pages, every link resolves\n", len(pages))
}

func hasID(html, fragment string) bool {
	for _, m := range id.FindAllStringSubmatch(html, -1) {
		if m[1] == fragment {
			return true
		}
	}
	return false
}
