package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/agenthooks"
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

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
