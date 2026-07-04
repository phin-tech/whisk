package app_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
)

func TestNormalizeUsageResolverResultValidatesMetrics(t *testing.T) {
	used := 125.0
	limit := 200.0
	remaining := 75.0
	resetAt := time.Date(2026, 7, 4, 12, 0, 0, 0, time.FixedZone("test", -4*60*60))

	result, err := app.NormalizeUsageResolverResult(app.UsageResolverResult{
		Summary: "  75 remaining  ",
		Metrics: []app.UsageResolverMetric{{
			ID:        " tokens ",
			Kind:      "rateLimit",
			Label:     " Tokens ",
			Unit:      " tokens ",
			Used:      &used,
			Limit:     &limit,
			Remaining: &remaining,
			ResetAt:   &resetAt,
		}},
		Meta: map[string]string{" source ": " plugin "},
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if result.Summary != "75 remaining" ||
		result.Metrics[0].ID != "tokens" ||
		result.Metrics[0].Kind != app.UsageMetricKindRateLimit ||
		result.Metrics[0].Unit != "tokens" ||
		result.Metrics[0].ResetAt.Location() != time.UTC ||
		result.Meta["source"] != "plugin" {
		t.Fatalf("normalized result = %#v", result)
	}

	_, err = app.NormalizeUsageResolverResult(app.UsageResolverResult{
		Metrics: []app.UsageResolverMetric{{ID: "bad", Kind: "quota"}},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported kind err = %v", err)
	}
}

func TestRuntimeRefreshUsageResolverCachesDaemonReadModel(t *testing.T) {
	ctx := context.Background()
	sink := &memoryEventSink{}
	used := 10.0
	limit := 100.0
	plugins := &memoryPluginRegistry{
		status: app.PluginStatus{
			ID:      "github",
			Name:    "GitHub",
			Version: "1.0.0",
			Trusted: true,
			Valid:   true,
			UsageResolvers: []app.PluginUsageResolver{{
				ID:           "github.usage",
				Provider:     "github",
				Label:        "GitHub",
				Profiles:     []string{"codex"},
				MinRefreshMs: 300000,
				StaleAfterMs: 1800000,
			}},
		},
		usageResult: app.UsageResolverResult{
			Summary: "10%",
			Metrics: []app.UsageResolverMetric{{
				ID:    "requests",
				Kind:  app.UsageMetricKindUsage,
				Used:  &used,
				Limit: &limit,
			}},
		},
	}
	runtime := app.NewRuntime(app.RuntimeConfig{Plugins: plugins, EventSink: sink})

	listed, err := runtime.ListUsageResolverResults(ctx)
	if err != nil || len(listed) != 1 || listed[0].Status != app.UsageResolverStatusPending {
		t.Fatalf("initial usage results = %#v, err = %v", listed, err)
	}
	refreshed, err := runtime.RefreshUsageResolver(ctx, app.RunUsageResolverRequest{
		PluginID:   "github",
		ResolverID: "github.usage",
	})
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.Status != app.UsageResolverStatusOK ||
		refreshed.Profile != "codex" ||
		refreshed.Result == nil ||
		refreshed.Result.Summary != "10%" ||
		refreshed.RefreshedAt == nil {
		t.Fatalf("refreshed = %#v", refreshed)
	}
	if plugins.usageReq.PluginID != "github" || plugins.usageReq.ResolverID != "github.usage" || plugins.usageReq.Profile != "codex" {
		t.Fatalf("usage request = %#v", plugins.usageReq)
	}
	if got := countRuntimeEvents(sink.events, app.EventPluginsChanged); got != 1 {
		t.Fatalf("plugin events = %d, want 1; events=%#v", got, sink.events)
	}
	listed, err = runtime.ListUsageResolverResults(ctx)
	if err != nil || len(listed) != 1 || listed[0].Status != app.UsageResolverStatusOK || listed[0].Result == nil {
		t.Fatalf("cached usage results = %#v, err = %v", listed, err)
	}

	plugins.usageErr = fmt.Errorf("usage backend unavailable")
	failed, err := runtime.RefreshUsageResolver(ctx, app.RunUsageResolverRequest{
		PluginID:   "github",
		ResolverID: "github.usage",
		Profile:    "codex",
	})
	if err != nil {
		t.Fatalf("refresh command failure should be cached, got err = %v", err)
	}
	if failed.Status != app.UsageResolverStatusError || !strings.Contains(failed.Error, "unavailable") {
		t.Fatalf("failed refresh = %#v", failed)
	}

	plugins.status.Trusted = false
	listed, err = runtime.ListUsageResolverResults(ctx)
	if err != nil || len(listed) != 1 || listed[0].Status != app.UsageResolverStatusUntrusted || listed[0].Result != nil {
		t.Fatalf("untrusted usage results should ignore cache = %#v, err = %v", listed, err)
	}
}
