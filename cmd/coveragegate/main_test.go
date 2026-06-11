package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCoveragePercent(t *testing.T) {
	profile := writeProfile(t, `mode: set
file.go:1.1,2.1 2 1
file.go:3.1,4.1 2 0
`)
	coverage, err := coveragePercent(profile)
	if err != nil {
		t.Fatalf("coveragePercent error: %v", err)
	}
	if coverage != 50 {
		t.Fatalf("coverage = %v", coverage)
	}
}

func TestCoveragePercentRejectsInvalidProfiles(t *testing.T) {
	tests := map[string]string{
		"empty":      `mode: set`,
		"bad fields": `file.go:1.1,2.1 2`,
		"bad stmts":  `file.go:1.1,2.1 nope 1`,
		"bad count":  `file.go:1.1,2.1 1 nope`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := coveragePercent(writeProfile(t, body)); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func writeProfile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cover.out")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	return path
}
