package workitem

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStateCreatesProjectWithCopiedDefaultWorkflow(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

	project, err := state.CreateProject(CreateProject{
		ID:      "proj_01",
		Name:    "My App",
		RootDir: "/repo/my-app",
		Now:     now,
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	if project.Slug != "my-app" || project.NextWorkItemNumber != 1 {
		t.Fatalf("project = %#v", project)
	}
	if project.Workflow.TemplateID != "default" || len(project.Workflow.Stages) != 7 {
		t.Fatalf("workflow = %#v", project.Workflow)
	}
	if project.Workflow.Stages[2].ID != "ready" || project.Workflow.Stages[2].ProvisionWorktree {
		t.Fatalf("ready stage = %#v", project.Workflow.Stages[2])
	}
	if project.Workflow.Stages[3].ID != "execution" || !project.Workflow.Stages[3].ProvisionWorktree {
		t.Fatalf("execution stage = %#v", project.Workflow.Stages[3])
	}
}

func TestStateCreatesProjectWithCopiedPreferences(t *testing.T) {
	state := NewState()
	agents := map[string]string{StagePlanning: "codex"}
	gates := []GateConfig{{ID: "gate_01", Name: "Review", Kind: "manual", Blocking: true, Phase: StageReview}}
	project, err := state.CreateProject(CreateProject{
		ID:      "proj_01",
		Name:    "My App",
		RootDir: "/repo/my-app",
		Preferences: ProjectPreferences{
			AutoRun:                  AutoRunAll,
			AutoWorktree:             true,
			UseInteractiveAgentShell: true,
			DefaultPhaseAgents:       agents,
			Gates:                    gates,
		},
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	agents[StagePlanning] = "mutated"
	gates[0].Name = "Mutated"
	if project.Preferences.AutoRun != AutoRunAll || !project.Preferences.AutoWorktree {
		t.Fatalf("preferences = %#v", project.Preferences)
	}
	if !project.Preferences.UseInteractiveAgentShell {
		t.Fatalf("interactive agent shell preference was not copied: %#v", project.Preferences)
	}
	if project.Preferences.DefaultPhaseAgents[StagePlanning] != "codex" || project.Preferences.Gates[0].Name != "Review" {
		t.Fatalf("preferences were not copied: %#v", project.Preferences)
	}
}

func TestStateUpdatesProjectDescription(t *testing.T) {
	state := NewState()
	project := mustProject(t, state, "proj_01", "One")
	name := "Updated"
	description := "Owns daemon project editing"

	updated, err := state.UpdateProject(UpdateProject{
		ID:          project.ID,
		Name:        &name,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("update project: %v", err)
	}
	if updated.Name != "Updated" || updated.Description != description || updated.RootDir != project.RootDir {
		t.Fatalf("updated project = %#v", updated)
	}

	clear := ""
	updated, err = state.UpdateProject(UpdateProject{ID: project.ID, Description: &clear})
	if err != nil {
		t.Fatalf("clear description: %v", err)
	}
	if updated.Description != "" {
		t.Fatalf("description = %q", updated.Description)
	}
}

func TestStateUpdatesProjectInteractiveAgentShellPreference(t *testing.T) {
	state := NewState()
	project := mustProject(t, state, "proj_01", "One")
	enabled := true

	updated, err := state.UpdateProject(UpdateProject{ID: project.ID, UseInteractiveAgentShell: &enabled})
	if err != nil {
		t.Fatalf("enable interactive agent shell: %v", err)
	}
	if !updated.Preferences.UseInteractiveAgentShell {
		t.Fatalf("preferences = %#v", updated.Preferences)
	}

	enabled = false
	updated, err = state.UpdateProject(UpdateProject{ID: project.ID, UseInteractiveAgentShell: &enabled})
	if err != nil {
		t.Fatalf("disable interactive agent shell: %v", err)
	}
	if updated.Preferences.UseInteractiveAgentShell {
		t.Fatalf("preferences = %#v", updated.Preferences)
	}
}

func TestStateManagesProjectAttachments(t *testing.T) {
	state := NewState()
	project := mustProject(t, state, "proj_01", "One")
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	updated, err := state.AddProjectAttachment(AddProjectAttachment{
		ID:               "att_01",
		ProjectID:        project.ID,
		Kind:             AttachmentKindExternal,
		Provider:         "github",
		Target:           "phin-tech/roux-next-gen#123",
		Title:            "Issue 123",
		IncludeInContext: true,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("add attachment: %v", err)
	}
	if len(updated.Attachments) != 1 || updated.Attachments[0].Provider != "github" || !updated.Attachments[0].IncludeInContext {
		t.Fatalf("attachments = %#v", updated.Attachments)
	}

	title := "Updated issue"
	include := false
	updated, err = state.UpdateProjectAttachment(UpdateProjectAttachment{
		ID:               "att_01",
		ProjectID:        project.ID,
		Title:            &title,
		IncludeInContext: &include,
		Now:              now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("update attachment: %v", err)
	}
	if updated.Attachments[0].Title != title || updated.Attachments[0].IncludeInContext {
		t.Fatalf("updated attachment = %#v", updated.Attachments[0])
	}

	updated, err = state.DeleteProjectAttachment(DeleteProjectAttachment{ID: "att_01", ProjectID: project.ID, Now: now.Add(2 * time.Minute)})
	if err != nil {
		t.Fatalf("delete attachment: %v", err)
	}
	if len(updated.Attachments) != 0 {
		t.Fatalf("attachments = %#v", updated.Attachments)
	}
}

func TestStateCreatesWorkItemsWithPerProjectNumbersAndHistory(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	mustProject(t, state, "proj_02", "Two")

	first, err := state.CreateWorkItem(CreateWorkItem{
		ID:           "wi_01",
		HistoryID:    "hist_01",
		ProjectID:    "proj_01",
		Title:        "Implement login",
		BodyMarkdown: "Body",
		Actor:        "user",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	second, err := state.CreateWorkItem(CreateWorkItem{
		ID:        "wi_02",
		HistoryID: "hist_02",
		ProjectID: "proj_01",
		Title:     "Fix tests",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	otherProject, err := state.CreateWorkItem(CreateWorkItem{
		ID:        "wi_03",
		HistoryID: "hist_03",
		ProjectID: "proj_02",
		Title:     "Other",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("create other: %v", err)
	}

	if first.Number != 1 || second.Number != 2 || otherProject.Number != 1 {
		t.Fatalf("numbers = %d, %d, %d", first.Number, second.Number, otherProject.Number)
	}
	if first.StageID != "backlog" || first.RunState != RunStateIdle {
		t.Fatalf("first = %#v", first)
	}
	if len(first.History) != 1 || first.History[0].Type != HistoryCreated || first.History[0].Actor != "user" {
		t.Fatalf("history = %#v", first.History)
	}
	project, _ := state.GetProject("proj_01")
	if project.NextWorkItemNumber != 3 {
		t.Fatalf("next number = %d", project.NextWorkItemNumber)
	}
}

func TestStateUpdatesWorkItemTitleAndBody(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item, err := state.CreateWorkItem(CreateWorkItem{
		ID:           "wi_01",
		HistoryID:    "hist_01",
		ProjectID:    "proj_01",
		Title:        "Old title",
		BodyMarkdown: "Old body",
		Actor:        "user",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	title := "  New title  "
	body := ""

	updated, err := state.UpdateWorkItem(UpdateWorkItem{
		ID:           item.ID,
		HistoryID:    "hist_02",
		Title:        &title,
		BodyMarkdown: &body,
		Actor:        "human",
		Now:          now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("update work item: %v", err)
	}
	if updated.Title != "New title" || updated.BodyMarkdown != "" || !updated.UpdatedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("updated item = %#v", updated)
	}
	if len(updated.History) != 2 || updated.History[1].Type != HistoryUpdated || updated.History[1].Actor != "human" {
		t.Fatalf("history = %#v", updated.History)
	}
	stored, ok := state.GetWorkItem(item.ID)
	if !ok || stored.Title != "New title" || stored.BodyMarkdown != "" {
		t.Fatalf("stored item = %#v, ok = %v", stored, ok)
	}
}

func TestStateRejectsInvalidWorkItemUpdates(t *testing.T) {
	state := NewState()
	mustProject(t, state, "proj_01", "One")
	item, err := state.CreateWorkItem(CreateWorkItem{
		ID:        "wi_01",
		HistoryID: "hist_01",
		ProjectID: "proj_01",
		Title:     "Task",
	})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	if _, err := state.UpdateWorkItem(UpdateWorkItem{ID: item.ID, HistoryID: "hist_02"}); err == nil {
		t.Fatalf("expected no-op update error")
	}
	blank := " "
	if _, err := state.UpdateWorkItem(UpdateWorkItem{ID: item.ID, HistoryID: "hist_03", Title: &blank}); err == nil {
		t.Fatalf("expected blank title error")
	}
}

func TestMoveToExecutionRequiresWorktreeAndBindWorktreeAllowsMove(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	definition := WorkflowDefinition{
		ID:      "worktree-only",
		Version: 1,
		Stages:  []string{StageBacklog, StageExecution},
		Actions: []WorkflowActionDefinition{{ID: "start", From: []string{StageBacklog}, To: StageExecution}},
	}
	if _, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{
		Definition: definition,
		Source:     "test",
		Now:        now,
	}); err != nil {
		t.Fatalf("import workflow: %v", err)
	}
	project := mustProject(t, state, "proj_01", "One")
	if _, err := state.SetProjectWorkflowDefinition(SetProjectWorkflowDefinition{
		ProjectID: project.ID,
		ID:        definition.ID,
		Version:   definition.Version,
		Now:       now,
	}); err != nil {
		t.Fatalf("set project workflow: %v", err)
	}
	item := mustWorkItem(t, state, "wi_01", project.ID)

	if _, err := state.MoveWorkItem(MoveWorkItem{
		ID:        item.ID,
		HistoryID: "hist_move_01",
		StageID:   "execution",
		Now:       now,
	}); err == nil || !strings.Contains(err.Error(), "requires worktree") {
		t.Fatalf("expected worktree requirement, got %v", err)
	}

	bound, err := state.BindWorktree(BindWorktree{
		ID:           item.ID,
		HistoryID:    "hist_bind_01",
		Branch:       "whisk/my-app-1-login",
		Base:         "main",
		WorktreePath: "/repo/.worktrees/login",
		Actor:        "user",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("bind worktree: %v", err)
	}
	if bound.Worktree == nil || bound.Worktree.Branch != "whisk/my-app-1-login" {
		t.Fatalf("bound = %#v", bound)
	}

	moved, err := state.MoveWorkItem(MoveWorkItem{
		ID:        item.ID,
		HistoryID: "hist_move_02",
		StageID:   "execution",
		Actor:     "user",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if moved.StageID != "execution" || len(moved.History) != 3 {
		t.Fatalf("moved = %#v", moved)
	}
}

func TestCreateWorkItemInProvisionedStageRequiresWorktree(t *testing.T) {
	state := NewState()
	mustProject(t, state, "proj_01", "One")

	_, err := state.CreateWorkItem(CreateWorkItem{
		ID:        "wi_01",
		HistoryID: "hist_01",
		ProjectID: "proj_01",
		Title:     "Execution item",
		StageID:   "execution",
	})
	if err == nil || !strings.Contains(err.Error(), "requires worktree") {
		t.Fatalf("expected worktree requirement, got %v", err)
	}
}

func TestAddAttachmentValidatesScopeAndUsesProjectRelativePaths(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")

	updated, err := state.AddAttachment(AddAttachment{
		ID:         "att_01",
		HistoryID:  "hist_att_01",
		WorkItemID: item.ID,
		Kind:       AttachmentKindFile,
		Path:       "docs/../docs/spec.md",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("add attachment: %v", err)
	}
	if len(updated.Attachments) != 1 || updated.Attachments[0].Scope != AttachmentScopeProject || updated.Attachments[0].Path != filepath.Clean("docs/spec.md") {
		t.Fatalf("attachments = %#v", updated.Attachments)
	}

	_, err = state.AddAttachment(AddAttachment{
		ID:         "att_02",
		HistoryID:  "hist_att_02",
		WorkItemID: item.ID,
		Kind:       AttachmentKindFile,
		Path:       "/etc/passwd",
		Now:        now,
	})
	if err == nil || !strings.Contains(err.Error(), "external scope") {
		t.Fatalf("expected absolute path error, got %v", err)
	}
}

func TestDeleteWorkItemRemovesItemButReturnsHistory(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")

	deleted, err := state.DeleteWorkItem(DeleteWorkItem{
		ID:        item.ID,
		HistoryID: "hist_delete_01",
		Actor:     "user",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(deleted.History) != 2 || deleted.History[1].Type != HistoryDeleted {
		t.Fatalf("deleted history = %#v", deleted.History)
	}
	if _, ok := state.GetWorkItem(item.ID); ok {
		t.Fatalf("work item still exists")
	}
	project, _ := state.GetProject("proj_01")
	if project.NextWorkItemNumber != 2 {
		t.Fatalf("number was reused or reset: %d", project.NextWorkItemNumber)
	}
}

func TestDeleteWorkItemRemovesOwnedWorkflowRecords(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")
	other := mustWorkItem(t, state, "wi_02", "proj_01")

	run, err := state.StartPlanning(StartPlanning{
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
		t.Fatalf("start planning: %v", err)
	}
	if _, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_01",
		WorkItemID: item.ID,
		RunID:      run.ID,
		Title:      "Plan",
		Body:       "Test then implement.",
		Actor:      "agent",
		Now:        now,
	}); err != nil {
		t.Fatalf("submit plan: %v", err)
	}
	if _, err := state.AskQuestion(AskQuestion{
		ID:         "question_01",
		WorkItemID: item.ID,
		RunID:      run.ID,
		Prompt:     "Which branch?",
		Actor:      "agent",
		Now:        now,
	}); err != nil {
		t.Fatalf("ask question: %v", err)
	}
	if _, err := state.ReportStatus(ReportStatus{
		ID:           "status_01",
		RunHistoryID: "run_hist_status_01",
		Kind:         StatusKindQuestion,
		Message:      "Need input.",
		Actor:        "agent",
		RunID:        run.ID,
		Now:          now,
	}); err != nil {
		t.Fatalf("report status: %v", err)
	}
	if _, err := state.StartPlanning(StartPlanning{
		ID:           "run_other",
		HistoryID:    "hist_run_other",
		RunHistoryID: "run_hist_other",
		WorkItemID:   other.ID,
		Actor:        "agent",
		Now:          now,
	}); err != nil {
		t.Fatalf("start other planning: %v", err)
	}

	if _, err := state.DeleteWorkItem(DeleteWorkItem{
		ID:        item.ID,
		HistoryID: "hist_delete_01",
		Actor:     "user",
		Now:       now,
	}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if got := state.ListRuns(item.ID); len(got) != 0 {
		t.Fatalf("runs remain = %#v", got)
	}
	if got := state.ListArtifacts(item.ID); len(got) != 0 {
		t.Fatalf("artifacts remain = %#v", got)
	}
	if got := state.ListQuestions(item.ID); len(got) != 0 {
		t.Fatalf("questions remain = %#v", got)
	}
	if got := state.ListWorkflowEvents(item.ID); len(got) != 0 {
		t.Fatalf("workflow events remain = %#v", got)
	}
	if got := state.ListStatusEvents(ListStatusEvents{WorkItemID: item.ID}); len(got) != 0 {
		t.Fatalf("status events remain = %#v", got)
	}
	if got := state.ListRuns(other.ID); len(got) != 1 || got[0].ID != "run_other" {
		t.Fatalf("other runs = %#v", got)
	}
}

func TestDeleteProjectCascadesOwnedRecordsOnly(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	otherProject := mustProject(t, state, "proj_02", "Two")
	item := mustWorkItem(t, state, "wi_01", project.ID)
	other := mustWorkItem(t, state, "wi_02", otherProject.ID)
	if _, err := state.AddProjectAttachment(AddProjectAttachment{
		ID:        "att_01",
		ProjectID: project.ID,
		Kind:      AttachmentKindNote,
		Note:      "delete with project",
		Now:       now,
	}); err != nil {
		t.Fatalf("add attachment: %v", err)
	}
	run, err := state.StartPlanning(StartPlanning{
		ID:           "run_01",
		HistoryID:    "hist_run_01",
		RunHistoryID: "run_hist_01",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	if _, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_01",
		WorkItemID: item.ID,
		RunID:      run.ID,
		Title:      "Plan",
		Body:       "Test then implement.",
		Actor:      "agent",
		Now:        now,
	}); err != nil {
		t.Fatalf("submit plan: %v", err)
	}
	if _, err := state.AskQuestion(AskQuestion{
		ID:         "question_01",
		WorkItemID: item.ID,
		RunID:      run.ID,
		Prompt:     "Which branch?",
		Actor:      "agent",
		Now:        now,
	}); err != nil {
		t.Fatalf("ask question: %v", err)
	}
	if _, err := state.ReportStatus(ReportStatus{
		ID:           "status_01",
		RunHistoryID: "run_hist_status_01",
		Kind:         StatusKindQuestion,
		Message:      "Need input.",
		Actor:        "agent",
		RunID:        run.ID,
		Now:          now,
	}); err != nil {
		t.Fatalf("report status: %v", err)
	}
	if _, err := state.StartPlanning(StartPlanning{
		ID:           "run_other",
		HistoryID:    "hist_run_other",
		RunHistoryID: "run_hist_other",
		WorkItemID:   other.ID,
		Actor:        "agent",
		Now:          now,
	}); err != nil {
		t.Fatalf("start other planning: %v", err)
	}

	deleted, err := state.DeleteProject(DeleteProject{ID: project.ID, Actor: "user", Now: now})
	if err != nil {
		t.Fatalf("delete project: %v", err)
	}
	if deleted.ID != project.ID {
		t.Fatalf("deleted project = %#v", deleted)
	}
	if _, ok := state.GetProject(project.ID); ok {
		t.Fatalf("project still exists")
	}
	if got := state.ListWorkItems(project.ID); len(got) != 0 {
		t.Fatalf("project items remain = %#v", got)
	}
	if got := state.ListRuns(item.ID); len(got) != 0 {
		t.Fatalf("project runs remain = %#v", got)
	}
	if got := state.ListArtifacts(item.ID); len(got) != 0 {
		t.Fatalf("project artifacts remain = %#v", got)
	}
	if got := state.ListQuestions(item.ID); len(got) != 0 {
		t.Fatalf("project questions remain = %#v", got)
	}
	if got := state.ListWorkflowEvents(item.ID); len(got) != 0 {
		t.Fatalf("project workflow events remain = %#v", got)
	}
	if got := state.ListStatusEvents(ListStatusEvents{ProjectID: project.ID}); len(got) != 0 {
		t.Fatalf("project status events remain = %#v", got)
	}
	if got := state.ListWorkItems(otherProject.ID); len(got) != 1 || got[0].ID != other.ID {
		t.Fatalf("other project items = %#v", got)
	}
	if got := state.ListRuns(other.ID); len(got) != 1 || got[0].ID != "run_other" {
		t.Fatalf("other project runs = %#v", got)
	}
}

func TestPlanPromptTellsAgentToExitPlanModeForDraftPlanReview(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")

	run, err := state.StartPlanning(StartPlanning{
		ID:           "run_01",
		HistoryID:    "hist_run_01",
		RunHistoryID: "run_hist_01",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	if !strings.Contains(run.PromptSnapshot, "call ExitPlanMode with the full plan markdown") ||
		!strings.Contains(run.PromptSnapshot, "Whisk will submit that plan for review") ||
		!strings.Contains(run.PromptSnapshot, "deny continuation") ||
		!strings.Contains(run.PromptSnapshot, "fallback CLI callback") ||
		!strings.Contains(run.PromptSnapshot, "${WHISK_CLI:-whisk} workflow submit-plan") ||
		!strings.Contains(run.PromptSnapshot, "-body '<plan markdown>'") ||
		!strings.Contains(run.PromptSnapshot, "If an external plan review tool opens") ||
		!strings.Contains(run.PromptSnapshot, "Do not submit drafts to Whisk before external plan approval") ||
		!strings.Contains(run.PromptSnapshot, "ExitPlanMode approval does not authorize implementation") ||
		!strings.Contains(run.PromptSnapshot, "Do not write files, edit code, run tests, install dependencies, or begin implementation in this stage") ||
		!strings.Contains(run.PromptSnapshot, "Do not treat the plan as complete until Whisk confirms it was submitted for review") {
		t.Fatalf("prompt snapshot = %q", run.PromptSnapshot)
	}
}

func TestSnapshotLoadRefreshesBuiltinPromptTemplates(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	base := NewState()
	project := mustProject(t, base, "proj_01", "One")
	oldPlan := PromptTemplate{
		ID:        PromptTemplatePlan,
		Name:      "Plan",
		Source:    "builtin",
		Body:      "Plan the work item.",
		CreatedAt: now,
		UpdatedAt: now,
	}
	custom := PromptTemplate{
		ID:        "custom",
		Name:      "Custom",
		Source:    "user",
		Body:      "Keep this custom prompt.",
		CreatedAt: now,
		UpdatedAt: now,
	}

	restored, err := NewStateFromSnapshot(Snapshot{
		Projects:        []Project{project},
		PromptTemplates: []PromptTemplate{oldPlan, custom},
	})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	templates := restored.ListPromptTemplates()
	byID := map[string]PromptTemplate{}
	for _, template := range templates {
		byID[template.ID] = template
	}
	if !strings.Contains(byID[PromptTemplatePlan].Body, "call ExitPlanMode with the full plan markdown") ||
		!strings.Contains(byID[PromptTemplatePlan].Body, "Whisk will submit that plan for review") ||
		!strings.Contains(byID[PromptTemplatePlan].Body, "fallback CLI callback") ||
		!strings.Contains(byID[PromptTemplatePlan].Body, "${WHISK_CLI:-whisk} workflow submit-plan") ||
		!strings.Contains(byID[PromptTemplatePlan].Body, "If an external plan review tool opens") ||
		!strings.Contains(byID[PromptTemplatePlan].Body, "Do not submit drafts to Whisk before external plan approval") ||
		!strings.Contains(byID[PromptTemplatePlan].Body, "ExitPlanMode approval does not authorize implementation") ||
		!strings.Contains(byID[PromptTemplatePlan].Body, "Do not write files, edit code, run tests, install dependencies, or begin implementation in this stage") {
		t.Fatalf("plan template = %q", byID[PromptTemplatePlan].Body)
	}
	if byID["custom"].Body != custom.Body {
		t.Fatalf("custom template = %q", byID["custom"].Body)
	}
}

func TestSnapshotLoadDropsWorkflowRecordsForMissingWorkItems(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	base := NewState()
	project := mustProject(t, base, "proj_01", "One")
	snapshot := Snapshot{
		Projects: []Project{project},
		Runs: []WorkItemRun{{
			ID:               "run_orphan",
			WorkItemID:       "wi_missing",
			ProjectID:        project.ID,
			Preset:           RunPresetReader,
			PromptTemplateID: PromptTemplatePlan,
			PromptSnapshot:   "Plan.",
			Status:           RunStateRunning,
			CreatedAt:        now,
			UpdatedAt:        now,
			History: []RunEvent{{
				ID:   "run_hist_01",
				Type: RunStateRunning,
				At:   now,
			}},
		}},
		Artifacts: []Artifact{{
			ID:         "artifact_orphan",
			ProjectID:  project.ID,
			WorkItemID: "wi_missing",
			Kind:       ArtifactKindPlan,
			Status:     ArtifactStatusDraft,
			Body:       "Plan.",
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		Questions: []Question{{
			ID:         "question_orphan",
			ProjectID:  project.ID,
			WorkItemID: "wi_missing",
			Prompt:     "Question?",
			Status:     QuestionStatusOpen,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		GateReports: []GateReport{{
			ID:         "gate_orphan",
			ProjectID:  project.ID,
			WorkItemID: "wi_missing",
			Name:       "review",
			Status:     GateStatusPending,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		WorkflowEvents: []WorkflowEvent{{
			ID:         "workflow_event_orphan",
			ProjectID:  project.ID,
			WorkItemID: "wi_missing",
			Type:       WorkflowEventPlanningStarted,
			At:         now,
		}},
		StatusEvents: []StatusEvent{{
			ID:         "status_orphan",
			Scope:      StatusScopeRun,
			Kind:       StatusKindQuestion,
			Message:    "Need input.",
			ProjectID:  project.ID,
			WorkItemID: "wi_missing",
			RunID:      "run_orphan",
			CreatedAt:  now,
		}},
	}

	restored, err := NewStateFromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := restored.ListRuns(""); len(got) != 0 {
		t.Fatalf("runs = %#v", got)
	}
	if got := restored.ListArtifacts(""); len(got) != 0 {
		t.Fatalf("artifacts = %#v", got)
	}
	if got := restored.ListQuestions(""); len(got) != 0 {
		t.Fatalf("questions = %#v", got)
	}
	if got := restored.ListGateReports(""); len(got) != 0 {
		t.Fatalf("gates = %#v", got)
	}
	if got := restored.ListWorkflowEvents(""); len(got) != 0 {
		t.Fatalf("events = %#v", got)
	}
	if got := restored.ListStatusEvents(ListStatusEvents{}); len(got) != 0 {
		t.Fatalf("status events = %#v", got)
	}
}

func TestListsAreDeterministic(t *testing.T) {
	state := NewState()
	mustProject(t, state, "proj_b", "B")
	mustProject(t, state, "proj_a", "A")
	mustWorkItem(t, state, "wi_b2", "proj_b")
	mustWorkItem(t, state, "wi_a1", "proj_a")
	mustWorkItem(t, state, "wi_b1", "proj_b")

	projects := state.ListProjects()
	if len(projects) != 2 || projects[0].ID != "proj_a" || projects[1].ID != "proj_b" {
		t.Fatalf("projects = %#v", projects)
	}
	items := state.ListWorkItems("")
	if len(items) != 3 || items[0].ID != "wi_a1" || items[1].ID != "wi_b2" || items[2].ID != "wi_b1" {
		t.Fatalf("items = %#v", items)
	}
}

func TestWorkflowRecordListsFilterAndSort(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")
	other := mustWorkItem(t, state, "wi_02", "proj_01")

	run, err := state.StartRun(StartRun{
		ID:           "run_01",
		HistoryID:    "hist_run_01",
		RunHistoryID: "run_hist_01",
		WorkItemID:   item.ID,
		Actor:        "agent",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	otherRun, err := state.StartRun(StartRun{
		ID:           "run_02",
		HistoryID:    "hist_run_02",
		RunHistoryID: "run_hist_02",
		WorkItemID:   other.ID,
		Actor:        "agent",
		Now:          now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("start other run: %v", err)
	}
	if _, err := state.AskQuestion(AskQuestion{
		ID:         "question_01",
		WorkItemID: item.ID,
		RunID:      run.ID,
		Prompt:     "Which branch?",
		Actor:      "agent",
		Now:        now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("ask question: %v", err)
	}
	if _, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_01",
		WorkItemID: item.ID,
		RunID:      run.ID,
		Body:       "Plan",
		Actor:      "agent",
		Now:        now.Add(3 * time.Minute),
	}); err != nil {
		t.Fatalf("submit draft: %v", err)
	}
	if _, err := state.ReportStatus(ReportStatus{
		ID:           "status_01",
		RunHistoryID: "run_hist_status_01",
		Kind:         StatusKindQuestion,
		ProjectID:    "proj_01",
		WorkItemID:   item.ID,
		RunID:        run.ID,
		Message:      "Need input",
		Actor:        "agent",
		Now:          now.Add(5 * time.Minute),
	}); err != nil {
		t.Fatalf("report status: %v", err)
	}
	if _, err := state.ReportStatus(ReportStatus{
		ID:           "status_02",
		RunHistoryID: "run_hist_status_02",
		Kind:         StatusKindBlocked,
		ProjectID:    "proj_01",
		WorkItemID:   other.ID,
		RunID:        otherRun.ID,
		Message:      "Working",
		Actor:        "agent",
		Now:          now.Add(6 * time.Minute),
	}); err != nil {
		t.Fatalf("report other status: %v", err)
	}

	if runs := state.ListRuns(""); len(runs) != 2 || runs[0].ID != run.ID || runs[1].ID != otherRun.ID {
		t.Fatalf("all runs = %#v", runs)
	}
	if runs := state.ListRuns(item.ID); len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("filtered runs = %#v", runs)
	}
	if artifacts := state.ListArtifacts(item.ID); len(artifacts) != 1 || artifacts[0].ID != "artifact_01" {
		t.Fatalf("artifacts = %#v", artifacts)
	}
	if artifacts := state.ListArtifacts(other.ID); len(artifacts) != 0 {
		t.Fatalf("other artifacts = %#v", artifacts)
	}
	if questions := state.ListQuestions(item.ID); len(questions) != 1 || questions[0].ID != "question_01" {
		t.Fatalf("questions = %#v", questions)
	}
	if gates := state.ListGateReports(item.ID); len(gates) != 0 {
		t.Fatalf("gates = %#v", gates)
	}
	if events := state.ListWorkflowEvents(item.ID); len(events) == 0 {
		t.Fatalf("events = %#v", events)
	}
	if events := state.ListStatusEvents(ListStatusEvents{ProjectID: "proj_01", UnreadOnly: true}); len(events) != 2 || events[0].ID != "status_01" || events[1].ID != "status_02" {
		t.Fatalf("unread status events = %#v", events)
	}
	if events := state.ListStatusEvents(ListStatusEvents{WorkItemID: other.ID, RunID: otherRun.ID}); len(events) != 1 || events[0].ID != "status_02" {
		t.Fatalf("other status events = %#v", events)
	}
}

func TestSnapshotRoundTripValidatesReferences(t *testing.T) {
	state := NewState()
	mustProject(t, state, "proj_01", "One")
	mustWorkItem(t, state, "wi_01", "proj_01")

	restored, err := NewStateFromSnapshot(state.Snapshot())
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	items := restored.ListWorkItems("proj_01")
	if len(items) != 1 || items[0].ID != "wi_01" {
		t.Fatalf("items = %#v", items)
	}

	snapshot := state.Snapshot()
	snapshot.Items[0].ProjectID = "missing"
	if _, err := NewStateFromSnapshot(snapshot); err == nil || !strings.Contains(err.Error(), "project missing not found") {
		t.Fatalf("expected missing project error, got %v", err)
	}
}

func TestSnapshotLoadRestoresFullWorkflowRecords(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	base := NewState()
	project := mustProject(t, base, "proj_01", "One")
	item := mustWorkItem(t, base, "wi_01", project.ID)
	readAt := now.Add(time.Minute)
	snapshot := Snapshot{
		Projects: []Project{project},
		Items:    []WorkItem{item},
		Runs: []WorkItemRun{{
			ID:               "run_01",
			WorkItemID:       item.ID,
			ProjectID:        project.ID,
			Preset:           RunPresetWriter,
			PromptTemplateID: PromptTemplateImplement,
			PromptSnapshot:   "Implement Task wi_01.",
			SessionID:        "sess_01",
			PTYID:            "pty_01",
			Status:           RunStateRunning,
			Metadata: map[string]MetadataValue{
				"agent/owned": {Type: MetadataTypeBool, Bool: true},
			},
			CreatedAt: now,
			UpdatedAt: now,
			History: []RunEvent{{
				ID:   "run_hist_01",
				Type: RunStateRunning,
				At:   now,
			}},
		}},
		Artifacts: []Artifact{{
			ID:         "artifact_01",
			ProjectID:  project.ID,
			WorkItemID: item.ID,
			RunID:      "run_01",
			Kind:       ArtifactKindPlan,
			Status:     ArtifactStatusApproved,
			Title:      "Plan",
			Body:       "Do it.",
			Metadata: map[string]MetadataValue{
				"review/risk": {Type: MetadataTypeNumber, Number: 0.25},
			},
			CreatedAt: now,
			UpdatedAt: now,
		}},
		Questions: []Question{{
			ID:         "question_01",
			ProjectID:  project.ID,
			WorkItemID: item.ID,
			RunID:      "run_01",
			SessionID:  "sess_01",
			PTYID:      "pty_01",
			Prompt:     "Which key?",
			Answer:     "Staging.",
			Status:     QuestionStatusAnswered,
			CreatedAt:  now,
			UpdatedAt:  now,
			AnsweredAt: &readAt,
		}},
		GateReports: []GateReport{{
			ID:             "gate_01",
			ProjectID:      project.ID,
			WorkItemID:     item.ID,
			RunID:          "run_01",
			Name:           "review",
			Blocking:       true,
			Status:         GateStatusOverridden,
			OverrideReason: "Manual review passed.",
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
		WorkflowEvents: []WorkflowEvent{{
			ID:         "event_01",
			ProjectID:  project.ID,
			WorkItemID: item.ID,
			RunID:      "run_01",
			Type:       WorkflowEventPlanApproved,
			Actor:      "human",
			Message:    "Looks good.",
			At:         now,
		}},
		StatusEvents: []StatusEvent{{
			ID:                "status_01",
			Scope:             StatusScopeRun,
			Kind:              StatusKindQuestion,
			Message:           "Need input.",
			Actor:             "agent",
			ProjectID:         project.ID,
			WorkItemID:        item.ID,
			RunID:             "run_01",
			SessionID:         "sess_01",
			PTYID:             "pty_01",
			RequiresAttention: true,
			CreatedAt:         now,
			ReadAt:            &readAt,
		}},
	}

	restored, err := NewStateFromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	run, ok := restored.GetRun("run_01")
	if !ok || run.Metadata["agent/owned"].Type != MetadataTypeBool || !run.Metadata["agent/owned"].Bool {
		t.Fatalf("run = %#v ok=%v", run, ok)
	}
	run.Metadata["agent/owned"] = MetadataValue{Type: MetadataTypeBool}
	again, _ := restored.GetRun("run_01")
	if !again.Metadata["agent/owned"].Bool {
		t.Fatalf("run metadata was not cloned: %#v", again.Metadata)
	}
	if got := restored.ListArtifacts(item.ID); len(got) != 1 || got[0].Metadata["review/risk"].Number != 0.25 {
		t.Fatalf("artifacts = %#v", got)
	}
	if got := restored.ListQuestions(item.ID); len(got) != 1 || got[0].AnsweredAt == nil {
		t.Fatalf("questions = %#v", got)
	}
	if got := restored.ListGateReports(item.ID); len(got) != 1 || got[0].Status != GateStatusOverridden {
		t.Fatalf("gates = %#v", got)
	}
	if got := restored.ListWorkflowEvents(item.ID); len(got) != 1 || got[0].Message != "Looks good." {
		t.Fatalf("workflow events = %#v", got)
	}
	if got := restored.ListStatusEvents(ListStatusEvents{RunID: "run_01"}); len(got) != 1 || got[0].ReadAt == nil {
		t.Fatalf("status events = %#v", got)
	}
}

func TestSetMetadataCoversOwnersAndValidation(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", project.ID)
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
	artifact, err := state.SubmitDraftPlan(SubmitDraftPlan{
		ID:         "artifact_01",
		WorkItemID: item.ID,
		RunID:      run.ID,
		Body:       "Do it.",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("submit plan: %v", err)
	}
	tests := []struct {
		name      string
		ownerType string
		ownerID   string
		value     MetadataValue
	}{
		{name: "project", ownerType: MetadataOwnerProject, ownerID: project.ID, value: MetadataValue{Type: MetadataTypeString, String: "high"}},
		{name: "work item", ownerType: MetadataOwnerWorkItem, ownerID: item.ID, value: MetadataValue{Type: MetadataTypeNumber, Number: 0.5}},
		{name: "run", ownerType: MetadataOwnerRun, ownerID: run.ID, value: MetadataValue{Type: MetadataTypeBool, Bool: true}},
		{name: "artifact", ownerType: MetadataOwnerArtifact, ownerID: artifact.ID, value: MetadataValue{Type: MetadataTypeJSON, JSON: json.RawMessage(`{"ok":true}`)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := state.SetMetadata(SetMetadata{
				OwnerType: tt.ownerType,
				OwnerID:   tt.ownerID,
				Namespace: "review",
				Key:       "risk",
				Value:     tt.value,
				Now:       now.Add(time.Minute),
			})
			if err != nil {
				t.Fatalf("set metadata: %v", err)
			}
			if got.Type != tt.value.Type {
				t.Fatalf("metadata = %#v", got)
			}
		})
	}
	if _, err := state.SetMetadata(SetMetadata{
		OwnerType: MetadataOwnerProject,
		OwnerID:   project.ID,
		Namespace: "bad namespace",
		Key:       "risk",
		Value:     MetadataValue{Type: MetadataTypeString, String: "high"},
	}); err == nil || !strings.Contains(err.Error(), "invalid metadata namespace") {
		t.Fatalf("expected namespace error, got %v", err)
	}
	if _, err := state.SetMetadata(SetMetadata{
		OwnerType: MetadataOwnerArtifact,
		OwnerID:   artifact.ID,
		Namespace: "review",
		Key:       "risk",
		Value:     MetadataValue{Type: MetadataTypeJSON, JSON: json.RawMessage(`{`)},
	}); err == nil || !strings.Contains(err.Error(), "valid json") {
		t.Fatalf("expected json error, got %v", err)
	}
	if _, err := state.SetMetadata(SetMetadata{
		OwnerType: "unknown",
		OwnerID:   "id",
		Namespace: "review",
		Key:       "risk",
		Value:     MetadataValue{Type: MetadataTypeString, String: "high"},
	}); err == nil || !strings.Contains(err.Error(), "unsupported metadata owner") {
		t.Fatalf("expected owner error, got %v", err)
	}
}

func TestSnapshotLoadRejectsInvalidRecords(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	base := NewState()
	project := mustProject(t, base, "proj_01", "One")
	item := mustWorkItem(t, base, "wi_01", project.ID)
	validRun := WorkItemRun{
		ID:               "run_01",
		WorkItemID:       item.ID,
		ProjectID:        project.ID,
		Preset:           RunPresetWriter,
		PromptTemplateID: PromptTemplateImplement,
		PromptSnapshot:   "Implement it.",
		Status:           RunStateQueued,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	validSnapshot := func() Snapshot {
		return Snapshot{
			Projects: []Project{project},
			Items:    []WorkItem{item},
			Runs:     []WorkItemRun{validRun},
			Artifacts: []Artifact{{
				ID:         "artifact_01",
				ProjectID:  project.ID,
				WorkItemID: item.ID,
				RunID:      validRun.ID,
				Kind:       ArtifactKindPlan,
				Status:     ArtifactStatusDraft,
				CreatedAt:  now,
				UpdatedAt:  now,
			}},
			Questions: []Question{{
				ID:         "question_01",
				ProjectID:  project.ID,
				WorkItemID: item.ID,
				RunID:      validRun.ID,
				Prompt:     "Question?",
				Status:     QuestionStatusOpen,
				CreatedAt:  now,
				UpdatedAt:  now,
			}},
			GateReports: []GateReport{{
				ID:         "gate_01",
				ProjectID:  project.ID,
				WorkItemID: item.ID,
				Name:       "review",
				Status:     GateStatusPending,
				CreatedAt:  now,
				UpdatedAt:  now,
			}},
			StatusEvents: []StatusEvent{{
				ID:        "status_01",
				Scope:     StatusScopeRun,
				Kind:      StatusKindQuestion,
				Message:   "Need input.",
				RunID:     validRun.ID,
				CreatedAt: now,
			}},
		}
	}
	tests := []struct {
		name string
		edit func(*Snapshot)
		want string
	}{
		{name: "project id", edit: func(s *Snapshot) { s.Projects[0].ID = "" }, want: "project id required"},
		{name: "project metadata key", edit: func(s *Snapshot) {
			s.Projects[0].Metadata = map[string]MetadataValue{"bad": {Type: MetadataTypeString, String: "x"}}
		}, want: "invalid metadata key"},
		{name: "workflow stage", edit: func(s *Snapshot) { s.Projects[0].Workflow.Stages = nil }, want: "workflow template stages required"},
		{name: "item id", edit: func(s *Snapshot) { s.Items[0].ID = "" }, want: "work item id required"},
		{name: "item project", edit: func(s *Snapshot) { s.Items[0].ProjectID = "missing" }, want: "project missing not found"},
		{name: "item attachment", edit: func(s *Snapshot) {
			s.Items[0].Attachments = []Attachment{{ID: "att_01", Kind: AttachmentKindURL}}
		}, want: "attachment url required"},
		{name: "run id", edit: func(s *Snapshot) { s.Runs[0].ID = "" }, want: "work item run id required"},
		{name: "run project", edit: func(s *Snapshot) { s.Runs[0].ProjectID = "other" }, want: "work item run project mismatch"},
		{name: "run preset", edit: func(s *Snapshot) { s.Runs[0].Preset = "bad" }, want: "unsupported run preset"},
		{name: "run prompt template", edit: func(s *Snapshot) { s.Runs[0].PromptTemplateID = "missing" }, want: "prompt template missing not found"},
		{name: "run prompt", edit: func(s *Snapshot) { s.Runs[0].PromptSnapshot = "" }, want: "prompt snapshot required"},
		{name: "run metadata", edit: func(s *Snapshot) {
			s.Runs[0].Metadata = map[string]MetadataValue{"run/risk": {Type: MetadataTypeJSON, JSON: json.RawMessage(`{`)}}
		}, want: "metadata json value must be valid json"},
		{name: "artifact id", edit: func(s *Snapshot) { s.Artifacts[0].ID = "" }, want: "artifact id required"},
		{name: "artifact kind", edit: func(s *Snapshot) { s.Artifacts[0].Kind = "bad" }, want: "unsupported artifact kind"},
		{name: "artifact status", edit: func(s *Snapshot) { s.Artifacts[0].Status = "bad" }, want: "unsupported artifact status"},
		{name: "question id", edit: func(s *Snapshot) { s.Questions[0].ID = "" }, want: "question id required"},
		{name: "question prompt", edit: func(s *Snapshot) { s.Questions[0].Prompt = "" }, want: "question prompt required"},
		{name: "question status", edit: func(s *Snapshot) { s.Questions[0].Status = "bad" }, want: "unsupported question status"},
		{name: "gate id", edit: func(s *Snapshot) { s.GateReports[0].ID = "" }, want: "gate report id required"},
		{name: "gate name", edit: func(s *Snapshot) { s.GateReports[0].Name = "" }, want: "gate report name required"},
		{name: "gate status", edit: func(s *Snapshot) { s.GateReports[0].Status = "bad" }, want: "unsupported gate status"},
		{name: "status id", edit: func(s *Snapshot) { s.StatusEvents[0].ID = "" }, want: "status event id required"},
		{name: "status kind", edit: func(s *Snapshot) { s.StatusEvents[0].Kind = "bad" }, want: "unsupported status kind"},
		{name: "status message", edit: func(s *Snapshot) { s.StatusEvents[0].Message = "" }, want: "status message required"},
		{name: "status run", edit: func(s *Snapshot) { s.StatusEvents[0].RunID = "" }, want: "run status event requires run id"},
		{name: "status pty", edit: func(s *Snapshot) {
			s.StatusEvents[0].Scope = StatusScopePTY
			s.StatusEvents[0].RunID = ""
			s.StatusEvents[0].PTYID = ""
		}, want: "pty status event requires pty id"},
		{name: "status session", edit: func(s *Snapshot) {
			s.StatusEvents[0].Scope = StatusScopeSession
			s.StatusEvents[0].RunID = ""
			s.StatusEvents[0].SessionID = ""
		}, want: "session status event requires session id"},
		{name: "status scope", edit: func(s *Snapshot) { s.StatusEvents[0].Scope = "bad" }, want: "unsupported status scope"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			test.edit(&snapshot)
			_, err := NewStateFromSnapshot(snapshot)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func mustProject(t *testing.T, state *State, id string, name string) Project {
	t.Helper()
	project, err := state.CreateProject(CreateProject{
		ID:      id,
		Name:    name,
		RootDir: "/repo/" + id,
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return project
}

func mustWorkItem(t *testing.T, state *State, id string, projectID string) WorkItem {
	t.Helper()
	item, err := state.CreateWorkItem(CreateWorkItem{
		ID:        id,
		HistoryID: "hist_" + id,
		ProjectID: projectID,
		Title:     "Task " + id,
	})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	return item
}
