package workitem

import (
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

func TestMoveToExecutionRequiresWorktreeAndBindWorktreeAllowsMove(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustProject(t, state, "proj_01", "One")
	item := mustWorkItem(t, state, "wi_01", "proj_01")

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
