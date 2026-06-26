package app_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/workitem"
)

func TestRuntimeImportsExportsAndDeletesWorkflowDefinitionFiles(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	ctx := context.Background()
	dir := t.TempDir()

	definition := workitem.DefaultWorkflowDefinition()
	definition.ID = "file-workflow"
	definition.Version = 1
	definition.Stages = []string{workitem.StageBacklog, workitem.StageDone}
	definition.Actions = []workitem.WorkflowActionDefinition{{
		ID:   "finish",
		From: []string{workitem.StageBacklog},
		To:   workitem.StageDone,
	}}
	definition.Gates = nil
	payload, err := json.Marshal(definition)
	if err != nil {
		t.Fatalf("marshal definition: %v", err)
	}
	inputPath := filepath.Join(dir, "workflow.json")
	if err := os.WriteFile(inputPath, payload, 0o600); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	report, err := runtime.ValidateWorkflowDefinitionFile(ctx, app.ValidateWorkflowDefinitionFileRequest{Path: inputPath})
	if err != nil {
		t.Fatalf("validate workflow file: %v", err)
	}
	if !report.Valid || report.Identity != "file-workflow@1" {
		t.Fatalf("validation report = %#v", report)
	}

	record, err := runtime.ImportWorkflowDefinitionFile(ctx, app.ImportWorkflowDefinitionFileRequest{Path: inputPath})
	if err != nil {
		t.Fatalf("import workflow file: %v", err)
	}
	if record.ID != definition.ID || record.SourcePath != inputPath {
		t.Fatalf("imported record = %#v", record)
	}

	outputPath := filepath.Join(dir, "exported.json")
	if err := runtime.ExportWorkflowDefinitionFile(ctx, app.ExportWorkflowDefinitionFileRequest{
		ID:      record.ID,
		Version: record.Version,
		Path:    outputPath,
	}); err != nil {
		t.Fatalf("export workflow file: %v", err)
	}
	exportedPayload, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read exported workflow file: %v", err)
	}
	var exported workitem.WorkflowDefinition
	if err := json.Unmarshal(exportedPayload, &exported); err != nil {
		t.Fatalf("decode exported workflow file: %v", err)
	}
	if exported.ID != definition.ID || exported.Version != definition.Version {
		t.Fatalf("exported definition = %#v", exported)
	}

	deleted, err := runtime.DeleteWorkflowDefinition(ctx, app.DeleteWorkflowDefinitionRequest{ID: record.ID, Version: record.Version})
	if err != nil {
		t.Fatalf("delete workflow definition: %v", err)
	}
	if deleted.ID != record.ID || deleted.Version != record.Version {
		t.Fatalf("deleted = %#v", deleted)
	}
}
