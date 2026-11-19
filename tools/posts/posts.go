// Command posts writes src/data/posts.json: the posts the blog has published, read from its
// public feed. A page links a post as [why](<post:the-slug>), and the build fails on a slug this
// list does not have (src/lib/post-links.mjs), so a page never links a post that is still a
// draft, nor one renamed or gone. The build reads the file, never the feed: it needs no network.
//
// A feed that cannot be read - the blog not live yet, or down for the day - keeps the file as it
// is, and writes one with no posts where there is none: the posts already linked stay linkable,
// and nothing is linkable that was not read.
package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Record is src/data/posts.json.
type Record struct {
	// Generated is the day the feed was last read, UTC; empty until it first is.
	Generated string `json:"generated"`
	// Blog is the blog's address: a post is at Blog + /posts/<slug>/.
	Blog  string `json:"blog"`
	Posts []Post `json:"posts"`
}

// Post is one published post, the oldest first.
type Post struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Date  string `json:"date"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "posts:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("posts", flag.ContinueOnError)
	blog := fs.String("blog", "https://blog.spinbox.dev", "the blog's address; its feed is /rss.xml")
	out := fs.String("out", "src/data/posts.json", "the record")
	if err := fs.Parse(args); err != nil {
		return err
	}
	*blog = strings.TrimSuffix(*blog, "/")
	feed, err := fetch(*blog + "/rss.xml")
	if err != nil {
		fmt.Fprintf(stdout, "posts: %v; %s kept as it is\n", err, *out)
		if _, statErr := os.Stat(*out); statErr == nil {
			return nil
		}
		return write(*out, Record{Blog: *blog, Posts: []Post{}})
	}
	posts, err := parse(feed, *blog)
	if err != nil {
		return err
	}
	if err := write(*out, Record{Generated: time.Now().UTC().Format(time.DateOnly), Blog: *blog, Posts: posts}); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "posts: %d published\n", len(posts))
	return nil
}

// fetch reads the feed. Any failure to reach it is an error the caller keeps the file over; what
// it says once read is parse's to refuse.
func fetch(url string) ([]byte, error) {
	c := &http.Client{Timeout: 30 * time.Second}
	resp, err := c.Get(url) //nolint:noctx // a command's one request
	if err != nil {
		return nil, fmt.Errorf("the feed cannot be read: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the feed cannot be read: %s answered %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// postPath is a post's address on the blog, as the feed links it: /posts/<slug>/.
var postPath = regexp.MustCompile(`^/posts/([a-z0-9-]+)/$`)

// parse reads the feed's items as the blog writes them (src/pages/rss.xml.ts there): a link to
// the post, its title and its date. An item that is not a post of the blog's fails: the feed's
// shape changed, and a list read from it wrong would pass links it should not. A feed with no
// items is the blog before its first post: a list nothing may link.
func parse(feed []byte, blog string) ([]Post, error) {
	var rss struct {
		Items []struct {
			Title   string `xml:"title"`
			Link    string `xml:"link"`
			PubDate string `xml:"pubDate"`
		} `xml:"channel>item"`
	}
	if err := xml.Unmarshal(feed, &rss); err != nil {
		return nil, fmt.Errorf("the feed: %w", err)
	}
	posts := []Post{}
	for _, it := range rss.Items {
		m := postPath.FindStringSubmatch(strings.TrimPrefix(it.Link, blog))
		if m == nil || !strings.HasPrefix(it.Link, blog+"/") {
			return nil, fmt.Errorf("the feed links %q, which is no post of %s", it.Link, blog)
		}
		date, err := time.Parse(time.RFC1123, it.PubDate)
		if err != nil {
			if date, err = time.Parse(time.RFC1123Z, it.PubDate); err != nil {
				return nil, fmt.Errorf("the feed dates %s %q: %w", m[1], it.PubDate, err)
			}
		}
		posts = append(posts, Post{Slug: m[1], Title: it.Title, Date: date.UTC().Format(time.DateOnly)})
	}
	sort.SliceStable(posts, func(i, j int) bool { return posts[i].Date < posts[j].Date })
	return posts, nil
}

func write(path string, r Record) error {
	b, err := json.MarshalIndent(r, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644) //nolint:gosec // the site's data file
}
