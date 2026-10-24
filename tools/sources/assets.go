package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/spin-stack/go-tools/versions"

	"github.com/spin-stack/website/tools/internal/github"
)

// asset is a file a source's release carries that the site takes as it is, fetched beside the
// source's checkout.
type asset struct {
	// name is the release asset, a .tar.gz of files with no directories.
	name string
	// dir is where it is unpacked, under the directory sources are fetched into.
	dir string
	// retake is the command that gives a release the asset, its tag as %s.
	retake string
}

// assets are the releases' files the site takes. spin's site-shots.yml takes the dashboard's
// screenshots of each release on the KVM fleet and puts them on it: refgen copies them from
// here, so the pictures are of the release the site documents, with nobody running a stack.
var assets = map[string]asset{
	"spin": {
		name:   "site-shots.tar.gz",
		dir:    "spin-site-shots",
		retake: "gh workflow run site-shots.yml -R spin-stack/spin -f tag=%s",
	},
}

// stamp is the file, in an asset's directory, that says which release it was unpacked from, as
// "<tag> <pin>\n". It is written last, so a directory without it is one a run did not finish;
// refgen reads it to hold the copy to the release it documents.
const stamp = ".release"

// shotName is what an unpacked file may be called: a PNG named as site.spec.ts's shoot() names
// one, and nothing a path could climb out of the directory with.
var shotName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*\.png$`)

// fetchAsset unpacks a's asset of e's release into dir. A directory already stamped with e's
// release and pin is left as it is, so a run after the one with the token needs none. A release
// without the asset fails, saying how to give it one: the site never keeps another release's
// screenshots under this one's name.
func fetchAsset(ctx context.Context, gh *github.Client, dir string, e versions.Entry, a asset) error {
	want := e.Version + " " + e.Pin + "\n"
	if have, err := os.ReadFile(filepath.Join(dir, stamp)); err == nil && string(have) == want {
		return nil
	}
	if !release.MatchString(e.Version) {
		return fmt.Errorf("%s is pinned at %s, which is no release, and %s is a release's: pin a release", e.Name, e.Version, a.name)
	}
	repo := strings.TrimPrefix(e.Source, "https://github.com/")
	raw, err := gh.ReleaseAsset(ctx, repo, e.Version, a.name)
	if errors.Is(err, github.ErrNoAsset) {
		return fmt.Errorf("%w: give it one with `%s` and fetch again "+
			"(site-shots.yml puts it on every release once Release has finished, about an hour after it)",
			err, fmt.Sprintf(a.retake, e.Version))
	}
	if err != nil {
		return err
	}
	files, err := untar(raw)
	if err != nil {
		return fmt.Errorf("%s %s's %s: %w", repo, e.Version, a.name, err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			return err
		}
		names = append(names, name)
	}
	slices.Sort(names)
	fmt.Printf("%s: %s of %s: %s\n", e.Name, a.name, e.Version, strings.Join(names, ", "))
	return os.WriteFile(filepath.Join(dir, stamp), []byte(want), 0o644)
}

// untar is the files of a .tar.gz that holds only PNGs, flat. Anything else in it - a directory,
// a link, another name - is refused rather than skipped: the archive is not this repository's,
// and one that is not what site-shots.yml writes is one to look at.
func untar(raw []byte) (map[string][]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	files := map[string][]byte{}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		name := strings.TrimPrefix(h.Name, "./")
		if h.Typeflag == tar.TypeDir && name == "" {
			continue
		}
		if h.Typeflag != tar.TypeReg || !shotName.MatchString(name) {
			return nil, fmt.Errorf("%q is not a PNG at the top of the archive", h.Name)
		}
		if h.Size > 16<<20 {
			return nil, fmt.Errorf("%s is %d bytes, more than a screenshot", name, h.Size)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		files[name] = body
	}
	if len(files) == 0 {
		return nil, errors.New("it holds no screenshot")
	}
	return files, nil
}
