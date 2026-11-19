package main

import (
	"path/filepath"
	"testing"
)

// A mark on a command's output is text the golden has: one the golden lost fails, wherever on
// the tag it is and whatever the caption holds, and an Output with no mark is not read.
func TestWhatAPageMarksInAnOutputIsInTheGolden(t *testing.T) {
	dir := writePage(t, map[string]string{"diff.txt": "A  /etc/greeting\nM  /etc/hosts\n"})
	docs := writePage(t, map[string]string{
		"use/agents.mdx": `<Output name="diff.txt" caption="spin snapshot diff <id>" mark="/etc/hosts" ins="A  /etc/greeting" />` + "\n" +
			`<Output name="diff.txt" caption="x" del="D  /etc/issue" />` + "\n" +
			`<Output name="gone.txt" caption="x" mark="y" />` + "\n" +
			`<Output name="gone.txt" caption="x" />` + "\n" +
			`<Output name="diff.txt" caption="x" mark="" />` + "\n",
		"reference/cli.md": `<Output name="diff.txt" caption="x" mark="nowhere" />` + "\n",
	})
	bad, err := checkOutputMarks(docs, dir)
	if err != nil {
		t.Fatal(err)
	}
	wantBad(t, bad,
		filepath.Join("use", "agents.mdx")+`:2: del="D  /etc/issue" is not in diff.txt`,
		`agents.mdx:3: no output "gone.txt"`,
		`agents.mdx:5: mark="" is not in diff.txt`)
}
