package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/spin-stack/go-tools/versions"

	"github.com/spin-stack/website/tools/internal/github"
)

// branch is the one branch, and so the one pull request, the sync writes: a run force-pushes it,
// and the pull request open from it is updated rather than another opened beside it.
const branch = "sync/sources"

// move is a pin that changed between main and the bump.
type move struct {
	Name           string
	FromV, FromPin string
	ToV, ToPin     string
}

// moves is every entry whose version or pin differs from base to now, in now's order.
func moves(base, now *versions.Versions) []move {
	var out []move
	for _, e := range now.Entries {
		was, err := base.Get(e.Name)
		if err == nil && was.Version == e.Version && was.Pin == e.Pin {
			continue
		}
		out = append(out, move{e.Name, was.Version, was.Pin, e.Version, e.Pin})
	}
	return out
}

// prBody is the sync pull request's description. The website is public and spin private, so it
// says which pins moved and which files changed, and nothing of what spin's commits say: that is
// in spin's issue, where only those who read spin read it.
func prBody(ms []move, changed []string, reference, today string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "The sources moved to their newest releases, and the reference regenerated from them, on %s.\n\n", today)
	b.WriteString("| source | from | to |\n|---|---|---|\n")
	for _, m := range ms {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", m.Name, at(m.FromV, m.FromPin), at(m.ToV, m.ToPin))
	}
	b.WriteString("\n")
	switch reference {
	case "success":
		b.WriteString("`task reference` wrote the files below and found nothing on the hand-written pages that the sources do not have.\n")
	default:
		fmt.Fprintf(&b, "`task reference` did not pass (%s): the hand-written pages say something the sources no longer do. Gate's `task check` names each line; fix them on this branch.\n", reference)
	}
	fmt.Fprintf(&b, "\n<details><summary>%d files changed</summary>\n\n", len(changed))
	for _, f := range changed {
		fmt.Fprintf(&b, "- `%s`\n", f)
	}
	b.WriteString("\n</details>\n\n")
	fmt.Fprintf(&b, "What spin's commits since the last pin change, grouped by the pages each feeds, is in spin's issue %q, beside the release chain. ", issueTitle)
	b.WriteString("Read the pages they feed against them: what is generated is here, what is written by hand is a person's to change. ")
	b.WriteString("Nothing merges this on its own; the next run writes over this branch.\n")
	return b.String()
}

func at(version, pin string) string {
	if version == "" {
		return "-"
	}
	return fmt.Sprintf("%s `%s`", version, short(pin))
}

// changedFiles is every path git sees changed or added in the work tree.
func changedFiles(ctx context.Context) ([]string, error) {
	out, err := gitOut(ctx, nil, "status", "--porcelain", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, rec := range strings.Split(strings.TrimSuffix(out, "\x00"), "\x00") {
		if len(rec) > 3 {
			files = append(files, rec[3:])
		}
	}
	return files, nil
}

// commitMessage is the sync commit's, in the repository's voice: what moved.
func commitMessage(ms []move) string {
	var parts []string
	for _, m := range ms {
		parts = append(parts, fmt.Sprintf("%s %s", m.Name, at(m.ToV, m.ToPin)))
	}
	return "sources: " + strings.ReplaceAll(strings.Join(parts, ", "), "`", "") + "\n\nWritten by sync.yml: versions.yaml bumped by task sources:bump, the reference by task reference.\n"
}

// publish commits the work tree as bot, force-pushes it to branch of remote, and opens the pull request
// from it, or updates the one open.
func publish(ctx context.Context, gh *github.Client, repo, remote, bot, token string, ms []move, body string) (string, error) {
	var user struct {
		ID int `json:"id"`
	}
	if err := gh.Get(ctx, "/users/"+url.PathEscape(bot), &user); err != nil {
		return "", err
	}
	id := []string{
		"GIT_AUTHOR_NAME=" + bot, "GIT_COMMITTER_NAME=" + bot,
		fmt.Sprintf("GIT_AUTHOR_EMAIL=%d+%s@users.noreply.github.com", user.ID, bot),
		fmt.Sprintf("GIT_COMMITTER_EMAIL=%d+%s@users.noreply.github.com", user.ID, bot),
	}
	for _, args := range [][]string{
		{"switch", "-q", "-C", branch},
		{"add", "-A"},
		{"commit", "-q", "-m", commitMessage(ms)},
	} {
		if _, err := gitOut(ctx, id, args...); err != nil {
			return "", err
		}
	}
	if _, err := gitOut(ctx, github.GitAuth(token), "push", "-q", "--force", remote, "HEAD:refs/heads/"+branch); err != nil {
		return "", err
	}
	owner, _, _ := strings.Cut(repo, "/")
	var open []struct {
		Number int    `json:"number"`
		URL    string `json:"html_url"`
	}
	q := url.Values{"head": {owner + ":" + branch}, "state": {"open"}}
	if err := gh.Get(ctx, "/repos/"+repo+"/pulls?"+q.Encode(), &open); err != nil {
		return "", err
	}
	title := strings.TrimSuffix(strings.SplitN(commitMessage(ms), "\n", 2)[0], ".")
	var pr struct {
		URL string `json:"html_url"`
	}
	if len(open) > 0 {
		err := gh.Do(ctx, http.MethodPatch, fmt.Sprintf("/repos/%s/pulls/%d", repo, open[0].Number), map[string]string{"title": title, "body": body}, &pr)
		return pr.URL, err
	}
	err := gh.Do(ctx, http.MethodPost, "/repos/"+repo+"/pulls", map[string]string{"title": title, "body": body, "head": branch, "base": "main"}, &pr)
	return pr.URL, err
}

// gitOut runs git here with env added, its output untrimmed. It never asks for a password.
func gitOut(ctx context.Context, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(append(os.Environ(), "GIT_TERMINAL_PROMPT=0"), env...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}
