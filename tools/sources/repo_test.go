package main

import (
	"regexp"
	"testing"

	"github.com/spin-stack/go-tools/versions/gate"
)

// The site is held to its versions.yaml as any spin-stack repository is: a commit written anywhere
// else is a pin no bump moves. What refgen writes records the commits the reference was generated
// from rather than pins them, and refgen fails when reference.json's are not the pins. A source
// is read where the Taskfile names it to this command.
func TestThisRepositoryPassesTheGate(t *testing.T) {
	g := gate.Gate{
		Root: "../..",
		Elsewhere: []string{
			"go.sum",
			// What refgen writes: reference.json names the commits, and the rest link spin's
			// tree at its commit (the status page's tests, the security pages, the diagrams) or
			// copy what spin and brand recorded (the CLI's goldens' digests, the screenshots').
			"src/data/reference.json",
			"src/data/figures.json",
			"src/data/shots.json",
			"src/data/output/",
			"src/content/docs/reference/",
		},
		Reads: append([]*regexp.Regexp{regexp.MustCompile(`go run \./tools/sources(?: -bump)?((?: [a-z][a-z0-9-]*)+)`)}, gate.DefaultReads...),
	}
	if err := g.Check(); err != nil {
		t.Error(err)
	}
}
