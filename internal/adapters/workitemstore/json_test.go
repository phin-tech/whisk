package workitemstore

import (
	"context"
	"path/filepath"
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
			ID:        "wi_01",
			ProjectID: "proj_01",
			Number:    1,
			Title:     "Task",
			StageID:   "backlog",
			RunState:  workitem.RunStateIdle,
			CreatedAt: now,
			UpdatedAt: now,
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
