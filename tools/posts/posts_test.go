package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// feed is the blog's feed as @astrojs/rss writes it, the newest first, with items of link.
func feed(links ...string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>Spinbox blog</title><link>https://blog.example/</link>`)
	dates := []string{"Wed, 25 Nov 2026 00:00:00 GMT", "Tue, 24 Nov 2026 00:00:00 GMT"}
	for i, l := range links {
		b.WriteString(`<item><title>A post</title><link>` + l + `</link><pubDate>` + dates[i%2] + `</pubDate></item>`)
	}
	b.WriteString(`</channel></rss>`)
	return b.String()
}

func serve(t *testing.T, body string, status int) string {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rss.xml" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)
	return s.URL
}

func read(t *testing.T, path string) Record {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var r Record
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

// The record is every post the feed has, the oldest first, by the slug its link names.
func TestTheRecordIsTheFeedsPosts(t *testing.T) {
	out := filepath.Join(t.TempDir(), "posts.json")
	if err := run([]string{"-blog", serve(t, feed(), http.StatusOK), "-out", out}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if r := read(t, out); r.Posts == nil || len(r.Posts) != 0 || r.Generated == "" {
		t.Errorf("a feed with no posts: %+v", r)
	}

	var blog string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(feed(blog+"/kitchen/the-second/", blog+"/spinbox/the-first/")))
	}))
	t.Cleanup(s.Close)
	blog = s.URL
	if err := run([]string{"-blog", blog, "-out", out}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	r := read(t, out)
	var slugs []string
	for _, p := range r.Posts {
		slugs = append(slugs, p.Slug+" "+p.Path+" "+p.Date)
	}
	if want := []string{"the-first /spinbox/the-first/ 2026-11-24", "the-second /kitchen/the-second/ 2026-11-25"}; !slices.Equal(slugs, want) {
		t.Errorf("posts %v, want %v", slugs, want)
	}
	if r.Generated == "" || r.Blog != blog {
		t.Errorf("record %+v", r)
	}
}

// A feed that cannot be read keeps the file as it is, and writes one with no posts where there is
// none: the blog is not live, or down for the day.
func TestAFeedThatCannotBeReadKeepsTheFile(t *testing.T) {
	for name, blog := range map[string]string{
		"not found":   serve(t, "", http.StatusNotFound),
		"unreachable": "http://127.0.0.1:1",
	} {
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "posts.json")
			if err := run([]string{"-blog", blog, "-out", out}, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
			if r := read(t, out); r.Posts == nil || len(r.Posts) != 0 || r.Generated != "" {
				t.Errorf("with no file, %+v", r)
			}
			kept := []byte(`{"generated":"2026-11-30","blog":"x","posts":[{"slug":"a","title":"A","date":"2026-11-24"}]}`)
			if err := os.WriteFile(out, kept, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := run([]string{"-blog", blog, "-out", out}, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
			if b, _ := os.ReadFile(out); !bytes.Equal(b, kept) {
				t.Errorf("the file was rewritten: %s", b)
			}
		})
	}
}

// A feed read but not as the blog writes it fails, rather than a list that would pass links it
// should not.
func TestAFeedOfAnotherShapeFails(t *testing.T) {
	for name, tc := range map[string]struct{ body, want string }{
		"another site's link": {feed("https://elsewhere.example/posts/a/"), "no post of"},
		"not a post":          {feed("BLOG/about/"), "no post of"},
		"not a feed":          {"<html>", "the feed"},
	} {
		t.Run(name, func(t *testing.T) {
			var blog string
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(strings.ReplaceAll(tc.body, "BLOG", blog)))
			}))
			t.Cleanup(s.Close)
			blog = s.URL
			err := run([]string{"-blog", blog, "-out", filepath.Join(t.TempDir(), "posts.json")}, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err %v, want %q", err, tc.want)
			}
		})
	}
}
