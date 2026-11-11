package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const statesDoc = `{"components":{"schemas":{"spin.v1.WorkspaceState":{
	"enum":["WORKSPACE_STATE_UNSPECIFIED","WORKSPACE_STATE_RUNNING","WORKSPACE_STATE_STOPPED"],
	"x-enum-descriptions":["Never a state.","Its machine is up,\n and a | is a pipe.","Not running."]}}}}`

const statesBefore = "---\ntitle: Workspaces\n---\n\n## States\n\nWhat a state is.\n\n" +
	"| State | Meaning |\n|---|---|\n| `RUNNING` | Up. |\n| `GONE` | A state the API no longer has. |\n\n" +
	"A runner tries a start three times.\n\n## Stop\n\n| Stop | Meaning |\n|---|---|\n"

// sources is a docs directory holding the workspaces page and a spin checkout holding the API's
// document.
func sources(t *testing.T, doc, page string) (docs, spin string) {
	t.Helper()
	root := t.TempDir()
	docs, spin = filepath.Join(root, "docs"), filepath.Join(root, "spin")
	for p, body := range map[string]string{filepath.Join(docs, statesPage): page, filepath.Join(spin, openAPIPath): doc} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return docs, spin
}

// The table of states is the API's, a row a value but the unspecified one, and only the table:
// the prose around it, and the next section's table, are the page's. -check finds a table the
// API no longer says and leaves it; a write makes it the API's.
func TestTheTableOfStatesIsTheAPIs(t *testing.T) {
	docs, spin := sources(t, statesDoc, statesBefore)
	page := filepath.Join(docs, statesPage)
	if stale, err := stateTable(docs, spin, true); err != nil || len(stale) != 1 {
		t.Fatalf("-check: %v, %v; want the page out of date", stale, err)
	}
	if got, _ := os.ReadFile(page); string(got) != statesBefore {
		t.Fatal("-check wrote the page")
	}
	if _, err := stateTable(docs, spin, false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(page)
	want := strings.Replace(statesBefore, "| `RUNNING` | Up. |\n| `GONE` | A state the API no longer has. |\n",
		"| `RUNNING` | Its machine is up, and a \\| is a pipe. |\n| `STOPPED` | Not running. |\n", 1)
	if string(got) != want {
		t.Errorf("the page is\n%s\nwant\n%s", got, want)
	}
	if stale, err := stateTable(docs, spin, true); err != nil || len(stale) != 0 {
		t.Errorf("-check after a write: %v, %v; want it current", stale, err)
	}
}

// A document without each value's description, as protoc-gen-connect-openapi writes one without
// with-enum-value-descriptions, has no table to give: refgen fails rather than keep the old one.
func TestAStateWithoutItsMeaningIsAnError(t *testing.T) {
	docs, spin := sources(t, `{"components":{"schemas":{"spin.v1.WorkspaceState":{"enum":["WORKSPACE_STATE_RUNNING"]}}}}`, statesBefore)
	if _, err := stateTable(docs, spin, false); err == nil || !strings.Contains(err.Error(), "with-enum-value-descriptions") {
		t.Errorf("err = %v; want it to name with-enum-value-descriptions", err)
	}
	docs, spin = sources(t, statesDoc, "## States\n\nNo table.\n\n## Stop\n\n| State | Meaning |\n")
	if _, err := stateTable(docs, spin, false); err == nil {
		t.Error("a page whose States section has no table: want an error, not a table under another section")
	}
}

// A placeholder is a word: what follows it is still read, where a redirection's target is not.
func TestShellCommandsKeepWhatFollowsAPlaceholder(t *testing.T) {
	for line, want := range map[string][]string{
		"spin login --endpoint https://app.<domain> --token <api-key>": {"spin", "login", "--endpoint", "https://app.<domain>", "--token", "<api-key>"},
		"spin exec api -- cat build.log > local.log":                   {"spin", "exec", "api", "--", "cat", "build.log"},
		"spin login --token-stdin < key.txt":                           {"spin", "login", "--token-stdin"},
	} {
		got := shellCommands(line)
		if len(got) != 1 || !slices.Equal(got[0], want) {
			t.Errorf("%s: %q, want %q", line, got, want)
		}
	}
}

// A document of spin's becomes a page: its heading the title, its bare blocks of spin commands
// named as shell so the CLI check reads them, a log's block left alone.
func TestSpinDocBecomesAPage(t *testing.T) {
	title, body, err := titled([]byte("# Who gets in\n\nYou.\n"))
	if err != nil || title != "Who gets in" || string(body) != "\nYou.\n" {
		t.Errorf("titled: %q, %q, %v", title, body, err)
	}
	if _, _, err := titled([]byte("Who gets in\n")); err == nil {
		t.Error("a document with no heading was given a title")
	}
	got := string(shellFences([]byte("```\nspin admin runner list\nspin admin runner activate <runner-id>\n```\n\n```\nWRN no suitable host\n```\n\n```yaml\nspin: true\n```\n")))
	want := "```bash\nspin admin runner list\nspin admin runner activate <runner-id>\n```\n\n```\nWRN no suitable host\n```\n\n```yaml\nspin: true\n```\n"
	if got != want {
		t.Errorf("shellFences:\n%s\nwant:\n%s", got, want)
	}
}

// A flag is read from the help whether it takes a value, takes none, or is printed with the
// values a flag that takes none may be given.
func TestFlagsOfReadsEveryFormAFlagIsPrintedIn(t *testing.T) {
	flags := flagsOf("Usage:\n  spin x [flags]\n\nFlags:\n" +
		"  -h, --help[=true|false]      help for x\n" +
		"      --name string            Workspace name\n" +
		"      --wait[=true|false]      Return once it runs\n" +
		"      --force                  Do not wait\n\nGlobal Flags:\n" +
		"  -o, --output string        text or json\n")
	for name, value := range map[string]bool{"help": false, "h": false, "name": true, "wait": false, "force": false, "output": true, "o": true} {
		f, ok := flags[name]
		if !ok || f.value != value {
			t.Errorf("--%s: read %v, takes a value %v; want read, takes a value %v", name, ok, f.value, value)
		}
	}
}
