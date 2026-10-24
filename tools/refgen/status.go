package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// feature is one row of spin's feature board, as `spin features` prints it
// (internal/features in spin, where a test holds every row to the tree).
type feature struct {
	Area  string `json:"area"`
	Name  string `json:"name"`
	State string `json:"state"`
	// Proof says the feature exists: a procedure, a command or a test by name. It does not say
	// the feature works; EndToEnd does.
	Proof string `json:"proof"`
	Note  string `json:"note"`
	// EndToEnd is the tests that drive the real stack and check a built row, each as
	// go:<dir>.<Test> or pw:<spec>:<title>.
	EndToEnd []string `json:"endToEnd,omitempty"`
	// EndToEndGap is what no test checks yet, of a built row with no EndToEnd.
	EndToEndGap string `json:"endToEndGap,omitempty"`
}

// board is spin's feature board, read from the built CLI so the page cannot say a feature
// the binary does not.
func board(bin, home string) ([]feature, error) {
	cmd := exec.Command(bin, "features")
	cmd.Env = []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "NO_COLOR=1"}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("spin features: %w", err)
	}
	return decodeBoard(out)
}

// decodeBoard refuses a field the site does not know: spin adds to the board's format, and a
// field dropped here is one the pages go on misreading without a word, as the end-to-end tests
// once were.
func decodeBoard(raw []byte) ([]feature, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var rows []feature
	if err := dec.Decode(&rows); err != nil {
		return nil, fmt.Errorf("spin features: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("spin features: more than one board")
	}
	return rows, nil
}

// tests is where spin declares each test a row names: a Go test by <dir>.<Test>, a dashboard
// test by <spec>:<title>, each as path#Lline.
type tests struct {
	goTests map[string]string
	pw      map[string]string
}

// e2eDir is where an e2e: proof's test lives, as spin's features test resolves it.
const e2eDir = "internal/e2e"

// specDir is where a pw: test's spec lives.
const specDir = "dashboard/e2e"

var pwTitle = regexp.MustCompile(`\btest\("([^"]+)"`)

// testFiles is every Go test function in spin and every dashboard test, as tests holds them.
func testFiles(spin string) (tests, error) {
	t := tests{goTests: map[string]string{}, pw: map[string]string{}}
	out, err := git(spin, "grep", "-n", "-E", "^func Test[A-Za-z0-9_]+\\(", "--", "*_test.go")
	if err != nil {
		return t, err
	}
	for _, line := range strings.Split(out, "\n") {
		path, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		n, decl, _ := strings.Cut(rest, ":")
		name := strings.TrimPrefix(decl, "func ")
		name = name[:strings.IndexByte(name, '(')]
		t.goTests[filepath.Dir(path)+"."+name] = fmt.Sprintf("%s#L%s", path, n)
	}
	out, err = git(spin, "grep", "-n", "-F", `test("`, "--", specDir+"/*.spec.ts")
	if err != nil {
		return t, err
	}
	for _, line := range strings.Split(out, "\n") {
		path, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		n, text, _ := strings.Cut(rest, ":")
		for _, m := range pwTitle.FindAllStringSubmatch(text, -1) {
			t.pw[filepath.Base(path)+":"+m[1]] = fmt.Sprintf("%s#L%s", path, n)
		}
	}
	return t, nil
}

// statusPage writes the status page from the board: what is built with what shows it exists and
// what checks it end to end, by area, what is partly there and what is missing, what is not
// built, and the platform's limits. Every test a row names links to where spin declares it at
// the documented commit, and refgen fails on a test the tree does not have.
func statusPage(rows []feature, t tests, commit string) (map[string][]byte, error) {
	var missing []string
	blob := func(at string) string {
		return fmt.Sprintf("https://github.com/spin-stack/spin/blob/%s/%s", commit, at)
	}
	link := func(p string) string {
		kind, what, _ := strings.Cut(p, ":")
		if kind != "e2e" {
			return proof(p)
		}
		at, ok := t.goTests[e2eDir+"."+what]
		if !ok {
			missing = append(missing, p)
			return proof(p)
		}
		return fmt.Sprintf("[`%s`](%s), a test of `%s`", what, blob(at), e2eDir)
	}
	endToEnd := func(r feature) string {
		if len(r.EndToEnd) == 0 {
			return "not yet: " + cell(r.EndToEndGap)
		}
		var out []string
		for _, e := range r.EndToEnd {
			kind, what, _ := strings.Cut(e, ":")
			switch kind {
			case "go":
				i := strings.LastIndex(what, ".")
				at, ok := t.goTests[what]
				if i < 0 || !ok {
					missing = append(missing, e)
					out = append(out, "`"+e+"`")
					continue
				}
				out = append(out, fmt.Sprintf("[`%s`](%s)", what[i+1:], blob(at)))
			case "pw":
				spec, title, _ := strings.Cut(what, ":")
				at, ok := t.pw[what]
				if !ok {
					missing = append(missing, e)
					out = append(out, "`"+e+"`")
					continue
				}
				out = append(out, fmt.Sprintf("[`%s`](%s): %s", spec, blob(at), cell(title)))
			default:
				missing = append(missing, e)
				out = append(out, "`"+e+"`")
			}
		}
		// One a line: a dashboard test's title has commas of its own.
		return strings.Join(out, "<br/>")
	}

	var b strings.Builder
	b.WriteString(frontmatter("spin:internal/features/features.go", "Status", "What Spinbox does today with what proves each, what is partly there, what is not built, and its limits, each row held to the code.", 0))
	b.WriteString("Every row of this page is a line of `internal/features` in spin, and a test there resolves " +
		"it against the tree. A row that says **built** names what shows the feature exists - a procedure " +
		"the control plane serves, a command the CLI has, or a test - and, under **End to end**, the tests " +
		"that drive a real runner and its guest and check what the feature did, or what no test checks " +
		"yet. spin's build checks that each of those tests is there and not quarantined; whether it " +
		"checks the whole feature is its review's. A row that says **not built** is an item of " +
		"[`docs/ROADMAP.md`](https://github.com/spin-stack/spin/blob/" + commit + "/docs/ROADMAP.md). " +
		"The page is written by `task reference` from `spin features` at spin `" + commit[:9] + "`, and " +
		"cannot say a feature the binary does not.\n\n")

	b.WriteString("## Built\n\n")
	var areas []string
	byArea := map[string][]feature{}
	for _, r := range rows {
		if r.State != "built" {
			continue
		}
		if _, ok := byArea[r.Area]; !ok {
			areas = append(areas, r.Area)
		}
		byArea[r.Area] = append(byArea[r.Area], r)
	}
	for _, a := range areas {
		fmt.Fprintf(&b, "### %s\n\n| Feature | Exists | End to end |\n|---|---|---|\n", a)
		for _, r := range byArea[a] {
			name := cell(r.Name)
			// A built row's note is a caveat on it, which a reader must see beside the claim.
			if r.Note != "" {
				name += "<br/>*" + strings.TrimSuffix(cell(r.Note), ".") + ".*"
			}
			fmt.Fprintf(&b, "| %s | %s | %s |\n", name, link(r.Proof), endToEnd(r))
		}
		b.WriteString("\n")
	}
	b.WriteString("## Partly there\n\n")
	partial := 0
	for _, r := range rows {
		if r.State == "partial" {
			if partial == 0 {
				b.WriteString("| Area | Feature | What exists | What is missing |\n|---|---|---|---|\n")
			}
			partial++
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", r.Area, cell(r.Name), link(r.Proof), cell(r.Note))
		}
	}
	if partial == 0 {
		b.WriteString("Nothing today.\n")
	}
	b.WriteString("\n## Not built\n\nEach is an item of `docs/ROADMAP.md`, with the command that checks it.\n\n")
	for _, r := range rows {
		if r.State == "not" {
			fmt.Fprintf(&b, "- **%s** (%s)\n", r.Name, r.Area)
		}
	}
	b.WriteString("\n## Limits\n\nNot planned: a limit of the platform or of where it runs today. " +
		"[Limitations](/overview/limitations/) says what each means.\n\n")
	for _, r := range rows {
		if r.State == "limit" {
			if r.Note == "" {
				fmt.Fprintf(&b, "- **%s** (%s)\n", r.Name, r.Area)
			} else {
				fmt.Fprintf(&b, "- **%s** (%s): %s\n", r.Name, r.Area, r.Note)
			}
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("the board names tests spin does not have: %v", missing)
	}
	return map[string][]byte{filepath.Join("status", "index.md"): []byte(b.String())}, nil
}

// proof is a proof as the page shows it: the procedure linked to its service's page, the
// command linked to its page, the test by name.
func proof(p string) string {
	kind, what, _ := strings.Cut(p, ":")
	switch kind {
	case "rpc":
		service, method, _ := strings.Cut(what, "/")
		short := service[strings.LastIndex(service, ".")+1:]
		return fmt.Sprintf("[`%s/%s`](/reference/api/%s/)", short, method, strings.ToLower(short))
	case "cli":
		top := strings.Fields(what)[0]
		return fmt.Sprintf("[`spin %s`](/reference/cli/%s/)", what, top)
	case "e2e":
		return fmt.Sprintf("`%s`, a test of `%s`", what, e2eDir)
	}
	return "`" + p + "`"
}

// limitLead is the lead of a bullet on the limitations page: bold, ending its sentence.
var limitLead = regexp.MustCompile(`(?m)^- \*\*(.+?)\.?\*\*`)

// checkLimits holds the limitations page to the board's limit rows, both ways: a bullet whose
// lead is no limit row, and a limit row no bullet leads with, are each one line to fix. The
// landing page's straight answers are limit rows too, a few of them: each must be one.
func checkLimits(docs string, rows []feature) ([]string, error) {
	limits := map[string]bool{}
	for _, r := range rows {
		if r.State == "limit" {
			limits[r.Name] = true
		}
	}
	page := filepath.Join(docs, "overview", "limitations.md")
	raw, err := os.ReadFile(page)
	if err != nil {
		return nil, err
	}
	var bad []string
	leads := bulletLeads(raw)
	for lead := range leads {
		if !limits[lead] {
			bad = append(bad, fmt.Sprintf("%s: the bullet %q is not a limit row of spin's feature board", page, lead))
		}
	}
	for limit := range limits {
		if !leads[limit] {
			bad = append(bad, fmt.Sprintf("%s: no bullet leads with %q, a limit row of spin's feature board", page, limit))
		}
	}

	landing := filepath.Join(docs, "index.mdx")
	raw, err = os.ReadFile(landing)
	if err != nil {
		return nil, err
	}
	answers := section(raw, "## Straight answers")
	if answers == nil {
		return append(bad, landing+": no \"## Straight answers\" section"), nil
	}
	leads = bulletLeads(answers)
	if len(leads) == 0 {
		bad = append(bad, landing+": the straight answers have no bullet")
	}
	for lead := range leads {
		if !limits[lead] {
			bad = append(bad, fmt.Sprintf("%s: the straight answer %q is not a limit row of spin's feature board", landing, lead))
		}
	}
	return bad, nil
}

// bulletLeads is the bold lead of every bullet in raw.
func bulletLeads(raw []byte) map[string]bool {
	leads := map[string]bool{}
	for _, m := range limitLead.FindAllSubmatch(raw, -1) {
		leads[string(bytes.TrimSpace(m[1]))] = true
	}
	return leads
}

// section is the text under a heading, up to the next heading of any level, or nil when the
// page has no such heading.
func section(raw []byte, heading string) []byte {
	_, after, ok := bytes.Cut(raw, []byte("\n"+heading+"\n"))
	if !ok {
		return nil
	}
	if i := bytes.Index(after, []byte("\n#")); i >= 0 {
		after = after[:i]
	}
	return after
}
