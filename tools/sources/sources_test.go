package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spin-stack/go-tools/versions"
	"github.com/spin-stack/go-tools/versions/upstream"

	"github.com/spin-stack/website/tools/internal/github"
)

// upstreamRepo is a bare repository with two commits, the first tagged v20261024.01, served from
// disk as GitHub serves one: any commit its history holds can be asked for.
func upstreamRepo(t *testing.T) (url, first, second string) {
	t.Helper()
	dir := t.TempDir()
	work, bare := filepath.Join(dir, "work"), filepath.Join(dir, "bare.git")
	run := func(d string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", d}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	run(work, "init", "-q", "-b", "main")
	for i, body := range []string{"one\n", "two\n"} {
		if err := os.WriteFile(filepath.Join(work, "f"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		run(work, "add", "f")
		run(work, "commit", "-q", "-m", body)
		if i == 0 {
			first = run(work, "rev-parse", "HEAD")
			run(work, "tag", "-a", "-m", "release", "v20261024.01")
		}
	}
	second = run(work, "rev-parse", "HEAD")
	run(dir, "clone", "-q", "--bare", work, bare)
	run(bare, "config", "uploadpack.allowReachableSHA1InWant", "true")
	return "file://" + bare, first, second
}

func head(t *testing.T, dir string) string {
	t.Helper()
	got, err := git(context.Background(), dir, nil, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// A source is fetched at exactly its pin, whole and with its tags, whatever its branch is at, and
// moved when the pin moves.
func TestFetchChecksOutThePinWithItsHistoryAndTags(t *testing.T) {
	ctx := context.Background()
	url, first, second := upstreamRepo(t)
	dir := filepath.Join(t.TempDir(), "src")

	if err := fetch(ctx, dir, versions.Entry{Name: "src", Source: url, Version: "main", Pin: second}, nil); err != nil {
		t.Fatal(err)
	}
	if got := head(t, dir); got != second {
		t.Errorf("HEAD is %s; want the pin %s", got, second)
	}
	if tags, _ := git(ctx, dir, nil, "tag", "--merged", "HEAD"); tags != "v20261024.01" {
		t.Errorf("tags merged into the pin: %q; want v20261024.01, which refgen reads the module's release from", tags)
	}
	if shallow, _ := git(ctx, dir, nil, "rev-parse", "--is-shallow-repository"); shallow != "false" {
		t.Error("the checkout is shallow; refgen reads its history")
	}

	if err := fetch(ctx, dir, versions.Entry{Name: "src", Source: url, Version: "v20261024.01", Pin: first}, nil); err != nil {
		t.Fatal(err)
	}
	if got := head(t, dir); got != first {
		t.Errorf("after the pin moved, HEAD is %s; want %s", got, first)
	}
}

// A checkout already at its pin is not fetched again: a run after the one with the token needs
// none, and asks nothing of the network.
func TestFetchLeavesACheckoutAtItsPin(t *testing.T) {
	ctx := context.Background()
	url, _, second := upstreamRepo(t)
	dir := filepath.Join(t.TempDir(), "src")
	e := versions.Entry{Name: "src", Source: url, Version: "main", Pin: second}
	if err := fetch(ctx, dir, e, nil); err != nil {
		t.Fatal(err)
	}
	e.Source = "file:///nowhere"
	if err := fetch(ctx, dir, e, nil); err != nil {
		t.Errorf("a checkout at its pin was fetched again: %v", err)
	}
}

// A release whose tag is not its pin is refused: the site would name one release and document
// another tree.
func TestFetchRefusesATagThatIsNotThePin(t *testing.T) {
	url, _, second := upstreamRepo(t)
	err := fetch(context.Background(), filepath.Join(t.TempDir(), "src"), versions.Entry{Name: "src", Source: url, Version: "v20261024.01", Pin: second}, nil)
	if err == nil || !strings.Contains(err.Error(), "tag v20261024.01 is") {
		t.Errorf("fetch = %v; want the tag named as not the pin", err)
	}
}

// A pin the repository does not have fails, and leaves nothing a later run takes for a checkout.
func TestFetchFailsOnACommitTheRepositoryDoesNotHave(t *testing.T) {
	url, _, _ := upstreamRepo(t)
	dir := filepath.Join(t.TempDir(), "src")
	missing := strings.Repeat("ab", 20)
	if err := fetch(context.Background(), dir, versions.Entry{Name: "src", Source: url, Version: "main", Pin: missing}, nil); err == nil {
		t.Fatal("fetched a commit the repository does not have")
	}
	if _, err := git(context.Background(), dir, nil, "rev-parse", "--verify", "-q", "HEAD"); err == nil {
		t.Error("a failed fetch left a checkout with a HEAD")
	}
}

// The token reaches git as configuration in the environment, never as a file, and only when there
// is one.
func TestTheTokenIsAnExtraHeaderInTheEnvironment(t *testing.T) {
	if got := github.GitAuth(""); got != nil {
		t.Errorf("GitAuth with no token = %q; want nothing", got)
	}
	env := strings.Join(github.GitAuth("ghs_x"), "\n")
	for _, want := range []string{"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=http.https://github.com/.extraheader", "GIT_CONFIG_VALUE_0=AUTHORIZATION: basic eC1hY2Nlc3MtdG9rZW46Z2hzX3g="} {
		if !strings.Contains(env, want) {
			t.Errorf("GitAuth lacks %s:\n%s", want, env)
		}
	}
}

// Where each kind of pin moves, and where it does not.
func TestNextMovesForwardOnly(t *testing.T) {
	const pin = "77380b13da641c24e3f79b9c7586c39888f4e7ba"
	main := versions.Entry{Name: "spin", Version: "main", Pin: pin, Track: "github-release"}
	tagged := versions.Entry{Name: "spin-terraform-aws", Version: "v20261024.01", Pin: pin, Track: "github-release"}
	branch := versions.Entry{Name: "brand", Version: "main", Pin: pin, Track: "branch"}
	holds := func(ok bool) func(string) (bool, error) { return func(string) (bool, error) { return ok, nil } }
	for _, c := range []struct {
		name  string
		st    upstream.Status
		holds bool
		want  string
	}{
		{"a branch that moved moves along it", upstream.Status{Entry: branch, Newest: "main", Behind: true}, false, "main"},
		{"a branch that did not stays", upstream.Status{Entry: branch, Newest: "main"}, false, ""},
		{"a newer release", upstream.Status{Entry: tagged, Newest: "v20261025.01", Behind: true}, false, "v20261025.01"},
		{"a newer release the same day, by number", upstream.Status{Entry: tagged, Newest: "v20261024.10", Behind: true}, false, "v20261024.10"},
		{"never back to an older release", upstream.Status{Entry: tagged, Newest: "v20261019.02", Behind: true}, false, ""},
		{"main to a release that holds the commit", upstream.Status{Entry: main, Newest: "v20261025.01", Behind: true}, true, "v20261025.01"},
		{"main to a release the track spells without its v", upstream.Status{Entry: main, Newest: "20261025.01", Behind: true}, true, "v20261025.01"},
		{"main stays past a release that does not", upstream.Status{Entry: main, Newest: "v20261019.02", Behind: true}, false, ""},
	} {
		got, err := next(c.st, holds(c.holds))
		if err != nil || got != c.want {
			t.Errorf("%s: next = %q, %v; want %q", c.name, got, err, c.want)
		}
	}
	for _, st := range []upstream.Status{
		{Entry: tagged, Note: "versions: the API answered 404"},
		{Entry: tagged, Newest: "latest", Behind: true},
		{Entry: tagged, Newest: "v20261024.01", Behind: true, Note: "v20261024.01 is now " + strings.Repeat("cd", 20)},
	} {
		if got, err := next(st, holds(true)); err == nil {
			t.Errorf("next(%+v) = %q; want an error", st, got)
		}
	}
	boom := errors.New("boom")
	if _, err := next(upstream.Status{Entry: main, Newest: "v20261025.01", Behind: true}, func(string) (bool, error) { return false, boom }); !errors.Is(err, boom) {
		t.Errorf("next lost the compare's error: %v", err)
	}
}

// Whether a release holds the pin is GitHub's compare from the pin to the tag: ahead or the same.
func TestHoldsIsTheCompareFromThePin(t *testing.T) {
	const pin = "77380b13da641c24e3f79b9c7586c39888f4e7ba"
	status := map[string]string{"v20261025.01": "ahead", "v20261026.01": "identical", "v20261019.02": "behind", "v20261020.01": "diverged"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ghs_x" {
			http.Error(w, "no token", http.StatusNotFound)
			return
		}
		tag, ok := strings.CutPrefix(r.URL.Path, "/repos/spin-stack/spin/compare/"+pin+"...")
		if !ok || status[tag] == "" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"status":"` + status[tag] + `"}`))
	}))
	defer srv.Close()
	gh := &github.Client{Base: srv.URL, Token: "ghs_x"}
	e := versions.Entry{Source: "https://github.com/spin-stack/spin", Pin: pin}
	for tag, want := range map[string]bool{"v20261025.01": true, "v20261026.01": true, "v20261019.02": false, "v20261020.01": false} {
		if got, err := holds(context.Background(), gh, e, tag); err != nil || got != want {
			t.Errorf("holds(%s) = %v, %v; want %v", tag, got, err, want)
		}
	}
	if _, err := holds(context.Background(), gh, e, "v20990101.01"); err == nil {
		t.Error("holds took a 404 for an answer")
	}
}
