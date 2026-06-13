package workitemstore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/domain/workitem"
)

func TestJSONStoreLoadMissingReturnsEmptySnapshot(t *testing.T) {
	store, err := NewJSONStore(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.LoadWorkItems(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(snapshot.Projects) != 0 || len(snapshot.Items) != 0 || len(snapshot.Templates) != 0 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestJSONStoreSavesAndLoadsSnapshot(t *testing.T) {
	store, err := NewJSONStore(filepath.Join(t.TempDir(), "work-items.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	snapshot := workitem.Snapshot{
		Templates: []workitem.WorkflowTemplate{workitem.DefaultWorkflowTemplate(now)},
		Projects: []workitem.Project{{
			ID:                 "proj_01",
			Name:               "App",
			Slug:               "app",
			RootDir:            "/repo/app",
			NextWorkItemNumber: 2,
			CreatedAt:          now,
			UpdatedAt:          now,
			Workflow: workitem.ProjectWorkflow{
				ID:         "wf_01",
				TemplateID: "default",
				Name:       "Default",
				Stages:     workitem.DefaultWorkflowTemplate(now).Stages,
			},
		}},
		Items: []workitem.WorkItem{{
			ID:              "wi_01",
			ProjectID:       "proj_01",
			WorkflowID:      workitem.WorkflowPlanExecuteReview,
			WorkflowVersion: 1,
			Number:          1,
			Title:           "Task",
			StageID:         "backlog",
			RunState:        workitem.RunStateIdle,
			CreatedAt:       now,
			UpdatedAt:       now,
		}},
	}

	if err := store.SaveWorkItems(context.Background(), snapshot); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := store.LoadWorkItems(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Projects) != 1 || loaded.Projects[0].ID != "proj_01" || len(loaded.Items) != 1 || loaded.Items[0].ID != "wi_01" {
		t.Fatalf("loaded = %#v", loaded)
	}
}

func TestJSONStoreDefaultPathUsesXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	if path != filepath.Join(configHome, "whisk", "work-items.json") {
		t.Fatalf("path = %q", path)
	}
	store, err := NewJSONStore("")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if store.path != path {
		t.Fatalf("store path = %q", store.path)
	}
}

func TestJSONStoreDefaultPathFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	if path != filepath.Join(home, ".config", "whisk", "work-items.json") {
		t.Fatalf("path = %q", path)
	}
}

func TestJSONStoreRejectsInvalidFiles(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "work-items.json")
	if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
		t.Fatalf("write invalid json: %v", err)
	}
	store, err := NewJSONStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if _, err := store.LoadWorkItems(ctx); err == nil {
		t.Fatalf("expected invalid json error")
	}
	if err := os.WriteFile(path, []byte(`{"version":2,"snapshot":{}}`), 0o600); err != nil {
		t.Fatalf("write invalid version: %v", err)
	}
	if _, err := store.LoadWorkItems(ctx); err == nil || !strings.Contains(err.Error(), "unsupported work item state version 2") {
		t.Fatalf("expected version error, got %v", err)
	}
}

func TestJSONStoreSaveReportsParentDirectoryErrors(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(parent, []byte("file"), 0o600); err != nil {
		t.Fatalf("write parent file: %v", err)
	}
	store, err := NewJSONStore(filepath.Join(parent, "work-items.json"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := store.SaveWorkItems(context.Background(), workitem.Snapshot{}); err == nil {
		t.Fatalf("expected parent directory error")
	}
}
