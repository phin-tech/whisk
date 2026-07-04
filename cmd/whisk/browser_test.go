package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
)

func TestRunBrowserDiagnosePassesFlagsAndPrintsJSON(t *testing.T) {
	var got app.BrowserDiagnosticRequest
	deps := runDeps{
		browserDiagnose: func(_ context.Context, req app.BrowserDiagnosticRequest) (app.BrowserDiagnostic, error) {
			got = req
			return app.BrowserDiagnostic{
				Enabled:     true,
				Status:      app.BrowserDiagnosticOK,
				CDPURL:      req.CDPURL,
				Browser:     "Chrome/126",
				TargetCount: 2,
			}, nil
		},
	}

	output, err := captureStdout(func() error {
		return runWithDeps([]string{"browser", "diagnose", "-cdp-url", "http://127.0.0.1:9222", "-timeout", "2s", "-json"}, deps)
	})
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if got.CDPURL != "http://127.0.0.1:9222" || got.Timeout != 2*time.Second {
		t.Fatalf("request = %#v", got)
	}
	var result app.BrowserDiagnostic
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output %q: %v", output, err)
	}
	if result.Status != app.BrowserDiagnosticOK || result.TargetCount != 2 {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunBrowserDiagnosePrintsDisabledTable(t *testing.T) {
	deps := runDeps{
		browserDiagnose: func(_ context.Context, req app.BrowserDiagnosticRequest) (app.BrowserDiagnostic, error) {
			if req.CDPURL != "" {
				t.Fatalf("request = %#v", req)
			}
			return app.BrowserDiagnostic{Status: app.BrowserDiagnosticDisabled, Error: "explicit -cdp-url required"}, nil
		},
	}

	output, err := captureStdout(func() error {
		return runWithDeps([]string{"browser", "diagnose"}, deps)
	})
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if !strings.Contains(output, "STATUS") || !strings.Contains(output, string(app.BrowserDiagnosticDisabled)) {
		t.Fatalf("output = %q", output)
	}
}

func TestFormatLaunchCommandUsesShellSafeQuoting(t *testing.T) {
	got := formatLaunchCommand(
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		[]string{
			"--remote-debugging-address=127.0.0.1",
			"--user-data-dir=/tmp/whisk profile/o'hare",
			"",
		},
	)
	want := `'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' --remote-debugging-address=127.0.0.1 '--user-data-dir=/tmp/whisk profile/o'\''hare' ''`
	if got != want {
		t.Fatalf("formatLaunchCommand = %q, want %q", got, want)
	}
}
