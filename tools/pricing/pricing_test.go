package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// module is a spin-terraform-aws checkout with the three defaults the record prices.
func module(t *testing.T, runnerTypes string) string {
	t.Helper()
	root := t.TempDir()
	write := func(path, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("modules/controlplane/variables.tf", "variable \"domain\" {\n  type = string\n}\n\nvariable \"instance_type\" {\n  description = \"The machine.\"\n  type        = string\n  default     = \"t8i.medium\"\n}\n\nvariable \"proxy_instance_type\" {\n  default     = \"t8i.micro\"\n}\n")
	write("modules/runners/variables.tf", "variable \"instance_types\" {\n  type    = list(string)\n  default = ["+runnerTypes+"]\n}\n\nvariable \"max_hosts\" {\n  type    = number\n  default = 3\n}\n")
	return root
}

// lists is a directory of each region's list as the pricing page loads it, with the machines'
// rows and others.
func lists(t *testing.T, published string) string {
	t.Helper()
	dir := t.TempDir()
	row := func(instanceType, price, vcpu, memory string) map[string]string {
		return map[string]string{"price": price, "vCPU": vcpu, "Memory": memory, "Instance Type": instanceType}
	}
	for code, name := range regionNames {
		clean := strings.NewReplacer("(", "", ")", "").Replace(name)
		l := map[string]any{
			"manifest": map[string]string{"hawkFilePublicationDate": published},
			"regions": map[string]any{name: map[string]any{
				"t8i medium " + clean + " Linux":   row("t8i.medium", "0.0499200000", "2", "4.0 GiB"),
				"t8i micro " + clean + " Linux":    row("t8i.micro", "0.0124800000", "2", "1.0 GiB"),
				"m8id 8xlarge " + clean + " Linux": row("m8id.8xlarge", "2.0883200000", "32", "128 GiB"),
				"g6 16xlarge " + clean + " Linux":  row("g6.16xlarge", "9", "64", "256 GiB"),
			}},
		}
		b, err := json.Marshal(l)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, code+".json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// The record prices the machines the module starts by default, in each region, dated by the
// day it was written and by AWS's publication; a machine the lists do not price fails rather
// than leaving a blank.
func TestTheRecordPricesTheModulesDefaults(t *testing.T) {
	out := filepath.Join(t.TempDir(), "pricing.json")
	var stdout bytes.Buffer
	if err := run([]string{"-terraform", module(t, `"m8id.8xlarge"`), "-lists", lists(t, "2026-10-23T22:13:55Z"), "-out", out}, &stdout); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "pricing: 2 regions, lists published 2026-10-23T22:13:55Z\n" {
		t.Errorf("said %q", got)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var r Record
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	if r.Generated != time.Now().UTC().Format("2006-01-02") || r.Published != "2026-10-23T22:13:55Z" {
		t.Errorf("dated %s, lists %s", r.Generated, r.Published)
	}
	want := Machines{ControlPlane: "t8i.medium", Proxy: "t8i.micro", Runners: []string{"m8id.8xlarge"}, MaxRunners: 3}
	if got, _ := json.Marshal(r.Machines); string(got) != string(must(json.Marshal(want))) {
		t.Errorf("machines %s", got)
	}
	for _, code := range []string{"us-east-1", "us-west-2"} {
		if p := r.Regions[code]["m8id.8xlarge"]; p.Hourly != 2.08832 || p.VCPU != 32 || p.Memory != "128 GiB" {
			t.Errorf("%s m8id.8xlarge = %+v", code, p)
		}
		if _, ok := r.Regions[code]["g6.16xlarge"]; ok {
			t.Errorf("%s prices a machine the module does not start", code)
		}
	}

	err = run([]string{"-terraform", module(t, `"m8id.8xlarge", "c8id.8xlarge"`), "-lists", lists(t, "2026-10-23T22:13:55Z"), "-out", out}, &stdout)
	if err == nil || !strings.Contains(err.Error(), "no on-demand price for c8id.8xlarge") {
		t.Errorf("a machine the list does not price: %v", err)
	}
}

// -check holds the record to the window and to the module: older than the window, or of other
// machines than the module starts now, is a record to rewrite.
func TestAStaleOrMovedRecordFails(t *testing.T) {
	out := filepath.Join(t.TempDir(), "pricing.json")
	tf := module(t, `"m8id.8xlarge"`)
	if err := run([]string{"-terraform", tf, "-lists", lists(t, "2026-10-23T22:13:55Z"), "-out", out}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-terraform", tf, "-out", out, "-check"}, &bytes.Buffer{}); err != nil {
		t.Errorf("a record written today, of the module's machines: %v", err)
	}
	moved := module(t, `"c8id.8xlarge"`)
	err := run([]string{"-terraform", moved, "-out", out, "-check"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "the module starts") {
		t.Errorf("the module's defaults moved: %v", err)
	}

	m, err := defaults(tf)
	if err != nil {
		t.Fatal(err)
	}
	err = checkRecord(out, m, time.Now().Add(window+24*time.Hour))
	if err == nil || !strings.Contains(err.Error(), "more than 30 days ago") {
		t.Errorf("a record past the window: %v", err)
	}
}

func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}
