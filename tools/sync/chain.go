package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
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
// release, the newest spin release images.json has an image of against spin's newest, when each
// link's workflow last ran and how, and what spinbox.dev runs. Each line says
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

	switch imaged, newest, err := images(ctx, gh); {
	case err != nil:
		row("images.json", "could not read: "+err.Error())
	case imaged == newest:
		row("images.json", fmt.Sprintf("has an image of spin's newest release, %s", newest))
	default:
		// The installation pages name the newest release with an image: until this one has its
		// own, the site documents a release nobody can install.
		row("images.json", fmt.Sprintf("the newest release with an image is %s; spin's newest is **%s**, which spin-stack/ami's `release.yml` builds one of", imaged, newest))
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

var release = regexp.MustCompile(`^v(\d{8})\.(\d+)$`)

// images is the newest spin release spin-terraform-aws's images.json has an image of, and spin's
// newest release.
func images(ctx context.Context, gh *github.Client) (string, string, error) {
	raw, err := gh.File(ctx, "spin-stack/spin-terraform-aws", "images.json", "main")
	if err != nil {
		return "", "", err
	}
	var byRelease map[string]json.RawMessage
	if err := json.Unmarshal(raw, &byRelease); err != nil {
		return "", "", fmt.Errorf("images.json: %w", err)
	}
	var have []string
	for r := range byRelease {
		if release.MatchString(r) {
			have = append(have, r)
		}
	}
	if len(have) == 0 {
		return "", "", fmt.Errorf("images.json names no release")
	}
	var latest struct {
		Tag string `json:"tag_name"`
	}
	if err := gh.Get(ctx, "/repos/spin-stack/spin/releases/latest", &latest); err != nil {
		return "", "", err
	}
	return slices.MaxFunc(have, compareReleases), latest.Tag, nil
}

// compareReleases orders two release names by date and then by number: v20261019.10 is after
// v20261019.9, which the strings' order would not say.
func compareReleases(a, b string) int {
	ma, mb := release.FindStringSubmatch(a), release.FindStringSubmatch(b)
	for i := 1; i <= 2; i++ {
		x, _ := strconv.Atoi(ma[i])
		y, _ := strconv.Atoi(mb[i])
		if x != y {
			return x - y
		}
	}
	return 0
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
