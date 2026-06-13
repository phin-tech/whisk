package workitemstore

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/domain/workitem"
)

func TestSQLiteStorePersistsWorkflowEntities(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "work-items.sqlite")
	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	snapshot := workitem.Snapshot{
		Projects: []workitem.Project{{
			ID:      "proj_01",
			Name:    "App",
			Slug:    "app",
			RootDir: "/repo/app",
			Workflow: workitem.ProjectWorkflow{
				ID:         "workflow_01",
				TemplateID: "default",
				Name:       "Default",
				Stages:     workitem.DefaultWorkflowTemplate(now).Stages,
			},
			Preferences: workitem.ProjectPreferences{
				AutoWorktree: true,
				AutoRun:      workitem.AutoRunNever,
				DefaultPhaseAgents: map[string]string{
					workitem.StagePlanning:  "codex",
					workitem.StageExecution: "codex",
				},
			},
			Metadata: map[string]workitem.MetadataValue{
				"plugin.review/risk": {Type: workitem.MetadataTypeNumber, Number: 0.75},
			},
			NextWorkItemNumber: 2,
			CreatedAt:          now,
			UpdatedAt:          now,
		}},
		Items: []workitem.WorkItem{{
			ID:              "wi_01",
			ProjectID:       "proj_01",
			WorkflowID:      workitem.WorkflowPlanExecuteReview,
			WorkflowVersion: 1,
			Number:          1,
			Title:           "Task",
			StageID:         workitem.StageReview,
			RunState:        workitem.RunStateCompleted,
			Metadata: map[string]workitem.MetadataValue{
				"plugin.review/risk": {Type: workitem.MetadataTypeNumber, Number: 0.75},
			},
			CreatedAt: now,
			UpdatedAt: now,
		}},
		Runs: []workitem.WorkItemRun{{
			ID:               "run_01",
			WorkItemID:       "wi_01",
			ProjectID:        "proj_01",
			Preset:           workitem.RunPresetWriter,
			PromptTemplateID: workitem.PromptTemplateImplement,
			PromptSnapshot:   "Implement Task",
			Status:           workitem.RunStateCompleted,
			CreatedAt:        now,
			UpdatedAt:        now,
		}},
		Artifacts: []workitem.Artifact{{
			ID:         "artifact_01",
			ProjectID:  "proj_01",
			WorkItemID: "wi_01",
			RunID:      "run_01",
			Kind:       workitem.ArtifactKindPlan,
			Status:     workitem.ArtifactStatusApproved,
			Title:      "Plan",
			Body:       "Do it.",
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		Questions: []workitem.Question{{
			ID:         "question_01",
			ProjectID:  "proj_01",
			WorkItemID: "wi_01",
			RunID:      "run_01",
			Prompt:     "Which key?",
			Answer:     "Staging.",
			Status:     workitem.QuestionStatusAnswered,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		GateReports: []workitem.GateReport{{
			ID:         "gate_01",
			ProjectID:  "proj_01",
			WorkItemID: "wi_01",
			Name:       "review",
			Blocking:   true,
			Status:     workitem.GateStatusPassed,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		WorkflowEvents: []workitem.WorkflowEvent{{
			ID:         "event_01",
			ProjectID:  "proj_01",
			WorkItemID: "wi_01",
			Type:       workitem.WorkflowEventPlanApproved,
			Actor:      "human",
			At:         now,
		}},
	}
	if err := store.SaveWorkItems(ctx, snapshot); err != nil {
		t.Fatalf("save: %v", err)
	}

	reopened, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("reopen sqlite store: %v", err)
	}
	loaded, err := reopened.LoadWorkItems(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Projects) != 1 || !loaded.Projects[0].Preferences.AutoWorktree || loaded.Projects[0].Metadata["plugin.review/risk"].Number != 0.75 {
		t.Fatalf("projects = %#v", loaded.Projects)
	}
	if len(loaded.Artifacts) != 1 || loaded.Artifacts[0].Status != workitem.ArtifactStatusApproved {
		t.Fatalf("artifacts = %#v", loaded.Artifacts)
	}
	if len(loaded.Questions) != 1 || loaded.Questions[0].Status != workitem.QuestionStatusAnswered {
		t.Fatalf("questions = %#v", loaded.Questions)
	}
	if len(loaded.GateReports) != 1 || loaded.GateReports[0].Status != workitem.GateStatusPassed {
		t.Fatalf("gates = %#v", loaded.GateReports)
	}
	if len(loaded.WorkflowEvents) != 1 || loaded.WorkflowEvents[0].Type != workitem.WorkflowEventPlanApproved {
		t.Fatalf("events = %#v", loaded.WorkflowEvents)
	}
}

func TestSQLiteStoreDefaultPathUsesXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path, err := DefaultSQLitePath()
	if err != nil {
		t.Fatalf("default sqlite path: %v", err)
	}
	if path != filepath.Join(configHome, "whisk", "work-items.sqlite") {
		t.Fatalf("path = %q", path)
	}
	store, err := NewSQLiteStore("")
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	if store.path != path {
		t.Fatalf("store path = %q", store.path)
	}
	loaded, err := store.LoadWorkItems(context.Background())
	if err != nil {
		t.Fatalf("load default store: %v", err)
	}
	if len(loaded.Projects) != 0 || len(loaded.Items) != 0 {
		t.Fatalf("loaded = %#v", loaded)
	}
}

func TestSQLiteStoreDefaultPathFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	path, err := DefaultSQLitePath()
	if err != nil {
		t.Fatalf("default sqlite path: %v", err)
	}
	if path != filepath.Join(home, ".config", "whisk", "work-items.sqlite") {
		t.Fatalf("path = %q", path)
	}
}

func TestSQLiteStoreRejectsCorruptSnapshotPayload(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "work-items.sqlite")
	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `insert into snapshots (id, payload, updated_at) values (1, ?, datetime('now'))`, []byte(`{`)); err != nil {
		t.Fatalf("write corrupt payload: %v", err)
	}
	if _, err := store.LoadWorkItems(ctx); err == nil {
		t.Fatalf("expected corrupt payload error")
	}
}

func TestSQLiteStoreReportsParentDirectoryErrors(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(parent, []byte("file"), 0o600); err != nil {
		t.Fatalf("write parent file: %v", err)
	}
	if _, err := NewSQLiteStore(filepath.Join(parent, "work-items.sqlite")); err == nil {
		t.Fatalf("expected parent directory error")
	}
}
