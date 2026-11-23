package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Commands the CLI carries that are not the product's: cobra's own.
var cliSkip = map[string]bool{"help": true, "completion": true}

// buildCLI builds the CLI from the checkout into dir.
func buildCLI(spin, dir string) (string, error) {
	bin := filepath.Join(dir, "spin")
	build := exec.Command("go", "build", "-o", bin, "./cmd/cli")
	build.Dir = spin
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return "", fmt.Errorf("building the CLI in %s: %w", spin, err)
	}
	return bin, nil
}

// cliPages writes a page per top-level command, each holding the --help of it and of every
// command under it. The help is the CLI's own words, so the page cannot say a flag the binary
// does not have. home is the empty HOME the CLI runs with.
func cliPages(bin, home string) (map[string][]byte, error) {
	top, err := help(bin, home, nil)
	if err != nil {
		return nil, err
	}
	pages := map[string][]byte{}
	var index strings.Builder
	index.WriteString(frontmatter("spin:cmd/cli", "CLI", "The spin CLI: every command, as its --help says it", "Every command of the spin CLI, as its own --help says it: workspaces, snapshots, ports, keys, teams and administration.", 0))
	index.WriteString("The `spin` command line, one page per command. Each page is the command's own `--help`, " +
		"and the help of every command under it.\n\n```\n" + top.text + "```\n\n")
	for i, c := range top.subs {
		var b strings.Builder
		b.WriteString(frontmatter("spin:cmd/cli", "spin "+c.name, cliMetaTitle(c), cliDescription(c), i+1))
		if err := walk(bin, home, []string{c.name}, &b, 1); err != nil {
			return nil, err
		}
		pages[filepath.Join("cli", c.name+".md")] = []byte(b.String())
		fmt.Fprintf(&index, "- [`spin %s`](/reference/cli/%s/): %s\n", c.name, c.name, c.short)
	}
	pages[filepath.Join("cli", "index.md")] = []byte(index.String())
	return pages, nil
}

// cliMetaTitle is a command's title in a search result: what its help says it does, or, where
// that is a sentence too long for one, what the page holds.
func cliMetaTitle(c command) string {
	t := "spin " + c.name + ": " + c.short
	if len(t) > 50 {
		t = "spin " + c.name + ": usage and flags"
	}
	return t + " | Spinbox CLI"
}

// cliDescription is a command's line in llms.txt and in a search result: its help's own short
// line, and what the page holds when there is room for it.
func cliDescription(c command) string {
	d := "spin " + c.name + ": " + c.short + "."
	if more := " Its usage and flags, from the CLI's own help."; len(d+more) <= maxDescription {
		d += more
	}
	return d
}

// walk writes a command's help, and under it each command below it with a heading a level
// deeper: the page's title is the top command, so the commands under it are its sections (h2).
func walk(bin, home string, path []string, b *strings.Builder, depth int) error {
	h, err := help(bin, home, path)
	if err != nil {
		return err
	}
	if depth > 1 {
		fmt.Fprintf(b, "%s spin %s\n\n", strings.Repeat("#", depth), strings.Join(path, " "))
	}
	fmt.Fprintf(b, "```\n%s```\n\n", h.text)
	for _, c := range h.subs {
		if err := walk(bin, home, append(append([]string{}, path...), c.name), b, depth+1); err != nil {
			return err
		}
	}
	return nil
}

type command struct{ name, short string }

type helpText struct {
	text string
	subs []command
}

// help runs `spin <path> --help` with a home of its own, so no profile of whoever runs this
// reaches the page, and reads the commands cobra lists under it.
func help(bin, home string, path []string) (helpText, error) {
	cmd := exec.Command(bin, append(append([]string{}, path...), "--help")...)
	cmd.Env = []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "NO_COLOR=1"}
	out, err := cmd.Output()
	if err != nil {
		return helpText{}, fmt.Errorf("spin %s --help: %w", strings.Join(path, " "), err)
	}
	h := helpText{text: string(out)}
	in := false
	sc := bufio.NewScanner(strings.NewReader(h.text))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasSuffix(line, "Commands:") {
			in = true
			continue
		}
		if !in {
			continue
		}
		if strings.TrimSpace(line) == "" {
			in = false
			continue
		}
		f := strings.Fields(line)
		if cliSkip[f[0]] {
			continue
		}
		h.subs = append(h.subs, command{name: f[0], short: strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), f[0]))})
	}
	return h, sc.Err()
}
