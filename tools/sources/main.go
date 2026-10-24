// Command sources fetches what the site is generated from - each repository versions.yaml pins -
// into .sources/<name>, at exactly the pinned commit, and with -bump moves the pins to the newest
// releases first:
//
//	go run ./tools/sources [-bump] <name>...
//
// A private repository is fetched with SOURCES_TOKEN, which reaches git only as GIT_CONFIG_* in
// the environment of the git commands this runs: nothing is written to a checkout's config, so
// the steps after this one, which build and run what was fetched, never see it.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/spin-stack/go-tools/versions"

	"github.com/spin-stack/website/tools/internal/github"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "sources:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	file := flag.String("file", versions.File, "the versions.yaml that pins the sources")
	dir := flag.String("dir", ".sources", "where each source is fetched, into a directory of its name")
	bump := flag.Bool("bump", false, "first move each pin named to the newest release it can move to, and write the file")
	flag.Parse()
	names := flag.Args()
	if len(names) == 0 {
		return errors.New("name the sources, as versions.yaml names them")
	}
	token := os.Getenv("SOURCES_TOKEN")
	if *bump {
		// What the go-tools resolvers ask of GitHub's API they ask as GITHUB_TOKEN, and of git
		// with what git's environment carries.
		if token != "" {
			if err := os.Setenv("GITHUB_TOKEN", token); err != nil {
				return err
			}
			for _, kv := range github.GitAuth(token) {
				k, v, _ := strings.Cut(kv, "=")
				if err := os.Setenv(k, v); err != nil {
					return err
				}
			}
		}
		if err := bumpFile(ctx, *file, names, github.New(token)); err != nil {
			return err
		}
	}
	v, err := versions.Load(*file)
	if err != nil {
		return err
	}
	for _, n := range names {
		e, err := v.Get(n)
		if err != nil {
			return err
		}
		if e.Kind != versions.Git {
			return fmt.Errorf("%s is a %s; a source is a git repository", n, e.Kind)
		}
		if err := fetch(ctx, filepath.Join(*dir, n), e, github.GitAuth(token)); err != nil {
			return fmt.Errorf("%s: %w", n, err)
		}
	}
	return nil
}
