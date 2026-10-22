package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// configPages puts the installation's example config file on a page as it is: its comments are
// the reference, and the control plane's `config check` is what holds the file to the code.
func configPages(spin string) (map[string][]byte, error) {
	src := filepath.Join(spin, "configs", "spin-example.yaml")
	body, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString(frontmatter("spin:configs/spin-example.yaml", "Installation config file",
		"The installation's config file: settings, registration, workspace presets and network policies, with the comment on each.", 1))
	b.WriteString("The file an installation starts from - `installation_config` in Terraform. It seeds the " +
		"control plane's database once; after that the dashboard edits the installation, and a later change " +
		"to the file shows up there as a proposal to apply or dismiss. Every block is optional. " +
		"Check a file before applying it with `controlplane config check <file>`.\n\n")
	fmt.Fprintf(&b, "```yaml title=\"spin.yaml\"\n%s```\n", strings.TrimRight(string(body), "\n")+"\n")
	return map[string][]byte{filepath.Join("configuration", "file.md"): []byte(b.String())}, nil
}
