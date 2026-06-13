package workitem

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultWorkflowUsesExplicitActionStages(t *testing.T) {
	template := DefaultWorkflowTemplate(time.Time{})
	want := []string{"backlog", "planning", "ready", "execution", "blocked", "review", "done"}
	if len(template.Stages) != len(want) {
		t.Fatalf("stages = %#v", template.Stages)
	}
	for i, stage := range template.Stages {
		if stage.ID != want[i] {
			t.Fatalf("stage[%d] = %#v, want %q", i, stage, want[i])
		}
	}
	if template.Stages[2].ProvisionWorktree {
		t.Fatalf("ready must not provision worktrees: %#v", template.Stages[2])
	}
	if !template.Stages[3].ProvisionWorktree {
		t.Fatalf("execution must provision worktrees by default: %#v", template.Stages[3])
	}
}

func TestWorkflowDefinitionValidationRejectsInvalidDefinitions(t *testing.T) {
	valid := DefaultWorkflowDefinition()
	tests := []struct {
		name string
		edit func(*WorkflowDefinition)
		want string
	}{
		{name: "missing id", edit: func(def *WorkflowDefinition) { def.ID = "" }, want: "workflow id required"},
		{name: "missing version", edit: func(def *WorkflowDefinition) { def.Version = 0 }, want: "workflow version must be positive"},
		{name: "missing stage", edit: func(def *WorkflowDefinition) { def.Stages = def.Stages[:len(def.Stages)-1] }, want: "workflow stages must match universal stages"},
		{name: "wrong stage order", edit: func(def *WorkflowDefinition) { def.Stages[0], def.Stages[1] = def.Stages[1], def.Stages[0] }, want: "workflow stages must match universal stages"},
		{name: "missing action id", edit: func(def *WorkflowDefinition) { def.Actions[0].ID = "" }, want: "workflow action id required"},
		{name: "duplicate action id", edit: func(def *WorkflowDefinition) { def.Actions[1].ID = def.Actions[0].ID }, want: "already exists"},
		{name: "unknown from stage", edit: func(def *WorkflowDefinition) { def.Actions[0].From = []string{"missing"} }, want: "unknown stage missing"},
		{name: "unknown to stage", edit: func(def *WorkflowDefinition) { def.Actions[0].To = "missing" }, want: "unknown stage missing"},
		{name: "bad required artifact", edit: func(def *WorkflowDefinition) {
			def.Actions[0].Requires = []WorkflowArtifactRequirement{{Kind: "bad", Status: ArtifactStatusDraft}}
		}, want: "unsupported artifact kind bad"},
		{name: "bad created artifact", edit: func(def *WorkflowDefinition) {
			def.Actions[0].CreatesArtifact = &WorkflowArtifactEffect{Kind: ArtifactKindPlan, Status: "bad"}
		}, want: "unsupported artifact status bad"},
		{name: "bad updated artifact", edit: func(def *WorkflowDefinition) {
			def.Actions[0].UpdatesArtifact = &WorkflowArtifactEffect{Kind: "bad", Status: ArtifactStatusDraft}
		}, want: "unsupported artifact kind bad"},
		{name: "bad run preset", edit: func(def *WorkflowDefinition) {
			def.Actions[0].CreatesRun = &WorkflowRunEffect{Preset: "bad", PromptTemplateID: PromptTemplatePlan}
		}, want: "unsupported run preset bad"},
		{name: "bad run prompt", edit: func(def *WorkflowDefinition) {
			def.Actions[0].CreatesRun = &WorkflowRunEffect{Preset: RunPresetReader, PromptTemplateID: "bad"}
		}, want: "unsupported prompt template bad"},
		{name: "missing gate id", edit: func(def *WorkflowDefinition) { def.Gates[0].ID = "" }, want: "workflow gate id required"},
		{name: "bad gate phase", edit: func(def *WorkflowDefinition) { def.Gates[0].Phase = "missing" }, want: "unknown stage missing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := valid
			definition.Stages = append([]string(nil), valid.Stages...)
			definition.Actions = append([]WorkflowActionDefinition(nil), valid.Actions...)
			definition.Gates = append([]WorkflowGateDefinition(nil), valid.Gates...)
			test.edit(&definition)
			err := ValidateWorkflowDefinition(definition)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
	if _, err := ParseWorkflowDefinition([]byte(`{`)); err == nil {
		t.Fatalf("expected invalid json error")
	}
}

func TestWorkflowActionsEnforcePlanExecutionReviewAndDoneRules(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")

	if _, err := state.StartExecution(StartExecution{
		ID:           "run_bad",
		HistoryID:    "hist_bad",
		RunHistoryID: "run_hist_bad",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now,
	}); err == nil || !strings.Contains(err.Error(), "approved plan required") {
		t.Fatalf("expected approved plan requirement, got %v", err)
	}

	planning, err := state.StartPlanning(StartPlanning{
		ID:           "run_plan",
		HistoryID:    "hist_plan",
		RunHistoryID: "run_hist_plan",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	if planning.PromptTemplateID != PromptTemplatePlan {
		t.Fatalf("planning run = %#v", planning)
	}
	item, _ = state.GetWorkItem(item.ID)
	if item.StageID != StagePlanning {
		t.Fatalf("item after planning = %#v", item)
	}

	draft, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_plan",
		WorkItemID: item.ID,
		RunID:      planning.ID,
		Title:      "Plan",
		Body:       "1. Add tests\n2. Implement",
		Actor:      "agent",
		Now:        now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("submit draft plan: %v", err)
	}
	if draft.Kind != ArtifactKindPlan || draft.Status != ArtifactStatusDraft {
		t.Fatalf("draft = %#v", draft)
	}

	approved, err := state.ApprovePlan(ApprovePlan{
		ArtifactID: draft.ID,
		WorkItemID: item.ID,
		Actor:      "human",
		Now:        now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	if approved.StageID != StageReady {
		t.Fatalf("approved item = %#v", approved)
	}

	execution, err := state.StartExecution(StartExecution{
		ID:           "run_exec",
		HistoryID:    "hist_exec",
		RunHistoryID: "run_hist_exec",
		WorkItemID:   item.ID,
		SessionID:    "sess_01",
		PTYID:        "pty_01",
		Actor:        "agent",
		Now:          now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("start execution: %v", err)
	}
	if execution.PromptTemplateID != PromptTemplateImplement || execution.Status != RunStateQueued {
		t.Fatalf("execution run = %#v", execution)
	}
	item, _ = state.GetWorkItem(item.ID)
	if item.StageID != StageExecution {
		t.Fatalf("item after execution = %#v", item)
	}

	question, err := state.AskQuestion(AskQuestion{
		ID:         "question_01",
		WorkItemID: item.ID,
		RunID:      execution.ID,
		Prompt:     "Which API key?",
		Actor:      "agent",
		Now:        now.Add(4 * time.Minute),
	})
	if err != nil {
		t.Fatalf("ask question: %v", err)
	}
	if question.Status != QuestionStatusOpen {
		t.Fatalf("question = %#v", question)
	}
	item, _ = state.GetWorkItem(item.ID)
	if item.StageID != StageExecution || item.RunState != RunStateAwaitingInput {
		t.Fatalf("questions must not move blocked: %#v", item)
	}

	answered, err := state.AnswerQuestion(AnswerQuestion{
		ID:     question.ID,
		Answer: "Use staging.",
		Actor:  "human",
		Now:    now.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("answer question: %v", err)
	}
	if answered.Status != QuestionStatusAnswered || answered.Answer != "Use staging." {
		t.Fatalf("answered = %#v", answered)
	}

	blocked, err := state.ReportBlocked(ReportBlocked{
		WorkItemID: item.ID,
		RunID:      execution.ID,
		Reason:     "Missing generated client.",
		Actor:      "agent",
		Now:        now.Add(6 * time.Minute),
	})
	if err != nil {
		t.Fatalf("report blocked: %v", err)
	}
	if blocked.StageID != StageBlocked {
		t.Fatalf("blocked = %#v", blocked)
	}
	unblocked, err := state.Unblock(Unblock{
		WorkItemID: item.ID,
		Actor:      "human",
		Now:        now.Add(7 * time.Minute),
	})
	if err != nil {
		t.Fatalf("unblock: %v", err)
	}
	if unblocked.StageID != StageExecution {
		t.Fatalf("unblocked = %#v", unblocked)
	}

	review, err := state.CompleteExecution(CompleteExecution{
		RunID:   execution.ID,
		Actor:   "agent",
		Message: "Implementation complete.",
		Now:     now.Add(8 * time.Minute),
	})
	if err != nil {
		t.Fatalf("complete execution: %v", err)
	}
	if review.StageID != StageReview {
		t.Fatalf("review = %#v", review)
	}
	if _, err := state.ApproveDone(ApproveDone{WorkItemID: item.ID, Actor: "human", Now: now.Add(9 * time.Minute)}); err == nil || !strings.Contains(err.Error(), "blocking gates") {
		t.Fatalf("expected blocking gate failure, got %v", err)
	}
	gates := state.ListGateReports(item.ID)
	if len(gates) != 1 || !gates[0].Blocking || gates[0].Status != GateStatusPending {
		t.Fatalf("gates = %#v", gates)
	}
	if _, err := state.CompleteGate(CompleteGate{
		ID:     gates[0].ID,
		Status: GateStatusPassed,
		Actor:  "agent",
		Now:    now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("complete gate: %v", err)
	}
	done, err := state.ApproveDone(ApproveDone{
		WorkItemID: item.ID,
		Actor:      "human",
		Now:        now.Add(11 * time.Minute),
	})
	if err != nil {
		t.Fatalf("approve done: %v", err)
	}
	if done.StageID != StageDone {
		t.Fatalf("done = %#v", done)
	}
}

func TestReviewFeedbackCreatesArtifactAndResumesExecution(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")
	mustApprovedPlan(t, state, item.ID, now)
	run, err := state.StartExecution(StartExecution{
		ID:           "run_exec",
		HistoryID:    "hist_exec",
		RunHistoryID: "run_hist_exec",
		WorkItemID:   item.ID,
		SessionID:    "sess_01",
		PTYID:        "pty_01",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start execution: %v", err)
	}
	if _, err := state.CompleteExecution(CompleteExecution{RunID: run.ID, Actor: "agent", Now: now.Add(time.Minute)}); err != nil {
		t.Fatalf("complete execution: %v", err)
	}

	feedback, err := state.SubmitReviewFeedback(SubmitReviewFeedback{
		ID:         "feedback_01",
		WorkItemID: item.ID,
		RunID:      run.ID,
		Body:       "Fix the missing validation.",
		Actor:      "human",
		Now:        now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("submit feedback: %v", err)
	}
	if feedback.Kind != ArtifactKindFeedback || feedback.Status != ArtifactStatusApproved {
		t.Fatalf("feedback = %#v", feedback)
	}
	item, _ = state.GetWorkItem(item.ID)
	if item.StageID != StageExecution {
		t.Fatalf("item after feedback = %#v", item)
	}
}

func TestMetadataValidationAndRoundTrip(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")

	if _, err := state.SetMetadata(SetMetadata{
		OwnerType: MetadataOwnerWorkItem,
		OwnerID:   item.ID,
		Namespace: "plugin.review",
		Key:       "risk",
		Value:     MetadataValue{Type: MetadataTypeNumber, Number: 0.75},
		Now:       now,
	}); err != nil {
		t.Fatalf("set metadata: %v", err)
	}
	updated, _ := state.GetWorkItem(item.ID)
	value := updated.Metadata["plugin.review/risk"]
	if value.Type != MetadataTypeNumber || value.Number != 0.75 {
		t.Fatalf("metadata = %#v", updated.Metadata)
	}
	if _, err := state.SetMetadata(SetMetadata{
		OwnerType: MetadataOwnerWorkItem,
		OwnerID:   item.ID,
		Namespace: "bad namespace",
		Key:       "risk",
		Value:     MetadataValue{Type: MetadataTypeBool, Bool: true},
		Now:       now,
	}); err == nil {
		t.Fatalf("expected invalid namespace error")
	}
}

func mustApprovedPlan(t *testing.T, state *State, workItemID string, now time.Time) Artifact {
	t.Helper()
	draft, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "plan_" + workItemID,
		WorkItemID: workItemID,
		Title:      "Plan",
		Body:       "Do the thing.",
		Actor:      "agent",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("submit draft plan: %v", err)
	}
	if _, err := state.ApprovePlan(ApprovePlan{ArtifactID: draft.ID, WorkItemID: workItemID, Actor: "human", Now: now}); err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	return draft
}
