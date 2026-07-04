package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phin-tech/whisk/internal/domain/workitem"
)

func TestResolveAgentProfileID(t *testing.T) {
	phaseAgents := map[string]string{workitem.RunPresetWriter: "codex"}

	cases := []struct {
		name        string
		explicit    string
		phaseAgents map[string]string
		preset      string
		want        string
	}{
		{name: "explicit wins over project default", explicit: "claude", phaseAgents: phaseAgents, preset: workitem.RunPresetWriter, want: "claude"},
		{name: "whitespace explicit is ignored", explicit: "  ", phaseAgents: phaseAgents, preset: workitem.RunPresetWriter, want: "codex"},
		{name: "project per-phase default used when no explicit", explicit: "", phaseAgents: phaseAgents, preset: workitem.RunPresetWriter, want: "codex"},
		{name: "explicit codex reader uses codex plan profile", explicit: "codex", phaseAgents: phaseAgents, preset: workitem.RunPresetReader, want: "codex-plan"},
		{name: "project codex reader default uses codex plan profile", explicit: "", phaseAgents: map[string]string{workitem.RunPresetReader: "codex"}, preset: workitem.RunPresetReader, want: "codex-plan"},
		{name: "no phase default falls back to preset default", explicit: "", phaseAgents: phaseAgents, preset: workitem.RunPresetReader, want: defaultAgentProfileForPreset(workitem.RunPresetReader)},
		{name: "nil phase agents falls back to preset default", explicit: "", phaseAgents: nil, preset: workitem.RunPresetWriter, want: defaultAgentProfileForPreset(workitem.RunPresetWriter)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveAgentProfileID(tc.explicit, tc.phaseAgents, tc.preset); got != tc.want {
				t.Fatalf("resolveAgentProfileID(%q, %#v, %q) = %q, want %q", tc.explicit, tc.phaseAgents, tc.preset, got, tc.want)
			}
		})
	}
}

func TestRuntimeListDetectedAgentsUsesProfileCatalog(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "codex"))
	t.Setenv("PATH", binDir)

	runtime := NewRuntime(RuntimeConfig{})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	detected, err := runtime.ListDetectedAgents(context.Background())
	if err != nil {
		t.Fatalf("ListDetectedAgents error: %v", err)
	}
	byID := map[string]string{}
	for _, agent := range detected {
		byID[agent.ProfileID] = agent.Path
	}
	if byID["codex"] != filepath.Join(binDir, "codex") {
		t.Fatalf("detected agents = %#v", detected)
	}
	if byID["codex-plan"] != filepath.Join(binDir, "codex") {
		t.Fatalf("codex-plan detection = %#v", detected)
	}
	if _, ok := byID["claude"]; ok {
		t.Fatalf("unexpected claude detection = %#v", detected)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}
