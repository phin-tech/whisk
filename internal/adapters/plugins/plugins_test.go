package plugins

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/agents"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/appsettings"
)

func TestReadManifestDefaultsV1AndAllowsAdHocFields(t *testing.T) {
	dir := t.TempDir()
	writeManifestOnly(t, dir, `{
		"id": "legacy",
		"name": "Legacy",
		"version": "0.1.0",
		"panels": [{"id": "legacy.panel"}],
		"agentProfiles": [{"id": "codex"}],
		"usageResolvers": [{"id": "legacy.usage", "provider": "legacy", "label": "Legacy", "command": "legacy"}],
		"events": [{"id": "legacy.event"}],
		"ui": {
			"reviewActions": [{
				"id": "legacy.review",
				"label": "Legacy Review",
				"urlTemplate": "https://example.test/review",
				"submitCommand": "node ./review.mjs",
				"blocking": true
			}]
		}
	}`)

	manifest, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if manifest.ManifestVersion != 1 {
		t.Fatalf("ManifestVersion = %d, want 1", manifest.ManifestVersion)
	}
	if len(manifest.Events) != 1 || manifest.Events[0].ID != "legacy.event" {
		t.Fatalf("events = %#v", manifest.Events)
	}
	if len(manifest.UI.ReviewActions) != 1 || !manifest.UI.ReviewActions[0].Blocking {
		t.Fatalf("review actions = %#v", manifest.UI.ReviewActions)
	}
	status := statusFromManifest("", dir, manifest, false)
	if len(status.ReviewActions) != 1 ||
		status.ReviewActions[0].ID != "legacy.review" ||
		status.ReviewActions[0].Scope != app.PluginUIScope(pluginUIScopeWorkItem) ||
		!status.ReviewActions[0].HasSubmit ||
		!status.ReviewActions[0].Blocking {
		t.Fatalf("legacy review action status = %#v", status.ReviewActions)
	}
	if len(manifest.AgentProfiles) != 0 {
		t.Fatalf("v1 agent profiles should be ignored, got %#v", manifest.AgentProfiles)
	}
	if len(manifest.UsageResolvers) != 0 {
		t.Fatalf("v1 usage resolvers should be ignored, got %#v", manifest.UsageResolvers)
	}
}

func TestReadManifestParsesAndNormalizesManifestV2(t *testing.T) {
	dir := t.TempDir()
	writeManifestOnly(t, dir, `{
		"manifestVersion": 2,
		"id": "linear",
		"name": "Linear",
		"version": "0.2.0",
		"agentProfiles": [{
			"id": "linear-agent",
			"provider": "linear",
			"label": "Linear Agent",
			"description": "Linear-aware agent CLI.",
			"command": "linear-agent",
			"args": ["--workspace", "acme"],
			"env": {"LINEAR_ENV": "test"},
			"detectCmd": "linear-agent",
			"detectAliases": ["linear"],
			"expectedProcess": "linear-agent",
			"promptInjectionMode": "argv",
			"draftPromptFlag": "--prompt",
			"hookProvider": "codex"
		}],
		"events": [{
			"id": "linear.sync-work",
			"subjects": ["workitem.stage.changed", "run.*"],
			"filter": {"entity.projectId": "proj_01", "data.requiresAttention": true},
			"command": "node ./on-event.mjs",
			"timeoutMs": 999999,
			"outputCapBytes": 999999999
		}],
		"hooks": [{
			"id": "linear.approval-policy",
			"point": "approval.evaluate",
			"command": "node ./policy.mjs"
		}],
		"gates": [{
			"id": "linear.issue-done",
			"label": "Linear issue done",
			"appliesTo": {"gateKinds": ["review"], "phases": ["review"]},
			"open": {"urlTemplate": "https://linear.app/acme/issue/{{work_item.number.url}}"},
			"resolve": {"command": "node ./resolve-gate.mjs"},
			"blocking": true
		}],
		"workflowActions": [{
			"id": "linear.sync",
			"label": "Sync Linear issue",
			"command": "node ./sync.mjs",
			"phases": ["planning"],
			"timeoutMs": 1,
			"outputCapBytes": 2
		}],
		"usageResolvers": [{
			"id": "linear.usage",
			"provider": "linear",
			"label": "Linear",
			"profiles": ["linear-agent", " linear-plan "],
			"command": "node ./usage.mjs",
			"timeoutMs": 999999,
			"outputCapBytes": 999999999,
			"minRefreshMs": 300000,
			"staleAfterMs": 1800000
		}],
		"permissions": {
			"ptyOutput": true,
			"envPrefixes": ["LINEAR_", "ACME_"],
			"network": ["api.linear.app"]
		}
	}`)

	manifest, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if manifest.ManifestVersion != 2 {
		t.Fatalf("ManifestVersion = %d, want 2", manifest.ManifestVersion)
	}
	if got := manifest.Events[0].TimeoutMs; got != manifestEventMaxTimeoutMs {
		t.Fatalf("event timeout = %d, want %d", got, manifestEventMaxTimeoutMs)
	}
	if got := manifest.Events[0].OutputCapBytes; got != manifestCommandMaxOutputCapBytes {
		t.Fatalf("event output cap = %d, want %d", got, manifestCommandMaxOutputCapBytes)
	}
	if got := manifest.Hooks[0].TimeoutMs; got != manifestHookDefaultTimeoutMs {
		t.Fatalf("hook timeout = %d, want %d", got, manifestHookDefaultTimeoutMs)
	}
	if got := manifest.Gates[0].TimeoutMs; got != manifestGateDefaultTimeoutMs {
		t.Fatalf("gate timeout = %d, want %d", got, manifestGateDefaultTimeoutMs)
	}
	if got := manifest.WorkflowActions[0].TimeoutMs; got != 1 {
		t.Fatalf("workflow action timeout = %d, want 1", got)
	}
	if got := manifest.WorkflowActions[0].OutputCapBytes; got != 2 {
		t.Fatalf("workflow action output cap = %d, want 2", got)
	}
	if len(manifest.UsageResolvers) != 1 {
		t.Fatalf("usage resolvers = %#v", manifest.UsageResolvers)
	}
	usageResolver := manifest.UsageResolvers[0]
	if usageResolver.ID != "linear.usage" ||
		usageResolver.Provider != "linear" ||
		usageResolver.Label != "Linear" ||
		usageResolver.Command != "node ./usage.mjs" ||
		len(usageResolver.Profiles) != 2 ||
		usageResolver.Profiles[1] != "linear-plan" ||
		usageResolver.TimeoutMs != manifestUsageResolverMaxTimeoutMs ||
		usageResolver.OutputCapBytes != manifestCommandMaxOutputCapBytes ||
		usageResolver.MinRefreshMs != 300000 ||
		usageResolver.StaleAfterMs != 1800000 {
		t.Fatalf("usage resolver = %#v", usageResolver)
	}
	if !manifest.Permissions.PTYOutput || len(manifest.Permissions.EnvPrefixes) != 2 || len(manifest.Permissions.Network) != 1 {
		t.Fatalf("permissions = %#v", manifest.Permissions)
	}
	if len(manifest.AgentProfiles) != 1 {
		t.Fatalf("agent profiles = %#v", manifest.AgentProfiles)
	}
	profile := manifest.AgentProfiles[0]
	if profile.ID != "linear-agent" ||
		profile.Provider != "linear" ||
		profile.Label != "Linear Agent" ||
		profile.Command != "linear-agent" ||
		len(profile.Args) != 2 ||
		profile.Env["LINEAR_ENV"] != "test" ||
		profile.PromptInjectionMode != agents.PromptInjectionArgv ||
		profile.HookProvider != "codex" {
		t.Fatalf("agent profile = %#v", profile)
	}
}

func TestReadManifestParsesAndNormalizesV2UIContributions(t *testing.T) {
	dir := t.TempDir()
	writeManifestOnly(t, dir, `{
		"manifestVersion": 2,
		"id": "linear",
		"name": "Linear",
		"version": "0.2.0",
		"ui": {
			"projectAttachments": [{
				"id": "linear.issue.attach",
				"label": "Attach Linear issue",
				"provider": "linear",
				"kind": "external",
				"command": "node ./attach.mjs",
				"fields": [{"id": "issue", "label": "Issue", "type": "text", "options": ["LIN-1"]}]
			}],
			"reviewActions": [{
				"id": "linear.review",
				"label": "Linear review",
				"scope": "workItem",
				"urlTemplate": "https://linear.app/acme/issue/{{work_item.id.url}}",
				"submitCommand": "node ./fetch-review.mjs",
				"blocking": true,
				"timeoutMs": 999999,
				"outputCapBytes": 999999999
			}],
			"panels": [
				{
					"id": "linear.issue.panel",
					"title": "Linear issue",
					"scope": "workItem",
					"read": {
						"command": "node ./render-issue.mjs",
						"timeoutMs": 999999,
						"outputCapBytes": 999999999
					},
					"actions": [{
						"id": "sync",
						"label": "Sync",
						"command": "node ./sync.mjs",
						"timeoutMs": 1,
						"outputCapBytes": 2
					}]
				},
				{
					"id": "linear.board.panel",
					"title": "Linear board",
					"scope": "project",
					"kind": "html",
					"entry": "./panel/"
				}
			],
			"commands": [{
				"id": "linear.open-triage",
				"label": "Linear: Open triage",
				"scope": "global",
				"command": "node ./triage.mjs"
			}]
		}
	}`)

	manifest, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if got := manifest.UI.ProjectAttachments[0].Fields[0].Options[0]; got != "LIN-1" {
		t.Fatalf("project attachment field option = %q", got)
	}
	reviewAction := manifest.UI.ReviewActions[0]
	if reviewAction.TimeoutMs != manifestUICommandMaxTimeoutMs ||
		reviewAction.OutputCapBytes != manifestCommandMaxOutputCapBytes {
		t.Fatalf("review action limits = %#v", reviewAction)
	}
	viewPanel := manifest.UI.Panels[0]
	if viewPanel.Kind != pluginUIPanelKindView ||
		viewPanel.Read.TimeoutMs != manifestUICommandMaxTimeoutMs ||
		viewPanel.Read.OutputCapBytes != manifestCommandMaxOutputCapBytes ||
		viewPanel.Actions[0].TimeoutMs != 1 ||
		viewPanel.Actions[0].OutputCapBytes != 2 {
		t.Fatalf("view panel = %#v", viewPanel)
	}
	htmlPanel := manifest.UI.Panels[1]
	if htmlPanel.Entry.Path != "./panel/" || htmlPanel.Entry.Forward != "" {
		t.Fatalf("html panel entry = %#v", htmlPanel.Entry)
	}
	command := manifest.UI.Commands[0]
	if command.TimeoutMs != manifestUICommandDefaultTimeoutMs ||
		command.OutputCapBytes != manifestCommandDefaultOutputCapBytes {
		t.Fatalf("ui command = %#v", command)
	}
}

func TestReadManifestRejectsUnsupportedFutureManifestVersion(t *testing.T) {
	dir := t.TempDir()
	writeManifestOnly(t, dir, `{"manifestVersion":3,"id":"future"}`)

	_, err := ReadManifest(dir)
	if err == nil || !strings.Contains(err.Error(), "unsupported manifestVersion 3") {
		t.Fatalf("ReadManifest error = %v", err)
	}
}

func TestReadManifestRejectsUnknownV2TopLevelField(t *testing.T) {
	dir := t.TempDir()
	writeManifestOnly(t, dir, `{"manifestVersion":2,"id":"future-ui","panels":[]}`)

	_, err := ReadManifest(dir)
	if err == nil || !strings.Contains(err.Error(), `unsupported top-level field "panels"`) {
		t.Fatalf("ReadManifest error = %v", err)
	}
}

func TestReadManifestRejectsInvalidV2Contributions(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		want     string
	}{
		{
			name: "agent profile duplicate id",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"agentProfiles": [
					{"id": "agent", "provider": "gemini", "label": "Gemini", "command": "gemini"},
					{"id": "agent", "provider": "gemini", "label": "Gemini", "command": "gemini"}
				]
			}`,
			want: `duplicate manifest contribution id "agent"`,
		},
		{
			name: "agent profile command required",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"agentProfiles": [{"id": "agent", "provider": "gemini", "label": "Gemini"}]
			}`,
			want: "agentProfiles[agent].command required",
		},
		{
			name: "agent profile id cannot contain slash",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"agentProfiles": [{"id": "gemini/plan", "provider": "gemini", "label": "Gemini", "command": "gemini"}]
			}`,
			want: "agentProfiles[gemini/plan].id must not contain /",
		},
		{
			name: "agent profile cannot shadow builtin",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"agentProfiles": [{"id": "codex", "provider": "gemini", "label": "Gemini", "command": "gemini"}]
			}`,
			want: "agentProfiles[codex].id shadows builtin agent profile",
		},
		{
			name: "agent profile label required",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"agentProfiles": [{"id": "agent", "provider": "gemini", "label": "  ", "command": "gemini"}]
			}`,
			want: "agentProfiles[agent].label required",
		},
		{
			name: "agent profile prompt mode must be known",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"agentProfiles": [{"id": "agent", "provider": "gemini", "label": "Gemini", "command": "gemini", "promptInjectionMode": "telepathy"}]
			}`,
			want: "promptInjectionMode",
		},
		{
			name: "agent profile alias cannot be empty",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"agentProfiles": [{"id": "agent", "provider": "gemini", "label": "Gemini", "command": "gemini", "detectAliases": ["gemini", " "]}]
			}`,
			want: "agentProfiles[agent].detectAliases contains empty value",
		},
		{
			name: "agent profile env key cannot be empty",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"agentProfiles": [{"id": "agent", "provider": "gemini", "label": "Gemini", "command": "gemini", "env": {"": "bad"}}]
			}`,
			want: "agentProfiles[agent].env contains empty key",
		},
		{
			name: "event command required",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"events": [{"id": "bad.event", "subjects": ["workitem.updated"]}]
			}`,
			want: "events[bad.event].command required",
		},
		{
			name: "event wildcard must trail",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"events": [{"id": "bad.event", "subjects": ["work*item.updated"], "command": "true"}]
			}`,
			want: "unsupported wildcard",
		},
		{
			name: "duplicate contribution id",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"events": [{"id": "dup", "subjects": ["workitem.updated"], "command": "true"}],
				"hooks": [{"id": "dup", "point": "approval.evaluate", "command": "true"}]
			}`,
			want: `duplicate manifest contribution id "dup"`,
		},
		{
			name: "usage resolver duplicate id",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"usageResolvers": [
					{"id": "usage", "provider": "codex", "label": "Codex", "command": "true"},
					{"id": "usage", "provider": "claude", "label": "Claude", "command": "true"}
				]
			}`,
			want: `duplicate manifest contribution id "usage"`,
		},
		{
			name: "usage resolver provider required",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"usageResolvers": [{"id": "usage", "label": "Codex", "command": "true"}]
			}`,
			want: "usageResolvers[usage].provider required",
		},
		{
			name: "usage resolver label required",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"usageResolvers": [{"id": "usage", "provider": "codex", "label": " ", "command": "true"}]
			}`,
			want: "usageResolvers[usage].label required",
		},
		{
			name: "usage resolver command required",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"usageResolvers": [{"id": "usage", "provider": "codex", "label": "Codex"}]
			}`,
			want: "usageResolvers[usage].command required",
		},
		{
			name: "usage resolver profile cannot be empty",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"usageResolvers": [{"id": "usage", "provider": "codex", "label": "Codex", "command": "true", "profiles": ["codex", " "]}]
			}`,
			want: "usageResolvers[usage].profiles[1] required",
		},
		{
			name: "usage resolver profile cannot duplicate",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"usageResolvers": [{"id": "usage", "provider": "codex", "label": "Codex", "command": "true", "profiles": ["codex", " codex "]}]
			}`,
			want: `usageResolvers[usage].profiles contains duplicate value "codex"`,
		},
		{
			name: "usage resolver negative output cap",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"usageResolvers": [{"id": "usage", "provider": "codex", "label": "Codex", "command": "true", "outputCapBytes": -1}]
			}`,
			want: "outputCapBytes must be non-negative",
		},
		{
			name: "usage resolver negative refresh",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"usageResolvers": [{"id": "usage", "provider": "codex", "label": "Codex", "command": "true", "minRefreshMs": -1}]
			}`,
			want: "usageResolvers[usage].minRefreshMs must be non-negative",
		},
		{
			name: "negative timeout",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"hooks": [{"id": "bad.hook", "point": "approval.evaluate", "command": "true", "timeoutMs": -1}]
			}`,
			want: "timeoutMs must be non-negative",
		},
		{
			name: "duplicate permission",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"permissions": {"envPrefixes": ["LINEAR_", "LINEAR_"]}
			}`,
			want: `permissions.envPrefixes contains duplicate value "LINEAR_"`,
		},
		{
			name: "project attachment label required",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"ui": {"projectAttachments": [{"id": "attach", "provider": "github", "kind": "external", "command": "true"}]}
			}`,
			want: "ui.projectAttachments[attach].label required",
		},
		{
			name: "project attachment duplicate field id",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"ui": {"projectAttachments": [{
					"id": "attach",
					"label": "Attach",
					"provider": "github",
					"kind": "external",
					"command": "true",
					"fields": [
						{"id": "url", "label": "URL", "type": "text"},
						{"id": " url ", "label": "URL", "type": "text"}
					]
				}]}
			}`,
			want: `ui.projectAttachments[attach].fields contains duplicate id "url"`,
		},
		{
			name: "review action url or submit required",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"ui": {"reviewActions": [{"id": "review", "label": "Review", "scope": "workItem"}]}
			}`,
			want: "ui.reviewActions[review].urlTemplate or submitCommand required",
		},
		{
			name: "review action scope required to be known",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"ui": {"reviewActions": [{"id": "review", "label": "Review", "scope": "planet", "urlTemplate": "https://example.test"}]}
			}`,
			want: `ui.reviewActions[review].scope "planet" is unsupported`,
		},
		{
			name: "ui panel read command required",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"ui": {"panels": [{"id": "panel", "title": "Panel", "scope": "workItem"}]}
			}`,
			want: "ui.panels[panel].read.command required",
		},
		{
			name: "ui panel unsupported kind",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"ui": {"panels": [{"id": "panel", "title": "Panel", "scope": "workItem", "kind": "native", "read": {"command": "true"}}]}
			}`,
			want: `ui.panels[panel].kind "native" is unsupported`,
		},
		{
			name: "ui html panel entry required",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"ui": {"panels": [{"id": "panel", "title": "Panel", "scope": "workItem", "kind": "html"}]}
			}`,
			want: "ui.panels[panel].entry.path or entry.forward required",
		},
		{
			name: "ui panel action duplicate id",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"ui": {"panels": [{
					"id": "panel",
					"title": "Panel",
					"scope": "workItem",
					"read": {"command": "true"},
					"actions": [
						{"id": "sync", "label": "Sync", "command": "true"},
						{"id": " sync ", "label": "Sync", "command": "true"}
					]
				}]}
			}`,
			want: `ui.panels[panel].actions contains duplicate id "sync"`,
		},
		{
			name: "ui command scope required",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"ui": {"commands": [{"id": "cmd", "label": "Command", "command": "true"}]}
			}`,
			want: "ui.commands[cmd].scope required",
		},
		{
			name: "duplicate ui contribution id",
			manifest: `{
				"manifestVersion": 2,
				"id": "bad",
				"ui": {
					"panels": [{"id": "dup", "title": "Panel", "scope": "workItem", "read": {"command": "true"}}],
					"commands": [{"id": "dup", "label": "Command", "scope": "global", "command": "true"}]
				}
			}`,
			want: `duplicate manifest contribution id "dup"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeManifestOnly(t, dir, tt.manifest)
			_, err := ReadManifest(dir)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ReadManifest error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestScanTrustedResolversRunsCommandResolver(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(`{
		"id": "github",
		"name": "GitHub",
		"version": "0.1.0",
		"resolvers": [{"provider": "github", "kinds": ["external"], "command": "printf '{\"delivery\":\"inline\",\"contentType\":\"text/markdown\",\"content\":\"ok\"}'"}]
	}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	resolvers, err := ScanTrustedResolvers([]string{dir}, map[string]bool{"github": true})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	resolver := resolvers["github"]
	if resolver == nil {
		t.Fatalf("resolver missing")
	}
	resolved, err := resolver.ResolveProjectAttachment(context.Background(), app.ResolveProjectAttachmentRequest{Provider: "github", Target: "owner/repo#1"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Delivery != "inline" || resolved.Content != "ok" {
		t.Fatalf("resolved = %#v", resolved)
	}

	resolvers, err = ScanTrustedResolvers([]string{dir}, nil)
	if err != nil {
		t.Fatalf("scan untrusted: %v", err)
	}
	if resolvers["github"] != nil {
		t.Fatalf("untrusted resolver registered")
	}
}

func TestManagerListsPluginAgentProfilesWithoutRegisteringLaunch(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	configPlugin := filepath.Join(configHome, "whisk", "plugins", "phin-tech", "gemini")
	writeManifestOnly(t, configPlugin, `{
		"manifestVersion": 2,
		"id": "gemini",
		"name": "Gemini",
		"version": "0.1.0",
		"agentProfiles": [{
			"id": "gemini-cli",
			"provider": "gemini",
			"label": "Gemini CLI",
			"description": "Gemini CLI from a plugin.",
			"command": "gemini",
			"detectCmd": "gemini",
			"promptInjectionMode": "argv"
		}]
	}`)

	store := appsettings.NewStore(filepath.Join(t.TempDir(), "whisk.json"))
	manager, err := NewManager(nil, store)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	profiles, err := manager.ListAgentProfiles(context.Background())
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("profiles = %#v", profiles)
	}
	profile := profiles[0]
	if profile.ID != "plugin:phin-tech/gemini/gemini-cli" ||
		profile.Source != agents.ProfileSourcePlugin ||
		profile.PluginID != "phin-tech/gemini" ||
		profile.Provider != "gemini" ||
		profile.Label != "Gemini CLI" ||
		profile.DetectCmd != "gemini" ||
		profile.Launchable ||
		!strings.Contains(profile.LaunchBlockedReason, "not trusted") {
		t.Fatalf("untrusted profile = %#v", profile)
	}

	if _, err := manager.TrustPlugin(context.Background(), "phin-tech/gemini"); err != nil {
		t.Fatalf("trust: %v", err)
	}
	profiles, err = manager.ListAgentProfiles(context.Background())
	if err != nil {
		t.Fatalf("list trusted profiles: %v", err)
	}
	if len(profiles) != 1 || profiles[0].Launchable || !strings.Contains(profiles[0].LaunchBlockedReason, "not implemented") {
		t.Fatalf("trusted foundation profile = %#v", profiles)
	}
}

func TestManagerScansEnvAndConfigPluginsAndTrustsLive(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	configPlugin := filepath.Join(configHome, "whisk", "plugins", "github")
	writePlugin(t, configPlugin, `{
		"manifestVersion": 2,
		"id": "github",
		"name": "GitHub Issues",
		"version": "0.1.0",
		"resolvers": [{"provider": "github", "kinds": ["external"], "command": "printf '{\"delivery\":\"inline\",\"content\":\"ok\"}'"}],
		"usageResolvers": [{
			"id": "github.usage",
			"provider": "github",
			"label": "GitHub",
			"profiles": ["codex"],
			"command": "printf '{\"summary\":\"ok\",\"metrics\":[{\"id\":\"requests\",\"kind\":\"rateLimit\",\"used\":1,\"limit\":10,\"remaining\":9}]}'"
		}],
		"permissions": {"network": ["api.github.com"]},
		"ui": {
			"projectAttachments": [{
				"id": "github.issue",
				"label": "GitHub Issue",
				"provider": "github",
				"kind": "external",
				"command": "printf '{\"kind\":\"external\",\"provider\":\"github\",\"target\":\"owner/repo#1\",\"url\":\"https://github.com/owner/repo/issues/1\",\"title\":\"Issue\",\"includeInContext\":true}'",
				"fields": [{"id":"url","label":"Issue URL","type":"text","required":true}]
			}],
			"reviewActions": [{
				"id": "github.review",
				"label": "GitHub Review",
				"scope": "workItem",
				"urlTemplate": "https://github.com/{{project.id.url}}",
				"submitCommand": "node ./fetch-review.mjs",
				"blocking": true
			}],
			"panels": [{
				"id": "github.issue.panel",
				"title": "GitHub Issue",
				"scope": "workItem",
				"read": {"command": "node ./render.mjs"}
			}],
			"commands": [{
				"id": "github.open",
				"label": "GitHub: Open",
				"scope": "global",
				"command": "node ./open.mjs"
			}]
		}
	}`)
	envPlugin := filepath.Join(t.TempDir(), "docs")
	writePlugin(t, envPlugin, `{"id":"docs","name":"Docs","version":"0.1.0"}`)

	store := appsettings.NewStore(filepath.Join(t.TempDir(), "whisk.json"))
	manager, err := NewManager([]string{envPlugin}, store)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	statuses, err := manager.ListPlugins(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("statuses = %#v", statuses)
	}
	var githubStatus app.PluginStatus
	for _, status := range statuses {
		if status.ID == "github" {
			githubStatus = status
		}
	}
	if len(githubStatus.UsageResolvers) != 1 ||
		githubStatus.UsageResolvers[0].ID != "github.usage" ||
		githubStatus.UsageResolvers[0].Provider != "github" ||
		githubStatus.UsageResolvers[0].Label != "GitHub" ||
		len(githubStatus.UsageResolvers[0].Profiles) != 1 ||
		githubStatus.UsageResolvers[0].Profiles[0] != "codex" ||
		githubStatus.UsageResolvers[0].TimeoutMs != manifestUsageResolverDefaultTimeoutMs ||
		githubStatus.UsageResolvers[0].OutputCapBytes != manifestCommandDefaultOutputCapBytes {
		t.Fatalf("github usage resolvers = %#v", githubStatus.UsageResolvers)
	}
	if resolver := manager.ResolveProjectAttachmentProvider("github"); resolver != nil {
		t.Fatalf("untrusted resolver registered")
	}
	if githubStatus.Trusted {
		t.Fatalf("github plugin should start untrusted")
	}
	if len(githubStatus.ProjectAttachmentTemplates) != 1 ||
		len(githubStatus.UIPanels) != 1 ||
		len(githubStatus.UICommands) != 1 ||
		len(githubStatus.ReviewActions) != 1 {
		t.Fatalf("untrusted catalog contributions = templates:%#v panels:%#v commands:%#v reviews:%#v",
			githubStatus.ProjectAttachmentTemplates, githubStatus.UIPanels, githubStatus.UICommands, githubStatus.ReviewActions)
	}
	if githubStatus.UIPanels[0].ID != "github.issue.panel" ||
		githubStatus.UIPanels[0].Kind != pluginUIPanelKindView ||
		githubStatus.UIPanels[0].Read == nil ||
		githubStatus.UICommands[0].ID != "github.open" ||
		!githubStatus.ReviewActions[0].HasSubmit ||
		!githubStatus.ReviewActions[0].Blocking ||
		githubStatus.Permissions == nil ||
		len(githubStatus.Permissions.Network) != 1 ||
		githubStatus.Permissions.Network[0] != "api.github.com" {
		t.Fatalf("untrusted github status = %#v", githubStatus)
	}
	if _, err := manager.TrustPlugin(context.Background(), "github"); err != nil {
		t.Fatalf("trust: %v", err)
	}
	if resolver := manager.ResolveProjectAttachmentProvider("github"); resolver == nil {
		t.Fatalf("trusted resolver missing")
	}
	usage, err := manager.RunUsageResolver(context.Background(), app.RunUsageResolverRequest{
		PluginID:   "github",
		ResolverID: "github.usage",
		Profile:    "codex",
	})
	if err != nil {
		t.Fatalf("run usage resolver: %v", err)
	}
	if usage.Summary != "ok" ||
		len(usage.Metrics) != 1 ||
		usage.Metrics[0].ID != "requests" ||
		usage.Metrics[0].Kind != app.UsageMetricKindRateLimit ||
		usage.Metrics[0].Remaining == nil ||
		*usage.Metrics[0].Remaining != 9 {
		t.Fatalf("usage = %#v", usage)
	}
	created, err := manager.RunProjectAttachmentTemplate(context.Background(), app.RunPluginProjectAttachmentTemplateRequest{
		PluginID:   "github",
		TemplateID: "github.issue",
		ProjectID:  "proj_01",
		Values:     map[string]string{"url": "https://github.com/owner/repo/issues/1"},
	})
	if err != nil {
		t.Fatalf("run template: %v", err)
	}
	if created.Provider != "github" || created.Target != "owner/repo#1" || created.URL == "" || !created.IncludeInContext {
		t.Fatalf("created = %#v", created)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if len(loaded.TrustedPlugins) != 1 || loaded.TrustedPlugins[0] != "github" {
		t.Fatalf("trusted plugins = %#v", loaded.TrustedPlugins)
	}
}

func TestManagerRunTemplateRejectsUntrustedPlugin(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, `{
		"id": "github",
		"name": "GitHub",
		"version": "0.1.0",
		"ui": {"projectAttachments": [{"id": "github.issue", "label": "GitHub Issue", "provider":"github", "kind":"external", "command":"printf '{}'"}]}
	}`)
	manager, err := NewManager([]string{dir}, appsettings.NewStore(filepath.Join(t.TempDir(), "whisk.json")))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if _, err := manager.RunProjectAttachmentTemplate(context.Background(), app.RunPluginProjectAttachmentTemplateRequest{PluginID: "github", TemplateID: "github.issue"}); err == nil {
		t.Fatalf("expected untrusted template error")
	}
}

func TestManagerRunUsageResolverRejectsUntrustedPlugin(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, `{
		"manifestVersion": 2,
		"id": "github",
		"name": "GitHub",
		"version": "0.1.0",
		"usageResolvers": [{
			"id": "github.usage",
			"provider": "github",
			"label": "GitHub",
			"command": "printf '{\"metrics\":[{\"id\":\"requests\",\"used\":1}]}'"
		}]
	}`)
	manager, err := NewManager([]string{dir}, appsettings.NewStore(filepath.Join(t.TempDir(), "whisk.json")))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	_, err = manager.RunUsageResolver(context.Background(), app.RunUsageResolverRequest{PluginID: "github", ResolverID: "github.usage"})
	if err == nil || !strings.Contains(err.Error(), "not trusted") {
		t.Fatalf("expected untrusted usage resolver error, got %v", err)
	}
}

func TestManagerRunUsageResolverHonorsOutputCap(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, `{
		"manifestVersion": 2,
		"id": "github",
		"name": "GitHub",
		"version": "0.1.0",
		"usageResolvers": [{
			"id": "github.usage",
			"provider": "github",
			"label": "GitHub",
			"command": "printf '{\"metrics\":[{\"id\":\"requests\",\"used\":1}]}'",
			"outputCapBytes": 2
		}]
	}`)
	store := appsettings.NewStore(filepath.Join(t.TempDir(), "whisk.json"))
	manager, err := NewManager([]string{dir}, store)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if _, err := manager.TrustPlugin(context.Background(), "github"); err != nil {
		t.Fatalf("trust: %v", err)
	}
	_, err = manager.RunUsageResolver(context.Background(), app.RunUsageResolverRequest{PluginID: "github", ResolverID: "github.usage"})
	if err == nil || !strings.Contains(err.Error(), "stdout exceeded 2 bytes") {
		t.Fatalf("expected output cap error, got %v", err)
	}
}

func writePlugin(t *testing.T, dir string, manifest string) {
	t.Helper()
	writeManifestOnly(t, dir, manifest)
	if runtime.GOOS == "windows" {
		t.Skip("shell command quoting in this test is POSIX-only")
	}
}

func writeManifestOnly(t *testing.T, dir string, manifest string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}
