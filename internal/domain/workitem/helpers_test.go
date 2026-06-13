package workitem

import (
	"strings"
	"testing"
	"time"
)

func TestStageDefaultHelpers(t *testing.T) {
	project := Project{
		Workflow: ProjectWorkflow{
			Stages: []WorkflowStage{
				{ID: "custom", Kind: StageKindExecution, DefaultRunPreset: RunPresetWriter, DefaultPromptTemplateID: PromptTemplateImplement},
				{ID: "review", Kind: StageKindReview},
			},
		},
	}

	if !project.hasStage("custom") || project.hasStage("missing") {
		t.Fatalf("stage lookup failed")
	}
	if got := reviewStageID(project); got != "review" {
		t.Fatalf("review stage = %q", got)
	}
	if got := defaultPresetForStage(project, "custom"); got != RunPresetWriter {
		t.Fatalf("preset = %q", got)
	}
	if got := defaultPromptForStage(project, "custom"); got != PromptTemplateImplement {
		t.Fatalf("prompt = %q", got)
	}
	if got := defaultPresetForStage(project, "missing"); got != RunPresetReader {
		t.Fatalf("fallback preset = %q", got)
	}
	if got := defaultPromptForStage(project, "missing"); got != PromptTemplatePlan {
		t.Fatalf("fallback prompt = %q", got)
	}
}

func TestMetadataValidationHelpers(t *testing.T) {
	key, err := metadataFullKey("plugin.review", "risk_score")
	if err != nil {
		t.Fatalf("metadata key: %v", err)
	}
	if key != "plugin.review/risk_score" {
		t.Fatalf("key = %q", key)
	}
	for _, input := range []string{"", "bad key", "bad/key"} {
		if validMetadataToken(input) {
			t.Fatalf("token %q should be invalid", input)
		}
	}
	if err := validateMetadataMap(map[string]MetadataValue{
		"plugin.review/json": {Type: MetadataTypeJSON, JSON: []byte(`{"ok":true}`)},
		"plugin.review/bool": {Type: MetadataTypeBool, Bool: true},
	}); err != nil {
		t.Fatalf("metadata map: %v", err)
	}
	if err := validateMetadataMap(map[string]MetadataValue{"bad": {Type: MetadataTypeString}}); err == nil {
		t.Fatalf("expected invalid key error")
	}
	if err := validateMetadataValue(MetadataValue{Type: MetadataTypeJSON, JSON: []byte(`{`)}); err == nil {
		t.Fatalf("expected invalid json error")
	}
	if err := validateMetadataValue(MetadataValue{Type: "unsupported"}); err == nil {
		t.Fatalf("expected unsupported metadata error")
	}
}

func TestValidateAttachmentNormalizesAndRejectsInvalidInputs(t *testing.T) {
	file, err := validateAttachment(Attachment{ID: "att_01", Kind: AttachmentKindFile, Path: "docs/../README.md"})
	if err != nil {
		t.Fatalf("file attachment: %v", err)
	}
	if file.Scope != AttachmentScopeProject || file.Path != "README.md" {
		t.Fatalf("file attachment = %#v", file)
	}

	if _, err := validateAttachment(Attachment{ID: "att_02", Kind: AttachmentKindFile, Path: "/tmp/file"}); err == nil {
		t.Fatalf("expected absolute path scope error")
	}
	if _, err := validateAttachment(Attachment{ID: "att_03", Kind: AttachmentKindURL, URL: " "}); err == nil {
		t.Fatalf("expected url error")
	}
	if _, err := validateAttachment(Attachment{ID: "att_04", Kind: AttachmentKindNote, Note: " "}); err == nil {
		t.Fatalf("expected note error")
	}
	if _, err := validateAttachment(Attachment{ID: "att_05", Kind: "other"}); err == nil {
		t.Fatalf("expected unsupported kind error")
	}
}

func TestAppendHistoryHelpersValidateRequiredFields(t *testing.T) {
	item := WorkItem{}
	if err := appendHistory(&item, HistoryEvent{ID: "hist_01", Type: HistoryCreated, At: time.Unix(1, 0)}); err != nil {
		t.Fatalf("append history: %v", err)
	}
	if len(item.History) != 1 {
		t.Fatalf("history = %#v", item.History)
	}
	if err := appendHistory(&item, HistoryEvent{}); err == nil || !strings.Contains(err.Error(), "history id required") {
		t.Fatalf("expected history id error, got %v", err)
	}

	run := WorkItemRun{}
	if err := appendRunEvent(&run, RunEvent{ID: "run_hist_01", Type: RunStateQueued, At: time.Unix(1, 0)}); err != nil {
		t.Fatalf("append run event: %v", err)
	}
	if len(run.History) != 1 {
		t.Fatalf("run history = %#v", run.History)
	}
	if err := appendRunEvent(&run, RunEvent{ID: "run_hist_02"}); err == nil || !strings.Contains(err.Error(), "run event type required") {
		t.Fatalf("expected run event type error, got %v", err)
	}
}
