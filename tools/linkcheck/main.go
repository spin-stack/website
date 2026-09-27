// Command linkcheck fails when a page of the built site links to a path on the site that the
// build did not produce, or to an anchor the target page does not have. A broken link is found
// here rather than by a reader.
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
	"strings"
)

var (
	href = regexp.MustCompile(`href="([^"]+)"`)
	id   = regexp.MustCompile(`\sid="([^"]+)"`)
)

func main() {
	base := flag.String("base", "", "the path the site is served under, such as /website")
	flag.Parse()
	dist := "dist"
	if flag.NArg() > 0 {
		dist = flag.Arg(0)
	}
	prefix := strings.TrimSuffix(*base, "/")
	pages := map[string]string{} // the path a file is served at -> its HTML
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
		if strings.HasSuffix(p, ".html") {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			body = string(b)
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
	sort.Strings(broken)
	for _, b := range broken {
		fmt.Fprintln(os.Stderr, b)
	}
	if len(broken) > 0 {
		fmt.Fprintf(os.Stderr, "linkcheck: %d broken links\n", len(broken))
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
