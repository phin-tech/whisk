package app

import (
	"testing"

	"github.com/phin-tech/whisk/internal/domain/workitem"
)

func TestRunPhaseLabel(t *testing.T) {
	tests := []struct {
		name string
		run  workitem.WorkItemRun
		want string
	}{
		{name: "planning", run: workitem.WorkItemRun{PromptTemplateID: workitem.PromptTemplatePlan}, want: "Planning"},
		{name: "execution", run: workitem.WorkItemRun{PromptTemplateID: workitem.PromptTemplateImplement}, want: "Execution"},
		{name: "review", run: workitem.WorkItemRun{PromptTemplateID: workitem.PromptTemplateReview}, want: "Review"},
		{name: "fallback", run: workitem.WorkItemRun{PromptTemplateID: "custom"}, want: "Run"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runPhaseLabel(tt.run); got != tt.want {
				t.Fatalf("label = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWorkItemRunSessionName(t *testing.T) {
	item := workitem.WorkItem{ID: "wi_01", Number: 42, Title: "Fix bug"}
	run := workitem.WorkItemRun{PromptTemplateID: workitem.PromptTemplateImplement}

	if got := workItemRunSessionName(item, run); got != "#42 Execution - Fix bug" {
		t.Fatalf("name = %q", got)
	}

	item.Title = " "
	if got := workItemRunSessionName(item, run); got != "#42 Execution - wi_01" {
		t.Fatalf("fallback name = %q", got)
	}
}

func TestDefaultAgentProfileForPreset(t *testing.T) {
	for _, preset := range []string{
		workitem.RunPresetReader,
		workitem.RunPresetManager,
		workitem.RunPresetReviewer,
		workitem.RunPresetWriter,
		"custom",
	} {
		if got := defaultAgentProfileForPreset(preset); got != "codex" {
			t.Fatalf("profile for %q = %q", preset, got)
		}
	}
}
