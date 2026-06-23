package workitem

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

const (
	WorkflowPlanExecuteReview = "plan-execute-review"

	WorkflowActionStartPlanning        = "start_planning"
	WorkflowActionSubmitDraftPlan      = "submit_draft_plan"
	WorkflowActionApprovePlan          = "approve_plan"
	WorkflowActionStartExecution       = "start_execution"
	WorkflowActionCompleteExecution    = "complete_execution"
	WorkflowActionSubmitReviewFeedback = "submit_review_feedback"
	WorkflowActionApproveDone          = "approve_done"
	WorkflowActionReportBlocked        = "report_blocked"
	WorkflowActionUnblock              = "unblock"
)

//go:embed workflows/plan_execute_review.json
var defaultWorkflowJSON []byte

type WorkflowDefinition struct {
	ID        string                     `json:"id"`
	Version   int                        `json:"version"`
	Stages    []string                   `json:"stages"`
	Actions   []WorkflowActionDefinition `json:"actions"`
	Questions WorkflowQuestionPolicy     `json:"questions"`
	Gates     []WorkflowGateDefinition   `json:"gates"`
}

type WorkflowActionDefinition struct {
	ID                           string                        `json:"id"`
	From                         []string                      `json:"from"`
	To                           string                        `json:"to"`
	Requires                     []WorkflowArtifactRequirement `json:"requires,omitempty"`
	CreatesArtifact              *WorkflowArtifactEffect       `json:"createsArtifact,omitempty"`
	UpdatesArtifact              *WorkflowArtifactEffect       `json:"updatesArtifact,omitempty"`
	CreatesRun                   *WorkflowRunEffect            `json:"createsRun,omitempty"`
	CompletesRun                 bool                          `json:"completesRun,omitempty"`
	CreatesGates                 []string                      `json:"createsGates,omitempty"`
	ResumesRun                   string                        `json:"resumesRun,omitempty"`
	RequiresPassingBlockingGates bool                          `json:"requiresPassingBlockingGates,omitempty"`
	RequiresHuman                bool                          `json:"requiresHuman,omitempty"`
	SideStage                    bool                          `json:"sideStage,omitempty"`
}

type WorkflowArtifactRequirement struct {
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

type WorkflowArtifactEffect struct {
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

type WorkflowRunEffect struct {
	Phase                 string `json:"phase"`
	Preset                string `json:"preset"`
	PromptTemplateID      string `json:"promptTemplateId"`
	WorkingDir            string `json:"workingDir"`
	AutoProvisionWorktree bool   `json:"autoProvisionWorktree,omitempty"`
}

type WorkflowQuestionPolicy struct {
	Enabled                                            bool   `json:"enabled"`
	MoveToBlocked                                      bool   `json:"moveToBlocked"`
	SetsRunState                                       string `json:"setsRunState"`
	AnswerClearsAwaitingInputWhenNoOpenQuestionsRemain bool   `json:"answerClearsAwaitingInputWhenNoOpenQuestionsRemain"`
}

type WorkflowGateDefinition struct {
	ID       string `json:"id"`
	Phase    string `json:"phase"`
	Blocking bool   `json:"blocking"`
}

func DefaultWorkflowDefinition() WorkflowDefinition {
	definition, err := ParseWorkflowDefinition(defaultWorkflowJSON)
	if err != nil {
		panic(err)
	}
	return definition
}

func ParseWorkflowDefinition(payload []byte) (WorkflowDefinition, error) {
	var definition WorkflowDefinition
	if err := json.Unmarshal(payload, &definition); err != nil {
		return WorkflowDefinition{}, err
	}
	if err := ValidateWorkflowDefinition(definition); err != nil {
		return WorkflowDefinition{}, err
	}
	return definition, nil
}

func ValidateWorkflowDefinition(definition WorkflowDefinition) error {
	if definition.ID == "" {
		return fmt.Errorf("workflow id required")
	}
	if definition.Version <= 0 {
		return fmt.Errorf("workflow version must be positive")
	}
	validStages := map[string]struct{}{}
	if definition.ID == WorkflowPlanExecuteReview {
		if len(definition.Stages) != len(UniversalStages()) {
			return fmt.Errorf("workflow stages must match universal stages")
		}
		for i, stage := range definition.Stages {
			if stage != UniversalStages()[i] {
				return fmt.Errorf("workflow stages must match universal stages")
			}
		}
	}
	for _, stage := range definition.Stages {
		if stage == "" {
			return fmt.Errorf("workflow stage id required")
		}
		if _, exists := validStages[stage]; exists {
			return fmt.Errorf("workflow stage %s already exists", stage)
		}
		validStages[stage] = struct{}{}
	}
	actions := map[string]struct{}{}
	for _, action := range definition.Actions {
		if action.ID == "" {
			return fmt.Errorf("workflow action id required")
		}
		if _, exists := actions[action.ID]; exists {
			return fmt.Errorf("workflow action %s already exists", action.ID)
		}
		actions[action.ID] = struct{}{}
		for _, stage := range action.From {
			if _, ok := validStages[stage]; !ok {
				return fmt.Errorf("unknown stage %s", stage)
			}
		}
		if action.To != "$previousStage" {
			if _, ok := validStages[action.To]; !ok {
				return fmt.Errorf("unknown stage %s", action.To)
			}
		}
		for _, requirement := range action.Requires {
			if err := validateWorkflowArtifact(requirement.Kind, requirement.Status); err != nil {
				return err
			}
		}
		if action.CreatesArtifact != nil {
			if err := validateWorkflowArtifact(action.CreatesArtifact.Kind, action.CreatesArtifact.Status); err != nil {
				return err
			}
		}
		if action.UpdatesArtifact != nil {
			if err := validateWorkflowArtifact(action.UpdatesArtifact.Kind, action.UpdatesArtifact.Status); err != nil {
				return err
			}
		}
		if action.CreatesRun != nil {
			if !validPreset(action.CreatesRun.Preset) {
				return fmt.Errorf("unsupported run preset %s", action.CreatesRun.Preset)
			}
			switch action.CreatesRun.PromptTemplateID {
			case PromptTemplatePlan, PromptTemplateImplement, PromptTemplateReview:
			default:
				return fmt.Errorf("unsupported prompt template %s", action.CreatesRun.PromptTemplateID)
			}
		}
	}
	for _, gate := range definition.Gates {
		if gate.ID == "" {
			return fmt.Errorf("workflow gate id required")
		}
		if _, ok := validStages[gate.Phase]; !ok {
			return fmt.Errorf("unknown stage %s", gate.Phase)
		}
	}
	return nil
}

func UniversalStages() []string {
	return []string{StageBacklog, StagePlanning, StageReady, StageExecution, StageBlocked, StageReview, StageDone}
}

func (d WorkflowDefinition) Action(id string) (WorkflowActionDefinition, bool) {
	for _, action := range d.Actions {
		if action.ID == id {
			return action, true
		}
	}
	return WorkflowActionDefinition{}, false
}

func (d WorkflowDefinition) Gate(id string) (WorkflowGateDefinition, bool) {
	for _, gate := range d.Gates {
		if gate.ID == id {
			return gate, true
		}
	}
	return WorkflowGateDefinition{}, false
}

func validateWorkflowArtifact(kind string, status string) error {
	switch kind {
	case ArtifactKindPlan, ArtifactKindFeedback, ArtifactKindGateReport:
	default:
		return fmt.Errorf("unsupported artifact kind %s", kind)
	}
	switch status {
	case ArtifactStatusDraft, ArtifactStatusApproved:
	default:
		return fmt.Errorf("unsupported artifact status %s", status)
	}
	return nil
}
