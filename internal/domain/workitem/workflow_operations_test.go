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
		Actor:     "agent",
		Now:       now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("move after gate override: %v", err)
	}
	if moved.StageID != StageDone {
		t.Fatalf("moved = %#v", moved)
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
