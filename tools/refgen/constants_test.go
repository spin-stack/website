package main

import (
	"go/constant"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A constant is worked out from the file as the compiler would: shifts, time's units, another
// constant of the file, integer division.
func TestAConstantIsWorkedOutFromItsFile(t *testing.T) {
	file := filepath.Join("testdata", "consts", "consts.go")
	for name, want := range map[string]int64{
		"Size":   1 << 30,
		"Window": int64(90 * 24 * time.Hour),
		"Half":   1 << 29,
		"Pair":   1 << 30,
	} {
		v, err := goConstant(file, name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got, ok := constant.Int64Val(v); !ok || got != want {
			t.Errorf("%s = %v, want %d", name, v, want)
		}
	}
	if _, err := goConstant(file, "Other"); err == nil || !strings.Contains(err.Error(), "another package") {
		t.Errorf("a constant the file cannot say gave %v", err)
	}
	if _, err := goConstant(file, "Missing"); err == nil {
		t.Error("a constant the file does not declare was found")
	}
}

// A figure is written in the largest unit it is a whole number of.
func TestAFigureIsWrittenInItsLargestWholeUnit(t *testing.T) {
	for got, want := range map[string]string{
		bytesText(1 << 30):                    "1 GiB",
		bytesText(3 << 29):                    "1536 MiB",
		bytesText(100):                        "100 bytes",
		durationText(90 * 24 * time.Hour):     "90 days",
		durationText(36 * time.Hour):          "36 hours",
		durationText(time.Minute):             "1 minute",
		durationText(1500 * time.Millisecond): "1.5s",
	} {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

// The module's half is its default, and where it is declared.
func TestTheLogRetentionIsTheModulesDefault(t *testing.T) {
	var c constants
	if err := terraformConstants(&c, fixtureTF); err != nil {
		t.Fatal(err)
	}
	if want := (codeValue{"30 days", "spin-terraform-aws:modules/controlplane/main.tf"}); c.LogRetentionDays != want {
		t.Errorf("logRetentionDays = %+v, want %+v", c.LogRetentionDays, want)
	}
}
