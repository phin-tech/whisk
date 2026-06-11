package workitem

import (
	"strings"
	"testing"
	"time"
)

func TestStartRunUsesStageDefaultsAndSnapshotsPrompt(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")
	item, err := state.AddAttachment(AddAttachment{
		ID:         "att_01",
		HistoryID:  "hist_att_01",
		WorkItemID: item.ID,
		Kind:       AttachmentKindFile,
		Path:       "docs/spec.md",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("add attachment: %v", err)
	}

	run, err := state.StartRun(StartRun{
		ID:           "run_01",
		HistoryID:    "hist_run_01",
		RunHistoryID: "run_hist_01",
		WorkItemID:   item.ID,
		SessionID:    "sess_01",
		PTYID:        "pty_01",
		Actor:        "agent",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.Status != RunStateQueued || run.Preset != RunPresetReader || run.PromptTemplateID != PromptTemplatePlan {
		t.Fatalf("run = %#v", run)
	}
	if !strings.Contains(run.PromptSnapshot, "Task wi_01") || !strings.Contains(run.PromptSnapshot, "docs/spec.md") {
		t.Fatalf("prompt snapshot = %q", run.PromptSnapshot)
	}
	updated, _ := state.GetWorkItem(item.ID)
	if updated.RunState != RunStateQueued {
		t.Fatalf("run state = %q", updated.RunState)
	}
}

func TestStartRunCanUseExplicitPresetAndTemplate(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")
	item, err := state.BindWorktree(BindWorktree{
		ID:           item.ID,
		HistoryID:    "hist_bind_01",
		Branch:       "feature",
		WorktreePath: "/repo/.worktrees/feature",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("bind worktree: %v", err)
	}

	run, err := state.StartRun(StartRun{
		ID:               "run_01",
		HistoryID:        "hist_run_01",
		RunHistoryID:     "run_hist_01",
		WorkItemID:       item.ID,
		Preset:           RunPresetWriter,
		PromptTemplateID: PromptTemplateImplement,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.Preset != RunPresetWriter || !strings.Contains(run.PromptSnapshot, "/repo/.worktrees/feature") {
		t.Fatalf("run = %#v", run)
	}
}

func TestCancelRunTransitionsRunAndWorkItem(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")
	run, err := state.StartRun(StartRun{
		ID:           "run_01",
		HistoryID:    "hist_run_01",
		RunHistoryID: "run_hist_01",
		WorkItemID:   item.ID,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	cancelled, err := state.CancelRun(CancelRun{
		ID:           run.ID,
		RunHistoryID: "run_hist_cancel_01",
		Actor:        "agent",
		Now:          now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("cancel run: %v", err)
	}
	if cancelled.Status != RunStateCancelled || cancelled.CompletedAt == nil || len(cancelled.History) != 2 {
		t.Fatalf("cancelled = %#v", cancelled)
	}
	updated, _ := state.GetWorkItem(item.ID)
	if updated.RunState != RunStateCancelled {
		t.Fatalf("run state = %q", updated.RunState)
	}
}

func TestMarkRunRunningStoresSessionPTYAndTransitionsWorkItem(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")
	run, err := state.StartRun(StartRun{
		ID:           "run_01",
		HistoryID:    "hist_run_01",
		RunHistoryID: "run_hist_01",
		WorkItemID:   item.ID,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	running, err := state.MarkRunRunning(MarkRunRunning{
		ID:           run.ID,
		RunHistoryID: "run_hist_running_01",
		SessionID:    "sess_01",
		PTYID:        "pty_01",
		Actor:        "agent",
		Now:          now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if running.Status != RunStateRunning || running.SessionID != "sess_01" || running.PTYID != "pty_01" || len(running.History) != 2 {
		t.Fatalf("running = %#v", running)
	}
	updated, _ := state.GetWorkItem(item.ID)
	if updated.RunState != RunStateRunning {
		t.Fatalf("run state = %q", updated.RunState)
	}
}

func TestFailRunTransitionsRunAndWorkItem(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")
	run, err := state.StartRun(StartRun{
		ID:           "run_01",
		HistoryID:    "hist_run_01",
		RunHistoryID: "run_hist_01",
		WorkItemID:   item.ID,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	failed, err := state.FailRun(FailRun{
		ID:           run.ID,
		RunHistoryID: "run_hist_failed_01",
		Message:      "launch failed",
		Now:          now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("fail run: %v", err)
	}
	if failed.Status != RunStateFailed || failed.CompletedAt == nil || failed.History[1].Message != "launch failed" {
		t.Fatalf("failed = %#v", failed)
	}
	updated, _ := state.GetWorkItem(item.ID)
	if updated.RunState != RunStateFailed {
		t.Fatalf("run state = %q", updated.RunState)
	}
}
