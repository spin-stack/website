package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// checkCommands holds the hand-written pages to the tree they describe: every `spin` command in
// a page's shell block must be one the CLI has, with flags its help lists, and every `task` one
// spin's Taskfile has. The reference pages are the CLI's own help; the guides only quote it, and
// a renamed flag would otherwise leave a page telling people to type something that fails. The
// commands the components under src show - the landing page's story - are held the same way.
func checkCommands(docs, src, spin, bin, home string) ([]string, error) {
	tasks, err := taskNames(spin)
	if err != nil {
		return nil, err
	}
	cli := cliHelp{bin: bin, home: home, seen: map[string]helpText{}}
	var bad []string
	err = handPages(docs, func(path string, _ []byte) error {
		lines, err := shellLines(path)
		if err != nil {
			return err
		}
		b, err := checkShell(path, lines, cli, tasks)
		bad = append(bad, b...)
		return err
	})
	if err != nil {
		return nil, err
	}
	components, err := componentCommands(src)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(components))
	for f := range components {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		b, err := checkShell(f, components[f], cli, tasks)
		if err != nil {
			return nil, err
		}
		bad = append(bad, b...)
	}
	return bad, nil
}

// checkShell says which of one file's commands spin's CLI or Taskfile does not have.
func checkShell(path string, lines []shellLine, cli cliHelp, tasks map[string]bool) ([]string, error) {
	var bad []string
	// A block that changes into another checkout - spin-stack/ami's, say - runs that one's
	// tasks from there on.
	block, elsewhere := -1, false
	for _, l := range lines {
		if l.block != block {
			block, elsewhere = l.block, false
		}
		for _, words := range shellCommands(l.text) {
			for len(words) > 0 && envAssign.MatchString(words[0]) {
				words = words[1:]
			}
			if len(words) == 0 {
				continue
			}
			var why string
			var err error
			switch words[0] {
			case "cd":
				elsewhere = len(words) > 1 && filepath.Base(words[1]) != "spin"
			case "spin":
				why, err = cli.check(words[1:])
			case "task":
				if !elsewhere {
					why = checkTask(words[1:], tasks)
				}
			}
			if err != nil {
				return nil, err
			}
			if why != "" {
				bad = append(bad, fmt.Sprintf("%s:%d: %s", path, l.n, why))
			}
		}
	}
	return bad, nil
}

// page says whether a file is one of the site's pages: Markdown, or MDX for the pages that use
// a component.
func page(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".md" || ext == ".mdx"
}

var (
	envAssign = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	fence     = regexp.MustCompile("^```(bash|sh|shell|console)\\s*$")
	// A cobra flag line: "  -o, --output string   text" or "      --wait   text". The type,
	// when there is one, is a single word before the description's run of spaces.
	flagLine = regexp.MustCompile(`^\s+(?:-(\w), )?--([\w-]+)(?: (\w+))?(?:\s{2,}|$)`)
)

type shellLine struct {
	n, block int
	text     string
}

// shellLines reads a page's shell blocks, a line ending in a backslash joined to the next.
func shellLines(path string) ([]shellLine, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []shellLine
	in, block := false, 0
	var cur *shellLine
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if strings.HasPrefix(line, "```") {
			in = !in && fence.MatchString(line)
			block++
			cur = nil
			continue
		}
		if !in {
			continue
		}
		if cur != nil {
			cur.text += " " + line
		} else {
			out = append(out, shellLine{n: n, block: block, text: line})
			cur = &out[len(out)-1]
		}
		if strings.HasSuffix(cur.text, `\`) {
			cur.text = strings.TrimSuffix(cur.text, `\`)
		} else {
			cur = nil
		}
	}
	return out, sc.Err()
}

// shellCommands splits a line into its commands, each its words with the quotes removed: a pipe
// or a list operator starts the next one, a redirection's target is nobody's word, and a
// comment ends the line.
func shellCommands(s string) [][]string {
	var cmds [][]string
	var words []string
	var w strings.Builder
	inWord, redirect := false, false
	var quote rune
	endWord := func() {
		if inWord && !redirect {
			words = append(words, w.String())
		}
		if inWord {
			redirect = false
		}
		w.Reset()
		inWord = false
	}
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				w.WriteRune(r)
			}
			continue
		case r == '\'' || r == '"':
			quote = r
			inWord = true
			continue
		case r == ' ' || r == '\t':
			endWord()
			continue
		case !inWord && r == '#':
			endWord()
			return append(cmds, words)
		case r == '<' || r == '>':
			endWord()
			redirect = true
			continue
		case strings.ContainsRune("|;&", r):
			endWord()
			if len(words) > 0 {
				cmds = append(cmds, words)
			}
			words = nil
			continue
		}
		w.WriteRune(r)
		inWord = true
	}
	endWord()
	return append(cmds, words)
}

type flagSpec struct{ value bool }

type cliHelp struct {
	bin, home string
	seen      map[string]helpText
}

func (c cliHelp) get(path []string) (helpText, error) {
	key := strings.Join(path, " ")
	if h, ok := c.seen[key]; ok {
		return h, nil
	}
	h, err := help(c.bin, c.home, path)
	if err != nil {
		return helpText{}, err
	}
	c.seen[key] = h
	return h, nil
}

// check walks the arguments the way cobra does: commands while the one reached has commands
// under it, then flags anywhere until "--". It says what the CLI would refuse, or "".
func (c cliHelp) check(args []string) (string, error) {
	var path []string
	h, err := c.get(nil)
	if err != nil {
		return "", err
	}
	flags := flagsOf(h.text)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			name, inline := strings.CutPrefix(a, "--")
			if !inline {
				name = strings.TrimPrefix(a, "-")
			}
			name, _, hasValue := strings.Cut(name, "=")
			f, ok := flags[name]
			if !ok {
				return fmt.Sprintf("spin %s has no flag %s", strings.Join(path, " "), a), nil
			}
			if f.value && !hasValue {
				i++
			}
			continue
		}
		if len(h.subs) == 0 {
			continue
		}
		known := false
		for _, s := range h.subs {
			known = known || s.name == a
		}
		if !known {
			return fmt.Sprintf("spin %s has no command %q", strings.Join(path, " "), a), nil
		}
		path = append(path, a)
		if h, err = c.get(path); err != nil {
			return "", err
		}
		flags = flagsOf(h.text)
	}
	return "", nil
}

// flagsOf reads the flags a command's help lists, its own and the global ones, by long name
// and by short.
func flagsOf(help string) map[string]flagSpec {
	flags := map[string]flagSpec{}
	in := false
	sc := bufio.NewScanner(strings.NewReader(help))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasSuffix(line, "Flags:") {
			in = true
			continue
		}
		if !in {
			continue
		}
		m := flagLine.FindStringSubmatch(line)
		if m == nil {
			in = strings.TrimSpace(line) != ""
			continue
		}
		f := flagSpec{value: m[3] != ""}
		flags[m[2]] = f
		if m[1] != "" {
			flags[m[1]] = f
		}
	}
	return flags
}

// taskNames lists every task spin's Taskfile has, as the Task spin's go.mod pins reads it: the
// one its Taskfile is written for, and one CI has without installing a Task of its own.
// --no-status: listing with status writes the checksums of the tasks it looks at.
func taskNames(spin string) (map[string]bool, error) {
	cmd := exec.Command("go", "tool", "task", "--list-all", "--json", "--no-status")
	cmd.Dir = spin
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go tool task --list-all in %s: %w", spin, err)
	}
	var list struct {
		Tasks []struct {
			Name    string   `json:"name"`
			Aliases []string `json:"aliases"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, fmt.Errorf("task --list-all in %s: %w", spin, err)
	}
	names := map[string]bool{}
	for _, t := range list.Tasks {
		names[t.Name] = true
		for _, a := range t.Aliases {
			names[a] = true
		}
	}
	return names, nil
}

// checkTask says which task a `task` line names that spin's Taskfile does not have.
func checkTask(args []string, tasks map[string]bool) string {
	for _, a := range args {
		if a == "--" {
			break
		}
		if strings.HasPrefix(a, "-") || strings.Contains(a, "=") {
			continue
		}
		if !tasks[a] {
			return fmt.Sprintf("spin's Taskfile has no task %q", a)
		}
	}
	return ""
}
