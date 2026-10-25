package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spin-stack/go-tools/versions"

	"github.com/spin-stack/website/tools/internal/github"
)

func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// releases serves spin's releases as GitHub's API does, to the token alone: each tag's asset
// list, and an asset's bytes from its id, when asked for them as octet-stream.
func releases(t *testing.T, assets map[string][]byte) *github.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ghs_x" {
			http.NotFound(w, r)
			return
		}
		if tag, ok := strings.CutPrefix(r.URL.Path, "/repos/spin-stack/spin/releases/tags/"); ok {
			if body, ok := assets[tag]; ok && body == nil {
				_, _ = w.Write([]byte(`{"assets":[{"id":1,"name":"checksums.txt"}]}`))
				return
			}
			if _, ok := assets[tag]; ok {
				_, _ = w.Write([]byte(`{"assets":[{"id":1,"name":"checksums.txt"},{"id":7,"name":"site-shots.tar.gz"}]}`))
				return
			}
		}
		if r.URL.Path == "/repos/spin-stack/spin/releases/assets/7" && r.Header.Get("Accept") == "application/octet-stream" {
			for _, body := range assets {
				if body != nil {
					_, _ = w.Write(body)
					return
				}
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return &github.Client{Base: srv.URL, Token: "ghs_x", HTTP: srv.Client()}
}

func spinAt(version string) versions.Entry {
	return versions.Entry{Name: "spin", Source: "https://github.com/spin-stack/spin", Version: version, Pin: strings.Repeat("b3", 20)}
}

// The release's screenshots are unpacked as the archive has them and stamped with the release
// and pin, and a later run - Sync's task reference, with no token - asks nothing again.
func TestTheReleasesScreenshotsAreUnpackedAndStamped(t *testing.T) {
	ctx := context.Background()
	gh := releases(t, map[string][]byte{"v20261024.01": tarGz(t, map[string]string{"workspace.png": "one", "./workspaces.png": "two"})})
	dir := filepath.Join(t.TempDir(), "spin-site-shots")
	e := spinAt("v20261024.01")
	if err := fetchAsset(ctx, gh, dir, e, assets["spin"]); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"workspace.png": "one", "workspaces.png": "two", stamp: "v20261024.01 " + e.Pin + "\n"} {
		if got, err := os.ReadFile(filepath.Join(dir, name)); err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", name, got, err, want)
		}
	}
	if err := fetchAsset(ctx, &github.Client{Base: "http://127.0.0.1:1"}, dir, e, assets["spin"]); err != nil {
		t.Errorf("a directory stamped with the pin was fetched again: %v", err)
	}
}

// What is not the documented release's screenshots fails, and a release that lacks them says
// how to give it them: nothing falls back to the pictures already there.
func TestScreenshotsThatAreNotTheReleasesFail(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		name    string
		version string
		archive []byte
		want    string
	}{
		{"a release without the asset", "v20261019.02", nil, "gh workflow run site-shots.yml -R spin-stack/spin -f tag=v20261019.02"},
		{"a pin that is no release", "main", nil, "which is no release"},
		{"an archive with a directory in it", "v20261024.01", tarGz(t, map[string]string{"shots/workspace.png": "x"}), "not a PNG at the top"},
		{"an archive that climbs out", "v20261024.01", tarGz(t, map[string]string{"../workspace.png": "x"}), "not a PNG at the top"},
		{"an empty archive", "v20261024.01", tarGz(t, nil), "no screenshot"},
	} {
		dir := filepath.Join(t.TempDir(), "spin-site-shots")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "workspace.png"), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
		gh := releases(t, map[string][]byte{c.version: c.archive})
		err := fetchAsset(ctx, gh, dir, spinAt(c.version), assets["spin"])
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: fetchAsset = %v; want an error saying %q", c.name, err, c.want)
		}
		if _, err := os.Stat(filepath.Join(dir, stamp)); err == nil {
			t.Errorf("%s: the directory is stamped as the release's", c.name)
		}
	}
}

// A bump waits for the release's screenshots: a release without them yet is not ready, one with
// them is, and a source the site takes no asset of is ready as it is.
func TestABumpWaitsForTheReleasesScreenshots(t *testing.T) {
	gh := releases(t, map[string][]byte{"v20261025.01": nil, "v20261026.01": []byte("shots")})
	for tag, want := range map[string]bool{"v20261025.01": false, "v20261026.01": true} {
		if got, err := ready(context.Background(), gh, spinAt("v20261024.01"), tag); err != nil || got != want {
			t.Errorf("ready(%s) = %v, %v; want %v", tag, got, err, want)
		}
	}
	if ok, err := ready(context.Background(), gh, versions.Entry{Name: "spin-terraform-aws"}, "v20261025.01"); err != nil || !ok {
		t.Errorf("a source with no asset: ready = %v, %v; want true", ok, err)
	}
	if _, err := ready(context.Background(), gh, spinAt("v20261024.01"), "v20990101.01"); err == nil {
		t.Error("ready took a 404 for an answer")
	}
}
