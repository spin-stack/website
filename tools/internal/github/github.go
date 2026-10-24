// Package github is the little of GitHub's REST API the site's tools ask: a client whose base
// is a field, so that a test hands it an httptest server and needs no network.
package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// API is the address every request is made to when a Client names none.
const API = "https://api.github.com"

// Client asks GitHub's REST API as Token, which may be empty for what is public.
type Client struct {
	Base  string
	Token string
	HTTP  *http.Client
}

// New is a client of api.github.com as token.
func New(token string) *Client {
	return &Client{Base: API, Token: token, HTTP: &http.Client{Timeout: time.Minute}}
}

// Do sends body, as JSON when it is not nil, to path and decodes the answer into out when out is
// not nil. An answer that is not a 2xx is an error that carries what GitHub said.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	raw, err := c.send(ctx, method, path, body, "application/vnd.github+json")
	if err != nil || out == nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	return nil
}

// Get is Do for a GET.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.Do(ctx, http.MethodGet, path, nil, out)
}

// File is a file of repo at ref, as its bytes.
func (c *Client) File(ctx context.Context, repo, path, ref string) ([]byte, error) {
	return c.send(ctx, http.MethodGet, "/repos/"+repo+"/contents/"+path+"?ref="+ref, nil, "application/vnd.github.raw")
}

// ErrNoAsset is a release that has no asset of the name asked for.
var ErrNoAsset = errors.New("no such asset")

// ReleaseAsset is the asset called name of repo's release tag, as its bytes, read through the
// API as the token can read a private repository's. The API redirects to storage on another
// host, and Go's client sends the Authorization header no further than GitHub.
func (c *Client) ReleaseAsset(ctx context.Context, repo, tag, name string) ([]byte, error) {
	var rel struct {
		Assets []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"assets"`
	}
	if err := c.Get(ctx, "/repos/"+repo+"/releases/tags/"+url.PathEscape(tag), &rel); err != nil {
		return nil, err
	}
	for _, a := range rel.Assets {
		if a.Name == name {
			return c.send(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/releases/assets/%d", repo, a.ID), nil, "application/octet-stream")
		}
	}
	return nil, fmt.Errorf("%s %s has no %s: %w", repo, tag, name, ErrNoAsset)
}

func (c *Client) send(ctx context.Context, method, path string, body any, accept string) ([]byte, error) {
	var in io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		in = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(c.Base, "/")+path, in)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }() // read-only
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(raw)))
	}
	return raw, nil
}

// GitAuth is what a git command's environment takes token as: an Authorization header for
// github.com, handed in GIT_CONFIG_* so it is never written to a repository's config, where a
// later step would find it. Basic, as actions/checkout sends it: git's HTTP answers no Bearer.
// No token, no header: what is public needs none.
func GitAuth(token string) []string {
	if token == "" {
		return nil
	}
	basic := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
	return []string{
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=http.https://github.com/.extraheader",
		"GIT_CONFIG_VALUE_0=AUTHORIZATION: basic " + basic,
	}
}
