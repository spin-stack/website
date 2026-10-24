// Command sync writes what the daily sync of the site with its sources says, and where:
//
//	go run ./tools/sync pr -bot <app>[bot]               the sync pull request, as the sync App (SYNC_TOKEN)
//	go run ./tools/sync issue -mode release -bot <app>[bot]  spin's issue: the commits the bump took in, and the release chain (SOURCES_TOKEN)
//	go run ./tools/sync issue -mode main -bot <app>[bot]     spin's issue: spin main past the pin, and the release chain
//
// The website is public and spin private: the pull request says which pins moved and which
// files changed, and what spin's commits say goes in one issue in spin.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spin-stack/go-tools/versions"

	"github.com/spin-stack/website/tools/internal/github"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, os.Args[1:])
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, argv []string) error {
	if len(argv) == 0 {
		return errors.New("pr or issue")
	}
	fs := flag.NewFlagSet(argv[0], flag.ContinueOnError)
	bot := fs.String("bot", "", "the App's bot login, <slug>[bot], that writes")
	base := fs.String("base", "origin/main", "the ref whose versions.yaml the pins moved from")
	mode := fs.String("mode", "", "issue: release, what the bump took in, or main, what spin main has past the pin")
	reference := fs.String("reference", "success", "pr: how task reference ended, as the step's outcome says")
	dry := fs.Bool("print", false, "issue: print the sections and write nothing, to read them before the issue does")
	docs := fs.String("docs", "src/content/docs", "the pages, whose sources the digest groups commits by")
	figures := fs.String("figures", "src/data/figures.json", "the diagrams' sources")
	if err := fs.Parse(argv[1:]); err != nil {
		return err
	}
	if *bot == "" {
		return errors.New("-bot names who writes")
	}
	now, err := versions.Load(versions.File)
	if err != nil {
		return err
	}
	raw, err := gitOut(ctx, nil, "show", *base+":"+versions.File)
	if err != nil {
		return err
	}
	was, err := versions.Parse([]byte(raw))
	if err != nil {
		return err
	}
	switch argv[0] {
	case "pr":
		ms := moves(was, now)
		changed, err := changedFiles(ctx)
		if err != nil {
			return err
		}
		if len(ms) == 0 && len(changed) == 0 {
			fmt.Println("nothing moved: every pin is at its newest, and the reference is what they say")
			return nil
		}
		token := os.Getenv("SYNC_TOKEN")
		link, err := publish(ctx, github.New(token), "spin-stack/website", "https://github.com/spin-stack/website", *bot, token, ms, prBody(ms, changed, *reference, today()))
		if err != nil {
			return err
		}
		fmt.Println(link)
		return nil
	case "issue":
		gh := github.New(os.Getenv("SOURCES_TOKEN"))
		fds, err := feeds(*docs, *figures)
		if err != nil {
			return err
		}
		text, err := issueText(ctx, gh, *mode, was, now, fds)
		if err != nil {
			return err
		}
		texts := map[string]string{*mode: text, "chain": chain(ctx, gh)}
		if *dry {
			fmt.Print(withSection(withSection("", *mode, text), "chain", texts["chain"]))
			return nil
		}
		link, err := writeIssue(ctx, gh, "spin-stack/spin", *bot, texts)
		if err != nil {
			return err
		}
		fmt.Println(link)
		return nil
	}
	return fmt.Errorf("%s is neither pr nor issue", argv[0])
}

// issueText is the issue's section for mode: the commits the bump took into the site, or those
// spin main has past the pin, each digest dated so a reader sees how old it is.
func issueText(ctx context.Context, gh *github.Client, mode string, was, now *versions.Versions, fds []feed) (string, error) {
	from, err := was.Get("spin")
	if err != nil {
		return "", err
	}
	to, err := now.Get("spin")
	if err != nil {
		return "", err
	}
	var head, base, intro string
	switch mode {
	case "release":
		if from.Pin == to.Pin {
			return fmt.Sprintf("%s: the pin stays at %s: no release newer than it is one the site can move to.\n", today(), at(to.Version, to.Pin)), nil
		}
		base, head = from.Pin, to.Pin
		intro = fmt.Sprintf("%s: the sync pull request moves spin from %s to %s.", today(), at(from.Version, from.Pin), at(to.Version, to.Pin))
	case "main":
		base, head = to.Pin, "main"
		intro = fmt.Sprintf("%s: the site documents spin %s; this is what main has past it, for the next release to bring.", today(), at(to.Version, to.Pin))
	default:
		return "", fmt.Errorf("-mode %q is neither release nor main", mode)
	}
	cs, total, err := commits(ctx, gh, "spin-stack/spin", base, head)
	if err != nil {
		return "", err
	}
	return intro + "\n\n" + digest(cs, total, fds), nil
}

func today() string { return time.Now().UTC().Format(time.DateOnly) }
