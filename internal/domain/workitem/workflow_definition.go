package workitem

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
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

	WorkflowActionInputNone              = "none"
	WorkflowActionInputRun               = "run"
	WorkflowActionInputArtifact          = "artifact"
	WorkflowActionInputArtifactSelection = "artifact_selection"
	WorkflowActionInputGate              = "gate"
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

type WorkflowDefinitionRecord struct {
	ID          string             `json:"id"`
	Version     int                `json:"version"`
	Source      string             `json:"source"`
	SourcePath  string             `json:"sourcePath,omitempty"`
	ContentHash string             `json:"contentHash"`
	Definition  WorkflowDefinition `json:"definition"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
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

type WorkflowActionAvailability struct {
	Action      WorkflowActionDefinition `json:"action"`
	Enabled     bool                     `json:"enabled"`
	Recommended bool                     `json:"recommended"`
	Reason      string                   `json:"reason,omitempty"`
	InputKind   string                   `json:"inputKind"`
}

type WorkflowValidationReport struct {
	Valid    bool                      `json:"valid"`
	Identity string                    `json:"identity,omitempty"`
	Errors   []WorkflowValidationError `json:"errors,omitempty"`
}

type WorkflowValidationError struct {
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

func DefaultWorkflowDefinition() WorkflowDefinition {
	definition, err := ParseWorkflowDefinition(defaultWorkflowJSON)
	if err != nil {
		panic(err)
	}
	return definition
}

func DefaultWorkflowDefinitionRecord(now time.Time) WorkflowDefinitionRecord {
	record, err := NewWorkflowDefinitionRecord(DefaultWorkflowDefinition(), "builtin", "internal/domain/workitem/workflows/plan_execute_review.json", now)
	if err != nil {
		panic(err)
	}
	return record
}

func NewWorkflowDefinitionRecord(definition WorkflowDefinition, source, sourcePath string, now time.Time) (WorkflowDefinitionRecord, error) {
	if err := ValidateWorkflowDefinition(definition); err != nil {
		return WorkflowDefinitionRecord{}, err
	}
	hash, err := WorkflowDefinitionHash(definition)
	if err != nil {
		return WorkflowDefinitionRecord{}, err
	}
	return WorkflowDefinitionRecord{
		ID:          definition.ID,
		Version:     definition.Version,
		Source:      source,
		SourcePath:  sourcePath,
		ContentHash: hash,
		Definition:  definition,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func WorkflowDefinitionHash(definition WorkflowDefinition) (string, error) {
	payload, err := json.Marshal(definition)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
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
	report := ValidateWorkflowDefinitionReport(definition)
	if report.Valid {
		return nil
	}
	return errors.New(report.Errors[0].Message)
}

func ValidateWorkflowDefinitionReport(definition WorkflowDefinition) WorkflowValidationReport {
	report := WorkflowValidationReport{
		Identity: workflowDefinitionIdentity(definition),
	}
	addError := func(path string, format string, args ...any) {
		report.Errors = append(report.Errors, WorkflowValidationError{
			Path:    path,
			Message: fmt.Sprintf(format, args...),
		})
	}
	if definition.ID == "" {
		addError("id", "workflow id required")
	}
	if definition.Version <= 0 {
		addError("version", "workflow version must be positive")
	}
	validStages := map[string]struct{}{}
	if definition.ID == WorkflowPlanExecuteReview {
		if len(definition.Stages) != len(UniversalStages()) {
			addError("stages", "workflow stages must match universal stages")
		} else {
			for i, stage := range definition.Stages {
				if stage != UniversalStages()[i] {
					addError("stages", "workflow stages must match universal stages")
					break
				}
			}
		}
	}
	for i, stage := range definition.Stages {
		if stage == "" {
			addError(fmt.Sprintf("stages[%d]", i), "workflow stage id required")
			continue
		}
		if _, exists := validStages[stage]; exists {
			addError(fmt.Sprintf("stages[%d]", i), "workflow stage %s already exists", stage)
			continue
		}
		validStages[stage] = struct{}{}
	}
	actions := map[string]struct{}{}
	for i, action := range definition.Actions {
		actionPath := fmt.Sprintf("actions[%d]", i)
		if action.ID == "" {
			addError(actionPath+".id", "workflow action id required")
		}
		if _, exists := actions[action.ID]; exists {
			addError(actionPath+".id", "workflow action %s already exists", action.ID)
		} else if action.ID != "" {
			actions[action.ID] = struct{}{}
		}
		for j, stage := range action.From {
			if _, ok := validStages[stage]; !ok {
				addError(fmt.Sprintf("%s.from[%d]", actionPath, j), "unknown stage %s", stage)
			}
		}
		if action.To != "$previousStage" {
			if _, ok := validStages[action.To]; !ok {
				addError(actionPath+".to", "unknown stage %s", action.To)
			}
		}
		for j, requirement := range action.Requires {
			if err := validateWorkflowArtifact(requirement.Kind, requirement.Status); err != nil {
				addError(fmt.Sprintf("%s.requires[%d]", actionPath, j), "%s", err.Error())
			}
		}
		if action.CreatesArtifact != nil {
			if err := validateWorkflowArtifact(action.CreatesArtifact.Kind, action.CreatesArtifact.Status); err != nil {
				addError(actionPath+".createsArtifact", "%s", err.Error())
			}
		}
		if action.UpdatesArtifact != nil {
			if err := validateWorkflowArtifact(action.UpdatesArtifact.Kind, action.UpdatesArtifact.Status); err != nil {
				addError(actionPath+".updatesArtifact", "%s", err.Error())
			}
		}
		if action.CreatesRun != nil {
			if !validPreset(action.CreatesRun.Preset) {
				addError(actionPath+".createsRun.preset", "unsupported run preset %s", action.CreatesRun.Preset)
			}
			switch action.CreatesRun.PromptTemplateID {
			case PromptTemplatePlan, PromptTemplateImplement, PromptTemplateReview:
			default:
				addError(actionPath+".createsRun.promptTemplateId", "unsupported prompt template %s", action.CreatesRun.PromptTemplateID)
			}
		}
	}
	gates := map[string]struct{}{}
	for i, gate := range definition.Gates {
		gatePath := fmt.Sprintf("gates[%d]", i)
		if gate.ID == "" {
			addError(gatePath+".id", "workflow gate id required")
		}
		if _, exists := gates[gate.ID]; exists {
			addError(gatePath+".id", "workflow gate %s already exists", gate.ID)
		} else if gate.ID != "" {
			gates[gate.ID] = struct{}{}
		}
		if _, ok := validStages[gate.Phase]; !ok {
			addError(gatePath+".phase", "unknown stage %s", gate.Phase)
		}
	}
	for i, action := range definition.Actions {
		for j, gateID := range action.CreatesGates {
			if _, ok := gates[gateID]; !ok {
				addError(fmt.Sprintf("actions[%d].createsGates[%d]", i, j), "workflow gate %s not found", gateID)
			}
		}
	}
	report.Valid = len(report.Errors) == 0
	return report
}

func workflowDefinitionIdentity(definition WorkflowDefinition) string {
	if definition.ID == "" || definition.Version <= 0 {
		return ""
	}
	return fmt.Sprintf("%s@%d", definition.ID, definition.Version)
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
