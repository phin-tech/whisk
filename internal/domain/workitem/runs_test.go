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

func TestStartRunUsesReadyAndReviewStageDefaults(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")
	planning, err := state.StartPlanning(StartPlanning{
		ID:           "run_plan",
		HistoryID:    "hist_plan",
		RunHistoryID: "run_hist_plan",
		WorkItemID:   item.ID,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	draft, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_plan",
		WorkItemID: item.ID,
		RunID:      planning.ID,
		Body:       "Do it.",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("submit plan: %v", err)
	}
	ready, err := state.ApprovePlan(ApprovePlan{ArtifactID: draft.ID, WorkItemID: item.ID, Now: now})
	if err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	if ready.StageID != StageReady {
		t.Fatalf("ready = %#v", ready)
	}
	readyRun, err := state.StartRun(StartRun{
		ID:           "run_ready",
		HistoryID:    "hist_ready",
		RunHistoryID: "run_hist_ready",
		WorkItemID:   item.ID,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start ready run: %v", err)
	}
	if readyRun.Preset != RunPresetManager || readyRun.PromptTemplateID != PromptTemplatePlan {
		t.Fatalf("ready run = %#v", readyRun)
	}
	review, err := state.CompleteExecution(CompleteExecution{
		RunID:   readyRun.ID,
		Message: "done",
		Now:     now,
	})
	if err != nil {
		t.Fatalf("complete execution: %v", err)
	}
	if review.StageID != StageReview {
		t.Fatalf("review = %#v", review)
	}
	reviewRun, err := state.StartRun(StartRun{
		ID:           "run_review",
		HistoryID:    "hist_review",
		RunHistoryID: "run_hist_review",
		WorkItemID:   item.ID,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start review run: %v", err)
	}
	if reviewRun.Preset != RunPresetReviewer || reviewRun.PromptTemplateID != PromptTemplateReview {
		t.Fatalf("review run = %#v", reviewRun)
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

func TestMarkRunRunningCanMarkDaemonOwnedSession(t *testing.T) {
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
		DaemonOwned:  true,
		Actor:        "agent",
		Now:          now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("mark running: %v", err)
	}
	owned := running.Metadata["whisk.daemon/owned_session"]
	if owned.Type != MetadataTypeBool || !owned.Bool {
		t.Fatalf("metadata = %#v", running.Metadata)
	}
}

func TestCompleteRunTransitionsRunWithoutChangingStage(t *testing.T) {
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

	completed, err := state.CompleteRun(CompleteRun{
		ID:           run.ID,
		RunHistoryID: "run_hist_complete_01",
		Actor:        "agent",
		Message:      "planning accepted",
		Now:          now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("complete run: %v", err)
	}
	if completed.Status != RunStateCompleted || completed.CompletedAt == nil {
		t.Fatalf("completed = %#v", completed)
	}
	updated, _ := state.GetWorkItem(item.ID)
	if updated.StageID != item.StageID || updated.RunState != RunStateCompleted {
		t.Fatalf("updated item = %#v", updated)
	}
}

func TestReportStatusQuestionBlockedAndDoneTransitionRunAndWorkItem(t *testing.T) {
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
	run, err = state.MarkRunRunning(MarkRunRunning{
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

	question, err := state.ReportStatus(ReportStatus{
		ID:           "status_01",
		RunHistoryID: "run_hist_question_01",
		Kind:         StatusKindQuestion,
		Actor:        "agent",
		Message:      "Need the staging API key.",
		ProjectID:    "proj_01",
		WorkItemID:   item.ID,
		RunID:        run.ID,
		SessionID:    "sess_01",
		PTYID:        "pty_01",
		Now:          now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("report question: %v", err)
	}
	if question.Kind != StatusKindQuestion || !question.RequiresAttention || question.Message != "Need the staging API key." {
		t.Fatalf("question = %#v", question)
	}
	runs := state.ListRuns(item.ID)
	if len(runs) != 1 || runs[0].Status != RunStateAwaitingInput || runs[0].CompletedAt != nil {
		t.Fatalf("runs after question = %#v", runs)
	}
	if got := runs[0].History[len(runs[0].History)-1]; got.Type != RunStateAwaitingInput || got.Actor != "agent" || got.Message != "Need the staging API key." {
		t.Fatalf("question event = %#v", got)
	}
	updated, _ := state.GetWorkItem(item.ID)
	if updated.RunState != RunStateAwaitingInput {
		t.Fatalf("run state = %q", updated.RunState)
	}

	blocked, err := state.ReportStatus(ReportStatus{
		ID:           "status_02",
		RunHistoryID: "run_hist_blocked_01",
		Kind:         StatusKindBlocked,
		Actor:        "agent",
		Message:      "Waiting on credentials.",
		ProjectID:    "proj_01",
		WorkItemID:   item.ID,
		RunID:        run.ID,
		SessionID:    "sess_01",
		PTYID:        "pty_01",
		Now:          now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("report blocked: %v", err)
	}
	if blocked.Kind != StatusKindBlocked || !blocked.RequiresAttention {
		t.Fatalf("blocked = %#v", blocked)
	}
	runs = state.ListRuns(item.ID)
	if len(runs) != 1 || runs[0].Status != RunStateAwaitingInput || runs[0].CompletedAt != nil {
		t.Fatalf("runs after blocked = %#v", runs)
	}

	done, err := state.ReportStatus(ReportStatus{
		ID:           "status_03",
		RunHistoryID: "run_hist_done_01",
		Kind:         StatusKindDone,
		Actor:        "agent",
		Message:      "Implementation complete and tests pass.",
		ProjectID:    "proj_01",
		WorkItemID:   item.ID,
		RunID:        run.ID,
		SessionID:    "sess_01",
		PTYID:        "pty_01",
		Now:          now.Add(4 * time.Minute),
	})
	if err != nil {
		t.Fatalf("report done: %v", err)
	}
	if done.Kind != StatusKindDone || done.RequiresAttention {
		t.Fatalf("done = %#v", done)
	}
	runs = state.ListRuns(item.ID)
	if len(runs) != 1 || runs[0].Status != RunStateCompleted || runs[0].CompletedAt == nil {
		t.Fatalf("runs after done = %#v", runs)
	}
	if got := runs[0].History[len(runs[0].History)-1]; got.Type != RunStateCompleted || got.Message != "Implementation complete and tests pass." {
		t.Fatalf("done event = %#v", got)
	}
	updated, _ = state.GetWorkItem(item.ID)
	if updated.RunState != RunStateCompleted || updated.StageID != "review" {
		t.Fatalf("updated item = %#v", updated)
	}
}

func TestGateCompletionBranchesAndApproveDoneBlocking(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")
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
	if _, err := state.CompleteExecution(CompleteExecution{
		RunID:   run.ID,
		Message: "ready",
		Now:     now,
	}); err != nil {
		t.Fatalf("complete execution: %v", err)
	}
	gates := state.ListGateReports(item.ID)
	if len(gates) != 1 || !gates[0].Blocking {
		t.Fatalf("gates = %#v", gates)
	}
	if _, err := state.ApproveDone(ApproveDone{WorkItemID: item.ID, Now: now}); err == nil || !strings.Contains(err.Error(), "blocking gates") {
		t.Fatalf("expected blocking gate error, got %v", err)
	}
	failed, err := state.CompleteGate(CompleteGate{ID: gates[0].ID, Status: GateStatusFailed, Now: now})
	if err != nil {
		t.Fatalf("complete failed gate: %v", err)
	}
	if failed.Status != GateStatusFailed {
		t.Fatalf("failed = %#v", failed)
	}
	if _, err := state.CompleteGate(CompleteGate{ID: gates[0].ID, Status: GateStatusOverridden, Now: now}); err == nil || !strings.Contains(err.Error(), "override reason required") {
		t.Fatalf("expected override reason error, got %v", err)
	}
	overridden, err := state.CompleteGate(CompleteGate{
		ID:             gates[0].ID,
		Status:         GateStatusOverridden,
		OverrideReason: "Manual approval.",
		Now:            now,
	})
	if err != nil {
		t.Fatalf("override gate: %v", err)
	}
	if overridden.Status != GateStatusOverridden || overridden.OverrideReason != "Manual approval." {
		t.Fatalf("overridden = %#v", overridden)
	}
	done, err := state.ApproveDone(ApproveDone{WorkItemID: item.ID, Reason: "override accepted", Now: now})
	if err != nil {
		t.Fatalf("approve done: %v", err)
	}
	if done.StageID != StageDone {
		t.Fatalf("done = %#v", done)
	}
}

func TestReportStatusStoresSessionScopedEventsWithoutMutatingWorkItems(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

	event, err := state.ReportStatus(ReportStatus{
		ID:        "status_01",
		Kind:      StatusKindQuestion,
		Actor:     "agent",
		Message:   "Which branch should I use?",
		SessionID: "sess_01",
		PTYID:     "pty_01",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("report session status: %v", err)
	}
	if event.Scope != StatusScopePTY || event.RunID != "" || !event.RequiresAttention {
		t.Fatalf("event = %#v", event)
	}
	events := state.ListStatusEvents(ListStatusEvents{SessionID: "sess_01", UnreadOnly: true})
	if len(events) != 1 || events[0].ID != event.ID {
		t.Fatalf("events = %#v", events)
	}

	read, err := state.MarkStatusEventRead(MarkStatusEventRead{ID: event.ID, Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if read.ReadAt == nil {
		t.Fatalf("read event = %#v", read)
	}
	if events := state.ListStatusEvents(ListStatusEvents{SessionID: "sess_01", UnreadOnly: true}); len(events) != 0 {
		t.Fatalf("unread events = %#v", events)
	}
}

func TestReportStatusRejectsTerminalRuns(t *testing.T) {
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
	if _, err := state.ReportStatus(ReportStatus{
		ID:           "status_01",
		RunHistoryID: "run_hist_done_01",
		Kind:         StatusKindDone,
		ProjectID:    "proj_01",
		WorkItemID:   item.ID,
		RunID:        run.ID,
		Now:          now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("report done: %v", err)
	}

	_, err = state.ReportStatus(ReportStatus{
		ID:           "status_02",
		RunHistoryID: "run_hist_question_01",
		Kind:         StatusKindQuestion,
		Message:      "One more thing.",
		ProjectID:    "proj_01",
		WorkItemID:   item.ID,
		RunID:        run.ID,
		Now:          now.Add(2 * time.Minute),
	})
	if err == nil || !strings.Contains(err.Error(), "already terminal") {
		t.Fatalf("expected terminal run error, got %v", err)
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
