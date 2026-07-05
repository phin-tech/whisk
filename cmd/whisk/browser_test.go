package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/protocol"
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

func TestRunBrowserAttachUsesDaemonRouteAndPrintsJSON(t *testing.T) {
	var got protocol.ConnectBrowserResourceRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/browser-resources" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"browser_1","name":"Main","cdpUrl":"http://127.0.0.1:9222","connected":true}`))
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return runWithDeps([]string{"browser", "attach", "-url", server.URL, "-cdp-url", "http://127.0.0.1:9222", "-name", "Main", "-ack", "-json"}, runDeps{})
	})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if got.CDPURL != "http://127.0.0.1:9222" || got.Name != "Main" || !got.AcknowledgeBrowserControlRisk {
		t.Fatalf("request = %#v", got)
	}
	var resource protocol.BrowserResource
	if err := json.Unmarshal([]byte(output), &resource); err != nil {
		t.Fatalf("decode output %q: %v", output, err)
	}
	if resource.ID != "browser_1" || !resource.Connected {
		t.Fatalf("resource = %#v", resource)
	}
}

func TestRunBrowserListPrintsTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/browser-resources" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"browser_1","name":"Main","cdpUrl":"http://127.0.0.1:9222","connected":true}]`))
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return runWithDeps([]string{"browser", "list", "-url", server.URL}, runDeps{})
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "browser_1") || !strings.Contains(output, "http://127.0.0.1:9222") {
		t.Fatalf("output = %q", output)
	}
}

func TestRunBrowserTargetsPrintsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/browser-resources/browser_1/targets" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"page_1","resourceId":"browser_1","type":"page","status":"available","url":"https://example.test","title":"Example"}]`))
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return runWithDeps([]string{"browser", "targets", "-url", server.URL, "-json", "browser_1"}, runDeps{})
	})
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	var targets []protocol.BrowserTarget
	if err := json.Unmarshal([]byte(output), &targets); err != nil {
		t.Fatalf("decode output %q: %v", output, err)
	}
	if len(targets) != 1 || targets[0].ID != "page_1" || targets[0].ResourceID != "browser_1" {
		t.Fatalf("targets = %#v", targets)
	}
}

func TestRunBrowserDetachUsesDaemonRouteAndPrintsJSON(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/browser-resources/browser_1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return runWithDeps([]string{"browser", "detach", "-url", server.URL, "-json", "browser_1"}, runDeps{})
	})
	if err != nil {
		t.Fatalf("detach: %v", err)
	}
	if !called {
		t.Fatalf("detach endpoint was not called")
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output %q: %v", output, err)
	}
	if result["resourceId"] != "browser_1" || result["detached"] != true {
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
