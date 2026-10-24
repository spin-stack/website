package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// The commands a component shows are its string literals that start with spin, not a comment's
// words or a template that builds a sentence.
func TestAComponentsCommandsAreItsSpinLiterals(t *testing.T) {
	src := writePage(t, map[string]string{
		"lib/story.ts": "// `spin snapshot diff` prints\n" +
			"commands: ['spin workspace create --name demo', \"spin exec demo -- sh -c 'echo hi; rm /x'\"],\n" +
			"const s = `spin main at ${commit}`;\n",
		"components/Hero.astro": "<p>{'spin login --nope'}</p>\n",
		"components/Other.vue":  "'spin never --read'\n",
	})
	got, err := componentCommands(src)
	if err != nil {
		t.Fatal(err)
	}
	story := got[filepath.Join(src, "lib", "story.ts")]
	if len(story) != 2 || story[0].text != "spin workspace create --name demo" || story[1].text != "spin exec demo -- sh -c 'echo hi; rm /x'" || story[1].n != 2 {
		t.Errorf("story.ts: %+v", story)
	}
	if hero := got[filepath.Join(src, "components", "Hero.astro")]; len(hero) != 1 || hero[0].text != "spin login --nope" {
		t.Errorf("Hero.astro: %+v", hero)
	}
	if len(got) != 2 {
		t.Errorf("read %d files, want story.ts and Hero.astro", len(got))
	}
}

// A card's and a story step's claim is a built row with an end-to-end test; a row that says
// what no test checks yet, or is only partly built, is no claim the landing page makes.
func TestTheLandingPageClaimsOnlyWhatATestChecksEndToEnd(t *testing.T) {
	rows := fixtureBoard(t)
	docs := writePage(t, map[string]string{
		"index.mdx": `<BentoCard title="A" feature="A machine per workspace">` + "\n" +
			`<BentoCard title="B" feature="Cores lent while the host has them">` + "\n" +
			`<BentoCard title="C" feature="Teams of teams">` + "\n" +
			`<BentoCard title="D" feature="Nothing of the board">` + "\n",
	})
	src := writePage(t, map[string]string{
		"lib/story.ts": "\t\tfeature: 'Create, start, stop and delete',\n\t\tfeature: 'VS Code over SSH\\'s',\n",
	})
	bad, err := checkCardFeatures(docs, src, rows)
	if err != nil {
		t.Fatal(err)
	}
	wantBad(t, bad,
		`index.mdx:2: "Cores lent while the host has them" names no end-to-end test (memory lent and taken back)`,
		`index.mdx:3: "Teams of teams" is partial on spin's feature board, not built`,
		`index.mdx:4: no row of spin's feature board is "Nothing of the board"`,
		`story.ts:2: no row of spin's feature board is "VS Code over SSH's"`)
	if unescape(`it\'s a \\ path`) != `it's a \ path` {
		t.Error(strings.Join([]string{"unescape:", unescape(`it\'s a \\ path`)}, " "))
	}
}
