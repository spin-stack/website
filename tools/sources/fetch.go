package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spin-stack/go-tools/versions"
)

// release is a version that names a release tag, v<YYYYMMDD>.<N>, as spin and spin-terraform-aws
// cut them; anything else (main) is a branch the pin is a commit of.
var release = regexp.MustCompile(`^v(20\d{6})\.(\d+)$`)

// fetch makes dir a checkout of e's pin, with the history that leads to it and every tag:
// refgen asks which release tags point at or are merged into it, and reads spin's versions.yaml
// at the release an installation runs. A checkout already at the pin, whole, is left as it is,
// so a second run asks nothing of the network and needs no token.
func fetch(ctx context.Context, dir string, e versions.Entry, auth []string) error {
	if head, err := git(ctx, dir, nil, "rev-parse", "HEAD"); err == nil && head == e.Pin {
		if shallow, err := git(ctx, dir, nil, "rev-parse", "--is-shallow-repository"); err == nil && shallow == "false" {
			return tagged(ctx, dir, e)
		}
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if _, err := git(ctx, dir, nil, "init", "-q"); err != nil {
		return err
	}
	if _, err := git(ctx, dir, auth, "fetch", "-q", "--tags", e.Source, e.Pin); err != nil {
		return err
	}
	if err := tagged(ctx, dir, e); err != nil {
		return err
	}
	_, err := git(ctx, dir, nil, "checkout", "-q", "--detach", e.Pin)
	return err
}

// tagged fails when e's version is a release whose tag is not its pin: a tag moved, or a pin
// written by hand, would have the site name one release while documenting another tree.
func tagged(ctx context.Context, dir string, e versions.Entry) error {
	if !release.MatchString(e.Version) {
		return nil
	}
	got, err := git(ctx, dir, nil, "rev-parse", "--verify", "-q", "refs/tags/"+e.Version+"^{commit}")
	if err != nil {
		return fmt.Errorf("%s has no tag %s", e.Source, e.Version)
	}
	if got != e.Pin {
		return fmt.Errorf("%s's tag %s is %s, and versions.yaml pins %s", e.Source, e.Version, got, e.Pin)
	}
	return nil
}

// git runs git in dir with env added to this process's, its output trimmed. It never asks for a
// password: a source the token cannot read fails rather than waits.
func git(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(append(os.Environ(), "GIT_TERMINAL_PROMPT=0"), env...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}
