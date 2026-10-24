package main

import (
	"fmt"

	"github.com/spin-stack/go-tools/versions"
)

// checkPins holds reference.json to versions.yaml: the commits the reference was generated from
// are the ones pinned, and a pin at a release is that release. Two records of the same thing
// that could disagree would have the sync move one and the site go on documenting the other; a
// reference generated from a checkout that is not the pin is one CI cannot reproduce.
func checkPins(file string, r reference) ([]string, error) {
	v, err := versions.Load(file)
	if err != nil {
		return nil, err
	}
	var bad []string
	for _, c := range []struct {
		name, commit, release, field string
	}{
		{"spin", r.Spin.Commit, r.Spin.Tag, "spin.tag"},
		{"spin-terraform-aws", r.Terraform.Commit, r.Terraform.Release, "terraform.release"},
		{"brand", r.Brand.Commit, "", ""},
	} {
		e, err := v.Get(c.name)
		if err != nil {
			return nil, err
		}
		if c.commit != e.Pin {
			bad = append(bad, fmt.Sprintf("reference.json: %s at %s; %s pins %s (task sources, then task reference)", c.name, c.commit, file, e.Pin))
		}
		if c.field != "" && releaseParts.MatchString(e.Version) && c.release != e.Version {
			bad = append(bad, fmt.Sprintf("reference.json: %s is %q; %s pins %s at the release %s", c.field, c.release, file, c.name, e.Version))
		}
	}
	return bad, nil
}
