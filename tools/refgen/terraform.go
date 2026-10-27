package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
)

// tfModule is one module of spin-terraform-aws the site has a page of.
type tfModule struct {
	dir, slug, title, what string
	// source is the file the page links as its source: the one its variables are declared in.
	source string
	// optional is a module an installation does not have unless it is added beside it: the root
	// does not call it, so what it serves is not what an installation needs.
	optional bool
}

// The modules an installation is made of, in the order a reader meets them, then the ones added
// beside it.
var tfModules = []tfModule{
	{".", "root", "The installation module", "the module an installation uses: every variable it takes and every output it gives", "variables.tf", false},
	{"modules/controlplane", "controlplane", "modules/controlplane", "the network, bucket, database, control plane and proxy, used on its own", "modules/controlplane/variables.tf", false},
	{"modules/runners", "runners", "modules/runners", "the runners' autoscaling group, used on its own", "modules/runners/variables.tf", false},
	{"bootstrap", "bootstrap", "bootstrap", "the once-per-account state bucket and key, and CI's roles", "bootstrap/variables.tf", false},
	{"modules/identity-issuer", "identity-issuer", "modules/identity-issuer", "the optional identity issuer at id.<domain>, added beside an installation", "modules/identity-issuer/issuer.tf", true},
}

// terraformPages reads each module's variables and outputs as OpenTofu would, so a variable
// renamed or a default changed is a page changed. release is the module's newest release, which
// each page's source pins.
func terraformPages(tf, release string) (map[string][]byte, error) {
	pages := map[string][]byte{}
	for i, m := range tfModules {
		mod, diags := tfconfig.LoadModule(filepath.Join(tf, m.dir))
		if diags.HasErrors() {
			return nil, fmt.Errorf("reading %s: %w", m.dir, diags.Err())
		}
		var b strings.Builder
		b.WriteString(frontmatter("spin-terraform-aws:"+m.source, m.title, "spin-terraform-aws, "+m.what+".", i+1))
		src := "github.com/spin-stack/spin-terraform-aws"
		if m.dir != "." {
			src += "//" + m.dir
		}
		if m.optional {
			fmt.Fprintf(&b, "Not part of an installation unless added beside it: the installation module does not call it. "+
				"Its [README](https://github.com/spin-stack/spin-terraform-aws/tree/%s/%s) says what it needs and how to add it.\n\n", release, m.dir)
		}
		fmt.Fprintf(&b, "Source: `%s?ref=%s`. Generated from its variables and outputs.\n\n", src, release)

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
			// Every variable at a glance first, each linked to its description below.
			b.WriteString("## Variables\n\n| Variable | Type | Default |\n|---|---|---|\n")
			for _, v := range vars {
				def := "**required**"
				switch {
				case v.Required:
				case v.Default == nil:
					def = "the module's own"
				default:
					def = "`" + strings.ReplaceAll(oneLine(fmt.Sprint(jsonish(v.Default))), "|", `\|`) + "`"
				}
				fmt.Fprintf(&b, "| [`%s`](#%s) | `%s` | %s |\n", v.Name, anchor(v.Name), strings.ReplaceAll(oneLine(v.Type), "|", `\|`), def)
			}
			b.WriteString("\n")
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
