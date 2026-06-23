package workitem

import (
	"strings"
	"testing"
)

func TestDefaultWorkflowDefinitionLoadsFromJSON(t *testing.T) {
	definition := DefaultWorkflowDefinition()

	if definition.ID != WorkflowPlanExecuteReview || definition.Version != 1 {
		t.Fatalf("definition identity = %#v", definition)
	}
	if got := definition.Stages; len(got) != len(UniversalStages()) || got[0] != StageBacklog || got[len(got)-1] != StageDone {
		t.Fatalf("stages = %#v", got)
	}

	startPlanning, ok := definition.Action(WorkflowActionStartPlanning)
	if !ok {
		t.Fatalf("missing start planning action")
	}
	if startPlanning.To != StagePlanning || startPlanning.CreatesRun == nil || startPlanning.CreatesRun.PromptTemplateID != PromptTemplatePlan {
		t.Fatalf("start planning = %#v", startPlanning)
	}

	startExecution, ok := definition.Action(WorkflowActionStartExecution)
	if !ok {
		t.Fatalf("missing start execution action")
	}
	if len(startExecution.Requires) != 1 || startExecution.Requires[0].Kind != ArtifactKindPlan || startExecution.Requires[0].Status != ArtifactStatusApproved {
		t.Fatalf("start execution requirements = %#v", startExecution.Requires)
	}
	if startExecution.To != StageExecution || startExecution.CreatesRun == nil || !startExecution.CreatesRun.AutoProvisionWorktree {
		t.Fatalf("start execution = %#v", startExecution)
	}

	completeExecution, ok := definition.Action(WorkflowActionCompleteExecution)
	if !ok {
		t.Fatalf("missing complete execution action")
	}
	if len(completeExecution.CreatesGates) != 1 || completeExecution.CreatesGates[0] != "review" {
		t.Fatalf("complete execution = %#v", completeExecution)
	}
	if !definition.Questions.Enabled || definition.Questions.MoveToBlocked {
		t.Fatalf("questions = %#v", definition.Questions)
	}
}

func TestWorkflowDefinitionValidationRejectsUnknownStagesAndRequirements(t *testing.T) {
	definition := DefaultWorkflowDefinition()
	definition.Actions[0].To = "custom"
	if err := ValidateWorkflowDefinition(definition); err == nil || !strings.Contains(err.Error(), "unknown stage custom") {
		t.Fatalf("expected unknown stage error, got %v", err)
	}

	definition = DefaultWorkflowDefinition()
	definition.Actions[0].Requires = []WorkflowArtifactRequirement{{Kind: "unknown", Status: ArtifactStatusApproved}}
	if err := ValidateWorkflowDefinition(definition); err == nil || !strings.Contains(err.Error(), "unsupported artifact kind") {
		t.Fatalf("expected artifact kind error, got %v", err)
	}
}

func TestWorkflowDefinitionValidationAcceptsCustomKanbanStages(t *testing.T) {
	definition := WorkflowDefinition{
		ID:      "uat-flex",
		Version: 1,
		Stages:  []string{"backlog", "triage", "doing", "done"},
		Actions: []WorkflowActionDefinition{
			{ID: "triage", From: []string{"backlog"}, To: "triage"},
			{ID: "start", From: []string{"triage"}, To: "doing"},
			{ID: "finish", From: []string{"doing"}, To: "done"},
		},
		Gates: []WorkflowGateDefinition{{ID: "uat", Phase: "doing", Blocking: true}},
	}

	if err := ValidateWorkflowDefinition(definition); err != nil {
		t.Fatalf("custom workflow should validate: %v", err)
	}
}

func TestCreateWorkItemStampsWorkflowDefinitionVersion(t *testing.T) {
	state := NewState()
	mustProject(t, state, "proj_01", "One")

	item := mustWorkItem(t, state, "wi_01", "proj_01")

	if item.WorkflowID != WorkflowPlanExecuteReview || item.WorkflowVersion != 1 {
		t.Fatalf("workflow stamp = %#v", item)
	}
	restored, err := NewStateFromSnapshot(state.Snapshot())
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	restoredItem, ok := restored.GetWorkItem(item.ID)
	if !ok {
		t.Fatalf("missing restored item")
	}
	if restoredItem.WorkflowID != item.WorkflowID || restoredItem.WorkflowVersion != item.WorkflowVersion {
		t.Fatalf("restored item = %#v", restoredItem)
	}
}
