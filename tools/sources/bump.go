package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/spin-stack/go-tools/versions"
	"github.com/spin-stack/go-tools/versions/upstream"

	"github.com/spin-stack/website/tools/internal/github"
)

// bumpFile moves each source named to the version next says, and writes file when one moved.
func bumpFile(ctx context.Context, file string, names []string, gh *github.Client) error {
	v, err := versions.Load(file)
	if err != nil {
		return err
	}
	var out []byte
	for _, st := range upstream.Check(ctx, v) {
		if !slices.Contains(names, st.Entry.Name) {
			continue
		}
		to, err := next(st, func(tag string) (bool, error) { return holds(ctx, gh, st.Entry, tag) })
		if err != nil {
			return fmt.Errorf("%s: %w", st.Entry.Name, err)
		}
		if to == "" {
			fmt.Printf("%s: stays at %s %s\n", st.Entry.Name, st.Entry.Version, st.Entry.Pin)
			continue
		}
		e, b, err := upstream.Bump(ctx, v, st.Entry.Name, to)
		if err != nil {
			return err
		}
		if v, err = versions.Parse(b); err != nil {
			return err
		}
		out = b
		fmt.Printf("%s: %s %s -> %s %s\n", e.Name, st.Entry.Version, st.Entry.Pin, e.Version, e.Pin)
	}
	if out == nil {
		return nil
	}
	return os.WriteFile(file, out, 0o644)
}

// next is the version a source moves to from where check found it, "" when it stays. holds says
// whether a release tag holds the commit pinned.
//
//   - A branch (brand's main) moves when the branch did: the newest of it is what it has to say.
//   - A release moves only to a newer release, never back.
//   - A commit of main, pinned until a release holds it, moves to the newest release only once
//     that release holds the commit: the release before it documents less than the site does,
//     and the site would go backwards.
func next(st upstream.Status, holds func(tag string) (bool, error)) (string, error) {
	e := st.Entry
	if st.Newest == "" {
		return "", fmt.Errorf("its newest is unknown: %s", st.Note)
	}
	if !st.Behind {
		return "", nil
	}
	if e.Track == "branch" {
		return e.Version, nil
	}
	// The go-tools track spells the tag as the version is spelled, and main has no v: the tag
	// does.
	newest := st.Newest
	if !strings.HasPrefix(newest, "v") && release.MatchString("v"+newest) {
		newest = "v" + newest
	}
	if !release.MatchString(newest) {
		return "", fmt.Errorf("its newest, %s, is not a release v<YYYYMMDD>.<N>", newest)
	}
	if release.MatchString(e.Version) {
		switch c := compareReleases(newest, e.Version); {
		case c > 0:
			return newest, nil
		case c == 0:
			// The same release at another commit: a tag moved, which is for a person to look at.
			return "", fmt.Errorf("%s is now %s; versions.yaml pins %s", e.Version, st.Note, e.Pin)
		}
		return "", nil
	}
	ok, err := holds(newest)
	if err != nil || !ok {
		return "", err
	}
	return newest, nil
}

// compareReleases orders two release names by date and then by number: v20261019.10 is after
// v20261019.9, which the strings' order would not say.
func compareReleases(a, b string) int {
	ma, mb := release.FindStringSubmatch(a), release.FindStringSubmatch(b)
	da, _ := strconv.Atoi(ma[1])
	db, _ := strconv.Atoi(mb[1])
	if da != db {
		return da - db
	}
	na, _ := strconv.Atoi(ma[2])
	nb, _ := strconv.Atoi(mb[2])
	return na - nb
}

// holds is whether tag of e's repository is e's pin or a descendant of it, as GitHub's compare
// says: a bump runs before the new pin is fetched, and the API knows the history without it.
func holds(ctx context.Context, gh *github.Client, e versions.Entry, tag string) (bool, error) {
	repo := strings.TrimPrefix(e.Source, "https://github.com/")
	var cmp struct {
		Status string `json:"status"`
	}
	if err := gh.Get(ctx, "/repos/"+repo+"/compare/"+e.Pin+"..."+tag, &cmp); err != nil {
		return false, err
	}
	return cmp.Status == "ahead" || cmp.Status == "identical", nil
}
