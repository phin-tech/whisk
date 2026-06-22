package app

import (
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
