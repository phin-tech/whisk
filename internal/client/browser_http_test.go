package client_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/client"
	domainbrowser "github.com/phin-tech/whisk/internal/domain/browser"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPClientBrowserResourceLifecycle(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{
		BrowserTargets: &fakeBrowserTargetBackend{},
		EventSink:      newFakeEventBus(),
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := daemon.ConnectBrowserResource(ctx, protocol.ConnectBrowserResourceRequest{
		Name:                          "Main",
		CDPURL:                        "http://127.0.0.1:9222",
		AcknowledgeBrowserControlRisk: true,
	})
	if err != nil {
		t.Fatalf("ConnectBrowserResource: %v", err)
	}
	if created.ID != "whisk_000001" || created.CDPURL != "http://127.0.0.1:9222" || !created.Connected {
		t.Fatalf("created = %#v", created)
	}

	resources, err := daemon.ListBrowserResources(ctx)
	if err != nil {
		t.Fatalf("ListBrowserResources: %v", err)
	}
	if len(resources) != 1 || resources[0] != created {
		t.Fatalf("resources = %#v", resources)
	}
	targets, err := daemon.ListBrowserTargets(ctx, created.ID)
	if err != nil {
		t.Fatalf("ListBrowserTargets: %v", err)
	}
	if len(targets) != 1 || targets[0].ID != "page_1" || targets[0].ResourceID != created.ID {
		t.Fatalf("targets = %#v", targets)
	}
	event, err := daemon.NextEvent(ctx, protocol.NextEventRequest{TimeoutMs: 10})
	if err != nil || event.Event.Type != "browser.changed" {
		t.Fatalf("event = %#v, err = %v", event, err)
	}

	if err := daemon.DisconnectBrowserResource(ctx, created.ID); err != nil {
		t.Fatalf("DisconnectBrowserResource: %v", err)
	}
	resources, err = daemon.ListBrowserResources(ctx)
	if err != nil {
		t.Fatalf("ListBrowserResources after disconnect: %v", err)
	}
	if len(resources) != 0 {
		t.Fatalf("resources after disconnect = %#v", resources)
	}
}

type fakeBrowserTargetBackend struct{}

func (b *fakeBrowserTargetBackend) ListTargets(_ context.Context, _ string, resourceID domainbrowser.ResourceID) ([]domainbrowser.Target, error) {
	return []domainbrowser.Target{{
		ID:         "page_1",
		ResourceID: resourceID,
		Type:       domainbrowser.TargetTypePage,
		Status:     domainbrowser.TargetStatusAvailable,
		URL:        "https://example.test",
		Title:      "Example",
	}}, nil
}
