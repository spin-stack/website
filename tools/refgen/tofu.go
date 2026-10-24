package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
)

// moduleCall is what the pages call the installation's module: `module "spin"`, as aws/overview
// writes it, so a state address on a page starts module.spin.
const moduleCall = "spin"

// notTerraformNames are the snake_case names an aws/ page writes in backticks that are no
// variable or output of a module, each with why it may be there.
var notTerraformNames = map[string]string{
	// The removal (spin-terraform-aws's README) says what decommission turns on in the plan: an
	// argument of the aws_s3_bucket resource, which a reader sees in that plan.
	"force_destroy": "an argument of aws_s3_bucket the removal's plan shows",
}

// tfNames is what the pages may name of spin-terraform-aws: each module's variables and
// outputs, the root's apart, since a -var and a tofu output reach only the root's.
type tfNames struct {
	root, bootstrap *tfconfig.Module
	// any is every variable and output of every module the site has a page of.
	any map[string]bool
}

func loadTFNames(tf string) (tfNames, error) {
	n := tfNames{any: map[string]bool{}}
	for _, m := range tfModules {
		mod, diags := tfconfig.LoadModule(filepath.Join(tf, m.dir))
		if diags.HasErrors() {
			return n, fmt.Errorf("reading %s: %w", m.dir, diags.Err())
		}
		switch m.dir {
		case ".":
			n.root = mod
		case "bootstrap":
			n.bootstrap = mod
		}
		for v := range mod.Variables {
			n.any[v] = true
		}
		for o := range mod.Outputs {
			n.any[o] = true
		}
	}
	return n, nil
}

var (
	// snakeName is a name in backticks on its own, or set to a value: `runner_spot = false`.
	snakeName = regexp.MustCompile("`([a-z][a-z0-9]*(?:_[a-z0-9]+)+)(?: = [^`]*)?`")
	codeBlock = regexp.MustCompile("(?ms)^```.*?^```")
	// resourceIndex is an instance's key in an address: [0], ["a"].
	resourceIndex = regexp.MustCompile(`\[[^\]]*\]`)
)

// checkTofu holds what the aws/ pages tell people to type and to set to spin-terraform-aws: a
// `tofu state rm` or `tofu import` address must be a resource the module has, a -var and a
// `tofu output` must name the root's variable and output, and a snake_case name in backticks
// must be a variable or an output of a module, or one of notTerraformNames. A renamed variable
// would otherwise leave a page telling people to set what the module no longer reads.
func checkTofu(docs, tf string) ([]string, error) {
	names, err := loadTFNames(tf)
	if err != nil {
		return nil, err
	}
	pages, err := filepath.Glob(filepath.Join(docs, "aws", "*.md*"))
	if err != nil {
		return nil, err
	}
	sort.Strings(pages)
	var bad []string
	for _, path := range pages {
		lines, err := shellLines(path)
		if err != nil {
			return nil, err
		}
		block, mod := -1, names.root
		for _, l := range lines {
			if l.block != block {
				block, mod = l.block, names.root
			}
			for _, words := range shellCommands(l.text) {
				if len(words) > 1 && words[0] == "cd" {
					mod = names.root
					if filepath.Base(words[1]) == "bootstrap" {
						mod = names.bootstrap
					}
				}
				for _, why := range checkTofuCommand(words, mod, names) {
					bad = append(bad, fmt.Sprintf("%s:%d: %s", path, l.n, why))
				}
			}
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		// Blanked, not removed, so a line number is still the page's.
		prose := codeBlock.ReplaceAllFunc(raw, func(b []byte) []byte { return bytes.Repeat([]byte("\n"), bytes.Count(b, []byte("\n"))) })
		for i, line := range bytes.Split(prose, []byte("\n")) {
			for _, m := range snakeName.FindAllSubmatch(line, -1) {
				if n := string(m[1]); !names.any[n] && notTerraformNames[n] == "" {
					bad = append(bad, fmt.Sprintf("%s:%d: `%s` is no variable or output of spin-terraform-aws's modules", path, i+1, n))
				}
			}
		}
	}
	return bad, nil
}

// checkTofuCommand says what of one tofu command the module mod does not have. mod is the
// module the command runs in: the root's, or bootstrap's after a cd into it or a -chdir.
func checkTofuCommand(words []string, mod *tfconfig.Module, names tfNames) []string {
	if len(words) == 0 || words[0] != "tofu" {
		return nil
	}
	var args []string
	for _, w := range words[1:] {
		// What a substitution prints is no word of the page's: the rest is the substitution's.
		if strings.HasPrefix(w, "$") {
			break
		}
		if dir, ok := strings.CutPrefix(w, "-chdir="); ok {
			if filepath.Base(dir) == "bootstrap" {
				mod = names.bootstrap
			}
			continue
		}
		args = append(args, w)
	}
	var bad []string
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-var" && i+1 < len(args):
			i++
			bad = append(bad, checkVar(args[i], mod)...)
		case strings.HasPrefix(a, "-var="):
			bad = append(bad, checkVar(strings.TrimPrefix(a, "-var="), mod)...)
		case strings.HasPrefix(a, "-"):
		default:
			positional = append(positional, a)
		}
	}
	switch {
	case len(positional) >= 2 && positional[0] == "output":
		if _, ok := mod.Outputs[positional[1]]; !ok {
			bad = append(bad, fmt.Sprintf("tofu output %s: the module has no output %s", positional[1], positional[1]))
		}
	case len(positional) >= 2 && positional[0] == "state" && positional[1] == "rm":
		for _, addr := range positional[2:] {
			bad = append(bad, checkAddress(addr, mod)...)
		}
	case len(positional) >= 2 && positional[0] == "import":
		bad = append(bad, checkAddress(positional[1], mod)...)
	}
	return bad
}

func checkVar(assign string, mod *tfconfig.Module) []string {
	name, _, _ := strings.Cut(assign, "=")
	if _, ok := mod.Variables[name]; !ok {
		return []string{fmt.Sprintf("-var %s: the module has no variable %s", assign, name)}
	}
	return nil
}

// checkAddress follows a state address from the installation's module call down the module
// calls it names, and says whether the resource it ends in is one that module declares.
func checkAddress(addr string, root *tfconfig.Module) []string {
	parts := strings.Split(resourceIndex.ReplaceAllString(addr, ""), ".")
	if len(parts) < 2 || parts[0] != "module" || parts[1] != moduleCall {
		return []string{fmt.Sprintf("%s: an address of the installation starts module.%s", addr, moduleCall)}
	}
	mod, dir, parts := root, ".", parts[2:]
	for len(parts) >= 2 && parts[0] == "module" {
		call, ok := mod.ModuleCalls[parts[1]]
		if !ok || !(strings.HasPrefix(call.Source, "./") || strings.HasPrefix(call.Source, "../")) {
			return []string{fmt.Sprintf("%s: the module calls no module %s of its own", addr, parts[1])}
		}
		next, diags := tfconfig.LoadModule(filepath.Join(mod.Path, call.Source))
		if diags.HasErrors() {
			return []string{fmt.Sprintf("%s: %v", addr, diags.Err())}
		}
		mod, dir, parts = next, filepath.Join(dir, call.Source), parts[2:]
	}
	resources := mod.ManagedResources
	if len(parts) == 3 && parts[0] == "data" {
		resources, parts = mod.DataResources, parts[1:]
	}
	if len(parts) != 2 {
		return []string{fmt.Sprintf("%s: not an address of one resource", addr)}
	}
	if _, ok := resources[parts[0]+"."+parts[1]]; !ok {
		return []string{fmt.Sprintf("%s: the module's %s declares no %s.%s", addr, dir, parts[0], parts[1])}
	}
	return nil
}
