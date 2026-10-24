package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spin-stack/go-tools/versions"

	"github.com/spin-stack/website/tools/internal/github"
)

// runs are the workflows whose last run the chain report gives: each is a link of the chain from a
// spin-machine release to an installation, and the one that stalls is the one nobody sees.
var runs = []struct{ repo, workflow, what string }{
	{"spin-stack/spin", "machine.yml", "a spin-machine release proposed to spin"},
	{"spin-stack/spin-terraform-aws", "propose.yml", "the module proposed to the deployments"},
	{"spin-stack/spin-terraform-aws", "fresh-install.yml", "an installation from nothing, weekly"},
}

var (
	tfvarsVersion = regexp.MustCompile(`(?m)^spin_version\s*=\s*"([^"]+)"`)
	moduleRef     = regexp.MustCompile(`spin-terraform-aws\?ref=([^"\s]+)`)
)

// chain is the release-chain report: spin-machine as spin main pins it against its newest
// release, when each link's workflow last ran and how, and what spinbox.dev runs. Each line says
// what it could not read rather than stopping the report: it is advice, and the rest of it holds.
func chain(ctx context.Context, gh *github.Client) string {
	var b strings.Builder
	b.WriteString("| | |\n|---|---|\n")
	row := func(k, v string) { fmt.Fprintf(&b, "| %s | %s |\n", k, v) }

	pinned, err := machinePin(ctx, gh)
	var latest struct {
		Tag string `json:"tag_name"`
	}
	if err == nil {
		err = gh.Get(ctx, "/repos/spin-stack/spin-machine/releases/latest", &latest)
	}
	switch {
	case err != nil:
		row("spin-machine", "could not read: "+err.Error())
	case pinned == latest.Tag:
		row("spin-machine", fmt.Sprintf("spin main pins %s, the newest release", pinned))
	default:
		row("spin-machine", fmt.Sprintf("spin main pins %s; the newest release is **%s**", pinned, latest.Tag))
	}

	for _, r := range runs {
		var list struct {
			Runs []struct {
				Created    string `json:"created_at"`
				Status     string `json:"status"`
				Conclusion string `json:"conclusion"`
				URL        string `json:"html_url"`
			} `json:"workflow_runs"`
		}
		key := fmt.Sprintf("`%s` %s", r.workflow, r.what)
		switch err := gh.Get(ctx, "/repos/"+r.repo+"/actions/workflows/"+r.workflow+"/runs?per_page=1", &list); {
		case err != nil:
			row(key, "could not read: "+err.Error())
		case len(list.Runs) == 0:
			row(key, "never ran")
		default:
			last := list.Runs[0]
			how := last.Conclusion
			if how == "" {
				how = last.Status
			}
			row(key, fmt.Sprintf("last ran %s, [%s](%s)", day(last.Created), how, last.URL))
		}
	}

	version, ref, err := deployment(ctx, gh)
	if err != nil {
		row("spinbox.dev", "could not read: "+err.Error())
	} else {
		row("spinbox.dev", fmt.Sprintf("runs %s, with the module at `%s`", version, ref))
	}
	return b.String()
}

// machinePin is the spin-machine release spin's main pins in its versions.yaml.
func machinePin(ctx context.Context, gh *github.Client) (string, error) {
	raw, err := gh.File(ctx, "spin-stack/spin", versions.File, "main")
	if err != nil {
		return "", err
	}
	v, err := versions.Parse(raw)
	if err != nil {
		return "", err
	}
	e, err := v.Get("spin-machine")
	return e.Version, err
}

// deployment is the release spinbox.dev runs and the module ref it takes the installation from.
func deployment(ctx context.Context, gh *github.Client) (string, string, error) {
	tfvars, err := gh.File(ctx, "spin-stack/spinbox.dev-deploy", "terraform.tfvars", "main")
	if err != nil {
		return "", "", err
	}
	main, err := gh.File(ctx, "spin-stack/spinbox.dev-deploy", "main.tf", "main")
	if err != nil {
		return "", "", err
	}
	v, r := tfvarsVersion.FindSubmatch(tfvars), moduleRef.FindSubmatch(main)
	if v == nil || r == nil {
		return "", "", fmt.Errorf("spinbox.dev-deploy names no spin_version in terraform.tfvars or no module ref in main.tf")
	}
	return string(v[1]), string(r[1]), nil
}

// day is the date of a GitHub timestamp.
func day(ts string) string {
	d, _, _ := strings.Cut(ts, "T")
	return d
}
