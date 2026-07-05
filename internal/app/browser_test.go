package app_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestRuntimeConnectBrowserResourceOwnsReadModel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sink := newRecordingEventSink()
	targets := &fakeBrowserTargets{
		targets: []domainbrowser.Target{
			{ID: "page_1", ResourceID: "whisk_000001", Type: domainbrowser.TargetTypePage, Status: domainbrowser.TargetStatusAvailable, URL: "https://example.test", Title: "Example"},
		},
	}
	runtime := app.NewRuntime(app.RuntimeConfig{BrowserTargets: targets, EventSink: sink})
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })

	resource, err := runtime.ConnectBrowserResource(ctx, app.ConnectBrowserResourceRequest{
		Name:                          " Main ",
		CDPURL:                        " http://127.0.0.1:9222/ ",
		AcknowledgeBrowserControlRisk: true,
	})
	if err != nil {
		t.Fatalf("ConnectBrowserResource: %v", err)
	}
	wantResource := domainbrowser.Resource{
		ID:        "whisk_000001",
		Name:      "Main",
		CDPURL:    "http://127.0.0.1:9222",
		Connected: true,
	}
	if resource != wantResource {
		t.Fatalf("resource = %#v, want %#v", resource, wantResource)
	}
	if !targets.called || targets.endpoint != "http://127.0.0.1:9222" || targets.resourceID != "whisk_000001" {
		t.Fatalf("target backend called=%v endpoint=%q resourceID=%q", targets.called, targets.endpoint, targets.resourceID)
	}

	resources, err := runtime.ListBrowserResources(ctx)
	if err != nil {
		t.Fatalf("ListBrowserResources: %v", err)
	}
	if !reflect.DeepEqual(resources, []domainbrowser.Resource{wantResource}) {
		t.Fatalf("resources = %#v", resources)
	}
	listedTargets, err := runtime.ListBrowserTargets(ctx, "whisk_000001")
	if err != nil {
		t.Fatalf("ListBrowserTargets: %v", err)
	}
	if !reflect.DeepEqual(listedTargets, targets.targets) {
		t.Fatalf("targets = %#v, want %#v", listedTargets, targets.targets)
	}
	sink.waitFor(t, ctx, app.EventBrowserChanged, "")
}

func TestRuntimeConnectBrowserResourceRequiresAcknowledgement(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{BrowserTargets: &fakeBrowserTargets{}})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	_, err := runtime.ConnectBrowserResource(context.Background(), app.ConnectBrowserResourceRequest{CDPURL: "http://127.0.0.1:9222"})
	if err == nil || !strings.Contains(err.Error(), "acknowledgement required") {
		t.Fatalf("err = %v", err)
	}
}

func TestRuntimeDisconnectBrowserResourceRemovesReadModel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sink := newRecordingEventSink()
	runtime := app.NewRuntime(app.RuntimeConfig{
		BrowserTargets: &fakeBrowserTargets{},
		EventSink:      sink,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })

	resource, err := runtime.ConnectBrowserResource(ctx, app.ConnectBrowserResourceRequest{
		CDPURL:                        "http://127.0.0.1:9222",
		AcknowledgeBrowserControlRisk: true,
	})
	if err != nil {
		t.Fatalf("ConnectBrowserResource: %v", err)
	}
	sink.waitFor(t, ctx, app.EventBrowserChanged, "")

	if err := runtime.DisconnectBrowserResource(ctx, string(resource.ID)); err != nil {
		t.Fatalf("DisconnectBrowserResource: %v", err)
	}
	sink.waitFor(t, ctx, app.EventBrowserChanged, "")
	resources, err := runtime.ListBrowserResources(ctx)
	if err != nil {
		t.Fatalf("ListBrowserResources: %v", err)
	}
	if len(resources) != 0 {
		t.Fatalf("resources = %#v", resources)
	}
	if _, err := runtime.ListBrowserTargets(ctx, string(resource.ID)); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("list targets err = %v", err)
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

type fakeBrowserTargets struct {
	called     bool
	endpoint   string
	resourceID domainbrowser.ResourceID
	targets    []domainbrowser.Target
	err        error
}

func (f *fakeBrowserTargets) ListTargets(_ context.Context, endpoint string, resourceID domainbrowser.ResourceID) ([]domainbrowser.Target, error) {
	f.called = true
	f.endpoint = endpoint
	f.resourceID = resourceID
	return f.targets, f.err
}
