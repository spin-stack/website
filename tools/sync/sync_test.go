package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/spin-stack/go-tools/versions"

	"github.com/spin-stack/website/tools/internal/github"
)

const (
	oldPin = "77380b13da641c24e3f79b9c7586c39888f4e7ba"
	newPin = "b3598142dbcd6ef24135066332441a400dd35980"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// api serves routes, each a path and its query as GitHub is asked them, and fails the test on
// anything else: a request the code was not meant to make is one to know about.
func api(t *testing.T, routes map[string]string) *github.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.RequestURI()]
		if !ok {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.RequestURI())
			http.NotFound(w, r)
			return
		}
		if body == "404" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &github.Client{Base: srv.URL, HTTP: srv.Client()}
}

// The digest puts each commit under every path of spin it touches that feeds a page, with the
// pages, and counts the commits that feed nothing rather than listing them.
func TestDigestGroupsCommitsByThePagesTheyFeed(t *testing.T) {
	dir := t.TempDir()
	docs := filepath.Join(dir, "docs")
	write(t, filepath.Join(docs, "reference/cli/workspace.md"), "---\ntitle: \"spin workspace\"\nsource: \"spin:cmd/cli\"\n---\n")
	write(t, filepath.Join(docs, "reference/api/index.md"), "---\nsource: \"spin:internal/controlplane/apidocs/spin.v1.openapi.json\"\n---\n")
	write(t, filepath.Join(docs, "concepts/network.mdx"), "Some words.\n\n<Figure name=\"datapath\" caption=\"x\" />\n")
	figures := filepath.Join(dir, "figures.json")
	write(t, figures, `{"datapath": {"source": "https://github.com/spin-stack/spin/blob/`+oldPin+`/docs/network/datapath"}}`)
	fds, err := feeds(docs, figures)
	if err != nil {
		t.Fatal(err)
	}

	gh := api(t, map[string]string{
		"/repos/spin-stack/spin/compare/" + oldPin + "...main?per_page=100&page=1": `{"total_commits": 3, "commits": [
			{"sha": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "commit": {"message": "cli: spin open\n\nbody"}},
			{"sha": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "commit": {"message": "network: the datapath drawn again"}},
			{"sha": "cccccccccccccccccccccccccccccccccccccccc", "commit": {"message": "runner: a fix"}}]}`,
		"/repos/spin-stack/spin/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": `{"files": [{"filename": "cmd/cli/cmd/open.go"}, {"filename": "hack/env-catalog.txt"}]}`,
		"/repos/spin-stack/spin/commits/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb": `{"files": [{"filename": "docs/network/datapath.dot"}]}`,
		"/repos/spin-stack/spin/commits/cccccccccccccccccccccccccccccccccccccccc": `{"files": [{"filename": "internal/runner/x.go"}]}`,
	})
	cs, total, err := commits(context.Background(), gh, "spin-stack/spin", oldPin, "main")
	if err != nil {
		t.Fatal(err)
	}
	got := digest(cs, total, fds)
	for _, want := range []string{
		"3 commits.",
		"#### `cmd/cli`\n\nFeeds: reference/cli/workspace\n\n- aaaaaaaaaa cli: spin open\n",
		"#### `hack/env-catalog.txt`\n\nFeeds: every page that names a SPIN_ variable\n\n- aaaaaaaaaa cli: spin open\n",
		"#### `docs/network/datapath`\n\nFeeds: concepts/network\n\n- bbbbbbbbbb network: the datapath drawn again\n",
		"1 of them touch no path the site reads.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("digest lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "runner: a fix") || strings.Contains(got, "apidocs") {
		t.Errorf("digest lists a commit or a path that feeds nothing it touched:\n%s", got)
	}
}

// A directory of more than three pages is said once, with how many: the CLI feeds a page per
// command.
func TestFeedsSayADirectoryOfManyPagesOnce(t *testing.T) {
	got := collapse([]string{"reference/cli/b", "reference/cli", "reference/cli/a", "reference/cli/c", "reference/cli/d", "reference/status", "reference/cli/a"})
	want := []string{"reference/cli", "reference/cli/ (4 pages)", "reference/status"}
	if !slices.Equal(got, want) {
		t.Errorf("collapse = %q; want %q", got, want)
	}
}

// The chain report reads each link, and says of one it cannot read that it could not, instead
// of giving up the rest.
func TestChainReportsEachLinkAndWhatItCouldNotRead(t *testing.T) {
	gh := api(t, map[string]string{
		"/repos/spin-stack/spin/contents/versions.yaml?ref=main": `- name: spin-machine
  kind: download
  source: https://github.com/spin-stack/spin-machine/releases/download/{version}/spin-machine-{version}-linux-x86_64.tar.gz
  version: v20261014.01
  pin: 10b1685e82b8b3e34785e0a7a17d618bac711aa9edee5f13856d1586e345a44e
  track: github-release https://github.com/spin-stack/spin-machine
`,
		"/repos/spin-stack/spin-machine/releases/latest":                                           `{"tag_name": "v20261023.01"}`,
		"/repos/spin-stack/spin-terraform-aws/contents/images.json?ref=main":                       `{"v20261019.9": {}, "v20261019.10": {}, "v20261018.01": {}}`,
		"/repos/spin-stack/spin/releases/latest":                                                   `{"tag_name": "v20261025.01"}`,
		"/repos/spin-stack/spin/actions/workflows/machine.yml/runs?per_page=1":                     `{"workflow_runs": [{"created_at": "2026-10-14T09:00:00Z", "status": "completed", "conclusion": "success", "html_url": "https://github.com/spin-stack/spin/actions/runs/1"}]}`,
		"/repos/spin-stack/spin-terraform-aws/actions/workflows/propose.yml/runs?per_page=1":       `{"workflow_runs": []}`,
		"/repos/spin-stack/spin-terraform-aws/actions/workflows/fresh-install.yml/runs?per_page=1": "404",
		"/repos/spin-stack/spinbox.dev-deploy/contents/terraform.tfvars?ref=main":                  "spin_version = \"v20260928.07\"\ndomain       = \"spinbox.dev\"\n",
		"/repos/spin-stack/spinbox.dev-deploy/contents/main.tf?ref=main":                           "module \"spin\" {\n  source = \"github.com/spin-stack/spin-terraform-aws?ref=1d7e2a2\"\n}\n",
	})
	got := chain(context.Background(), gh)
	for _, want := range []string{
		"| spin-machine | spin main pins v20261014.01; the newest release is **v20261023.01** |",
		"| images.json | the newest release with an image is v20261019.10; spin's newest is **v20261025.01**",
		"| `machine.yml` a spin-machine release proposed to spin | last ran 2026-10-14, [success](https://github.com/spin-stack/spin/actions/runs/1) |",
		"| `propose.yml` the module proposed to the deployments | never ran |",
		"| `fresh-install.yml` an installation from nothing, weekly | could not read: GET /repos/spin-stack/spin-terraform-aws/actions/workflows/fresh-install.yml/runs?per_page=1: 404 Not Found",
		"| spinbox.dev | runs v20260928.07, with the module at `1d7e2a2` |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("chain report lacks %q:\n%s", want, got)
		}
	}
}

// issues is spin's issues as the sync sees them: the bot's, and the requests that write them.
type issues struct {
	mu      sync.Mutex
	byNum   map[int]map[string]any
	created int
}

func (s *issues) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/spin-stack/spin/issues":
			if r.URL.Query().Get("creator") != "sources[bot]" || r.URL.Query().Get("state") != "open" {
				t.Errorf("issues listed by %q", r.URL.RawQuery)
			}
			list := []map[string]any{{"number": 7, "title": "Something else", "body": "x"}}
			for _, i := range s.byNum {
				list = append(list, i)
			}
			_ = json.NewEncoder(w).Encode(list)
		case r.Method == http.MethodPost && r.URL.Path == "/repos/spin-stack/spin/issues":
			var in map[string]any
			_ = json.NewDecoder(r.Body).Decode(&in)
			s.created++
			n := 100 + s.created
			in["number"], in["html_url"] = n, fmt.Sprintf("https://github.com/spin-stack/spin/issues/%d", n)
			s.byNum[n] = in
			_ = json.NewEncoder(w).Encode(in)
		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/repos/spin-stack/spin/issues/"):
			var n int
			_, _ = fmt.Sscanf(strings.TrimPrefix(r.URL.Path, "/repos/spin-stack/spin/issues/"), "%d", &n)
			var in map[string]any
			_ = json.NewDecoder(r.Body).Decode(&in)
			s.byNum[n]["body"] = in["body"]
			_ = json.NewEncoder(w).Encode(s.byNum[n])
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.RequestURI())
			http.NotFound(w, r)
		}
	}
}

// One issue: the first write opens it, the next finds it and writes its own section, keeping the
// others.
func TestTheIssueIsOneAndEachWriteKeepsTheOtherSections(t *testing.T) {
	st := &issues{byNum: map[int]map[string]any{}}
	srv := httptest.NewServer(st.handler(t))
	defer srv.Close()
	gh := &github.Client{Base: srv.URL, HTTP: srv.Client()}
	ctx := context.Background()

	if _, err := writeIssue(ctx, gh, "spin-stack/spin", "sources[bot]", map[string]string{"release": "moved to v20261025.01", "chain": "chain one"}); err != nil {
		t.Fatal(err)
	}
	link, err := writeIssue(ctx, gh, "spin-stack/spin", "sources[bot]", map[string]string{"main": "main is 40 past", "chain": "chain two"})
	if err != nil {
		t.Fatal(err)
	}
	if st.created != 1 || len(st.byNum) != 1 {
		t.Fatalf("%d issues opened; want one", st.created)
	}
	if link != "https://github.com/spin-stack/spin/issues/101" {
		t.Errorf("link = %s", link)
	}
	body := st.byNum[101]["body"].(string)
	if st.byNum[101]["title"] != issueTitle {
		t.Errorf("title = %v", st.byNum[101]["title"])
	}
	for _, want := range []string{"moved to v20261025.01", "main is 40 past", "chain two"} {
		if !strings.Contains(body, want) {
			t.Errorf("the body lacks %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "chain one") || strings.Index(body, "## The last release sync") > strings.Index(body, "## spin main past the pin") {
		t.Errorf("a section was kept that was written over, or the order moved:\n%s", body)
	}
}

func pins(t *testing.T, spin, brand string) *versions.Versions {
	t.Helper()
	v, err := versions.Parse([]byte(fmt.Sprintf(`- name: spin
  kind: git
  source: https://github.com/spin-stack/spin
  version: %s
  pin: %s
  track: github-release
- name: brand
  kind: git
  source: https://github.com/spin-stack/brand
  version: main
  pin: %s
  track: branch
`, map[string]string{oldPin: "main", newPin: "v20261025.01"}[spin], spin, brand)))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// The public pull request names the pins that moved and the files that changed, and nothing of
// what spin's commits say.
func TestThePullRequestSaysWhatMovedAndNothingOfSpinsCommits(t *testing.T) {
	brand := "c882b6f0f3901787525ef16ef29058c9c1418cbe"
	ms := moves(pins(t, oldPin, brand), pins(t, newPin, brand))
	if len(ms) != 1 || ms[0].Name != "spin" || ms[0].FromV != "main" || ms[0].ToV != "v20261025.01" {
		t.Fatalf("moves = %+v; want spin alone, from main to v20261025.01", ms)
	}
	body := prBody(ms, []string{"versions.yaml", "src/data/reference.json"}, "failure", "2026-10-25")
	for _, want := range []string{
		"| spin | main `77380b13da` | v20261025.01 `b3598142db` |",
		"- `src/data/reference.json`",
		"`task reference` did not pass (failure)",
		"on 2026-10-25",
		`spin's issue "Site vs spin main"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the body lacks %q:\n%s", want, body)
		}
	}
	if got := commitMessage(ms); !strings.HasPrefix(got, "sources: spin v20261025.01 b3598142db\n") {
		t.Errorf("commit message = %q", got)
	}
}

// The sync commits as the App's bot, force-pushes the one branch, and updates the pull request
// open from it rather than opening a second.
func TestPublishPushesTheOneBranchAndUpdatesItsPullRequest(t *testing.T) {
	dir := t.TempDir()
	work, bare := filepath.Join(dir, "work"), filepath.Join(dir, "bare.git")
	git := func(d string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", d}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write(t, filepath.Join(work, "versions.yaml"), "old\n")
	git(dir, "init", "-q", "--bare", bare)
	git(work, "init", "-q", "-b", "main")
	git(work, "add", "-A")
	git(work, "commit", "-q", "-m", "init")
	write(t, filepath.Join(work, "versions.yaml"), "new\n")
	t.Chdir(work)

	var patched map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.RequestURI() {
		case "GET /users/sync%5Bbot%5D":
			_, _ = w.Write([]byte(`{"id": 42}`))
		case "GET /repos/spin-stack/website/pulls?head=spin-stack%3Async%2Fsources&state=open":
			_, _ = w.Write([]byte(`[{"number": 9, "html_url": "https://github.com/spin-stack/website/pull/9"}]`))
		case "PATCH /repos/spin-stack/website/pulls/9":
			_ = json.NewDecoder(r.Body).Decode(&patched)
			_, _ = w.Write([]byte(`{"html_url": "https://github.com/spin-stack/website/pull/9"}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.RequestURI())
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	gh := &github.Client{Base: srv.URL, HTTP: srv.Client()}
	ms := []move{{Name: "brand", FromV: "main", FromPin: oldPin, ToV: "main", ToPin: newPin}}
	link, err := publish(context.Background(), gh, "spin-stack/website", bare, "sync[bot]", "", ms, "the body")
	if err != nil {
		t.Fatal(err)
	}
	if link != "https://github.com/spin-stack/website/pull/9" || patched["body"] != "the body" || patched["title"] != "sources: brand main b3598142db" {
		t.Errorf("publish = %s, patched %v", link, patched)
	}
	if got := git(bare, "show", "sync/sources:versions.yaml"); got != "new" {
		t.Errorf("the branch has versions.yaml %q; want the bumped one", got)
	}
	if got := git(bare, "log", "-1", "--format=%an <%ae>", "sync/sources"); got != "sync[bot] <42+sync[bot]@users.noreply.github.com>" {
		t.Errorf("the commit is %s's; want the App's bot", got)
	}
}
