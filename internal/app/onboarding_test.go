package app

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/agenthooks"
	"github.com/phin-tech/whisk/internal/domain/onboarding"
)

func TestApplyOnboardingInstallsSelectedSkillAndRecordsSkips(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex", "skills"), 0o755); err != nil {
		t.Fatalf("mkdir codex skills: %v", err)
	}
	source := filepath.Join(t.TempDir(), "whisk")
	writeFile(t, filepath.Join(source, "SKILL.md"), "---\nname: whisk\nversion: \"1\"\n---\n")
	writeFile(t, filepath.Join(source, "README.md"), "# Whisk\n")

	runtime := NewRuntime(RuntimeConfig{
		DaemonURL:           "http://127.0.0.1:8787",
		DaemonAPIVersion:    18,
		OnboardingSkillDir:  source,
		OnboardingStatePath: filepath.Join(t.TempDir(), "onboarding.json"),
		AgentHookPaths: &agenthooks.Paths{
			ConfigRoot:         filepath.Join(t.TempDir(), "whisk"),
			HelperSourcePath:   os.Args[0],
			ClaudeSettingsPath: filepath.Join(t.TempDir(), "claude.json"),
			CodexHooksPath:     filepath.Join(t.TempDir(), "codex.json"),
		},
	})

	status, err := runtime.OnboardingStatus(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !status.ShouldShow {
		t.Fatalf("first-run onboarding did not show")
	}

	status, err = runtime.ApplyOnboarding(context.Background(), OnboardingApplyRequest{ItemIDs: []string{"skill:codex"}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if status.ShouldShow {
		t.Fatalf("onboarding still shows after selected apply and skipped rest")
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "skills", "whisk", "SKILL.md")); err != nil {
		t.Fatalf("installed skill: %v", err)
	}
}

func TestBundledSkillDirsPreferResources(t *testing.T) {
	got := bundledSkillDirs("/Applications/Whisk.app/Contents/MacOS/whisk-app")
	want := []string{
		"/Applications/Whisk.app/Contents/Resources/skills/whisk",
		"/Applications/Whisk.app/Contents/MacOS/skills/whisk",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("bundledSkillDirs = %#v, want %#v", got, want)
	}
}

func TestOnboardingCoversPluginHookAndRemoteDaemonBranches(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex", "skills"), 0o755); err != nil {
		t.Fatalf("mkdir codex skills: %v", err)
	}
	source := filepath.Join(t.TempDir(), "whisk")
	writeFile(t, filepath.Join(source, "SKILL.md"), "---\nname: whisk\nversion: \"2\"\n---\n")
	writeFile(t, filepath.Join(source, "README.md"), "# Whisk\n")
	hookRoot := t.TempDir()
	plugins := &onboardingPluginRegistry{
		statuses: []PluginStatus{
			{ID: "github", Version: "1.0.0", Dir: filepath.Join(t.TempDir(), "github"), Trusted: false, Valid: true},
			{ID: "broken", Name: "Broken Plugin", Version: "0.1.0", Trusted: false, Valid: false, Error: "bad manifest"},
		},
	}
	runtime := NewRuntime(RuntimeConfig{
		DaemonURL:           "http://localhost:8787",
		DaemonAPIVersion:    19,
		OnboardingSkillDir:  source,
		OnboardingStatePath: filepath.Join(t.TempDir(), "onboarding.json"),
		Plugins:             plugins,
		AgentHookPaths: &agenthooks.Paths{
			ConfigRoot:         filepath.Join(hookRoot, "whisk"),
			HelperSourcePath:   os.Args[0],
			ClaudeSettingsPath: filepath.Join(hookRoot, "claude", "settings.json"),
			CodexHooksPath:     filepath.Join(hookRoot, "codex", "hooks.json"),
		},
	})

	status, err := runtime.OnboardingStatus(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	seenPlugin, seenBroken, seenHook := false, false, false
	for _, item := range status.Items {
		switch item.ID {
		case "plugin:github":
			seenPlugin = item.Label == "github" && item.Status == onboarding.StatusUntrusted
		case "plugin:broken":
			seenBroken = item.Label == "Broken Plugin" && item.Status == onboarding.StatusUnavailable && item.Detail == "bad manifest"
		case "hook:claude":
			seenHook = item.Target == "claude"
		}
	}
	if !status.LocalDaemon || !seenPlugin || !seenBroken || !seenHook {
		t.Fatalf("status = %#v", status)
	}

	status, err = runtime.ApplyOnboarding(context.Background(), OnboardingApplyRequest{ItemIDs: []string{"plugin:github", "hook:claude", "daemon:version"}})
	if err != nil {
		t.Fatalf("apply local onboarding: %v", err)
	}
	if plugins.trustedID != "github" {
		t.Fatalf("trusted plugin = %q", plugins.trustedID)
	}
	if status.StatePath == "" {
		t.Fatalf("missing state path")
	}

	remote := NewRuntime(RuntimeConfig{
		DaemonURL:           "https://whisk.example.test",
		OnboardingSkillDir:  source,
		OnboardingStatePath: filepath.Join(t.TempDir(), "onboarding.json"),
		Plugins:             plugins,
		AgentHookPaths: &agenthooks.Paths{
			ConfigRoot:         filepath.Join(t.TempDir(), "whisk"),
			HelperSourcePath:   os.Args[0],
			ClaudeSettingsPath: filepath.Join(t.TempDir(), "claude.json"),
			CodexHooksPath:     filepath.Join(t.TempDir(), "codex.json"),
		},
	})
	remoteStatus, err := remote.OnboardingStatus(context.Background())
	if err != nil {
		t.Fatalf("remote status: %v", err)
	}
	if remoteStatus.LocalDaemon {
		t.Fatalf("remote daemon reported local")
	}
	if _, err := remote.ApplyOnboarding(context.Background(), OnboardingApplyRequest{ItemIDs: []string{"plugin:github"}}); err == nil {
		t.Fatalf("expected remote apply error")
	}
}

type onboardingPluginRegistry struct {
	statuses  []PluginStatus
	trustedID string
}

func (r *onboardingPluginRegistry) ListPlugins(context.Context) ([]PluginStatus, error) {
	return append([]PluginStatus(nil), r.statuses...), nil
}

func (r *onboardingPluginRegistry) RescanPlugins(context.Context) ([]PluginStatus, error) {
	return append([]PluginStatus(nil), r.statuses...), nil
}

func (r *onboardingPluginRegistry) TrustPlugin(_ context.Context, id string) (PluginStatus, error) {
	r.trustedID = id
	for _, status := range r.statuses {
		if status.ID == id {
			status.Trusted = true
			return status, nil
		}
	}
	return PluginStatus{ID: id, Trusted: true, Valid: true}, nil
}

func (r *onboardingPluginRegistry) UntrustPlugin(_ context.Context, id string) (PluginStatus, error) {
	return PluginStatus{ID: id, Trusted: false, Valid: true}, nil
}

func (r *onboardingPluginRegistry) ListRegistryPlugins(context.Context) ([]RegistryPlugin, error) {
	return nil, nil
}

func (r *onboardingPluginRegistry) InstallPlugin(context.Context, string, string) (PluginStatus, error) {
	return PluginStatus{}, nil
}

func (r *onboardingPluginRegistry) RunProjectAttachmentTemplate(context.Context, RunPluginProjectAttachmentTemplateRequest) (AddProjectAttachmentRequest, error) {
	return AddProjectAttachmentRequest{}, nil
}

func (r *onboardingPluginRegistry) ResolveProjectAttachmentProvider(string) ProjectContextResolver {
	return nil
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
