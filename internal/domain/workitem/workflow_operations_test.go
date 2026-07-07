package workitem

import (
	"strings"
	"testing"
	"time"
)

func TestWorkflowActionAvailabilityReflectsCurrentItemState(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", project.ID)

	actions, err := state.ListWorkflowActionAvailability(item.ID)
	if err != nil {
		t.Fatalf("list backlog actions: %v", err)
	}
	if got := availabilityByID(actions, WorkflowActionStartPlanning); got == nil || !got.Enabled || got.InputKind != WorkflowActionInputRun {
		t.Fatalf("start planning availability = %#v", got)
	}
	if got := availabilityByID(actions, WorkflowActionStartPlanning); got == nil || !got.Recommended {
		t.Fatalf("start planning should be recommended, got %#v", got)
	}
	if got := recommendedAvailabilityIDs(actions); len(got) != 1 || got[0] != WorkflowActionStartPlanning {
		t.Fatalf("recommended backlog actions = %#v", got)
	}
	if got := availabilityByID(actions, WorkflowActionApprovePlan); got != nil {
		t.Fatalf("approve plan should not be available from backlog: %#v", got)
	}

	if _, err := state.StartPlanning(StartPlanning{
		ID:           "run_01",
		HistoryID:    "hist_run_01",
		RunHistoryID: "run_hist_01",
		WorkItemID:   item.ID,
		Now:          now,
	}); err != nil {
		t.Fatalf("start planning: %v", err)
	}
	actions, err = state.ListWorkflowActionAvailability(item.ID)
	if err != nil {
		t.Fatalf("list planning actions: %v", err)
	}
	if got := availabilityByID(actions, WorkflowActionSubmitDraftPlan); got == nil || !got.Enabled || got.InputKind != WorkflowActionInputArtifact {
		t.Fatalf("submit draft availability = %#v", got)
	}
	if got := recommendedAvailabilityIDs(actions); len(got) != 1 || got[0] != WorkflowActionSubmitDraftPlan {
		t.Fatalf("recommended planning actions = %#v", got)
	}
	if got := availabilityByID(actions, WorkflowActionApprovePlan); got == nil || got.Enabled || !strings.Contains(got.Reason, "plan draft") {
		t.Fatalf("approve plan without draft = %#v", got)
	}
	if got := availabilityByID(actions, WorkflowActionReportBlocked); got == nil || !got.Enabled || got.InputKind != WorkflowActionInputNone {
		t.Fatalf("report blocked availability = %#v", got)
	}

	draft, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_01",
		WorkItemID: item.ID,
		Title:      "Plan",
		Body:       "Do the thing.",
		Now:        now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("submit draft: %v", err)
	}
	actions, err = state.ListWorkflowActionAvailability(item.ID)
	if err != nil {
		t.Fatalf("list with draft: %v", err)
	}
	if got := availabilityByID(actions, WorkflowActionApprovePlan); got == nil || !got.Enabled || got.InputKind != WorkflowActionInputArtifactSelection {
		t.Fatalf("approve plan with draft = %#v", got)
	}
	if got := recommendedAvailabilityIDs(actions); len(got) != 1 || got[0] != WorkflowActionApprovePlan {
		t.Fatalf("recommended draft actions = %#v", got)
	}

	if _, err := state.ApprovePlan(ApprovePlan{
		WorkItemID: item.ID,
		ArtifactID: draft.ID,
		Actor:      "human",
		Now:        now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	actions, err = state.ListWorkflowActionAvailability(item.ID)
	if err != nil {
		t.Fatalf("list ready actions: %v", err)
	}
	if got := availabilityByID(actions, WorkflowActionStartExecution); got == nil || !got.Enabled || got.InputKind != WorkflowActionInputRun {
		t.Fatalf("start execution availability = %#v", got)
	}
	if got := recommendedAvailabilityIDs(actions); len(got) != 1 || got[0] != WorkflowActionStartExecution {
		t.Fatalf("recommended ready actions = %#v", got)
	}

	run, err := state.StartExecution(StartExecution{
		ID:           "run_exec_01",
		HistoryID:    "hist_run_exec_01",
		RunHistoryID: "run_hist_exec_01",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("start execution: %v", err)
	}
	if _, err := state.CompleteExecution(CompleteExecution{
		RunID:   run.ID,
		Message: "ready for review",
		Actor:   "agent",
		Now:     now.Add(4 * time.Minute),
	}); err != nil {
		t.Fatalf("complete execution: %v", err)
	}
	actions, err = state.ListWorkflowActionAvailability(item.ID)
	if err != nil {
		t.Fatalf("list review actions: %v", err)
	}
	if got := availabilityByID(actions, WorkflowActionApproveDone); got == nil || got.Enabled || got.Recommended || !strings.Contains(got.Reason, "blocking gates") {
		t.Fatalf("approve done with blocking gate = %#v", got)
	}
	if got := recommendedAvailabilityIDs(actions); len(got) != 1 || got[0] != WorkflowActionSubmitReviewFeedback {
		t.Fatalf("recommended review actions = %#v", got)
	}
}

func TestHumanWorkflowActionsRejectAgentActors(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	item, gate := mustReviewWorkItemWithPendingGate(t, state, now)

	if _, err := state.CompleteGate(CompleteGate{
		ID:     gate.ID,
		Status: GateStatusPassed,
		Actor:  "agent:codex",
		Now:    now.Add(time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "requires human actor") {
		t.Fatalf("expected agent gate denial, got %v", err)
	}
	gates := state.ListGateReports(item.ID)
	if len(gates) != 1 || gates[0].Status != GateStatusPending {
		t.Fatalf("gate mutated after denied agent completion: %#v", gates)
	}

	if _, err := state.CompleteGate(CompleteGate{
		ID:     gate.ID,
		Status: GateStatusPassed,
		Actor:  "human",
		Now:    now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("human complete gate: %v", err)
	}
	if _, err := state.ApproveDone(ApproveDone{
		WorkItemID: item.ID,
		Actor:      "agent:codex",
		Now:        now.Add(3 * time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "requires human actor") {
		t.Fatalf("expected agent approve done denial, got %v", err)
	}
	if stored, ok := state.GetWorkItem(item.ID); !ok || stored.StageID != StageReview {
		t.Fatalf("item mutated after denied approve done: %#v, ok = %v", stored, ok)
	}
	if _, err := state.MoveWorkItem(MoveWorkItem{
		ID:        item.ID,
		HistoryID: "hist_move_done",
		StageID:   StageDone,
		Actor:     "agent:codex",
		Now:       now.Add(4 * time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "requires human actor") {
		t.Fatalf("expected agent direct move denial, got %v", err)
	}
	if _, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   WorkflowActionApproveDone,
		Actor:      "agent:codex",
		Now:        now.Add(5 * time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "requires human actor") {
		t.Fatalf("expected agent workflow action denial, got %v", err)
	}

	done, err := state.ApproveDone(ApproveDone{
		WorkItemID: item.ID,
		Actor:      "human",
		Now:        now.Add(6 * time.Minute),
	})
	if err != nil {
		t.Fatalf("human approve done: %v", err)
	}
	if done.StageID != StageDone {
		t.Fatalf("done = %#v", done)
	}
}

func TestApplyWorkflowActionUpdatesSelectedArtifact(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	definition := WorkflowDefinition{
		ID:      "generic-artifact-selection",
		Version: 1,
		Stages:  []string{StageBacklog, StagePlanning, StageReady},
		Actions: []WorkflowActionDefinition{
			{ID: "enter_planning", From: []string{StageBacklog}, To: StagePlanning},
			{
				ID:   WorkflowActionSubmitDraftPlan,
				From: []string{StagePlanning},
				To:   StagePlanning,
				CreatesArtifact: &WorkflowArtifactEffect{
					Kind:   ArtifactKindPlan,
					Status: ArtifactStatusDraft,
				},
			},
			{
				ID:       "accept_plan",
				From:     []string{StagePlanning},
				To:       StageReady,
				Requires: []WorkflowArtifactRequirement{{Kind: ArtifactKindPlan, Status: ArtifactStatusDraft}},
				UpdatesArtifact: &WorkflowArtifactEffect{
					Kind:   ArtifactKindPlan,
					Status: ArtifactStatusApproved,
				},
				RequiresHuman: true,
			},
		},
	}
	project := mustProjectWithWorkflow(t, state, "proj_01", definition, now)
	item := mustWorkItem(t, state, "wi_01", project.ID)
	other := mustWorkItem(t, state, "wi_02", project.ID)
	if _, err := state.ApplyWorkflowAction(ApplyWorkflowAction{WorkItemID: item.ID, ActionID: "enter_planning", Now: now.Add(time.Minute)}); err != nil {
		t.Fatalf("enter planning: %v", err)
	}
	if _, err := state.ApplyWorkflowAction(ApplyWorkflowAction{WorkItemID: other.ID, ActionID: "enter_planning", Now: now.Add(2 * time.Minute)}); err != nil {
		t.Fatalf("enter other planning: %v", err)
	}
	draft, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_01",
		WorkItemID: item.ID,
		Title:      "Plan",
		Body:       "Ship it.",
		Actor:      "agent",
		Now:        now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("submit draft: %v", err)
	}
	otherDraft, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_02",
		WorkItemID: other.ID,
		Title:      "Other plan",
		Body:       "Do not use.",
		Actor:      "agent",
		Now:        now.Add(4 * time.Minute),
	})
	if err != nil {
		t.Fatalf("submit other draft: %v", err)
	}

	if _, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   "accept_plan",
		Actor:      "human",
		Now:        now.Add(5 * time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "artifact id required") {
		t.Fatalf("expected missing artifact id error, got %v", err)
	}
	if _, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   "accept_plan",
		ArtifactID: otherDraft.ID,
		Actor:      "human",
		Now:        now.Add(6 * time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "plan artifact required") {
		t.Fatalf("expected foreign artifact error, got %v", err)
	}

	ready, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   "accept_plan",
		ArtifactID: draft.ID,
		Actor:      "human",
		Now:        now.Add(7 * time.Minute),
	})
	if err != nil {
		t.Fatalf("accept plan: %v", err)
	}
	if ready.StageID != StageReady {
		t.Fatalf("ready = %#v", ready)
	}
	artifacts := state.ListArtifacts("")
	if len(artifacts) != 2 ||
		artifacts[0].ID != draft.ID ||
		artifacts[0].Status != ArtifactStatusApproved ||
		artifacts[1].ID != otherDraft.ID ||
		artifacts[1].Status != ArtifactStatusDraft {
		t.Fatalf("artifacts = %#v", artifacts)
	}
}

func TestQuestionPolicyControlsRunStateTransitions(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	definition := WorkflowDefinition{
		ID:      "question-policy",
		Version: 1,
		Stages:  []string{StageBacklog, StagePlanning},
		Actions: []WorkflowActionDefinition{{
			ID:   WorkflowActionStartPlanning,
			From: []string{StageBacklog},
			To:   StagePlanning,
			CreatesRun: &WorkflowRunEffect{
				Phase:            "planning",
				Preset:           RunPresetReader,
				PromptTemplateID: PromptTemplatePlan,
				WorkingDir:       "projectRoot",
			},
		}},
		Questions: WorkflowQuestionPolicy{
			Enabled:      true,
			SetsRunState: RunStateRunning,
			AnswerClearsAwaitingInputWhenNoOpenQuestionsRemain: false,
		},
	}
	if _, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{Definition: definition, Source: "test", Now: now}); err != nil {
		t.Fatalf("import workflow: %v", err)
	}
	project := mustProject(t, state, "proj_questions", "Questions")
	if _, err := state.SetProjectWorkflowDefinition(SetProjectWorkflowDefinition{
		ProjectID: project.ID,
		ID:        definition.ID,
		Version:   definition.Version,
		Now:       now,
	}); err != nil {
		t.Fatalf("set workflow: %v", err)
	}
	item := mustWorkItem(t, state, "wi_questions", project.ID)
	run, err := state.StartPlanning(StartPlanning{
		ID:           "run_question",
		HistoryID:    "hist_question",
		RunHistoryID: "run_hist_question",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}

	question, err := state.AskQuestion(AskQuestion{
		ID:         "question_01",
		WorkItemID: item.ID,
		RunID:      run.ID,
		Prompt:     "Which endpoint?",
		Actor:      "agent",
		Now:        now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("ask question: %v", err)
	}
	storedRun, ok := state.GetRun(run.ID)
	if !ok || storedRun.Status != RunStateRunning {
		t.Fatalf("run after ask = %#v, ok = %v", storedRun, ok)
	}
	storedItem, ok := state.GetWorkItem(item.ID)
	if !ok || storedItem.RunState != RunStateRunning {
		t.Fatalf("item after ask = %#v, ok = %v", storedItem, ok)
	}
	if _, err := state.AnswerQuestion(AnswerQuestion{
		ID:     question.ID,
		Answer: "Use v2.",
		Actor:  "human",
		Now:    now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("answer question: %v", err)
	}
	storedRun, ok = state.GetRun(run.ID)
	if !ok || storedRun.Status != RunStateRunning {
		t.Fatalf("run after answer = %#v, ok = %v", storedRun, ok)
	}
}

func TestWorkflowRunEffectsArePersistedOnRuns(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	definition := WorkflowDefinition{
		ID:      "run-effect-metadata",
		Version: 1,
		Stages:  []string{StageBacklog, StagePlanning, StageReady, StageExecution},
		Actions: []WorkflowActionDefinition{
			{
				ID:   WorkflowActionStartPlanning,
				From: []string{StageBacklog},
				To:   StagePlanning,
				CreatesRun: &WorkflowRunEffect{
					Phase:            "planning",
					Preset:           RunPresetReader,
					PromptTemplateID: PromptTemplatePlan,
					WorkingDir:       "projectRoot",
				},
			},
			{
				ID:              WorkflowActionSubmitDraftPlan,
				From:            []string{StagePlanning},
				To:              StagePlanning,
				CreatesArtifact: &WorkflowArtifactEffect{Kind: ArtifactKindPlan, Status: ArtifactStatusDraft},
			},
			{
				ID:              WorkflowActionApprovePlan,
				From:            []string{StagePlanning},
				To:              StageReady,
				RequiresHuman:   true,
				UpdatesArtifact: &WorkflowArtifactEffect{Kind: ArtifactKindPlan, Status: ArtifactStatusApproved},
			},
			{
				ID:   WorkflowActionStartExecution,
				From: []string{StageReady},
				To:   StageExecution,
				Requires: []WorkflowArtifactRequirement{
					{Kind: ArtifactKindPlan, Status: ArtifactStatusApproved},
				},
				CreatesRun: &WorkflowRunEffect{
					Phase:                 "execution",
					Preset:                RunPresetWriter,
					PromptTemplateID:      PromptTemplateImplement,
					WorkingDir:            "projectRoot",
					AutoProvisionWorktree: false,
				},
			},
		},
	}
	if _, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{Definition: definition, Source: "test", Now: now}); err != nil {
		t.Fatalf("import workflow: %v", err)
	}
	project := mustProject(t, state, "proj_run_effect", "Run effect")
	if _, err := state.SetProjectWorkflowDefinition(SetProjectWorkflowDefinition{
		ProjectID: project.ID,
		ID:        definition.ID,
		Version:   definition.Version,
		Now:       now,
	}); err != nil {
		t.Fatalf("set workflow: %v", err)
	}
	item := mustWorkItem(t, state, "wi_run_effect", project.ID)

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
	if value := planning.Metadata[RunMetadataWorkflowWorkingDir]; value.Type != MetadataTypeString || value.String != "projectRoot" {
		t.Fatalf("planning working dir metadata = %#v", planning.Metadata)
	}

	draft, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_plan",
		WorkItemID: item.ID,
		RunID:      planning.ID,
		Title:      "Plan",
		Body:       "Implement it.",
		Actor:      "agent",
		Now:        now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("submit draft: %v", err)
	}
	if _, err := state.ApprovePlan(ApprovePlan{
		WorkItemID: item.ID,
		ArtifactID: draft.ID,
		Actor:      "human",
		Now:        now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	execution, err := state.StartExecution(StartExecution{
		ID:           "run_execution",
		HistoryID:    "hist_execution",
		RunHistoryID: "run_hist_execution",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("start execution: %v", err)
	}
	if value := execution.Metadata[RunMetadataWorkflowWorkingDir]; value.Type != MetadataTypeString || value.String != "projectRoot" {
		t.Fatalf("execution working dir metadata = %#v", execution.Metadata)
	}
	if value := execution.Metadata[RunMetadataWorkflowAutoProvisionWorktree]; value.Type != MetadataTypeBool || value.Bool {
		t.Fatalf("execution auto provision metadata = %#v", execution.Metadata)
	}
}

func TestMoveWorkItemRejectsUndefinedWorkflowTransition(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", project.ID)

	if _, err := state.MoveWorkItem(MoveWorkItem{
		ID:        item.ID,
		HistoryID: "hist_move_01",
		StageID:   StageReview,
		Actor:     "agent",
		Now:       now,
	}); err == nil || !strings.Contains(err.Error(), "workflow action from backlog to review not found") {
		t.Fatalf("expected missing workflow action error, got %v", err)
	}
	stored, ok := state.GetWorkItem(item.ID)
	if !ok || stored.StageID != StageBacklog {
		t.Fatalf("stored item = %#v, ok = %v", stored, ok)
	}
}

func TestMoveWorkItemRejectsWorkflowActionRequirements(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", project.ID)
	item.StageID = StageReady
	state.items[item.ID] = item
	if _, err := state.BindWorktree(BindWorktree{
		ID:           item.ID,
		HistoryID:    "hist_bind_01",
		Branch:       "whisk/proj-01-1-task",
		Base:         "main",
		WorktreePath: "/repo/proj_01/.worktrees/task",
		Actor:        "human",
		Now:          now,
	}); err != nil {
		t.Fatalf("bind worktree: %v", err)
	}

	if _, err := state.MoveWorkItem(MoveWorkItem{
		ID:        item.ID,
		HistoryID: "hist_move_01",
		StageID:   StageExecution,
		Actor:     "agent",
		Now:       now.Add(time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "approved plan required") {
		t.Fatalf("expected approved plan requirement, got %v", err)
	}
	stored, ok := state.GetWorkItem(item.ID)
	if !ok || stored.StageID != StageReady {
		t.Fatalf("stored item = %#v, ok = %v", stored, ok)
	}
}

func TestMoveWorkItemRejectsArtifactSelectionWorkflowAction(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	definition := WorkflowDefinition{
		ID:      "direct-move-input",
		Version: 1,
		Stages:  []string{StageBacklog, StagePlanning, "accepted"},
		Actions: []WorkflowActionDefinition{
			{ID: "enter_planning", From: []string{StageBacklog}, To: StagePlanning},
			{
				ID:   WorkflowActionSubmitDraftPlan,
				From: []string{StagePlanning},
				To:   StagePlanning,
				CreatesArtifact: &WorkflowArtifactEffect{
					Kind:   ArtifactKindPlan,
					Status: ArtifactStatusDraft,
				},
			},
			{
				ID:       "accept_plan",
				From:     []string{StagePlanning},
				To:       "accepted",
				Requires: []WorkflowArtifactRequirement{{Kind: ArtifactKindPlan, Status: ArtifactStatusDraft}},
				UpdatesArtifact: &WorkflowArtifactEffect{
					Kind:   ArtifactKindPlan,
					Status: ArtifactStatusApproved,
				},
			},
		},
	}
	project := mustProjectWithWorkflow(t, state, "proj_01", definition, now)
	item := mustWorkItem(t, state, "wi_01", project.ID)
	if _, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   "enter_planning",
		Actor:      "human",
		Now:        now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("enter planning: %v", err)
	}
	draft, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_01",
		WorkItemID: item.ID,
		Title:      "Plan",
		Body:       "Ship it.",
		Actor:      "agent",
		Now:        now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("submit draft: %v", err)
	}

	if _, err := state.MoveWorkItem(MoveWorkItem{
		ID:        item.ID,
		HistoryID: "hist_move_01",
		StageID:   "accepted",
		Actor:     "human",
		Now:       now.Add(3 * time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "workflow action accept_plan requires artifact_selection input") {
		t.Fatalf("expected artifact-selection direct move rejection, got %v", err)
	}
	stored, ok := state.GetWorkItem(item.ID)
	if !ok || stored.StageID != StagePlanning {
		t.Fatalf("stored item = %#v, ok = %v", stored, ok)
	}
	artifacts := state.ListArtifacts(item.ID)
	if len(artifacts) != 1 || artifacts[0].ID != draft.ID || artifacts[0].Status != ArtifactStatusDraft {
		t.Fatalf("artifacts = %#v", artifacts)
	}
}

func TestMoveWorkItemRequiresBlockingGatesToPass(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", project.ID)
	item.StageID = StageReview
	state.items[item.ID] = item
	state.gateReports["gate_01"] = GateReport{
		ID:         "gate_01",
		ProjectID:  project.ID,
		WorkItemID: item.ID,
		Name:       "Review",
		Blocking:   true,
		Status:     GateStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if _, err := state.MoveWorkItem(MoveWorkItem{
		ID:        item.ID,
		HistoryID: "hist_move_01",
		StageID:   StageDone,
		Actor:     "agent",
		Now:       now.Add(time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "blocking gates must pass or be overridden") {
		t.Fatalf("expected blocking gate requirement, got %v", err)
	}
	if _, err := state.CompleteGate(CompleteGate{
		ID:             "gate_01",
		Status:         GateStatusOverridden,
		OverrideReason: "Manual review passed.",
		Actor:          "human",
		Now:            now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("override gate: %v", err)
	}
	moved, err := state.MoveWorkItem(MoveWorkItem{
		ID:        item.ID,
		HistoryID: "hist_move_02",
		StageID:   StageDone,
		Actor:     "human",
		Now:       now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("move after gate override: %v", err)
	}
	if moved.StageID != StageDone {
		t.Fatalf("moved = %#v", moved)
	}
}

func TestMoveWorkItemRunsNoInputWorkflowActionEffects(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", project.ID)
	if _, err := state.StartPlanning(StartPlanning{
		ID:           "run_plan",
		HistoryID:    "hist_plan",
		RunHistoryID: "hist_run_plan",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now,
	}); err != nil {
		t.Fatalf("start planning: %v", err)
	}

	blocked, err := state.MoveWorkItem(MoveWorkItem{
		ID:        item.ID,
		HistoryID: "hist_move_blocked",
		StageID:   StageBlocked,
		Actor:     "agent",
		Now:       now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("move blocked: %v", err)
	}
	if blocked.StageID != StageBlocked ||
		blocked.PreviousStageID != StagePlanning ||
		blocked.RunState != RunStateAwaitingInput ||
		blocked.History[len(blocked.History)-1].Type != HistoryStageMoved {
		t.Fatalf("blocked = %#v", blocked)
	}
	events := state.ListWorkflowEvents(item.ID)
	if len(events) == 0 || events[len(events)-1].Type != WorkflowEventBlocked {
		t.Fatalf("events = %#v", events)
	}
}

func TestApplyWorkflowActionRunsBlockedLifecycle(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", project.ID)
	if _, err := state.StartPlanning(StartPlanning{
		ID:           "run_plan",
		HistoryID:    "hist_plan",
		RunHistoryID: "hist_run_plan",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now,
	}); err != nil {
		t.Fatalf("start planning: %v", err)
	}

	blocked, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   WorkflowActionReportBlocked,
		Reason:     "Waiting for credentials.",
		Actor:      "agent",
		Now:        now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("report blocked action: %v", err)
	}
	if blocked.StageID != StageBlocked || blocked.PreviousStageID != StagePlanning || blocked.RunState != RunStateAwaitingInput {
		t.Fatalf("blocked = %#v", blocked)
	}
	events := state.ListWorkflowEvents(item.ID)
	if len(events) == 0 || events[len(events)-1].Type != WorkflowEventBlocked || events[len(events)-1].Message != "Waiting for credentials." {
		t.Fatalf("events = %#v", events)
	}

	unblocked, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   WorkflowActionUnblock,
		Actor:      "human",
		Now:        now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("unblock action: %v", err)
	}
	if unblocked.StageID != StagePlanning || unblocked.PreviousStageID != "" {
		t.Fatalf("unblocked = %#v", unblocked)
	}
	events = state.ListWorkflowEvents(item.ID)
	if len(events) == 0 || events[len(events)-1].Type != WorkflowEventUnblocked {
		t.Fatalf("events = %#v", events)
	}
	_ = project
}

func TestApplyWorkflowActionRunsCustomNoInputTransition(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	definition := WorkflowDefinition{
		ID:      "custom-actions",
		Version: 1,
		Stages:  []string{StageBacklog, StageReady, StageDone},
		Actions: []WorkflowActionDefinition{
			{ID: "triage", From: []string{StageBacklog}, To: StageReady},
			{ID: "ship", From: []string{StageReady}, To: StageDone},
		},
	}
	if _, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{Definition: definition, Source: "test", Now: now}); err != nil {
		t.Fatalf("import workflow: %v", err)
	}
	project := mustProject(t, state, "proj_01", "One")
	if _, err := state.SetProjectWorkflowDefinition(SetProjectWorkflowDefinition{ProjectID: project.ID, ID: definition.ID, Version: definition.Version, Now: now}); err != nil {
		t.Fatalf("set workflow: %v", err)
	}
	item := mustWorkItem(t, state, "wi_01", project.ID)

	ready, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   "triage",
		Actor:      "human",
		Now:        now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("triage action: %v", err)
	}
	if ready.StageID != StageReady {
		t.Fatalf("ready = %#v", ready)
	}
	if _, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   "missing",
		Actor:      "human",
		Now:        now.Add(2 * time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "workflow action missing not found") {
		t.Fatalf("expected missing action error, got %v", err)
	}
}

func TestApplyWorkflowActionRejectsUnavailableInputAndInvalidRun(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", project.ID)

	if _, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   WorkflowActionApprovePlan,
		Actor:      "human",
		Now:        now,
	}); err == nil || !strings.Contains(err.Error(), "cannot start from backlog") {
		t.Fatalf("expected unavailable action error, got %v", err)
	}
	if _, err := state.StartPlanning(StartPlanning{
		ID:           "run_plan",
		HistoryID:    "hist_plan",
		RunHistoryID: "hist_run_plan",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("start planning: %v", err)
	}
	if _, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   WorkflowActionSubmitDraftPlan,
		Actor:      "agent",
		Now:        now.Add(2 * time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "requires artifact input") {
		t.Fatalf("expected input requirement error, got %v", err)
	}
	if _, err := state.ApplyWorkflowAction(ApplyWorkflowAction{
		WorkItemID: item.ID,
		ActionID:   WorkflowActionReportBlocked,
		RunID:      "missing-run",
		Actor:      "agent",
		Now:        now.Add(3 * time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "work item run missing-run not found") {
		t.Fatalf("expected missing run error, got %v", err)
	}
	stored, ok := state.GetWorkItem(item.ID)
	if !ok || stored.StageID != StagePlanning {
		t.Fatalf("stored = %#v, ok = %v", stored, ok)
	}
}

func TestMoveWorkItemAllowsExplicitWorkflowTransition(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	definition := WorkflowDefinition{
		ID:      "simple",
		Version: 1,
		Stages:  []string{StageBacklog, StageDone},
		Actions: []WorkflowActionDefinition{{ID: "finish", From: []string{StageBacklog}, To: StageDone}},
	}
	if _, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{
		Definition: definition,
		Source:     "test",
		Now:        now,
	}); err != nil {
		t.Fatalf("import workflow: %v", err)
	}
	project, err := state.CreateProject(CreateProject{
		ID:      "proj_01",
		Name:    "One",
		RootDir: "/repo/proj_01",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := state.SetProjectWorkflowDefinition(SetProjectWorkflowDefinition{
		ProjectID: project.ID,
		ID:        definition.ID,
		Version:   definition.Version,
		Now:       now,
	}); err != nil {
		t.Fatalf("set project workflow: %v", err)
	}
	item := mustWorkItem(t, state, "wi_01", project.ID)

	moved, err := state.MoveWorkItem(MoveWorkItem{
		ID:        item.ID,
		HistoryID: "hist_move_01",
		StageID:   StageDone,
		Actor:     "human",
		Now:       now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if moved.StageID != StageDone || moved.History[len(moved.History)-1].Type != HistoryStageMoved {
		t.Fatalf("moved = %#v", moved)
	}
}

func TestMoveWorkItemRejectsAmbiguousWorkflowTransition(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	definition := WorkflowDefinition{
		ID:      "ambiguous",
		Version: 1,
		Stages:  []string{StageBacklog, StageDone},
		Actions: []WorkflowActionDefinition{
			{ID: "archive", From: []string{StageBacklog}, To: StageDone},
			{ID: "ship", From: []string{StageBacklog}, To: StageDone},
		},
	}
	if _, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{
		Definition: definition,
		Source:     "test",
		Now:        now,
	}); err != nil {
		t.Fatalf("import workflow: %v", err)
	}
	project := mustProject(t, state, "proj_ambiguous", "Ambiguous")
	if _, err := state.SetProjectWorkflowDefinition(SetProjectWorkflowDefinition{
		ProjectID: project.ID,
		ID:        definition.ID,
		Version:   definition.Version,
		Now:       now,
	}); err != nil {
		t.Fatalf("set project workflow: %v", err)
	}
	item := mustWorkItem(t, state, "wi_ambiguous", project.ID)

	if _, err := state.MoveWorkItem(MoveWorkItem{
		ID:        item.ID,
		HistoryID: "hist_move_ambiguous",
		StageID:   StageDone,
		Actor:     "human",
		Now:       now.Add(time.Minute),
	}); err == nil || !strings.Contains(err.Error(), "ambiguous workflow action from backlog to done") ||
		!strings.Contains(err.Error(), "archive") ||
		!strings.Contains(err.Error(), "ship") {
		t.Fatalf("expected ambiguous transition error naming candidates, got %v", err)
	}
	stored, ok := state.GetWorkItem(item.ID)
	if !ok || stored.StageID != StageBacklog {
		t.Fatalf("stored after ambiguous move = %#v, ok = %v", stored, ok)
	}
}

func TestWorkflowDefinitionValidationReportListsMultipleErrors(t *testing.T) {
	definition := DefaultWorkflowDefinition()
	definition.ID = ""
	definition.Actions[0].From = []string{"missing"}
	definition.Actions[1].CreatesArtifact = &WorkflowArtifactEffect{Kind: "bad", Status: ArtifactStatusDraft}

	report := ValidateWorkflowDefinitionReport(definition)
	if report.Valid {
		t.Fatalf("report should be invalid: %#v", report)
	}
	if len(report.Errors) < 3 {
		t.Fatalf("expected multiple errors, got %#v", report.Errors)
	}
	if !containsWorkflowValidationError(report.Errors, "workflow id required") ||
		!containsWorkflowValidationError(report.Errors, "unknown stage missing") ||
		!containsWorkflowValidationError(report.Errors, "unsupported artifact kind bad") {
		t.Fatalf("errors = %#v", report.Errors)
	}
}

func TestProjectWorkflowMigrationPlanKeepsExistingItemsPinnedAndReportsCompatibility(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	backlog := mustWorkItem(t, state, "wi_backlog", project.ID)
	planning := mustWorkItem(t, state, "wi_planning", project.ID)
	if _, err := state.StartPlanning(StartPlanning{
		ID:           "run_01",
		HistoryID:    "hist_run_01",
		RunHistoryID: "run_hist_01",
		WorkItemID:   planning.ID,
		Now:          now,
	}); err != nil {
		t.Fatalf("start planning: %v", err)
	}

	definition := WorkflowDefinition{
		ID:      "lean",
		Version: 1,
		Stages:  []string{StageBacklog, StageReady, StageDone},
		Actions: []WorkflowActionDefinition{{ID: "ship", From: []string{StageBacklog}, To: StageDone}},
		Questions: WorkflowQuestionPolicy{
			Enabled: true,
		},
	}
	if _, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{
		Definition: definition,
		Source:     "file",
		SourcePath: "/tmp/lean.json",
		Now:        now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("import lean workflow: %v", err)
	}

	plan, err := state.PlanProjectWorkflowMigration(PlanProjectWorkflowMigration{
		ProjectID: project.ID,
		ID:        definition.ID,
		Version:   definition.Version,
	})
	if err != nil {
		t.Fatalf("migration plan: %v", err)
	}
	if plan.ProjectID != project.ID || plan.TargetID != definition.ID || plan.ExistingItems != 2 || plan.ItemsPinnedToCurrentVersion != 2 {
		t.Fatalf("plan summary = %#v", plan)
	}
	if got := migrationItemByID(plan.Items, backlog.ID); got == nil || !got.Compatible || got.TargetStageID != StageBacklog {
		t.Fatalf("backlog migration item = %#v", got)
	}
	if got := migrationItemByID(plan.Items, planning.ID); got == nil || got.Compatible || !strings.Contains(got.Reason, "stage planning not present") {
		t.Fatalf("planning migration item = %#v", got)
	}
}

func TestDeleteWorkflowDefinitionRejectsActiveOrPinnedDefinitions(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", project.ID)

	if _, err := state.DeleteWorkflowDefinition(DeleteWorkflowDefinition{
		ID:      item.WorkflowID,
		Version: item.WorkflowVersion,
	}); err == nil || !strings.Contains(err.Error(), "used by work item") {
		t.Fatalf("expected pinned definition delete failure, got %v", err)
	}

	definition := WorkflowDefinition{
		ID:      "scratch",
		Version: 1,
		Stages:  []string{StageBacklog, StageDone},
		Actions: []WorkflowActionDefinition{{ID: "finish", From: []string{StageBacklog}, To: StageDone}},
	}
	record, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{
		Definition: definition,
		Source:     "file",
		SourcePath: "/tmp/scratch.json",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("import scratch: %v", err)
	}
	deleted, err := state.DeleteWorkflowDefinition(DeleteWorkflowDefinition{ID: record.ID, Version: record.Version})
	if err != nil {
		t.Fatalf("delete unused definition: %v", err)
	}
	if deleted.ID != record.ID || deleted.Version != record.Version {
		t.Fatalf("deleted = %#v", deleted)
	}
	if _, ok := state.WorkflowDefinition(record.ID, record.Version); ok {
		t.Fatalf("definition still present after delete")
	}
}

func availabilityByID(actions []WorkflowActionAvailability, id string) *WorkflowActionAvailability {
	for i := range actions {
		if actions[i].Action.ID == id {
			return &actions[i]
		}
	}
	return nil
}

func recommendedAvailabilityIDs(actions []WorkflowActionAvailability) []string {
	out := []string{}
	for _, action := range actions {
		if action.Recommended {
			out = append(out, action.Action.ID)
		}
	}
	return out
}

func mustReviewWorkItemWithPendingGate(t *testing.T, state *State, now time.Time) (WorkItem, GateReport) {
	t.Helper()
	project := mustProject(t, state, "proj_review_gate", "Review Gate")
	item := mustWorkItem(t, state, "wi_review_gate", project.ID)
	planningRun, err := state.StartPlanning(StartPlanning{
		ID:           "run_review_plan",
		HistoryID:    "hist_review_plan",
		RunHistoryID: "run_hist_review_plan",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	draft, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_review_plan",
		WorkItemID: item.ID,
		RunID:      planningRun.ID,
		Title:      "Plan",
		Body:       "Do it.",
		Actor:      "agent",
		Now:        now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("submit draft: %v", err)
	}
	if _, err := state.ApprovePlan(ApprovePlan{
		WorkItemID: item.ID,
		ArtifactID: draft.ID,
		Actor:      "human",
		Now:        now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	executionRun, err := state.StartExecution(StartExecution{
		ID:           "run_review_exec",
		HistoryID:    "hist_review_exec",
		RunHistoryID: "run_hist_review_exec",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("start execution: %v", err)
	}
	item, err = state.CompleteExecution(CompleteExecution{
		RunID:   executionRun.ID,
		Actor:   "agent",
		Message: "ready for review",
		Now:     now.Add(4 * time.Minute),
	})
	if err != nil {
		t.Fatalf("complete execution: %v", err)
	}
	gates := state.ListGateReports(item.ID)
	if len(gates) != 1 {
		t.Fatalf("gates = %#v", gates)
	}
	return item, gates[0]
}

func containsWorkflowValidationError(errors []WorkflowValidationError, message string) bool {
	for _, item := range errors {
		if strings.Contains(item.Message, message) {
			return true
		}
	}
	return false
}

func migrationItemByID(items []WorkflowMigrationItem, id string) *WorkflowMigrationItem {
	for i := range items {
		if items[i].WorkItemID == id {
			return &items[i]
		}
	}
	return nil
}
