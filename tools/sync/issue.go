package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spin-stack/website/tools/internal/github"
)

// issueTitle is the one issue in spin that the sync keeps: spin is private and the website public,
// so what quotes spin's commits is written where only those who read spin read it.
const issueTitle = "Site vs spin main"

// issueHead opens the issue's body, above its sections.
const issueHead = "What the site at https://spin-stack.github.io/website/ documents of spin, against what spin has. " +
	"Written by spin-stack/website's `.github/workflows/sync.yml`, section by section; an edit by hand is written over. " +
	"Advice: nothing here fails a check.\n"

// sections, in the order the body has them: each job writes its own and leaves the others.
var sections = []struct{ key, title string }{
	{"release", "The last release sync"},
	{"main", "spin main past the pin"},
	{"chain", "The release chain"},
}

// withSection is body with the section key's text replaced by text, the others as they were;
// a body that has none of them is begun again with the head.
func withSection(body, key, text string) string {
	have := map[string]string{}
	for _, s := range sections {
		start, end := "<!-- sync:"+s.key+" -->", "<!-- /sync:"+s.key+" -->"
		if _, rest, ok := strings.Cut(body, start); ok {
			if in, _, ok := strings.Cut(rest, end); ok {
				have[s.key] = in
			}
		}
	}
	have[key] = "\n" + strings.TrimSpace(text) + "\n"
	var b strings.Builder
	b.WriteString(issueHead)
	for _, s := range sections {
		in, ok := have[s.key]
		if !ok {
			in = "\nNot written yet.\n"
		}
		fmt.Fprintf(&b, "\n## %s\n\n<!-- sync:%s -->%s<!-- /sync:%s -->\n", s.title, s.key, in, s.key)
	}
	return b.String()
}

// writeIssue writes the sections given into the open issue issueTitle in repo, which bot opened,
// and opens it when there is none. It is found by its author, not searched for: GitHub's search
// is behind its writes, and a second issue is what that would make.
func writeIssue(ctx context.Context, gh *github.Client, repo, bot string, texts map[string]string) (string, error) {
	var open []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		URL    string `json:"html_url"`
		PR     any    `json:"pull_request"`
	}
	q := url.Values{"state": {"open"}, "creator": {bot}, "per_page": {"100"}}
	if err := gh.Get(ctx, "/repos/"+repo+"/issues?"+q.Encode(), &open); err != nil {
		return "", err
	}
	body, number, link := "", 0, ""
	for _, i := range open {
		if i.Title == issueTitle && i.PR == nil {
			body, number, link = i.Body, i.Number, i.URL
			break
		}
	}
	for _, s := range sections {
		if t, ok := texts[s.key]; ok {
			body = withSection(body, s.key, t)
		}
	}
	var out struct {
		URL string `json:"html_url"`
	}
	if number == 0 {
		err := gh.Do(ctx, http.MethodPost, "/repos/"+repo+"/issues", map[string]string{"title": issueTitle, "body": body}, &out)
		return out.URL, err
	}
	err := gh.Do(ctx, http.MethodPatch, fmt.Sprintf("/repos/%s/issues/%d", repo, number), map[string]string{"body": body}, &out)
	if out.URL == "" {
		out.URL = link
	}
	return out.URL, err
}
