package app

import "testing"

func TestAggregateUIContributions(t *testing.T) {
	plugins := []PluginStatus{
		{
			ID:      " plugin-a ",
			Name:    "Plugin A",
			Version: "1.0.0",
			Trusted: true,
			Valid:   true,
			UIPanels: []PluginUIPanel{
				{ID: "panel-project", Title: "Project Panel", Scope: PluginUIScope("project")},
				{ID: " panel-global ", Title: "Global Panel", Scope: PluginUIScope("global")},
			},
			UICommands: []PluginUICommand{
				{ID: "cmd-workitem", Label: "Work Item Command", Scope: PluginUIScope("workItem")},
			},
			ReviewActions: []PluginReviewAction{
				{ID: "review-run", Label: "Run Review", Scope: PluginUIScope("run")},
			},
		},
		{
			ID:      "plugin-b",
			Name:    "Plugin B",
			Version: "2.0.0",
			Trusted: false,
			Valid:   false,
			Error:   "invalid manifest",
			UIPanels: []PluginUIPanel{
				{ID: "panel-global-b", Title: "Global Panel B", Scope: PluginUIScope("global")},
			},
		},
		{
			ID:      "plugin-c",
			Name:    "Plugin C",
			Version: "3.0.0",
			Trusted: true,
		},
	}

	t.Run("omitted scope returns trusted global contributions only", func(t *testing.T) {
		result := AggregateUIContributions(plugins, UIContributionScope{})
		if len(result) != 1 {
			t.Fatalf("expected 1 plugin, got %d", len(result))
		}
		if result[0].PluginID != "plugin-a" {
			t.Fatalf("expected plugin-a first, got %s", result[0].PluginID)
		}
		if len(result[0].Panels) != 1 || result[0].Panels[0].ID != "panel-global" {
			t.Fatalf("expected only global panel, got %#v", result[0].Panels)
		}
		if len(result[0].Commands) != 0 || len(result[0].ReviewActions) != 0 {
			t.Fatalf("expected no entity-scoped commands/actions, got commands=%#v actions=%#v", result[0].Commands, result[0].ReviewActions)
		}
	})

	t.Run("scope filters contributions per plugin", func(t *testing.T) {
		result := AggregateUIContributions(plugins, UIContributionScope{ProjectID: " proj_01 "})
		if len(result) != 1 {
			t.Fatalf("expected 1 plugin, got %d", len(result))
		}
		if len(result[0].Panels) != 2 {
			t.Fatalf("plugin-a: expected 2 panels (global + project), got %d", len(result[0].Panels))
		}
		if len(result[0].Commands) != 0 {
			t.Fatalf("plugin-a: expected 0 commands, got %d", len(result[0].Commands))
		}
		if len(result[0].ReviewActions) != 0 {
			t.Fatalf("plugin-a: expected 0 review actions, got %d", len(result[0].ReviewActions))
		}
		if result[0].Panels[0].ID != "panel-global" || result[0].Panels[1].ID != "panel-project" {
			t.Fatalf("plugin-a: panels not normalized and sorted: %#v", result[0].Panels)
		}
	})

	t.Run("workItem scope includes global + workItem contributions", func(t *testing.T) {
		result := AggregateUIContributions(plugins, UIContributionScope{WorkItemID: " wi_01 "})
		if len(result) != 1 {
			t.Fatalf("expected 1 plugin, got %d", len(result))
		}
		if len(result[0].Panels) != 1 || result[0].Panels[0].ID != "panel-global" {
			t.Fatalf("plugin-a: expected 1 global panel, got %d panels", len(result[0].Panels))
		}
		if len(result[0].Commands) != 1 || result[0].Commands[0].ID != "cmd-workitem" {
			t.Fatalf("plugin-a: expected 1 workItem command, got %d", len(result[0].Commands))
		}
	})

	t.Run("scope filters per-plugin contributions", func(t *testing.T) {
		result := AggregateUIContributions(plugins, UIContributionScope{GateReportID: "gate_01"})
		if len(result) != 1 {
			t.Fatalf("expected 1 plugin, got %d", len(result))
		}
		if len(result[0].Panels) != 1 || result[0].Panels[0].ID != "panel-global" {
			t.Fatalf("plugin-a: expected 1 global panel, got %d panels", len(result[0].Panels))
		}
		if len(result[0].Commands) != 0 {
			t.Fatalf("plugin-a: expected 0 commands, got %d", len(result[0].Commands))
		}
		if len(result[0].ReviewActions) != 0 {
			t.Fatalf("plugin-a: expected 0 review actions, got %d", len(result[0].ReviewActions))
		}
	})

	t.Run("disabled plugins do not expose renderable contributions", func(t *testing.T) {
		result := AggregateUIContributions(plugins, UIContributionScope{WorkItemID: "wi_01"})
		for _, plugin := range result {
			if plugin.PluginID == "plugin-b" {
				t.Fatalf("disabled plugin must be omitted from active contributions: %#v", plugin)
			}
			if !plugin.Trusted || !plugin.Enabled || plugin.DisabledReason != "" {
				t.Fatalf("active plugin should be enabled and trusted: %#v", plugin)
			}
		}
	})

	t.Run("phase-only and explicit empty entity scope return global contributions", func(t *testing.T) {
		result := BuildUIContributions(plugins, UIContributionScope{WorkItemID: " ", Phase: " review "})
		if result.Scope.WorkItemID != "" || result.Scope.Phase != "review" {
			t.Fatalf("scope = %#v", result.Scope)
		}
		if len(result.Plugins) != 1 || result.Plugins[0].PluginID != "plugin-a" {
			t.Fatalf("plugins = %#v", result.Plugins)
		}
		if len(result.Plugins[0].Panels) != 1 || result.Plugins[0].Panels[0].ID != "panel-global" {
			t.Fatalf("expected phase-only global panel, got %#v", result.Plugins[0].Panels)
		}
		if len(result.Plugins[0].Commands) != 0 || len(result.Plugins[0].ReviewActions) != 0 {
			t.Fatalf("expected phase-only to omit entity contributions, got commands=%#v actions=%#v", result.Plugins[0].Commands, result.Plugins[0].ReviewActions)
		}
	})

	t.Run("build read model normalizes echoed scope", func(t *testing.T) {
		result := BuildUIContributions(plugins, UIContributionScope{WorkItemID: " wi_01 ", Phase: " review "})
		if result.Scope.WorkItemID != "wi_01" || result.Scope.Phase != "review" {
			t.Fatalf("scope = %#v", result.Scope)
		}
		if len(result.Plugins) != 1 || len(result.Plugins[0].Commands) != 1 {
			t.Fatalf("plugins = %#v", result.Plugins)
		}
	})
}
