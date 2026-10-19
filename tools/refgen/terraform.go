package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
)

// The modules an installation is made of, in the order a reader meets them.
var tfModules = []struct{ dir, slug, title, what string }{
	{".", "root", "The installation module", "the module an installation uses: every variable it takes and every output it gives"},
	{"modules/controlplane", "controlplane", "modules/controlplane", "the network, bucket, database, control plane and proxy, used on its own"},
	{"modules/runners", "runners", "modules/runners", "the runners' autoscaling group, used on its own"},
	{"bootstrap", "bootstrap", "bootstrap", "the once-per-account state bucket and key, and CI's roles"},
}

// terraformPages reads each module's variables and outputs as OpenTofu would, so a variable
// renamed or a default changed is a page changed.
func terraformPages(tf string) (map[string][]byte, error) {
	pages := map[string][]byte{}
	for i, m := range tfModules {
		mod, diags := tfconfig.LoadModule(filepath.Join(tf, m.dir))
		if diags.HasErrors() {
			return nil, fmt.Errorf("reading %s: %w", m.dir, diags.Err())
		}
		var b strings.Builder
		b.WriteString(frontmatter(m.title, "spin-terraform-aws, "+m.what+".", i+1))
		src := "github.com/spin-stack/spin-terraform-aws"
		if m.dir != "." {
			src += "//" + m.dir
		}
		fmt.Fprintf(&b, "Source: `%s?ref=<tag>`. Generated from its `variables.tf` and `outputs.tf`.\n\n", src)

		vars := make([]*tfconfig.Variable, 0, len(mod.Variables))
		for _, v := range mod.Variables {
			vars = append(vars, v)
		}
		sort.Slice(vars, func(a, b int) bool {
			if vars[a].Required != vars[b].Required {
				return vars[a].Required
			}
			return before(vars[a].Pos, vars[b].Pos)
		})
		if len(vars) > 0 {
			b.WriteString("## Variables\n\n")
		}
		for _, v := range vars {
			fmt.Fprintf(&b, "### %s\n\n", v.Name)
			fmt.Fprintf(&b, "%s\n\n", prose(v.Description))
			fmt.Fprintf(&b, "- Type: `%s`\n", oneLine(v.Type))
			switch {
			case v.Required:
				b.WriteString("- **Required**\n")
			case v.Default == nil:
				b.WriteString("- Default: `null` (the module's own default)\n")
			default:
				fmt.Fprintf(&b, "- Default: `%s`\n", oneLine(fmt.Sprint(jsonish(v.Default))))
			}
			if v.Sensitive {
				b.WriteString("- Sensitive\n")
			}
			b.WriteString("\n")
		}

		outs := make([]*tfconfig.Output, 0, len(mod.Outputs))
		for _, o := range mod.Outputs {
			outs = append(outs, o)
		}
		sort.Slice(outs, func(a, b int) bool { return before(outs[a].Pos, outs[b].Pos) })
		if len(outs) > 0 {
			b.WriteString("## Outputs\n\n| Output | What it is |\n|---|---|\n")
		}
		for _, o := range outs {
			fmt.Fprintf(&b, "| `%s` | %s |\n", o.Name, cell(o.Description))
		}
		pages[filepath.Join("terraform", m.slug+".md")] = []byte(b.String())
	}
	return pages, nil
}

// before orders declarations by file, then by line: a module's are spread over several files,
// and tfconfig hands them over in a map, so the line alone orders two files' at random.
func before(a, b tfconfig.SourcePos) bool {
	if a.Filename != b.Filename {
		return a.Filename < b.Filename
	}
	return a.Line < b.Line
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func cell(s string) string {
	return strings.ReplaceAll(prose(oneLine(s)), "|", `\|`)
}

// prose makes a description safe to put in Markdown as text: `<domain>` would otherwise be read
// as a tag and vanish, and `*.ws.` would open an emphasis.
func prose(s string) string {
	return strings.NewReplacer("<", "&lt;", "*", `\*`).Replace(strings.TrimSpace(s))
}
