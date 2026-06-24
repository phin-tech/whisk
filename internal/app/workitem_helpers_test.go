package app

import (
	"strings"
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
	tests := map[string]string{
		workitem.RunPresetReader:   "claude-plan",
		workitem.RunPresetManager:  "claude",
		workitem.RunPresetReviewer: "claude",
		workitem.RunPresetWriter:   "claude",
		"custom":                   "",
	}

	for preset, want := range tests {
		if got := defaultAgentProfileForPreset(preset); got != want {
			t.Fatalf("profile for %q = %q, want %q", preset, got, want)
		}
	}
}

func TestBracketedPaste(t *testing.T) {
	got := string(bracketedPaste("line one\nline two\r\nline three"))
	want := "\x1b[200~line one\rline two\rline three\r\x1b[201~"
	if got != want {
		t.Fatalf("bracketedPaste = %q, want %q", got, want)
	}
	// Multi-line content must not carry a bare LF — that would let the TUI submit
	// or split on a line instead of keeping the paste as one editable block.
	if strings.Contains(got, "\n") {
		t.Fatalf("bracketed paste leaked a line feed: %q", got)
	}
	// The payload is wrapped so the submit Enter (sent separately) lands after the
	// closing marker, not inside the paste.
	if !strings.HasPrefix(got, "\x1b[200~") || !strings.HasSuffix(got, "\x1b[201~") {
		t.Fatalf("bracketed paste markers missing: %q", got)
	}
}

