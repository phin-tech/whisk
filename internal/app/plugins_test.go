package app_test

import (
	"context"
	"testing"

	"github.com/phin-tech/whisk/internal/app"
)

func TestRuntimePluginMutationsPublishPluginsChanged(t *testing.T) {
	ctx := context.Background()
	sink := &memoryEventSink{}
	plugins := &memoryPluginRegistry{
		status: app.PluginStatus{
			ID:      "github",
			Name:    "GitHub",
			Version: "1.0.0",
			Valid:   true,
		},
		registry: app.RegistryPlugin{Registry: "phin-tech", ID: "github", Name: "GitHub", SourceType: "path"},
	}
	runtime := app.NewRuntime(app.RuntimeConfig{Plugins: plugins, EventSink: sink})

	if _, err := runtime.ListPlugins(ctx); err != nil {
		t.Fatalf("list plugins: %v", err)
	}
	if got := countRuntimeEvents(sink.events, app.EventPluginsChanged); got != 0 {
		t.Fatalf("list plugins published %d plugin events, want 0", got)
	}
	if _, err := runtime.RescanPlugins(ctx); err != nil {
		t.Fatalf("rescan plugins: %v", err)
	}
	if _, err := runtime.TrustPlugin(ctx, "github"); err != nil {
		t.Fatalf("trust plugin: %v", err)
	}
	if _, err := runtime.UntrustPlugin(ctx, "github"); err != nil {
		t.Fatalf("untrust plugin: %v", err)
	}
	if _, err := runtime.ListRegistryPlugins(ctx); err != nil {
		t.Fatalf("list registry plugins: %v", err)
	}
	if _, err := runtime.InstallPlugin(ctx, "phin-tech", "github"); err != nil {
		t.Fatalf("install plugin: %v", err)
	}

	if got := countRuntimeEvents(sink.events, app.EventPluginsChanged); got != 4 {
		t.Fatalf("plugin mutations published %d plugin events, want 4; events=%#v", got, sink.events)
	}
}

func countRuntimeEvents(events []app.RuntimeEvent, eventType app.RuntimeEventType) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}
