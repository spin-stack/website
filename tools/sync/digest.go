package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/spin-stack/website/tools/internal/github"
)

// feed is a path of spin's tree and the pages of the site it feeds: a commit that touches it is
// one a reviewer reads those pages against.
type feed struct {
	Path  string
	Pages []string
}

// read are the paths refgen reads that no page names as its source: what it holds the hand-written
// pages to. Each is every page of a kind, not a list of them, since the check is what finds which.
var read = []feed{
	{"internal/features/features.go", []string{"the landing page's count and cards", "every feature: link"}},
	{"hack/env-catalog.txt", []string{"every page that names a SPIN_ variable"}},
	{"Taskfile.yml", []string{"every page that quotes a task command"}},
	{"internal/storage/qcow/manager.go", []string{"every page with a <Value> of constants.json"}},
	{"internal/controlplane/compat/compat.go", []string{"every page with a <Value> of constants.json"}},
	{"versions.yaml", []string{"reference/release"}},
}

var (
	sourceLine = regexp.MustCompile(`(?m)^source: "spin:([^"]+)"$`)
	figureName = regexp.MustCompile(`<Figure\s+name="([^"]+)"`)
	blobPath   = regexp.MustCompile(`^https://github\.com/spin-stack/spin/blob/[0-9a-f]{40}/(.+)$`)
)

// feeds is every path of spin the site reads, with its pages: the generated pages' own source,
// the diagrams' and the pages that show each, and what refgen holds the rest to.
func feeds(docs, figures string) ([]feed, error) {
	by := map[string][]string{}
	for _, f := range read {
		by[f.Path] = slices.Clone(f.Pages)
	}
	shows := map[string][]string{}
	err := filepath.WalkDir(docs, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || (filepath.Ext(path) != ".md" && filepath.Ext(path) != ".mdx") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		page := pageName(docs, path)
		for _, m := range sourceLine.FindAllSubmatch(raw, -1) {
			by[string(m[1])] = append(by[string(m[1])], page)
		}
		for _, m := range figureName.FindAllSubmatch(raw, -1) {
			shows[string(m[1])] = append(shows[string(m[1])], page)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(figures)
	if err != nil {
		return nil, err
	}
	var figs map[string]struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal(raw, &figs); err != nil {
		return nil, fmt.Errorf("%s: %w", figures, err)
	}
	for name, f := range figs {
		if m := blobPath.FindStringSubmatch(f.Source); m != nil {
			by[m[1]] = append(by[m[1]], shows[name]...)
		}
	}
	var out []feed
	for path, pages := range by {
		out = append(out, feed{path, collapse(pages)})
	}
	slices.SortFunc(out, func(a, b feed) int { return strings.Compare(a.Path, b.Path) })
	return out, nil
}

// collapse is pages sorted, once each, with a directory of more than three said as one: the CLI
// feeds a page per command, and a reviewer reads them as one.
func collapse(pages []string) []string {
	slices.Sort(pages)
	pages = slices.Compact(pages)
	in := map[string]int{}
	for _, p := range pages {
		in[path.Dir(p)]++
	}
	var out []string
	for _, p := range pages {
		d := path.Dir(p)
		switch n := in[d]; {
		case n < 0:
			// said already
		case n <= 3:
			out = append(out, p)
		default:
			out = append(out, fmt.Sprintf("%s/ (%d pages)", d, n))
			in[d] = -1
		}
	}
	return out
}

// pageName is a page as a reviewer finds it: its path under docs, without the extension and
// index.
func pageName(docs, path string) string {
	rel, _ := filepath.Rel(docs, path)
	rel = strings.TrimSuffix(strings.TrimSuffix(filepath.ToSlash(rel), filepath.Ext(rel)), "/index")
	return rel
}

// feeds says whether file is under the feed's path: the path itself, a file under it as a
// directory, or the same name with an extension (a diagram's source beside its picture).
func (f feed) feeds(file string) bool {
	return file == f.Path || strings.HasPrefix(file, f.Path+"/") || strings.HasPrefix(file, f.Path+".")
}

// commit is one commit of the digest: its short hash, its first line and the files it touched.
type commit struct {
	SHA, Subject string
	Files        []string
}

// maxCommits is how many commits a digest reads, each a request of its own for its files: a
// week of spin's main, and a stop well short of the token's hourly allowance.
const maxCommits = 300

// commits is what repo's history holds from base to head, oldest first, each with its files.
func commits(ctx context.Context, gh *github.Client, repo, base, head string) ([]commit, int, error) {
	var out []commit
	total := 0
	for page := 1; len(out) < maxCommits; page++ {
		var cmp struct {
			Total   int `json:"total_commits"`
			Commits []struct {
				SHA    string `json:"sha"`
				Commit struct {
					Message string `json:"message"`
				} `json:"commit"`
			} `json:"commits"`
		}
		if err := gh.Get(ctx, fmt.Sprintf("/repos/%s/compare/%s...%s?per_page=100&page=%d", repo, base, head, page), &cmp); err != nil {
			return nil, 0, err
		}
		total = cmp.Total
		for _, c := range cmp.Commits {
			subject, _, _ := strings.Cut(c.Commit.Message, "\n")
			out = append(out, commit{SHA: c.SHA, Subject: subject})
		}
		if len(cmp.Commits) < 100 || len(out) >= total {
			break
		}
	}
	out = out[:min(len(out), maxCommits)]
	for i := range out {
		var c struct {
			Files []struct {
				Name string `json:"filename"`
			} `json:"files"`
		}
		if err := gh.Get(ctx, "/repos/"+repo+"/commits/"+out[i].SHA, &c); err != nil {
			return nil, 0, err
		}
		for _, f := range c.Files {
			out[i].Files = append(out[i].Files, f.Name)
		}
	}
	return out, total, nil
}

// digest is the commits grouped by the path of spin each touches that feeds the site, each group
// headed by the pages that path feeds, and a count of the rest: what a reviewer reads the site
// against, and nothing they need not.
func digest(cs []commit, total int, fs []feed) string {
	var b strings.Builder
	if total == 0 {
		return "No commits.\n"
	}
	fmt.Fprintf(&b, "%d commit%s", total, map[bool]string{true: "s"}[total != 1])
	if total > len(cs) {
		fmt.Fprintf(&b, ", the first %d read", len(cs))
	}
	b.WriteString(".\n")
	touched := map[string]bool{}
	for _, f := range fs {
		var lines []string
		for _, c := range cs {
			if slices.ContainsFunc(c.Files, f.feeds) {
				lines = append(lines, fmt.Sprintf("- %s %s", short(c.SHA), c.Subject))
				touched[c.SHA] = true
			}
		}
		if lines == nil {
			continue
		}
		fmt.Fprintf(&b, "\n#### `%s`\n\nFeeds: %s\n\n%s\n", f.Path, strings.Join(f.Pages, ", "), strings.Join(lines, "\n"))
	}
	if n := len(cs) - len(touched); n > 0 {
		fmt.Fprintf(&b, "\n%d of them touch no path the site reads.\n", n)
	}
	return b.String()
}

func short(sha string) string { return sha[:min(len(sha), 10)] }
