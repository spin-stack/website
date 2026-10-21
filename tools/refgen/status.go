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
	Proof string `json:"proof"`
	Note  string `json:"note"`
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
	var rows []feature
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, fmt.Errorf("spin features: %w", err)
	}
	return rows, nil
}

// statusPage writes the status page from the board: what is built with what proves it, what is
// partly there and what is missing, what is not built, and the platform's limits.
func statusPage(rows []feature) map[string][]byte {
	var b strings.Builder
	b.WriteString(frontmatter("Status", "What Spin does today with what proves each, what is partly there, what is not built, and the platform's limits - every row held to the tree by a test.", 0))
	b.WriteString("Every row of this page is a line of `internal/features` in spin, and a test there resolves " +
		"its proof against the tree: a procedure the control plane serves, a command the CLI has, or a " +
		"test the end-to-end lane runs on real machines. A row that says **built** names one; a row that " +
		"says **not built** is an item of [`docs/ROADMAP.md`](https://github.com/spin-stack/spin/blob/main/docs/ROADMAP.md), " +
		"each checkable in one command. The page is written by `task reference` from `spin features` and " +
		"cannot say a feature the binary does not.\n\n")

	b.WriteString("## Built\n\n| Area | Feature | Proof |\n|---|---|---|\n")
	for _, r := range rows {
		if r.State == "built" {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", r.Area, cell(r.Name), proof(r.Proof))
		}
	}
	b.WriteString("\n## Partly there\n\n| Area | Feature | What exists | What is missing |\n|---|---|---|---|\n")
	for _, r := range rows {
		if r.State == "partial" {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", r.Area, cell(r.Name), proof(r.Proof), cell(r.Note))
		}
	}
	b.WriteString("\n## Not built\n\nEach is an item of `docs/ROADMAP.md`, with the command that checks it.\n\n")
	for _, r := range rows {
		if r.State == "not" {
			fmt.Fprintf(&b, "- **%s** (%s)\n", r.Name, r.Area)
		}
	}
	b.WriteString("\n## Limits\n\nNot planned: a limit of the platform or of where it runs today. " +
		"[Status and limitations](/overview/limitations/) says what each means.\n\n")
	for _, r := range rows {
		if r.State == "limit" {
			if r.Note == "" {
				fmt.Fprintf(&b, "- **%s** (%s)\n", r.Name, r.Area)
			} else {
				fmt.Fprintf(&b, "- **%s** (%s): %s\n", r.Name, r.Area, r.Note)
			}
		}
	}
	return map[string][]byte{filepath.Join("status", "index.md"): []byte(b.String())}
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
		return fmt.Sprintf("`%s`, an end-to-end test", what)
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
