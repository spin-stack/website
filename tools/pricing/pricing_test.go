package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// module is a spin-terraform-aws checkout with the defaults the record prices: runnerTypes as
// instance_types' list, and dataVolume as data_volume_gb's.
func module(t *testing.T, runnerTypes, dataVolume string) string {
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
	write("modules/runners/variables.tf", "variable \"instance_types\" {\n  type    = list(string)\n  default = ["+runnerTypes+"]\n}\n\n"+
		"variable \"vcpus\" {\n  type    = object({ min = number, max = number })\n  default = { min = 32, max = 48 }\n}\n\n"+
		"variable \"data_volume_gb\" {\n  type    = number\n  default = "+dataVolume+"\n}\n\n"+
		"variable \"max_hosts\" {\n  type    = number\n  default = 3\n}\n")
	return root
}

// lists is a directory of each region's list as the pricing page loads it, with the machines'
// rows and others; extra are rows one region alone has.
func lists(t *testing.T, published string, extra map[string][4]string) string {
	t.Helper()
	dir := t.TempDir()
	for code, name := range regionNames {
		clean := strings.NewReplacer("(", "", ")", "").Replace(name)
		prices := map[string]any{}
		add := func(instanceType, price, vcpu, memory, storage string) {
			family, size, _ := strings.Cut(instanceType, ".")
			prices[family+" "+size+" "+clean+" Linux"] = map[string]string{
				"price": price, "vCPU": vcpu, "Memory": memory, "Instance Type": instanceType, "Storage": storage,
			}
		}
		add("t8i.medium", "0.0499200000", "2", "4.0 GiB", "EBS only")
		add("t8i.micro", "0.0124800000", "2", "1.0 GiB", "EBS only")
		add("m8id.8xlarge", "2.0883200000", "32", "128 GiB", "1 x 1900 NVMe SSD")
		add("c8id.12xlarge", "2.6611200000", "48", "96 GiB", "1 x 2850 NVMe SSD")
		add("c8id.8xlarge", "1.7740800000", "32", "64 GiB", "1 x 1900 NVMe SSD")
		add("c8id.16xlarge", "3.5481600000", "64", "128 GiB", "1 x 3800 NVMe SSD")
		add("c8i.8xlarge", "1.4995200000", "32", "64 GiB", "EBS only")
		add("g6.16xlarge", "9", "64", "256 GiB", "1 x 1900 NVMe SSD")
		if code == "us-west-2" {
			for t, r := range extra {
				add(t, r[0], r[1], r[2], r[3])
			}
		}
		b, err := json.Marshal(map[string]any{
			"manifest": map[string]string{"hawkFilePublicationDate": published},
			"regions":  map[string]any{name: prices},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, code+".json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// record writes the record of tf's defaults against ls and reads it back.
func record(t *testing.T, tf, ls string) (Record, error) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "pricing.json")
	if err := run([]string{"-terraform", tf, "-lists", ls, "-out", out}, &bytes.Buffer{}); err != nil {
		return Record{}, err
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var r Record
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return r, nil
}

// The record prices the machines the module starts by default, in each region, dated by the
// day it was written and by AWS's publication; a machine the lists do not price fails rather
// than leaving a blank.
func TestTheRecordPricesTheModulesDefaults(t *testing.T) {
	out := filepath.Join(t.TempDir(), "pricing.json")
	var stdout bytes.Buffer
	if err := run([]string{"-terraform", module(t, `"m8id.8xlarge"`, "0"), "-lists", lists(t, "2026-10-23T22:13:55Z", nil), "-out", out}, &stdout); err != nil {
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
	want := Machines{ControlPlane: "t8i.medium", Proxy: "t8i.micro", RunnerTypes: []string{"m8id.8xlarge"}, RunnerVCPUs: [2]int{32, 48}, RunnerLocalDisk: true, MaxRunners: 3}
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

	_, err = record(t, module(t, `"m9id.8xlarge"`, "0"), lists(t, "2026-10-23T22:13:55Z", nil))
	if err == nil || !strings.Contains(err.Error(), "no type every region prices matches") {
		t.Errorf("a runner's type the list does not price: %v", err)
	}
}

// A runner's type is every one the module's names and patterns match within its vCPUs - with a
// local NVMe while the data is on the instance store, or any disk with a data volume - that every
// region prices, the cheapest first.
func TestTheRunnersAreTheTypesTheDefaultsMatch(t *testing.T) {
	onlyWest := map[string][4]string{"c8id.10xlarge": {"2.2", "40", "80 GiB", "1 x 2400 NVMe SSD"}}
	for _, tc := range []struct {
		name, types, dataVolume string
		want                    []string
	}{
		{"on the instance store", `"c8i.*", "c8id.*", "m8id.*"`, "0", []string{"c8id.8xlarge", "m8id.8xlarge", "c8id.12xlarge"}},
		{"on a data volume", `"c8i.*", "c8id.*"`, "100", []string{"c8i.8xlarge", "c8id.8xlarge", "c8id.12xlarge"}},
		{"a name beside a pattern", `"m8id.8xlarge", "c8id.*"`, "0", []string{"c8id.8xlarge", "m8id.8xlarge", "c8id.12xlarge"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := record(t, module(t, tc.types, tc.dataVolume), lists(t, "2026-10-23T22:13:55Z", onlyWest))
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(r.Runners, tc.want) {
				t.Errorf("runners %v, want %v", r.Runners, tc.want)
			}
			for _, code := range []string{"us-east-1", "us-west-2"} {
				for _, rt := range tc.want {
					if _, ok := r.Regions[code][rt]; !ok {
						t.Errorf("%s does not price %s", code, rt)
					}
				}
			}
		})
	}
}

// -check holds the record to the window and to the module: older than the window, or of other
// machines than the module starts now, is a record to rewrite.
func TestAStaleOrMovedRecordFails(t *testing.T) {
	out := filepath.Join(t.TempDir(), "pricing.json")
	tf := module(t, `"m8id.*"`, "0")
	if err := run([]string{"-terraform", tf, "-lists", lists(t, "2026-10-23T22:13:55Z", nil), "-out", out}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-terraform", tf, "-out", out, "-check"}, &bytes.Buffer{}); err != nil {
		t.Errorf("a record written today, of the module's machines: %v", err)
	}
	for name, moved := range map[string]string{
		"the types":     module(t, `"c8id.*"`, "0"),
		"the data disk": module(t, `"m8id.*"`, "100"),
	} {
		err := run([]string{"-terraform", moved, "-out", out, "-check"}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "the module starts") {
			t.Errorf("the module's defaults moved, %s: %v", name, err)
		}
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
