package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	// A link someone wrote, with words of its own: an address Markdown linked only because it
	// was in the text - an example in a flag's help, say - has itself for its words.
	externalLink = regexp.MustCompile(`<a href="(https?://[^"#]+)(?:#[^"]*)?"[^>]*>([\s\S]*?)</a>`)
	ghcrRef      = regexp.MustCompile(`ghcr\.io/(spin-stack/[a-z0-9._-]+):([A-Za-z0-9._-]+)`)
)

// external opens every link out of the built site and every ghcr.io reference a page quotes, as
// a reader would - with no credential - and returns those that do not open. It is for the day
// the site goes public: links into a private repository answer 404 until then.
func external(dist, site string) ([]string, error) {
	urls := map[string]bool{}
	refs := map[string]bool{}
	err := filepath.WalkDir(dist, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(p)
		if ext != ".html" && ext != ".md" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if ext == ".html" {
			for _, m := range externalLink.FindAllStringSubmatch(string(b), -1) {
				if strings.TrimSpace(m[2]) == m[1] || strings.HasPrefix(m[1], site) {
					continue
				}
				if u, err := url.Parse(m[1]); err == nil && reachable(u) {
					urls[m[1]] = true
				}
			}
		}
		for _, m := range ghcrRef.FindAllString(string(b), -1) {
			refs[m] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 20 * time.Second}
	var mu sync.Mutex
	var failed []string
	var wg sync.WaitGroup
	slots := make(chan struct{}, 8)
	check := func(what string, fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			if err := fn(); err != nil {
				mu.Lock()
				failed = append(failed, fmt.Sprintf("%s: %v", what, err))
				mu.Unlock()
			}
		}()
	}
	for u := range urls {
		check(u, func() error { return open(client, u, "") })
	}
	for r := range refs {
		check(r, func() error { return pullable(client, r) })
	}
	wg.Wait()
	sort.Strings(failed)
	return failed, nil
}

// reachable leaves out the addresses a page writes as an example: a placeholder in angle
// brackets, and the domains reserved for examples.
func reachable(u *url.URL) bool {
	h := u.Hostname()
	return !strings.ContainsAny(u.String(), "<&") && !strings.Contains(u.String(), "%3C") &&
		h != "localhost" && !strings.HasSuffix(h, "example.com") && !strings.HasSuffix(h, ".example") &&
		!strings.HasSuffix(h, ".internal")
}

func open(client *http.Client, u, bearer string) error {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "spin-website-linkcheck")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
		req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json")
		req.Method = http.MethodHead
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", resp.Status)
	}
	return nil
}

// pullable asks ghcr.io, anonymously, for a reference's manifest: what `oras pull` does first.
func pullable(client *http.Client, ref string) error {
	m := ghcrRef.FindStringSubmatch(ref)
	resp, err := client.Get("https://ghcr.io/token?scope=repository:" + m[1] + ":pull")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("no anonymous pull: %s", resp.Status)
	}
	var tok struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return err
	}
	return open(client, fmt.Sprintf("https://ghcr.io/v2/%s/manifests/%s", m[1], m[2]), tok.Token)
}
