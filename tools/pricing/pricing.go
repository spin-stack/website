// Command pricing writes src/data/pricing.json: what the machines an installation starts by
// default cost an hour, on demand, in each region Spin OS has images in, read from the price list
// the AWS pricing page itself loads (one JSON a region and an operating system, dated by AWS) and
// the module's defaults at the commit the site pins. The cost page draws its scenarios from it,
// each number with its region and the date the list was published; `-check` fails on a record
// older than the window or of other machines than the module's, so a number on the page is never
// quietly a month old.
//
// Spot prices are not here: AWS publishes none without credentials, and a runner's is the hour's.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// listURL is the on-demand Linux price list of one region, as the AWS pricing page loads it:
// %s is the region's name as AWS writes it, percent-encoded.
const listURL = "https://b0.p.awsstatic.com/pricing/2.0/meteredUnitMaps/ec2/USD/current/ec2-ondemand-without-sec-sel/%s/Linux/index.json"

// regionNames is each region code as the list names it: the list is keyed by name, not code.
var regionNames = map[string]string{
	"us-east-1": "US East (N. Virginia)",
	"us-west-2": "US West (Oregon)",
}

// window is how old the record may be before -check refuses it: the daily sync rewrites it,
// and a month without a sync merged is a site nobody is keeping.
const window = 30 * 24 * time.Hour

// Record is src/data/pricing.json.
type Record struct {
	// Generated is the day this was written, UTC.
	Generated string `json:"generated"`
	// Published is when AWS published the lists read, as each list's manifest says.
	Published string `json:"published"`
	// Source is where the lists come from, for a reader.
	Source   string   `json:"source"`
	Machines Machines `json:"machines"`
	// Runners are the types Machines' runner defaults match in every region's list, the cheapest
	// first in the first region: what a runner may be.
	Runners []string          `json:"runners"`
	Regions map[string]Region `json:"regions"`
}

// Machines is what the module starts by default, at the commit the site pins: the control
// plane's and the proxy's machine, what a runner's type is drawn from - the names and patterns
// of instance_types, within vcpus, with a local NVMe while data_volume_gb is 0 - and the most
// runners at once.
type Machines struct {
	ControlPlane    string   `json:"controlPlane"`
	Proxy           string   `json:"proxy"`
	RunnerTypes     []string `json:"runnerTypes"`
	RunnerVCPUs     [2]int   `json:"runnerVcpus"`
	RunnerLocalDisk bool     `json:"runnerLocalDisk"`
	MaxRunners      int      `json:"maxRunners"`
}

// Region is each machine's price in one region.
type Region map[string]Price

// Price is one instance type on demand, an hour, with what it is.
type Price struct {
	Hourly float64 `json:"hourly"`
	VCPU   int     `json:"vcpu"`
	Memory string  `json:"memory"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "pricing:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("pricing", flag.ContinueOnError)
	terraform := fs.String("terraform", ".sources/spin-terraform-aws", "a checkout of spin-terraform-aws at the commit the site pins")
	out := fs.String("out", "src/data/pricing.json", "the record")
	check := fs.Bool("check", false, "write nothing; fail when the record is older than the window or of other machines than the module's defaults")
	lists := fs.String("lists", "", "test: a directory holding <region>.json in place of the lists on the web")
	if err := fs.Parse(args); err != nil {
		return err
	}
	machines, err := defaults(*terraform)
	if err != nil {
		return err
	}
	if *check {
		return checkRecord(*out, machines, time.Now())
	}
	r, err := build(machines, *lists, time.Now())
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "\t")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil { //nolint:gosec // the site's data file
		return err
	}
	fmt.Fprintf(stdout, "pricing: %d regions, lists published %s\n", len(r.Regions), r.Published)
	return nil
}

var (
	variable    = regexp.MustCompile(`(?m)^variable "([a-z_]+)" \{$`)
	stringValue = regexp.MustCompile(`(?m)^\s*default\s*=\s*"([^"]+)"`)
	listValue   = regexp.MustCompile(`(?m)^\s*default\s*=\s*\[([^\]]*)\]`)
	intValue    = regexp.MustCompile(`(?m)^\s*default\s*=\s*(\d+)`)
	rangeValue  = regexp.MustCompile(`(?m)^\s*default\s*=\s*\{\s*min\s*=\s*(\d+)\s*,\s*max\s*=\s*(\d+)\s*\}`)
)

// defaults reads what the module starts by default: the control plane's and the proxy's
// instance type from modules/controlplane, and from modules/runners what a runner's type is
// drawn from and the most runners (max_hosts, which the root calls max_runners). A variable
// without a default is a mistake here, not a value to guess.
func defaults(terraform string) (Machines, error) {
	cp, err := os.ReadFile(filepath.Join(terraform, "modules", "controlplane", "variables.tf"))
	if err != nil {
		return Machines{}, err
	}
	runners, err := os.ReadFile(filepath.Join(terraform, "modules", "runners", "variables.tf"))
	if err != nil {
		return Machines{}, err
	}
	var m Machines
	if m.ControlPlane, err = stringDefault(cp, "instance_type"); err != nil {
		return Machines{}, err
	}
	if m.Proxy, err = stringDefault(cp, "proxy_instance_type"); err != nil {
		return Machines{}, err
	}
	block, err := variableBlock(runners, "instance_types")
	if err != nil {
		return Machines{}, err
	}
	l := listValue.FindSubmatch(block)
	if l == nil {
		return Machines{}, errors.New("modules/runners instance_types: no list default")
	}
	for _, item := range strings.Split(string(l[1]), ",") {
		if item = strings.Trim(strings.TrimSpace(item), `"`); item != "" {
			m.RunnerTypes = append(m.RunnerTypes, item)
		}
	}
	if len(m.RunnerTypes) == 0 {
		return Machines{}, errors.New("modules/runners instance_types: an empty default")
	}
	block, err = variableBlock(runners, "vcpus")
	if err != nil {
		return Machines{}, err
	}
	v := rangeValue.FindSubmatch(block)
	if v == nil {
		return Machines{}, errors.New("modules/runners vcpus: no { min, max } default")
	}
	m.RunnerVCPUs[0], _ = strconv.Atoi(string(v[1])) // digits, by the expression
	m.RunnerVCPUs[1], _ = strconv.Atoi(string(v[2]))
	if m.RunnerLocalDisk, err = zeroDefault(runners, "data_volume_gb"); err != nil {
		return Machines{}, err
	}
	block, err = variableBlock(runners, "max_hosts")
	if err != nil {
		return Machines{}, err
	}
	n := intValue.FindSubmatch(block)
	if n == nil {
		return Machines{}, errors.New("modules/runners max_hosts: no number default")
	}
	m.MaxRunners, _ = strconv.Atoi(string(n[1])) // digits, by the expression
	return m, nil
}

func stringDefault(src []byte, name string) (string, error) {
	block, err := variableBlock(src, name)
	if err != nil {
		return "", err
	}
	m := stringValue.FindSubmatch(block)
	if m == nil {
		return "", fmt.Errorf("%s: no string default", name)
	}
	return string(m[1]), nil
}

// zeroDefault is whether variable name's number default is 0.
func zeroDefault(src []byte, name string) (bool, error) {
	block, err := variableBlock(src, name)
	if err != nil {
		return false, err
	}
	m := intValue.FindSubmatch(block)
	if m == nil {
		return false, fmt.Errorf("%s: no number default", name)
	}
	return string(m[1]) == "0", nil
}

// variableBlock is the text of variable "name" up to the next variable.
func variableBlock(src []byte, name string) ([]byte, error) {
	all := variable.FindAllSubmatchIndex(src, -1)
	for i, m := range all {
		if string(src[m[2]:m[3]]) != name {
			continue
		}
		end := len(src)
		if i+1 < len(all) {
			end = all[i+1][0]
		}
		return src[m[1]:end], nil
	}
	return nil, fmt.Errorf("no variable %q", name)
}

// row is one instance type of a region's list.
type row struct {
	Price   string `json:"price"`
	VCPU    string `json:"vCPU"`
	Memory  string `json:"Memory"`
	Type    string `json:"Instance Type"`
	Storage string `json:"Storage"`
}

// list is one region's price list as the pricing page loads it: a manifest with its
// publication date, and the region's prices keyed "<family> <size> <region name> Linux".
type list struct {
	Manifest struct {
		Published string `json:"hawkFilePublicationDate"`
	} `json:"manifest"`
	Regions map[string]map[string]row `json:"regions"`
}

// build reads each region's list and keeps the machines' prices: the control plane's and the
// proxy's, and every type a runner may be there. A machine the list does not price fails, and so
// does a runner's defaults that match no type every region prices: a type the module starts that
// a region lacks is for the module to fix, not a blank on the page.
func build(m Machines, lists string, now time.Time) (Record, error) {
	r := Record{Generated: now.UTC().Format("2006-01-02"), Source: fmt.Sprintf(listURL, "<region>"), Machines: m, Regions: map[string]Region{}}
	codes := make([]string, 0, len(regionNames))
	for code := range regionNames {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	rows := map[string]map[string]row{}
	for _, code := range codes {
		raw, err := fetch(code, lists)
		if err != nil {
			return Record{}, err
		}
		var l list
		if err := json.Unmarshal(raw, &l); err != nil {
			return Record{}, fmt.Errorf("%s's list: %w", code, err)
		}
		if l.Manifest.Published == "" {
			return Record{}, fmt.Errorf("%s's list: no publication date in its manifest", code)
		}
		if r.Published == "" || l.Manifest.Published > r.Published {
			r.Published = l.Manifest.Published
		}
		prices, ok := l.Regions[regionNames[code]]
		if !ok {
			return Record{}, fmt.Errorf("%s's list holds no region %q", code, regionNames[code])
		}
		rows[code] = map[string]row{}
		for _, p := range prices {
			rows[code][p.Type] = p
		}
	}
	runners, err := runnerTypes(m, codes, rows)
	if err != nil {
		return Record{}, err
	}
	r.Runners = runners
	for _, code := range codes {
		region := Region{}
		for _, t := range append([]string{m.ControlPlane, m.Proxy}, runners...) {
			p, ok := rows[code][t]
			if !ok {
				return Record{}, fmt.Errorf("%s: no on-demand price for %s, which the module starts by default", code, t)
			}
			price, err := priced(code, p)
			if err != nil {
				return Record{}, err
			}
			region[t] = price
		}
		r.Regions[code] = region
	}
	return r, nil
}

// runnerTypes is every type m's runner defaults match - a name or pattern of RunnerTypes, within
// RunnerVCPUs, with a local NVMe where RunnerLocalDisk says - that every region prices, the
// cheapest first in the first region.
func runnerTypes(m Machines, codes []string, rows map[string]map[string]row) ([]string, error) {
	first := rows[codes[0]]
	var types []string
	for t, p := range first {
		if !slices.ContainsFunc(m.RunnerTypes, func(pattern string) bool { ok, _ := path.Match(pattern, t); return ok }) {
			continue
		}
		vcpu, err := strconv.Atoi(p.VCPU)
		if err != nil || vcpu < m.RunnerVCPUs[0] || vcpu > m.RunnerVCPUs[1] {
			continue
		}
		if m.RunnerLocalDisk && (p.Storage == "" || p.Storage == "EBS only") {
			continue
		}
		if slices.ContainsFunc(codes, func(code string) bool { _, ok := rows[code][t]; return !ok }) {
			continue
		}
		types = append(types, t)
	}
	if len(types) == 0 {
		return nil, fmt.Errorf("no type every region prices matches the runners' defaults %v, %d to %d vCPUs", m.RunnerTypes, m.RunnerVCPUs[0], m.RunnerVCPUs[1])
	}
	hourly := func(t string) float64 { h, _ := strconv.ParseFloat(first[t].Price, 64); return h }
	sort.Slice(types, func(a, b int) bool {
		if ha, hb := hourly(types[a]), hourly(types[b]); ha != hb {
			return ha < hb
		}
		return types[a] < types[b]
	})
	return types, nil
}

// priced is a list's row as the record keeps it.
func priced(code string, p row) (Price, error) {
	hourly, err := strconv.ParseFloat(p.Price, 64)
	if err != nil {
		return Price{}, fmt.Errorf("%s %s: price %q: %w", code, p.Type, p.Price, err)
	}
	vcpu, err := strconv.Atoi(p.VCPU)
	if err != nil {
		return Price{}, fmt.Errorf("%s %s: vCPU %q: %w", code, p.Type, p.VCPU, err)
	}
	return Price{Hourly: hourly, VCPU: vcpu, Memory: p.Memory}, nil
}

func fetch(code, lists string) ([]byte, error) {
	if lists != "" {
		return os.ReadFile(filepath.Join(lists, code+".json"))
	}
	u := fmt.Sprintf(listURL, strings.ReplaceAll(strings.ReplaceAll(regionNames[code], " ", "%20"), "(", "%28"))
	u = strings.ReplaceAll(u, ")", "%29")
	resp, err := http.Get(u) //nolint:gosec,noctx // a public list, one request
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }() // read-only
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", u, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// checkRecord holds the record to the window and to the module's defaults: one older than the
// window, or of other machines than the module starts at the pinned commit, fails.
func checkRecord(path string, m Machines, now time.Time) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var r Record
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	day, err := time.Parse("2006-01-02", r.Generated)
	if err != nil {
		return fmt.Errorf("%s: generated %q: %w", path, r.Generated, err)
	}
	if now.Sub(day) > window {
		return fmt.Errorf("%s was generated on %s, more than %d days ago: run task pricing", path, r.Generated, int(window.Hours()/24))
	}
	have, _ := json.Marshal(r.Machines)
	want, _ := json.Marshal(m)
	if string(have) != string(want) {
		return fmt.Errorf("%s prices %s, and the module starts %s by default: run task pricing", path, have, want)
	}
	if len(r.Runners) == 0 {
		return fmt.Errorf("%s names no type a runner may be: run task pricing", path)
	}
	for code := range regionNames {
		if _, ok := r.Regions[code]; !ok {
			return fmt.Errorf("%s has no prices for %s: run task pricing", path, code)
		}
	}
	return nil
}
