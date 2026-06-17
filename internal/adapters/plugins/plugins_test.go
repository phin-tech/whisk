package plugins

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/appsettings"
)

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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("shell command quoting in this test is POSIX-only")
	}
}
