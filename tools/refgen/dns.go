package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// domainPlaceholder is var.domain as the pages write it.
const domainPlaceholder = "<domain>"

// servedNames is the names one module's own Route 53 records give, read from its HCL with
// var.domain as the pages write it. A record whose name the module cannot say alone - an ACM
// validation record, named by the certificate AWS issues - is no name a reader points anywhere,
// and evaluates to nothing here.
func servedNames(dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	if err != nil {
		return nil, err
	}
	p := hclparse.NewParser()
	locals := map[string]hcl.Expression{}
	var records []*hclsyntax.Body
	for _, f := range files {
		file, diags := p.ParseHCLFile(f)
		if diags.HasErrors() {
			return nil, fmt.Errorf("%s: %w", f, diags)
		}
		for _, b := range file.Body.(*hclsyntax.Body).Blocks {
			switch {
			case b.Type == "locals":
				for name, a := range b.Body.Attributes {
					locals[name] = a.Expr
				}
			case b.Type == "resource" && len(b.Labels) == 2 && b.Labels[0] == "aws_route53_record":
				records = append(records, b.Body)
			}
		}
	}
	ctx := &hcl.EvalContext{
		Variables: map[string]cty.Value{"var": cty.ObjectVal(map[string]cty.Value{"domain": cty.StringVal(domainPlaceholder)})},
		// toset is how a record set is written; what it is a set of is all that matters here.
		Functions: map[string]function.Function{"toset": function.New(&function.Spec{
			Params: []function.Parameter{{Name: "v", Type: cty.DynamicPseudoType}},
			Type:   function.StaticReturnType(cty.DynamicPseudoType),
			Impl:   func(args []cty.Value, _ cty.Type) (cty.Value, error) { return args[0], nil },
		})},
	}
	// A local may name another: each pass resolves those whose inputs are resolved, and what
	// needs more than var.domain never is.
	known := map[string]cty.Value{}
	for progress := true; progress; {
		progress = false
		ctx.Variables["local"] = cty.ObjectVal(known)
		for name, e := range locals {
			if _, ok := known[name]; ok {
				continue
			}
			if v, ok := evalKnown(e, ctx); ok {
				known[name], progress = v, true
			}
		}
	}
	ctx.Variables["local"] = cty.ObjectVal(known)
	var names []string
	for _, r := range records {
		name, ok := r.Attributes["name"]
		if !ok {
			continue
		}
		each, ok := r.Attributes["for_each"]
		if !ok {
			if v, ok := evalKnown(name.Expr, ctx); ok && v.Type() == cty.String {
				names = append(names, v.AsString())
			}
			continue
		}
		set, ok := evalKnown(each.Expr, ctx)
		if !ok || !set.CanIterateElements() {
			continue
		}
		for it := set.ElementIterator(); it.Next(); {
			k, v := it.Element()
			if set.Type().IsSetType() || set.Type().IsListType() || set.Type().IsTupleType() {
				k = v
			}
			c := ctx.NewChild()
			c.Variables = map[string]cty.Value{"each": cty.ObjectVal(map[string]cty.Value{"key": k, "value": v})}
			if n, ok := evalKnown(name.Expr, c); ok && n.Type() == cty.String {
				names = append(names, n.AsString())
			}
		}
	}
	sort.Strings(names)
	return slices.Compact(names), nil
}

// evalKnown is e's value when the context says all of it.
func evalKnown(e hcl.Expression, ctx *hcl.EvalContext) (cty.Value, bool) {
	v, diags := e.Value(ctx)
	if diags.HasErrors() || !v.IsWhollyKnown() || v.IsNull() {
		return cty.NilVal, false
	}
	return v, true
}

// installationNames is what the root module serves: its own records and those of every module it
// calls from this repository, as far down as they go.
func installationNames(tf string) ([]string, error) {
	var names []string
	var walk func(dir string) error
	walk = func(dir string) error {
		n, err := servedNames(dir)
		if err != nil {
			return err
		}
		names = append(names, n...)
		mod, diags := tfconfig.LoadModule(dir)
		if diags.HasErrors() {
			return fmt.Errorf("reading %s: %w", dir, diags.Err())
		}
		for _, c := range mod.ModuleCalls {
			if strings.HasPrefix(c.Source, "./") || strings.HasPrefix(c.Source, "../") {
				if err := walk(filepath.Join(dir, c.Source)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(tf); err != nil {
		return nil, err
	}
	sort.Strings(names)
	return slices.Compact(names), nil
}

// domainRow is a row of a table of names: the name, and the module that serves it when the
// table says.
var domainRow = regexp.MustCompile("(?m)^\\| `([^`]+)` \\|(?: \\[`(modules/[\\w-]+)`\\])?")

// checkDomains holds aws/prerequisites.md's names to the modules: the table under "A domain" is
// every name the installation serves and nothing else, and the one under "Optional modules" is
// every name each optional module serves, beside that module. A name the module adds is one a
// reader must point the domain at, and the page is where they learn it.
func checkDomains(docs, tf string) ([]string, error) {
	page := filepath.Join(docs, "aws", "prerequisites.md")
	raw, err := os.ReadFile(page)
	if err != nil {
		return nil, err
	}
	want, err := installationNames(tf)
	if err != nil {
		return nil, err
	}
	required := section(raw, "## A domain")
	if required == nil {
		return []string{page + `: no "## A domain" section`}, nil
	}
	var have []string
	for _, m := range domainRow.FindAllSubmatch(required, -1) {
		have = append(have, string(m[1]))
	}
	bad := diffNames(page, "the installation", want, have)

	optional := section(raw, "### Optional modules")
	if optional == nil {
		return append(bad, page+`: no "### Optional modules" section`), nil
	}
	byModule := map[string][]string{}
	for _, m := range domainRow.FindAllSubmatch(optional, -1) {
		byModule[string(m[2])] = append(byModule[string(m[2])], string(m[1]))
	}
	for _, m := range tfModules {
		if !m.optional {
			continue
		}
		names, err := servedNames(filepath.Join(tf, m.dir))
		if err != nil {
			return nil, err
		}
		bad = append(bad, diffNames(page, m.dir, names, byModule[m.dir])...)
		delete(byModule, m.dir)
	}
	for mod, names := range byModule {
		bad = append(bad, fmt.Sprintf("%s: the optional modules name %v under %q, which is no optional module of spin-terraform-aws", page, names, mod))
	}
	sort.Strings(bad)
	return bad, nil
}

// diffNames says which names of what serves the page leaves out, and which the page has that it
// does not serve.
func diffNames(page, what string, want, have []string) []string {
	var bad []string
	for _, n := range want {
		if !slices.Contains(have, n) {
			bad = append(bad, fmt.Sprintf("%s: %s serves %s, and the page does not name it", page, what, n))
		}
	}
	for _, n := range have {
		if !slices.Contains(want, n) {
			bad = append(bad, fmt.Sprintf("%s: names %s, which %s does not serve", page, n, what))
		}
	}
	return bad
}
