package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/app"
	domainbrowser "github.com/phin-tech/whisk/internal/domain/browser"
)

func TestBrowserDiagnosticServiceReportsDisabledWithoutEndpoint(t *testing.T) {
	probe := &fakeBrowserProbe{}
	service := app.NewBrowserDiagnosticService(probe)

	result, err := service.Diagnose(context.Background(), app.BrowserDiagnosticRequest{})
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	if result.Status != app.BrowserDiagnosticDisabled || result.Enabled {
		t.Fatalf("result = %#v", result)
	}
	if probe.called {
		t.Fatalf("probe was called for disabled browser diagnostic")
	}
}

func TestBrowserDiagnosticServiceProbesEndpoint(t *testing.T) {
	probe := &fakeBrowserProbe{
		result: domainbrowser.CDPProbeResult{
			Endpoint:        "http://127.0.0.1:9222",
			Browser:         "Chrome/126",
			ProtocolVersion: "1.3",
			Targets: []domainbrowser.CDPTarget{
				{ID: "page_1", Type: "page", URL: "https://example.test", Title: "Example"},
			},
		},
	}
	service := app.NewBrowserDiagnosticService(probe)

	result, err := service.Diagnose(context.Background(), app.BrowserDiagnosticRequest{CDPURL: "http://127.0.0.1:9222"})
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	if !probe.called || probe.endpoint != "http://127.0.0.1:9222" {
		t.Fatalf("probe called=%v endpoint=%q", probe.called, probe.endpoint)
	}
	if result.Status != app.BrowserDiagnosticOK || result.TargetCount != 1 || result.Browser != "Chrome/126" {
		t.Fatalf("result = %#v", result)
	}
}

func TestBrowserDiagnosticServiceCapturesProbeError(t *testing.T) {
	service := app.NewBrowserDiagnosticService(&fakeBrowserProbe{err: errors.New("dial tcp: connection refused")})

	result, err := service.Diagnose(context.Background(), app.BrowserDiagnosticRequest{CDPURL: "http://127.0.0.1:9222"})
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	if result.Status != app.BrowserDiagnosticError || !strings.Contains(result.Error, "connection refused") {
		t.Fatalf("result = %#v", result)
	}
}

func TestBrowserDiagnosticServiceBuildsLaunchCommand(t *testing.T) {
	service := app.NewBrowserDiagnosticService(&fakeBrowserProbe{})

	result, err := service.Diagnose(context.Background(), app.BrowserDiagnosticRequest{
		ChromePath:    "/chrome",
		UserDataDir:   "/tmp/whisk-browser",
		DebuggingPort: 9223,
	})
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	if result.LaunchCommand == nil || result.LaunchCommand.Endpoint != "http://127.0.0.1:9223" {
		t.Fatalf("launch command = %#v", result.LaunchCommand)
	}
}

func TestRuntimeDiagnoseBrowserUsesConfiguredProbe(t *testing.T) {
	probe := &fakeBrowserProbe{
		result: domainbrowser.CDPProbeResult{
			Endpoint: "http://127.0.0.1:9222",
			Browser:  "Chrome/126",
		},
	}
	runtime := app.NewRuntime(app.RuntimeConfig{BrowserProbe: probe})

	result, err := runtime.DiagnoseBrowser(context.Background(), app.BrowserDiagnosticRequest{CDPURL: "http://127.0.0.1:9222"})
	if err != nil {
		t.Fatalf("DiagnoseBrowser: %v", err)
	}
	if !probe.called || result.Status != app.BrowserDiagnosticOK {
		t.Fatalf("called=%v result=%#v", probe.called, result)
	}
}

type fakeBrowserProbe struct {
	called   bool
	endpoint string
	result   domainbrowser.CDPProbeResult
	err      error
}

func (f *fakeBrowserProbe) ProbeCDP(_ context.Context, endpoint string) (domainbrowser.CDPProbeResult, error) {
	f.called = true
	f.endpoint = endpoint
	return f.result, f.err
}
