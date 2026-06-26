package workitem

import (
	"strings"
	"testing"
	"time"
)

func TestWorkflowDefinitionRecordsSeedAndRoundTrip(t *testing.T) {
	state := NewState()

	definitions := state.ListWorkflowDefinitions()
	if len(definitions) != 1 {
		t.Fatalf("definitions = %#v", definitions)
	}
	if definitions[0].ID != WorkflowPlanExecuteReview || definitions[0].Version != 1 {
		t.Fatalf("builtin definition = %#v", definitions[0])
	}
	if definitions[0].ContentHash == "" {
		t.Fatalf("content hash required")
	}

	restored, err := NewStateFromSnapshot(state.Snapshot())
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	restoredDefinitions := restored.ListWorkflowDefinitions()
	if len(restoredDefinitions) != 1 || restoredDefinitions[0].ContentHash != definitions[0].ContentHash {
		t.Fatalf("restored definitions = %#v", restoredDefinitions)
	}
}

func TestOldSnapshotsBackfillBuiltinWorkflowDefinition(t *testing.T) {
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	snapshot := Snapshot{
		Projects: []Project{{
			ID: "proj_01", Name: "One", Slug: "one", RootDir: "/tmp/one",
			Workflow: ProjectWorkflow{
				ID: "wf_01", TemplateID: "default", Name: "Default",
				Stages: DefaultWorkflowTemplate(now).Stages,
			},
			Preferences:        defaultProjectPreferences(ProjectPreferences{}),
			NextWorkItemNumber: 1,
			CreatedAt:          now,
			UpdatedAt:          now,
		}},
		Items: []WorkItem{{
			ID: "wi_01", ProjectID: "proj_01", WorkflowID: WorkflowPlanExecuteReview,
			WorkflowVersion: 1, Number: 1, Title: "Task", StageID: StageBacklog,
			RunState: RunStateIdle, CreatedAt: now, UpdatedAt: now,
			History: []HistoryEvent{{ID: "hist_01", Type: HistoryCreated, At: now, StageID: StageBacklog}},
		}},
	}

	state, err := NewStateFromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("restore old snapshot: %v", err)
	}
	if _, ok := state.WorkflowDefinition(WorkflowPlanExecuteReview, 1); !ok {
		t.Fatalf("missing backfilled builtin definition")
	}
}

func TestImportWorkflowDefinitionRejectsDifferentPayloadForSameVersion(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	definition := DefaultWorkflowDefinition()
	definition.ID = "custom"
	definition.Version = 1
	definition.Stages = []string{"backlog", "doing", "done"}
	definition.Actions = []WorkflowActionDefinition{{ID: "start", From: []string{"backlog"}, To: "doing"}}
	definition.Gates = nil

	first, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{
		Definition: definition,
		Source:     "file",
		SourcePath: "/tmp/custom.json",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("import first: %v", err)
	}

	same, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{
		Definition: definition,
		Source:     "file",
		SourcePath: "/tmp/custom.json",
		Now:        now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("import same: %v", err)
	}
	if same.ContentHash != first.ContentHash {
		t.Fatalf("same import changed hash: %#v %#v", first, same)
	}

	definition.Actions[0].To = "done"
	_, err = state.ImportWorkflowDefinition(ImportWorkflowDefinition{
		Definition: definition,
		Source:     "file",
		SourcePath: "/tmp/custom.json",
		Now:        now.Add(2 * time.Minute),
	})
	if err == nil || !strings.Contains(err.Error(), "already exists with different content") {
		t.Fatalf("expected immutable version error, got %v", err)
	}
}

func TestProjectWorkflowSelectionStampsNewWorkItemsOnly(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	project := mustProject(t, state, "proj_01", "One")
	first := mustWorkItem(t, state, "wi_01", project.ID)

	definition := DefaultWorkflowDefinition()
	definition.Version = 2
	definition.Actions[0].To = StageReady
	if _, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{
		Definition: definition,
		Source:     "file",
		SourcePath: "/tmp/plan_execute_review_v2.json",
		Now:        now,
	}); err != nil {
		t.Fatalf("import v2: %v", err)
	}
	if _, err := state.SetProjectWorkflowDefinition(SetProjectWorkflowDefinition{
		ProjectID: project.ID,
		ID:        definition.ID,
		Version:   definition.Version,
		Now:       now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("set project workflow: %v", err)
	}

	second, err := state.CreateWorkItem(CreateWorkItem{
		ID:        "wi_02",
		HistoryID: "hist_02",
		ProjectID: project.ID,
		Title:     "Second",
		Now:       now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	if first.WorkflowVersion != 1 {
		t.Fatalf("first item version changed: %#v", first)
	}
	if second.WorkflowVersion != 2 {
		t.Fatalf("second item version = %#v", second)
	}
}

func TestWorkflowActionsUseItemStampedDefinitionVersion(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	project := mustProject(t, state, "proj_01", "One")

	definition := DefaultWorkflowDefinition()
	definition.Version = 2
	definition.Actions[0].To = StageReady
	if _, err := state.ImportWorkflowDefinition(ImportWorkflowDefinition{
		Definition: definition,
		Source:     "file",
		SourcePath: "/tmp/custom.json",
		Now:        now,
	}); err != nil {
		t.Fatalf("import v2: %v", err)
	}
	if _, err := state.SetProjectWorkflowDefinition(SetProjectWorkflowDefinition{
		ProjectID: project.ID,
		ID:        definition.ID,
		Version:   definition.Version,
		Now:       now,
	}); err != nil {
		t.Fatalf("set project workflow: %v", err)
	}

	item, err := state.CreateWorkItem(CreateWorkItem{
		ID:        "wi_01",
		HistoryID: "hist_01",
		ProjectID: project.ID,
		Title:     "Task",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	if _, err := state.StartPlanning(StartPlanning{
		ID:           "run_01",
		HistoryID:    "hist_run_01",
		RunHistoryID: "run_hist_01",
		WorkItemID:   item.ID,
		Now:          now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("start planning: %v", err)
	}

	updated, _ := state.GetWorkItem(item.ID)
	if updated.StageID != StageReady {
		t.Fatalf("stage should follow v2 action target, got %#v", updated)
	}
}
