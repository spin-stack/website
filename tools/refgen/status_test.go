package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureBoard is testdata/board.json: a board as `spin features` prints it, a row of each kind.
func fixtureBoard(t *testing.T) []feature {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "board.json"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := decodeBoard(raw)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// fixtureTests is where the fixture board's tests are declared, as testFiles reads them from spin.
var fixtureTests = tests{
	goTests: map[string]string{
		"internal/e2e.TestAWorkspaceBootsARealGuest": "internal/e2e/vm_test.go#L167",
		"internal/e2e.TestABusyWorkspaceIsLentCores": "internal/e2e/background_test.go#L342",
		"internal/e2e.TestTheVSCodeServerAnswers":    "internal/e2e/vscodeserver_test.go#L86",
		"cmd/cli.TestE2E_WorkspaceCRUD":              "cmd/cli/e2e_test.go#L397",
	},
	pw: map[string]string{
		"auth.spec.ts:the CLI's device page stands alone, and keeps its code": "dashboard/e2e/auth.spec.ts#L19",
	},
}

const fixtureCommit = "0677b0410423d07e9a4623a22fc8e0ececc6f5e0"

// A field the site does not know is an error, not a field dropped: the end-to-end tests were
// once dropped so, and the landing page counted the wrong rows for it.
func TestTheBoardRefusesAFieldTheSiteDoesNotKnow(t *testing.T) {
	_, err := decodeBoard([]byte(`[{"area": "A", "name": "N", "state": "built", "proof": "cli:x", "endToEnd": ["go:a.TestB"], "provedBy": "x"}]`))
	if err == nil || !strings.Contains(err.Error(), "provedBy") {
		t.Fatalf("decodeBoard took an unknown field: %v", err)
	}
	rows := fixtureBoard(t)
	if got := rows[1].EndToEnd; len(got) != 2 || got[0] != "go:cmd/cli.TestE2E_WorkspaceCRUD" {
		t.Errorf("endToEnd = %q, want the row's two tests", got)
	}
	if got := rows[2].EndToEndGap; got != "memory lent and taken back" {
		t.Errorf("endToEndGap = %q", got)
	}
}

// The landing page's number is the built rows that name an end-to-end test, whatever kind their
// proof is: an e2e: proof with a gap is not one, a cli: or rpc: proof with a test is.
func TestCountsAreTheBuiltRowsWithAnEndToEndTest(t *testing.T) {
	c, err := countRows(fixtureBoard(t))
	if err != nil {
		t.Fatal(err)
	}
	want := counts{Built: 4, Partial: 1, NotBuilt: 1, Limits: 1, EndToEnd: 3, Gap: 1}
	if c != want {
		t.Errorf("counts = %+v, want %+v", c, want)
	}
}

func TestABuiltRowNamesItsTestsOrItsGapNotBothNorNeither(t *testing.T) {
	for name, row := range map[string]feature{
		"both":    {Name: "both", State: "built", EndToEnd: []string{"go:a.TestB"}, EndToEndGap: "x"},
		"neither": {Name: "neither", State: "built"},
		"partial": {Name: "partial", State: "partial", EndToEnd: []string{"go:a.TestB"}},
		"limit":   {Name: "limit", State: "limit", EndToEndGap: "x"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := countRows([]feature{row}); err == nil {
				t.Errorf("countRows took a %s row", name)
			}
		})
	}
}

func TestTheStatusPageShowsWhatChecksEachBuiltRowEndToEnd(t *testing.T) {
	pages, err := statusPage(fixtureBoard(t), fixtureTests, fixtureCommit)
	if err != nil {
		t.Fatal(err)
	}
	page := string(pages[filepath.Join("status", "index.md")])
	blob := "https://github.com/spin-stack/spin/blob/" + fixtureCommit + "/"
	for _, want := range []string{
		"| Feature | Exists | End to end |",
		// A go: test linked where its directory declares it, a pw: test by its spec and title,
		// one a line.
		"[`TestE2E_WorkspaceCRUD`](" + blob + "cmd/cli/e2e_test.go#L397)<br/>[`auth.spec.ts`](" + blob + "dashboard/e2e/auth.spec.ts#L19): the CLI's device page stands alone, and keeps its code |",
		"| not yet: memory lent and taken back |",
		// A built row's note is on the page.
		"| VS Code over SSH<br/>*spin code itself is run by its unit tests only.* |",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page has no %q:\n%s", want, page)
		}
	}
	// An e2e: proof says the feature exists; it is not called an end-to-end test.
	if strings.Contains(page, "an end-to-end test") {
		t.Errorf("the page calls a proof an end-to-end test:\n%s", page)
	}
}

func TestTheStatusPageFailsOnATestSpinDoesNotHave(t *testing.T) {
	for name, row := range map[string]feature{
		"go":   {Area: "A", Name: "go", State: "built", Proof: "cli:x", EndToEnd: []string{"go:internal/e2e.TestGone"}},
		"pw":   {Area: "A", Name: "pw", State: "built", Proof: "cli:x", EndToEnd: []string{"pw:auth.spec.ts:a title gone"}},
		"kind": {Area: "A", Name: "kind", State: "built", Proof: "cli:x", EndToEnd: []string{"sh:hack/e2e.sh"}},
		// A Go test of that name, but in another directory than the row names.
		"dir":   {Area: "A", Name: "dir", State: "built", Proof: "cli:x", EndToEnd: []string{"go:cmd/cli.TestAWorkspaceBootsARealGuest"}},
		"proof": {Area: "A", Name: "proof", State: "built", Proof: "e2e:TestGone", EndToEndGap: "x"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := statusPage([]feature{row}, fixtureTests, fixtureCommit); err == nil {
				t.Errorf("statusPage took %q", row.EndToEnd)
			}
		})
	}
}
