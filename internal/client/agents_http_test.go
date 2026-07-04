package client_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func newAgentTestDaemon(t *testing.T) *client.HTTPClient {
	t.Helper()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)
	return client.NewHTTP(httpServer.URL, httpServer.Client())
}

func TestHTTPClientListsAgentProfiles(t *testing.T) {
	daemon := newAgentTestDaemon(t)

	profiles, err := daemon.ListAgentProfiles(context.Background())
	if err != nil {
		t.Fatalf("list agent profiles: %v", err)
	}
	if len(profiles) == 0 {
		t.Fatalf("expected builtin agent profiles, got none")
	}

	byID := map[string]protocol.AgentProfile{}
	for _, p := range profiles {
		byID[p.ID] = p
	}
	claude, ok := byID["claude"]
	if !ok ||
		claude.Label == "" ||
		claude.Provider != "claude" ||
		claude.DetectCmd != "claude" ||
		claude.ExpectedProcess != "claude" ||
		claude.PromptInjectionMode != "argv" ||
		claude.DraftPromptFlag != "--prefill" {
		t.Fatalf("claude profile = %#v (ok=%v)", claude, ok)
	}
	codex, ok := byID["codex"]
	if !ok ||
		codex.DetectCmd != "codex" ||
		codex.PromptInjectionMode != "argv" ||
		codex.PreflightTrust != "codex" ||
		codex.ReadySignal != "codex-composer-prompt" {
		t.Fatalf("codex profile = %#v (ok=%v), all = %#v", codex, ok, byID)
	}
}

func TestHTTPClientListsDetectedAgents(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "codex"))
	t.Setenv("PATH", binDir)
	daemon := newAgentTestDaemon(t)

	detected, err := daemon.ListDetectedAgents(context.Background())
	if err != nil {
		t.Fatalf("list detected agents: %v", err)
	}
	byID := map[string]protocol.DetectedAgent{}
	for _, agent := range detected {
		byID[agent.ProfileID] = agent
	}
	codex, ok := byID["codex"]
	if !ok ||
		codex.Provider != "codex" ||
		codex.Label == "" ||
		codex.DetectCommand != "codex" ||
		codex.Path != filepath.Join(binDir, "codex") {
		t.Fatalf("codex detection = %#v (ok=%v), all = %#v", codex, ok, detected)
	}
	if _, ok := byID["codex-plan"]; !ok {
		t.Fatalf("expected codex-plan detection, got %#v", detected)
	}
	if _, ok := byID["claude"]; ok {
		t.Fatalf("unexpected claude detection = %#v", detected)
	}
}

func TestHTTPClientPersistsDefaultPhaseAgents(t *testing.T) {
	daemon := newAgentTestDaemon(t)
	ctx := context.Background()

	project, err := daemon.CreateProject(ctx, protocol.CreateProjectRequest{
		Name:    "Agent Defaults",
		RootDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	updated, err := daemon.UpdateProject(ctx, project.ID, protocol.UpdateProjectRequest{
		DefaultPhaseAgents: map[string]string{workitem.RunPresetWriter: "codex"},
	})
	if err != nil {
		t.Fatalf("update project phase agents: %v", err)
	}
	if got := updated.Preferences.DefaultPhaseAgents[workitem.RunPresetWriter]; got != "codex" {
		t.Fatalf("DefaultPhaseAgents[writer] = %q, want codex (prefs=%#v)", got, updated.Preferences.DefaultPhaseAgents)
	}

	// Patch is additive: setting one phase must not drop the others, and an empty patch leaves it intact.
	again, err := daemon.UpdateProject(ctx, project.ID, protocol.UpdateProjectRequest{
		DefaultPhaseAgents: map[string]string{workitem.RunPresetReviewer: "claude"},
	})
	if err != nil {
		t.Fatalf("second update: %v", err)
	}
	if again.Preferences.DefaultPhaseAgents[workitem.RunPresetWriter] != "codex" ||
		again.Preferences.DefaultPhaseAgents[workitem.RunPresetReviewer] != "claude" {
		t.Fatalf("merged phase agents = %#v", again.Preferences.DefaultPhaseAgents)
	}
}

func TestHTTPClientPersistsInteractiveAgentShellPreference(t *testing.T) {
	daemon := newAgentTestDaemon(t)
	ctx := context.Background()

	project, err := daemon.CreateProject(ctx, protocol.CreateProjectRequest{
		Name:    "Agent Shell",
		RootDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	enabled := true
	updated, err := daemon.UpdateProject(ctx, project.ID, protocol.UpdateProjectRequest{
		UseInteractiveAgentShell: &enabled,
	})
	if err != nil {
		t.Fatalf("enable interactive agent shell: %v", err)
	}
	if !updated.Preferences.UseInteractiveAgentShell {
		t.Fatalf("preferences = %#v", updated.Preferences)
	}

	enabled = false
	updated, err = daemon.UpdateProject(ctx, project.ID, protocol.UpdateProjectRequest{
		UseInteractiveAgentShell: &enabled,
	})
	if err != nil {
		t.Fatalf("disable interactive agent shell: %v", err)
	}
	if updated.Preferences.UseInteractiveAgentShell {
		t.Fatalf("preferences = %#v", updated.Preferences)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}
