package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// fixtureTF is testdata/tf: a module tree of spin-terraform-aws's shape, small enough to read.
var fixtureTF = filepath.Join("testdata", "tf")

// writePage writes a page under a fresh docs directory and answers the directory.
func writePage(t *testing.T, pages map[string]string) string {
	t.Helper()
	docs := t.TempDir()
	for rel, body := range pages {
		p := filepath.Join(docs, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return docs
}

// wantBad fails unless bad holds a line with each of want, and nothing else.
func wantBad(t *testing.T, bad []string, want ...string) {
	t.Helper()
	if len(bad) != len(want) {
		t.Errorf("got %d lines, want %d:\n%s", len(bad), len(want), strings.Join(bad, "\n"))
	}
	for _, w := range want {
		if !slices.ContainsFunc(bad, func(b string) bool { return strings.Contains(b, w) }) {
			t.Errorf("no line says %q:\n%s", w, strings.Join(bad, "\n"))
		}
	}
}

// A module serves the names its records give with the domain as the pages write it: a set of
// them, a local naming another local, and not a validation record, which the certificate names.
func TestTheNamesAModuleServesAreItsRecords(t *testing.T) {
	got, err := installationNames(fixtureTF)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"*.ws.<domain>", "app.<domain>", "tunnel.app.<domain>"}
	if !slices.Equal(got, want) {
		t.Errorf("installationNames = %q, want %q", got, want)
	}
	if got, _ := servedNames(filepath.Join(fixtureTF, "modules", "terminal")); !slices.Equal(got, []string{"term.<domain>"}) {
		t.Errorf("the terminal serves %q", got)
	}
}

const prerequisites = "---\ntitle: Before you start\n---\n\n## A domain\n\n| Name | What |\n|---|---|\n" +
	"| `app.<domain>` | The dashboard |\n| `tunnel.app.<domain>` | The relay |\n| `*.ws.<domain>` | Ports |\n\n" +
	"### Optional modules\n\n| Name | Module | What |\n|---|---|---|\n" +
	"| `term.<domain>` | [`modules/terminal`](/reference/terraform/terminal/) | The terminal |\n" +
	"| `id.<domain>` | [`modules/identity-issuer`](/reference/terraform/identity-issuer/) | The issuer |\n\n## Next\n"

// The page's names are the modules', both ways, and each optional one beside its module.
func TestThePrerequisitesNameWhatTheModulesServe(t *testing.T) {
	bad, err := checkDomains(writePage(t, map[string]string{"aws/prerequisites.md": prerequisites}), fixtureTF)
	if err != nil {
		t.Fatal(err)
	}
	wantBad(t, bad)

	wrong := strings.NewReplacer("| `tunnel.app.<domain>` | The relay |\n", "| `old.<domain>` | Gone |\n",
		"[`modules/identity-issuer`](/reference/terraform/identity-issuer/)", "[`modules/terminal`](/reference/terraform/terminal/)").Replace(prerequisites)
	bad, err = checkDomains(writePage(t, map[string]string{"aws/prerequisites.md": wrong}), fixtureTF)
	if err != nil {
		t.Fatal(err)
	}
	wantBad(t, bad,
		"the installation serves tunnel.app.<domain>, and the page does not name it",
		"names old.<domain>, which the installation does not serve",
		"modules/identity-issuer serves id.<domain>, and the page does not name it",
		"names id.<domain>, which modules/terminal does not serve")
}

// What an aws/ page tells people to type and to set is what the module has.
func TestTofuOnThePagesIsWhatTheModuleHas(t *testing.T) {
	page := "Set `decommission = true`, `runner_spot` and `force_destroy`; not `deletion_protection`.\n\n" +
		"```bash\n" +
		"tofu output -raw update_status\n" +
		"tofu output next_steps\n" +
		"tofu apply -var 'decommission=true' -var=spin_version=v1\n" +
		"tofu apply -var 'database={deletion_protection=false}'\n" +
		"tofu state rm module.spin.module.controlplane.aws_kms_key.encryption module.spin.module.controlplane.tls_private_key.ca\n" +
		"tofu state rm module.spin.module.controlplane.aws_ssm_parameter.encryption_key\n" +
		"tofu state rm $(tofu state list | grep -E '\\.aws_kms_key\\.')\n" +
		"tofu import module.spin.module.nothing.aws_kms_key.x id\n" +
		"tofu -chdir=../../bootstrap output -raw init\n" +
		"```\n\n" +
		"```bash\ncd bootstrap\ntofu output init\n```\n"
	bad, err := checkTofu(writePage(t, map[string]string{"aws/remove.md": page}), fixtureTF)
	if err != nil {
		t.Fatal(err)
	}
	wantBad(t, bad,
		"remove.md:1: `deletion_protection` is no variable",
		"remove.md:5: tofu output next_steps: the module has no output next_steps",
		"remove.md:7: -var database={deletion_protection=false}: the module has no variable database",
		"remove.md:9: module.spin.module.controlplane.aws_ssm_parameter.encryption_key: the module's modules/controlplane declares no aws_ssm_parameter.encryption_key",
		"remove.md:11: module.spin.module.nothing.aws_kms_key.x: the module calls no module nothing")
}

// The update and remove pages are the README's sections, the markers' own words left out, with a
// link that is the README's own pointed at the repository.
func TestTheProceduresAreTheReadmes(t *testing.T) {
	pages, err := includedPages(fixtureTF, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	update := string(pages["aws/update.md"])
	for _, want := range []string{
		"title: \"Update\"",
		"source: \"spin-terraform-aws:README.md\"",
		"Move the ref, then see [the examples](https://github.com/spin-stack/spin-terraform-aws/blob/abc123/examples/complete)",
		"[removing](https://github.com/spin-stack/spin-terraform-aws/blob/abc123/README.md#removing-an-installation)",
		"[its releases](https://github.com/spin-stack/spin-terraform-aws/releases)",
		"[a file](https://github.com/spin-stack/spin-terraform-aws/blob/abc123/docs/x.md)",
	} {
		if !strings.Contains(update, want) {
			t.Errorf("aws/update.md has no %q:\n%s", want, update)
		}
	}
	if strings.Contains(update, "markers.") || strings.Contains(update, "END website") {
		t.Errorf("aws/update.md carries the markers:\n%s", update)
	}
	if !strings.HasSuffix(string(pages["aws/remove.md"]), "-->\n\nSet `decommission = true`.\n") {
		t.Errorf("aws/remove.md:\n%s", pages["aws/remove.md"])
	}
	if _, err := readmeSection([]byte("<!-- BEGIN website:aws/update x -->\nno end\n"), "aws/update"); err == nil {
		t.Error("a section with no END marker was taken")
	}
}

// gitRepo makes a repository in a temporary directory with one commit of files, and tags it.
func gitRepo(t *testing.T, files map[string]string, tags ...string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("add", ".")
	run("commit", "-q", "-m", "x")
	for _, tag := range tags {
		run("tag", tag)
	}
	return dir
}

// The module is pinned at the newest release the commit holds, by date and then number.
func TestTheModuleReleaseIsTheNewestTagOfTheCommit(t *testing.T) {
	got, err := moduleRelease(gitRepo(t, map[string]string{"a": "a"}, "v20261024.01", "v20261024.10", "v20261023.11"))
	if err != nil || got != "v20261024.10" {
		t.Errorf("moduleRelease = %q, %v", got, err)
	}
	if _, err := moduleRelease(gitRepo(t, map[string]string{"a": "a"})); err == nil {
		t.Error("a checkout with no tags gave a release")
	}
}

// A page pins the module at its newest release and names the spin release installations run; a
// ref's pattern, explained, is neither.
func TestPagesPinTheModuleAndTheRelease(t *testing.T) {
	page := "source = \"github.com/spin-stack/spin-terraform-aws?ref=<tag>\"\n" +
		"spin_version = \"v20261019.02\"\n" +
		"pin one (`?ref=v<YYYYMMDD>.<NN>`)\n" +
		"`//modules/terminal?ref=v20261001.01` and v20261001.01\n"
	// The dashboard's page names its screenshots' release, the documented one.
	shotsLine := "These screenshots are of spin v20261024.01.\n"
	docs := writePage(t, map[string]string{"aws/overview.mdx": page, shotsPageDoc: shotsLine})
	bad, err := pinReleases(docs, "v20261019.02", "v20261024.01", true)
	if err != nil {
		t.Fatal(err)
	}
	wantBad(t, bad,
		"overview.mdx:1: ?ref=<tag> pins the module at no release; its newest is v20261024.01",
		"overview.mdx:4: pins the module at v20261001.01; its newest release is v20261024.01",
		"overview.mdx:4: names release v20261001.01; installations run v20261019.02")
	if _, err := pinReleases(docs, "v20261019.02", "v20261024.01", false); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(docs, "aws", "overview.mdx"))
	want := "source = \"github.com/spin-stack/spin-terraform-aws?ref=v20261024.01\"\n" +
		"spin_version = \"v20261019.02\"\n" +
		"pin one (`?ref=v<YYYYMMDD>.<NN>`)\n" +
		"`//modules/terminal?ref=v20261024.01` and v20261019.02\n"
	if string(raw) != want {
		t.Errorf("rewritten:\n%s\nwant:\n%s", raw, want)
	}
	if raw, _ := os.ReadFile(filepath.Join(docs, shotsPageDoc)); string(raw) != shotsLine {
		t.Errorf("the dashboard's page was rewritten to %q", raw)
	}
}
