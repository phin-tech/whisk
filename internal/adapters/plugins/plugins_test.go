package plugins

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
}

func TestReadManifestParsesAndNormalizesManifestV2(t *testing.T) {
	dir := t.TempDir()
	writeManifestOnly(t, dir, `{
		"manifestVersion": 2,
		"id": "linear",
		"name": "Linear",
		"version": "0.2.0",
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
	if !manifest.Permissions.PTYOutput || len(manifest.Permissions.EnvPrefixes) != 2 || len(manifest.Permissions.Network) != 1 {
		t.Fatalf("permissions = %#v", manifest.Permissions)
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

func TestManagerScansEnvAndConfigPluginsAndTrustsLive(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	configPlugin := filepath.Join(configHome, "whisk", "plugins", "github")
	writePlugin(t, configPlugin, `{
		"id": "github",
		"name": "GitHub Issues",
		"version": "0.1.0",
		"resolvers": [{"provider": "github", "kinds": ["external"], "command": "printf '{\"delivery\":\"inline\",\"content\":\"ok\"}'"}],
		"ui": {"projectAttachments": [{
			"id": "github.issue",
			"label": "GitHub Issue",
			"provider": "github",
			"kind": "external",
			"command": "printf '{\"kind\":\"external\",\"provider\":\"github\",\"target\":\"owner/repo#1\",\"url\":\"https://github.com/owner/repo/issues/1\",\"title\":\"Issue\",\"includeInContext\":true}'",
			"fields": [{"id":"url","label":"Issue URL","type":"text","required":true}]
		}]}
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
	if resolver := manager.ResolveProjectAttachmentProvider("github"); resolver != nil {
		t.Fatalf("untrusted resolver registered")
	}
	if _, err := manager.TrustPlugin(context.Background(), "github"); err != nil {
		t.Fatalf("trust: %v", err)
	}
	if resolver := manager.ResolveProjectAttachmentProvider("github"); resolver == nil {
		t.Fatalf("trusted resolver missing")
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
