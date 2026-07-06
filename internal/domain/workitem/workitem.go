package workitem

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	StageKindBacklog   = "backlog"
	StageKindPlanning  = "planning"
	StageKindReady     = "ready"
	StageKindActive    = "active"
	StageKindExecution = "execution"
	StageKindBlocked   = "blocked"
	StageKindReview    = "review"
	StageKindDone      = "done"
	StageKindArchived  = "archived"

	StageBacklog   = "backlog"
	StagePlanning  = "planning"
	StageReady     = "ready"
	StageExecution = "execution"
	StageBlocked   = "blocked"
	StageReview    = "review"
	StageDone      = "done"

	RunStateIdle          = "idle"
	RunStateQueued        = "queued"
	RunStateRunning       = "running"
	RunStateAwaitingInput = "awaiting_input"
	RunStateFailed        = "failed"
	RunStateCompleted     = "completed"
	RunStateCancelled     = "cancelled"

	StatusKindQuestion = "question"
	StatusKindDone     = "done"
	StatusKindBlocked  = "blocked"

	StatusScopeRun     = "run"
	StatusScopePTY     = "pty"
	StatusScopeSession = "session"

	StatusNotificationSeverityAttention = "attention"
	StatusNotificationSeverityWarning   = "warning"
	StatusNotificationSeverityInfo      = "info"

	AttachmentKindFile     = "file"
	AttachmentKindURL      = "url"
	AttachmentKindNote     = "note"
	AttachmentKindExternal = "external"

	AttachmentScopeProject  = "project"
	AttachmentScopeWorktree = "worktree"
	AttachmentScopeExternal = "external"

	AutoRunNever = "never"
	AutoRunPlan  = "plan"
	AutoRunAll   = "all"

	MetadataOwnerProject  = "project"
	MetadataOwnerWorkItem = "work_item"
	MetadataOwnerRun      = "run"
	MetadataOwnerArtifact = "artifact"

	MetadataTypeString = "string"
	MetadataTypeNumber = "number"
	MetadataTypeBool   = "bool"
	MetadataTypeJSON   = "json"

	ArtifactKindPlan       = "plan"
	ArtifactKindFeedback   = "feedback"
	ArtifactKindGateReport = "gate_report"

	ArtifactStatusDraft    = "draft"
	ArtifactStatusApproved = "approved"

	QuestionStatusOpen     = "open"
	QuestionStatusAnswered = "answered"

	GateStatusPending    = "pending"
	GateStatusPassed     = "passed"
	GateStatusFailed     = "failed"
	GateStatusOverridden = "overridden"

	WorkflowEventPlanningStarted     = "planning_started"
	WorkflowEventDraftPlanSubmitted  = "draft_plan_submitted"
	WorkflowEventPlanApproved        = "plan_approved"
	WorkflowEventExecutionStarted    = "execution_started"
	WorkflowEventQuestionAsked       = "question_asked"
	WorkflowEventQuestionAnswered    = "question_answered"
	WorkflowEventBlocked             = "blocked"
	WorkflowEventUnblocked           = "unblocked"
	WorkflowEventExecutionCompleted  = "execution_completed"
	WorkflowEventReviewFeedbackAdded = "review_feedback_added"
	WorkflowEventGateCompleted       = "gate_completed"
	WorkflowEventRunFailed           = "run_failed"
	WorkflowEventRunCancelled        = "run_cancelled"
	WorkflowEventDoneApproved        = "done_approved"

	HistoryCreated          = "created"
	HistoryUpdated          = "updated"
	HistoryStageMoved       = "stage_moved"
	HistoryAttachmentAdded  = "attachment_added"
	HistoryWorktreeBound    = "worktree_bound"
	HistoryDeleted          = "deleted"
	HistoryRunEventAppended = "run_event_appended"

	RunPresetReader   = "reader"
	RunPresetWriter   = "writer"
	RunPresetReviewer = "reviewer"
	RunPresetManager  = "manager"

	PromptTemplatePlan      = "plan"
	PromptTemplateImplement = "implement"
	PromptTemplateReview    = "review"

	WorkItemLinkBlocks      = "blocks"
	WorkItemLinkParentChild = "parent-child"
	WorkItemLinkRelated     = "related"
	WorkItemLinkDuplicates  = "duplicates"
	WorkItemLinkSupersedes  = "supersedes"
)

type Project struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	Description        string                   `json:"description,omitempty"`
	Slug               string                   `json:"slug"`
	RootDir            string                   `json:"rootDir"`
	Workflow           ProjectWorkflow          `json:"workflow"`
	Preferences        ProjectPreferences       `json:"preferences"`
	Attachments        []Attachment             `json:"attachments"`
	Metadata           map[string]MetadataValue `json:"metadata,omitempty"`
	NextWorkItemNumber int                      `json:"nextWorkItemNumber"`
	CreatedAt          time.Time                `json:"createdAt"`
	UpdatedAt          time.Time                `json:"updatedAt"`
}

type ProjectPreferences struct {
	AutoRun                  string            `json:"autoRun"`
	AutoWorktree             bool              `json:"autoWorktree"`
	UseInteractiveAgentShell bool              `json:"useInteractiveAgentShell,omitempty"`
	DefaultPhaseAgents       map[string]string `json:"defaultPhaseAgents,omitempty"`
	Gates                    []GateConfig      `json:"gates,omitempty"`
}

type GateConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Command  string `json:"command,omitempty"`
	Skill    string `json:"skill,omitempty"`
	Blocking bool   `json:"blocking"`
	Phase    string `json:"phase,omitempty"`
}

type MetadataValue struct {
	Type   string          `json:"type"`
	String string          `json:"string,omitempty"`
	Number float64         `json:"number,omitempty"`
	Bool   bool            `json:"bool,omitempty"`
	JSON   json.RawMessage `json:"json,omitempty"`
}

type WorkflowTemplate struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Source          string           `json:"source"`
	Stages          []WorkflowStage  `json:"stages"`
	TransitionRules []TransitionRule `json:"transitionRules"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

type ProjectWorkflow struct {
	ID                string           `json:"id"`
	TemplateID        string           `json:"templateId"`
	DefinitionID      string           `json:"definitionId,omitempty"`
	DefinitionVersion int              `json:"definitionVersion,omitempty"`
	DefinitionHash    string           `json:"definitionHash,omitempty"`
	Name              string           `json:"name"`
	Stages            []WorkflowStage  `json:"stages"`
	TransitionRules   []TransitionRule `json:"transitionRules"`
}

type WorkflowStage struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	Kind                    string `json:"kind"`
	DefaultRunPreset        string `json:"defaultRunPreset,omitempty"`
	DefaultPromptTemplateID string `json:"defaultPromptTemplateId,omitempty"`
	ProvisionWorktree       bool   `json:"provisionWorktree,omitempty"`
	WIPLimit                int    `json:"wipLimit,omitempty"`
}

type TransitionRule struct {
	FromStageID           string `json:"fromStageId"`
	ToStageID             string `json:"toStageId"`
	RequiresApproval      bool   `json:"requiresApproval,omitempty"`
	RequiresChecks        bool   `json:"requiresChecks,omitempty"`
	RequiresNoRunningRuns bool   `json:"requiresNoRunningRuns,omitempty"`
}

type WorkItem struct {
	ID              string                   `json:"id"`
	ProjectID       string                   `json:"projectId"`
	WorkflowID      string                   `json:"workflowId"`
	WorkflowVersion int                      `json:"workflowVersion"`
	Number          int                      `json:"number"`
	Title           string                   `json:"title"`
	BodyMarkdown    string                   `json:"bodyMarkdown"`
	StageID         string                   `json:"stageId"`
	PreviousStageID string                   `json:"previousStageId,omitempty"`
	RunState        string                   `json:"runState"`
	Worktree        *WorktreeBinding         `json:"worktree,omitempty"`
	Attachments     []Attachment             `json:"attachments"`
	Metadata        map[string]MetadataValue `json:"metadata,omitempty"`
	History         []HistoryEvent           `json:"history"`
	CreatedAt       time.Time                `json:"createdAt"`
	UpdatedAt       time.Time                `json:"updatedAt"`
}

type WorkItemLink struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectId"`
	SourceWorkItemID string    `json:"sourceWorkItemId"`
	TargetWorkItemID string    `json:"targetWorkItemId"`
	Type             string    `json:"type"`
	CreatedBy        string    `json:"createdBy,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

type PromptTemplate struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Source    string    `json:"source"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type WorkItemRun struct {
	ID               string                   `json:"id"`
	WorkItemID       string                   `json:"workItemId"`
	ProjectID        string                   `json:"projectId"`
	Preset           string                   `json:"preset"`
	PromptTemplateID string                   `json:"promptTemplateId"`
	PromptSnapshot   string                   `json:"promptSnapshot"`
	SessionID        string                   `json:"sessionId,omitempty"`
	PTYID            string                   `json:"ptyId,omitempty"`
	Status           string                   `json:"status"`
	Metadata         map[string]MetadataValue `json:"metadata,omitempty"`
	CreatedAt        time.Time                `json:"createdAt"`
	UpdatedAt        time.Time                `json:"updatedAt"`
	CompletedAt      *time.Time               `json:"completedAt,omitempty"`
	History          []RunEvent               `json:"history"`
}

type Artifact struct {
	ID         string                   `json:"id"`
	ProjectID  string                   `json:"projectId"`
	WorkItemID string                   `json:"workItemId"`
	RunID      string                   `json:"runId,omitempty"`
	Kind       string                   `json:"kind"`
	Status     string                   `json:"status"`
	Title      string                   `json:"title,omitempty"`
	Body       string                   `json:"body,omitempty"`
	Metadata   map[string]MetadataValue `json:"metadata,omitempty"`
	CreatedAt  time.Time                `json:"createdAt"`
	UpdatedAt  time.Time                `json:"updatedAt"`
}

type Question struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"projectId"`
	WorkItemID string     `json:"workItemId"`
	RunID      string     `json:"runId,omitempty"`
	SessionID  string     `json:"sessionId,omitempty"`
	PTYID      string     `json:"ptyId,omitempty"`
	Prompt     string     `json:"prompt"`
	Answer     string     `json:"answer,omitempty"`
	Status     string     `json:"status"`
	Actor      string     `json:"actor,omitempty"`
	AnsweredBy string     `json:"answeredBy,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	AnsweredAt *time.Time `json:"answeredAt,omitempty"`
}

type GateReport struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"projectId"`
	WorkItemID     string    `json:"workItemId"`
	RunID          string    `json:"runId,omitempty"`
	Name           string    `json:"name"`
	Blocking       bool      `json:"blocking"`
	Status         string    `json:"status"`
	OverrideReason string    `json:"overrideReason,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type WorkflowEvent struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"projectId"`
	WorkItemID string    `json:"workItemId,omitempty"`
	RunID      string    `json:"runId,omitempty"`
	Type       string    `json:"type"`
	Actor      string    `json:"actor,omitempty"`
	Message    string    `json:"message,omitempty"`
	At         time.Time `json:"at"`
}

type StatusEvent struct {
	ID                   string     `json:"id"`
	Scope                string     `json:"scope"`
	Kind                 string     `json:"kind"`
	Message              string     `json:"message"`
	Actor                string     `json:"actor,omitempty"`
	ProjectID            string     `json:"projectId,omitempty"`
	WorkItemID           string     `json:"workItemId,omitempty"`
	RunID                string     `json:"runId,omitempty"`
	SessionID            string     `json:"sessionId,omitempty"`
	PaneID               string     `json:"paneId,omitempty"`
	PTYID                string     `json:"ptyId,omitempty"`
	RequiresAttention    bool       `json:"requiresAttention"`
	NotificationKey      string     `json:"notificationKey,omitempty"`
	NotificationSeverity string     `json:"notificationSeverity,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	ReadAt               *time.Time `json:"readAt,omitempty"`
}

type RunEvent struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	At      time.Time `json:"at"`
	Actor   string    `json:"actor,omitempty"`
	Message string    `json:"message,omitempty"`
}

type WorktreeBinding struct {
	Branch       string    `json:"branch"`
	Base         string    `json:"base"`
	WorktreePath string    `json:"worktreePath"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Attachment struct {
	ID               string                   `json:"id"`
	Kind             string                   `json:"kind"`
	Scope            string                   `json:"scope"`
	Title            string                   `json:"title,omitempty"`
	Path             string                   `json:"path,omitempty"`
	URL              string                   `json:"url,omitempty"`
	Note             string                   `json:"note,omitempty"`
	Provider         string                   `json:"provider,omitempty"`
	Target           string                   `json:"target,omitempty"`
	IncludeInContext bool                     `json:"includeInContext,omitempty"`
	Meta             map[string]MetadataValue `json:"meta,omitempty"`
	CreatedAt        time.Time                `json:"createdAt"`
}

type HistoryEvent struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	At           time.Time `json:"at"`
	Actor        string    `json:"actor,omitempty"`
	Message      string    `json:"message,omitempty"`
	StageID      string    `json:"stageId,omitempty"`
	AttachmentID string    `json:"attachmentId,omitempty"`
	Branch       string    `json:"branch,omitempty"`
	WorktreePath string    `json:"worktreePath,omitempty"`
}

type State struct {
	projects            map[string]Project
	templates           map[string]WorkflowTemplate
	workflowDefinitions map[workflowDefinitionKey]WorkflowDefinitionRecord
	promptTemplates     map[string]PromptTemplate
	items               map[string]WorkItem
	links               map[string]WorkItemLink
	runs                map[string]WorkItemRun
	artifacts           map[string]Artifact
	questions           map[string]Question
	gateReports         map[string]GateReport
	workflowEvents      map[string]WorkflowEvent
	statusEvents        map[string]StatusEvent
}

type Snapshot struct {
	Projects            []Project                  `json:"projects"`
	Templates           []WorkflowTemplate         `json:"workflowTemplates"`
	WorkflowDefinitions []WorkflowDefinitionRecord `json:"workflowDefinitions,omitempty"`
	PromptTemplates     []PromptTemplate           `json:"promptTemplates"`
	Items               []WorkItem                 `json:"workItems"`
	Links               []WorkItemLink             `json:"workItemLinks,omitempty"`
	Runs                []WorkItemRun              `json:"workItemRuns"`
	Artifacts           []Artifact                 `json:"artifacts"`
	Questions           []Question                 `json:"questions"`
	GateReports         []GateReport               `json:"gateReports"`
	WorkflowEvents      []WorkflowEvent            `json:"workflowEvents"`
	StatusEvents        []StatusEvent              `json:"statusEvents"`
}

type workflowDefinitionKey struct {
	id      string
	version int
}

type CreateProject struct {
	ID                 string
	WorkflowID         string
	ProjectWorkflowID  string
	Name               string
	Description        string
	Slug               string
	RootDir            string
	Preferences        ProjectPreferences
	NextWorkItemNumber int
	Now                time.Time
}

type ImportWorkflowDefinition struct {
	Definition WorkflowDefinition
	Source     string
	SourcePath string
	Now        time.Time
}

type SetProjectWorkflowDefinition struct {
	ProjectID string
	ID        string
	Version   int
	Now       time.Time
}

type PlanProjectWorkflowMigration struct {
	ProjectID string
	ID        string
	Version   int
}

type WorkflowMigrationPlan struct {
	ProjectID                   string                  `json:"projectId"`
	CurrentID                   string                  `json:"currentId"`
	CurrentVersion              int                     `json:"currentVersion"`
	TargetID                    string                  `json:"targetId"`
	TargetVersion               int                     `json:"targetVersion"`
	ExistingItems               int                     `json:"existingItems"`
	ItemsPinnedToCurrentVersion int                     `json:"itemsPinnedToCurrentVersion"`
	CompatibleItems             int                     `json:"compatibleItems"`
	IncompatibleItems           int                     `json:"incompatibleItems"`
	Items                       []WorkflowMigrationItem `json:"items"`
}

type WorkflowMigrationItem struct {
	WorkItemID             string `json:"workItemId"`
	Number                 int    `json:"number"`
	Title                  string `json:"title"`
	CurrentWorkflowID      string `json:"currentWorkflowId"`
	CurrentWorkflowVersion int    `json:"currentWorkflowVersion"`
	CurrentStageID         string `json:"currentStageId"`
	TargetStageID          string `json:"targetStageId,omitempty"`
	Compatible             bool   `json:"compatible"`
	Reason                 string `json:"reason,omitempty"`
}

type DeleteWorkflowDefinition struct {
	ID      string
	Version int
}

type UpdateProject struct {
	ID                       string
	Name                     *string
	Description              *string
	Slug                     *string
	UseInteractiveAgentShell *bool
	// DefaultPhaseAgents merges per-phase default agent overrides (keyed by run
	// preset) into the project's preferences. Only supplied keys are touched; an
	// empty value clears that phase's override.
	DefaultPhaseAgents map[string]string
	Now                time.Time
}

type DeleteProject struct {
	ID    string
	Actor string
	Now   time.Time
}

type AddProjectAttachment struct {
	ID               string
	ProjectID        string
	Kind             string
	Scope            string
	Title            string
	Path             string
	URL              string
	Note             string
	Provider         string
	Target           string
	IncludeInContext bool
	Meta             map[string]MetadataValue
	Now              time.Time
}

type UpdateProjectAttachment struct {
	ID               string
	ProjectID        string
	Title            *string
	Path             *string
	URL              *string
	Note             *string
	Provider         *string
	Target           *string
	IncludeInContext *bool
	Meta             map[string]MetadataValue
	Now              time.Time
}

type DeleteProjectAttachment struct {
	ID        string
	ProjectID string
	Now       time.Time
}

type CreateWorkItem struct {
	ID           string
	HistoryID    string
	ProjectID    string
	WorkflowID   string
	Title        string
	BodyMarkdown string
	StageID      string
	Actor        string
	Now          time.Time
}

type UpdateWorkItem struct {
	ID           string
	HistoryID    string
	Title        *string
	BodyMarkdown *string
	Actor        string
	Now          time.Time
}

type MoveWorkItem struct {
	ID        string
	HistoryID string
	StageID   string
	Actor     string
	Now       time.Time
}

type ApplyWorkflowAction struct {
	WorkItemID string
	ActionID   string
	RunID      string
	Reason     string
	Actor      string
	Now        time.Time
}

type BindWorktree struct {
	ID           string
	HistoryID    string
	Branch       string
	Base         string
	WorktreePath string
	Actor        string
	Now          time.Time
}

type AddAttachment struct {
	ID         string
	HistoryID  string
	WorkItemID string
	Kind       string
	Scope      string
	Path       string
	URL        string
	Note       string
	Actor      string
	Now        time.Time
}

type DeleteWorkItem struct {
	ID        string
	HistoryID string
	Actor     string
	Now       time.Time
}

type AddWorkItemLink struct {
	ID               string
	SourceWorkItemID string
	TargetWorkItemID string
	Type             string
	Actor            string
	Now              time.Time
}

type StartRun struct {
	ID               string
	HistoryID        string
	RunHistoryID     string
	WorkItemID       string
	Preset           string
	PromptTemplateID string
	SessionID        string
	PTYID            string
	Actor            string
	Now              time.Time
}

type StartPlanning struct {
	ID           string
	HistoryID    string
	RunHistoryID string
	WorkItemID   string
	SessionID    string
	PTYID        string
	Actor        string
	Now          time.Time
}

type SubmitDraftPlan struct {
	ID         string
	WorkItemID string
	RunID      string
	Title      string
	Body       string
	Actor      string
	Now        time.Time
}

type ApprovePlan struct {
	ArtifactID string
	WorkItemID string
	Actor      string
	Now        time.Time
}

type StartExecution struct {
	ID           string
	HistoryID    string
	RunHistoryID string
	WorkItemID   string
	SessionID    string
	PTYID        string
	Actor        string
	Now          time.Time
}

type AskQuestion struct {
	ID         string
	WorkItemID string
	RunID      string
	SessionID  string
	PTYID      string
	Prompt     string
	Actor      string
	Now        time.Time
}

type AnswerQuestion struct {
	ID     string
	Answer string
	Actor  string
	Now    time.Time
}

type ReportBlocked struct {
	WorkItemID string
	RunID      string
	Reason     string
	Actor      string
	Now        time.Time
}

type Unblock struct {
	WorkItemID string
	Actor      string
	Now        time.Time
}

type CompleteExecution struct {
	RunID   string
	Actor   string
	Message string
	Now     time.Time
}

type SubmitReviewFeedback struct {
	ID         string
	WorkItemID string
	RunID      string
	Body       string
	Actor      string
	Now        time.Time
}

type ApproveDone struct {
	WorkItemID string
	Actor      string
	Reason     string
	Now        time.Time
}

type CompleteGate struct {
	ID             string
	Status         string
	OverrideReason string
	Actor          string
	Now            time.Time
}

type SetMetadata struct {
	OwnerType string
	OwnerID   string
	Namespace string
	Key       string
	Value     MetadataValue
	Now       time.Time
}

type CancelRun struct {
	ID           string
	RunHistoryID string
	Actor        string
	Now          time.Time
}

type CompleteRun struct {
	ID           string
	RunHistoryID string
	Actor        string
	Message      string
	Now          time.Time
}

type MarkRunRunning struct {
	ID           string
	RunHistoryID string
	SessionID    string
	PTYID        string
	DaemonOwned  bool
	Actor        string
	Now          time.Time
}

type FailRun struct {
	ID           string
	RunHistoryID string
	Actor        string
	Message      string
	Now          time.Time
}

type ReportStatus struct {
	ID           string
	RunHistoryID string
	Kind         string
	Message      string
	Actor        string
	ProjectID    string
	WorkItemID   string
	RunID        string
	SessionID    string
	PaneID       string
	PTYID        string
	Now          time.Time
}

type ListStatusEvents struct {
	ProjectID  string
	WorkItemID string
	RunID      string
	SessionID  string
	PTYID      string
	UnreadOnly bool
}

type MarkStatusEventRead struct {
	ID  string
	Now time.Time
}

type ReadyWorkInput struct {
	ProjectID string
	Projects  []Project
	WorkItems []WorkItem
	Links     []WorkItemLink
}

type ReadyWorkExplanation struct {
	Ready   []ReadyWorkItem   `json:"ready"`
	Blocked []BlockedWorkItem `json:"blocked"`
	Cycles  [][]string        `json:"cycles,omitempty"`
	Summary ReadyWorkSummary  `json:"summary"`
}

type ReadyWorkItem struct {
	WorkItem         WorkItem `json:"workItem"`
	Reason           string   `json:"reason"`
	ResolvedBlockers []string `json:"resolvedBlockers,omitempty"`
	DependencyCount  int      `json:"dependencyCount"`
	DependentCount   int      `json:"dependentCount"`
	ParentWorkItemID *string  `json:"parentWorkItemId,omitempty"`
}

type BlockedWorkItem struct {
	WorkItem       WorkItem           `json:"workItem"`
	BlockedBy      []ReadyBlockerInfo `json:"blockedBy"`
	BlockedByCount int                `json:"blockedByCount"`
}

type ReadyBlockerInfo struct {
	ID       string `json:"id"`
	Number   int    `json:"number,omitempty"`
	Title    string `json:"title,omitempty"`
	StageID  string `json:"stageId,omitempty"`
	RunState string `json:"runState,omitempty"`
}

type ReadyWorkSummary struct {
	TotalReady   int `json:"totalReady"`
	TotalBlocked int `json:"totalBlocked"`
	CycleCount   int `json:"cycleCount"`
}

func NewState() *State {
	state := &State{
		projects:            map[string]Project{},
		templates:           map[string]WorkflowTemplate{},
		workflowDefinitions: map[workflowDefinitionKey]WorkflowDefinitionRecord{},
		promptTemplates:     map[string]PromptTemplate{},
		items:               map[string]WorkItem{},
		links:               map[string]WorkItemLink{},
		runs:                map[string]WorkItemRun{},
		artifacts:           map[string]Artifact{},
		questions:           map[string]Question{},
		gateReports:         map[string]GateReport{},
		workflowEvents:      map[string]WorkflowEvent{},
		statusEvents:        map[string]StatusEvent{},
	}
	template := DefaultWorkflowTemplate(time.Time{})
	state.templates[template.ID] = template
	defaultDefinition := DefaultWorkflowDefinitionRecord(time.Time{})
	state.workflowDefinitions[workflowDefinitionKey{id: defaultDefinition.ID, version: defaultDefinition.Version}] = defaultDefinition
	for _, prompt := range DefaultPromptTemplates(time.Time{}) {
		state.promptTemplates[prompt.ID] = prompt
	}
	return state
}

func NewStateFromSnapshot(snapshot Snapshot) (*State, error) {
	state := &State{
		projects:            map[string]Project{},
		templates:           map[string]WorkflowTemplate{},
		workflowDefinitions: map[workflowDefinitionKey]WorkflowDefinitionRecord{},
		promptTemplates:     map[string]PromptTemplate{},
		items:               map[string]WorkItem{},
		links:               map[string]WorkItemLink{},
		runs:                map[string]WorkItemRun{},
		artifacts:           map[string]Artifact{},
		questions:           map[string]Question{},
		gateReports:         map[string]GateReport{},
		workflowEvents:      map[string]WorkflowEvent{},
		statusEvents:        map[string]StatusEvent{},
	}
	if len(snapshot.Templates) == 0 {
		template := DefaultWorkflowTemplate(time.Time{})
		state.templates[template.ID] = template
	}
	if len(snapshot.PromptTemplates) == 0 {
		for _, prompt := range DefaultPromptTemplates(time.Time{}) {
			state.promptTemplates[prompt.ID] = prompt
		}
	}
	for _, template := range snapshot.Templates {
		if err := validateWorkflowTemplate(template); err != nil {
			return nil, err
		}
		if _, exists := state.templates[template.ID]; exists {
			return nil, fmt.Errorf("workflow template %s already exists", template.ID)
		}
		state.templates[template.ID] = cloneTemplate(template)
	}
	if len(snapshot.WorkflowDefinitions) == 0 {
		record := DefaultWorkflowDefinitionRecord(time.Time{})
		state.workflowDefinitions[workflowDefinitionKey{id: record.ID, version: record.Version}] = record
	}
	for _, record := range snapshot.WorkflowDefinitions {
		if err := validateWorkflowDefinitionRecord(record); err != nil {
			return nil, err
		}
		key := workflowDefinitionKey{id: record.ID, version: record.Version}
		if _, exists := state.workflowDefinitions[key]; exists {
			return nil, fmt.Errorf("workflow definition %s@%d already exists", record.ID, record.Version)
		}
		state.workflowDefinitions[key] = cloneWorkflowDefinitionRecord(record)
	}
	builtinPrompts := map[string]PromptTemplate{}
	for _, prompt := range DefaultPromptTemplates(time.Time{}) {
		builtinPrompts[prompt.ID] = prompt
	}
	for _, prompt := range snapshot.PromptTemplates {
		if prompt.Source == "builtin" {
			if builtin, ok := builtinPrompts[prompt.ID]; ok {
				builtin.CreatedAt = prompt.CreatedAt
				builtin.UpdatedAt = prompt.UpdatedAt
				prompt = builtin
			}
		}
		if err := validatePromptTemplate(prompt); err != nil {
			return nil, err
		}
		if _, exists := state.promptTemplates[prompt.ID]; exists {
			return nil, fmt.Errorf("prompt template %s already exists", prompt.ID)
		}
		state.promptTemplates[prompt.ID] = prompt
	}
	for _, project := range snapshot.Projects {
		project = state.backfillProjectWorkflowDefinition(project)
		if err := validateProject(project); err != nil {
			return nil, err
		}
		if _, exists := state.projects[project.ID]; exists {
			return nil, fmt.Errorf("project %s already exists", project.ID)
		}
		state.projects[project.ID] = cloneProject(project)
	}
	for _, item := range snapshot.Items {
		if item.WorkflowID == "" {
			item.WorkflowID = WorkflowPlanExecuteReview
		}
		if item.WorkflowVersion <= 0 {
			item.WorkflowVersion = 1
		}
		if err := state.validateWorkItem(item); err != nil {
			return nil, err
		}
		if _, exists := state.items[item.ID]; exists {
			return nil, fmt.Errorf("work item %s already exists", item.ID)
		}
		state.items[item.ID] = cloneWorkItem(item)
	}
	for _, link := range snapshot.Links {
		if err := state.validateWorkItemLink(link); err != nil {
			return nil, err
		}
		if _, exists := state.links[link.ID]; exists {
			return nil, fmt.Errorf("work item link %s already exists", link.ID)
		}
		if link.Type == WorkItemLinkBlocks && state.wouldCreateBlockingCycle(link.SourceWorkItemID, link.TargetWorkItemID) {
			return nil, fmt.Errorf("work item link creates blocking cycle")
		}
		state.links[link.ID] = cloneWorkItemLink(link)
	}
	for _, run := range snapshot.Runs {
		if _, ok := state.items[run.WorkItemID]; !ok {
			continue
		}
		if err := state.validateRun(run); err != nil {
			return nil, err
		}
		if _, exists := state.runs[run.ID]; exists {
			return nil, fmt.Errorf("work item run %s already exists", run.ID)
		}
		state.runs[run.ID] = cloneRun(run)
	}
	for _, artifact := range snapshot.Artifacts {
		if _, ok := state.items[artifact.WorkItemID]; !ok {
			continue
		}
		if err := state.validateArtifact(artifact); err != nil {
			return nil, err
		}
		if _, exists := state.artifacts[artifact.ID]; exists {
			return nil, fmt.Errorf("artifact %s already exists", artifact.ID)
		}
		state.artifacts[artifact.ID] = cloneArtifact(artifact)
	}
	for _, question := range snapshot.Questions {
		if _, ok := state.items[question.WorkItemID]; !ok {
			continue
		}
		if err := state.validateQuestion(question); err != nil {
			return nil, err
		}
		if _, exists := state.questions[question.ID]; exists {
			return nil, fmt.Errorf("question %s already exists", question.ID)
		}
		state.questions[question.ID] = cloneQuestion(question)
	}
	for _, gate := range snapshot.GateReports {
		if _, ok := state.items[gate.WorkItemID]; !ok {
			continue
		}
		if err := state.validateGateReport(gate); err != nil {
			return nil, err
		}
		if _, exists := state.gateReports[gate.ID]; exists {
			return nil, fmt.Errorf("gate report %s already exists", gate.ID)
		}
		state.gateReports[gate.ID] = cloneGateReport(gate)
	}
	for _, event := range snapshot.WorkflowEvents {
		if event.WorkItemID != "" {
			if _, ok := state.items[event.WorkItemID]; !ok {
				continue
			}
		}
		if event.RunID != "" {
			if _, ok := state.runs[event.RunID]; !ok {
				continue
			}
		}
		if event.ID == "" {
			return nil, fmt.Errorf("workflow event id required")
		}
		if _, exists := state.workflowEvents[event.ID]; exists {
			return nil, fmt.Errorf("workflow event %s already exists", event.ID)
		}
		state.workflowEvents[event.ID] = event
	}
	for _, event := range snapshot.StatusEvents {
		if event.WorkItemID != "" {
			if _, ok := state.items[event.WorkItemID]; !ok {
				continue
			}
		}
		if event.RunID != "" {
			if _, ok := state.runs[event.RunID]; !ok {
				continue
			}
		}
		if err := state.validateStatusEvent(event); err != nil {
			return nil, err
		}
		if _, exists := state.statusEvents[event.ID]; exists {
			return nil, fmt.Errorf("status event %s already exists", event.ID)
		}
		state.statusEvents[event.ID] = cloneStatusEvent(event)
	}
	return state, nil
}

func DefaultWorkflowTemplate(now time.Time) WorkflowTemplate {
	stages := []WorkflowStage{
		{ID: StageBacklog, Name: "Backlog", Kind: StageKindBacklog, DefaultRunPreset: RunPresetReader, DefaultPromptTemplateID: PromptTemplatePlan},
		{ID: StagePlanning, Name: "Planning", Kind: StageKindPlanning, DefaultRunPreset: RunPresetReader, DefaultPromptTemplateID: PromptTemplatePlan},
		{ID: StageReady, Name: "Ready", Kind: StageKindReady, DefaultRunPreset: RunPresetManager, DefaultPromptTemplateID: PromptTemplatePlan},
		{ID: StageExecution, Name: "Execution", Kind: StageKindExecution, DefaultRunPreset: RunPresetWriter, DefaultPromptTemplateID: PromptTemplateImplement, ProvisionWorktree: true},
		{ID: StageBlocked, Name: "Blocked", Kind: StageKindBlocked},
		{ID: StageReview, Name: "Review", Kind: StageKindReview, DefaultRunPreset: RunPresetReviewer, DefaultPromptTemplateID: PromptTemplateReview},
		{ID: StageDone, Name: "Done", Kind: StageKindDone},
	}
	return WorkflowTemplate{
		ID:        "default",
		Name:      "Default",
		Source:    "builtin",
		Stages:    stages,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func DefaultPromptTemplates(now time.Time) []PromptTemplate {
	return []PromptTemplate{
		{
			ID:        PromptTemplatePlan,
			Name:      "Plan",
			Source:    "builtin",
			Body:      "Plan the work item.\n\nProject: {{project.name}}\nRoot: {{project.rootDir}}\nWork item: #{{work_item.number}} {{work_item.title}}\n\n{{work_item.bodyMarkdown}}\n\nAttachments:\n{{attachments}}\n\nIf an external plan review tool opens, wait for its approval before submitting the plan to Whisk. If it requests changes, revise the plan and resubmit it there first. Do not submit drafts to Whisk before external plan approval.\n\nExitPlanMode approval does not authorize implementation. When the approved plan is ready, call ExitPlanMode with the full plan markdown. Whisk will submit that plan for review and deny continuation so implementation happens only after human plan approval. Do not write files, edit code, run tests, install dependencies, or begin implementation in this stage.\n\nIf ExitPlanMode fails to include the plan text, use the fallback CLI callback:\n${WHISK_CLI:-whisk} workflow submit-plan -body '<plan markdown>'\n\nDo not treat the plan as complete until Whisk confirms it was submitted for review.\n",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        PromptTemplateImplement,
			Name:      "Implement",
			Source:    "builtin",
			Body:      "Implement the work item.\n\nProject: {{project.name}}\nRoot: {{project.rootDir}}\nWorktree: {{worktree.path}}\nBranch: {{worktree.branch}}\nWork item: #{{work_item.number}} {{work_item.title}}\n\n{{work_item.bodyMarkdown}}\n\nAttachments:\n{{attachments}}\n",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        PromptTemplateReview,
			Name:      "Review",
			Source:    "builtin",
			Body:      "Review the work item output.\n\nProject: {{project.name}}\nRoot: {{project.rootDir}}\nWorktree: {{worktree.path}}\nBranch: {{worktree.branch}}\nWork item: #{{work_item.number}} {{work_item.title}}\n\n{{work_item.bodyMarkdown}}\n\nAttachments:\n{{attachments}}\n",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

func (s *State) Snapshot() Snapshot {
	return Snapshot{
		Projects:            s.ListProjects(),
		Templates:           s.ListWorkflowTemplates(),
		WorkflowDefinitions: s.ListWorkflowDefinitions(),
		PromptTemplates:     s.ListPromptTemplates(),
		Items:               s.ListWorkItems(""),
		Links:               s.ListWorkItemLinks(""),
		Runs:                s.ListRuns(""),
		Artifacts:           s.ListArtifacts(""),
		Questions:           s.ListQuestions(""),
		GateReports:         s.ListGateReports(""),
		WorkflowEvents:      s.ListWorkflowEvents(""),
		StatusEvents:        s.ListStatusEvents(ListStatusEvents{}),
	}
}

func (s *State) ListProjects() []Project {
	out := make([]Project, 0, len(s.projects))
	for _, project := range s.projects {
		out = append(out, cloneProject(project))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Slug < out[j].Slug
	})
	return out
}

func (s *State) ListWorkflowTemplates() []WorkflowTemplate {
	out := make([]WorkflowTemplate, 0, len(s.templates))
	for _, template := range s.templates {
		out = append(out, cloneTemplate(template))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) ListWorkflowDefinitions() []WorkflowDefinitionRecord {
	out := make([]WorkflowDefinitionRecord, 0, len(s.workflowDefinitions))
	for _, record := range s.workflowDefinitions {
		out = append(out, cloneWorkflowDefinitionRecord(record))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID == out[j].ID {
			return out[i].Version < out[j].Version
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) WorkflowDefinition(id string, version int) (WorkflowDefinitionRecord, bool) {
	record, ok := s.workflowDefinitions[workflowDefinitionKey{id: id, version: version}]
	return cloneWorkflowDefinitionRecord(record), ok
}

func (s *State) ImportWorkflowDefinition(req ImportWorkflowDefinition) (WorkflowDefinitionRecord, error) {
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "imported"
	}
	record, err := NewWorkflowDefinitionRecord(req.Definition, source, strings.TrimSpace(req.SourcePath), req.Now)
	if err != nil {
		return WorkflowDefinitionRecord{}, err
	}
	key := workflowDefinitionKey{id: record.ID, version: record.Version}
	existing, exists := s.workflowDefinitions[key]
	if exists {
		if existing.ContentHash != record.ContentHash {
			return WorkflowDefinitionRecord{}, fmt.Errorf("workflow definition %s@%d already exists with different content", record.ID, record.Version)
		}
		return cloneWorkflowDefinitionRecord(existing), nil
	}
	s.workflowDefinitions[key] = record
	return cloneWorkflowDefinitionRecord(record), nil
}

func (s *State) DeleteWorkflowDefinition(req DeleteWorkflowDefinition) (WorkflowDefinitionRecord, error) {
	key := workflowDefinitionKey{id: req.ID, version: req.Version}
	record, ok := s.workflowDefinitions[key]
	if !ok {
		return WorkflowDefinitionRecord{}, fmt.Errorf("workflow definition %s@%d not found", req.ID, req.Version)
	}
	for _, item := range s.items {
		if item.WorkflowID == req.ID && item.WorkflowVersion == req.Version {
			return WorkflowDefinitionRecord{}, fmt.Errorf("workflow definition %s@%d used by work item %s", req.ID, req.Version, item.ID)
		}
	}
	for _, project := range s.projects {
		if project.Workflow.DefinitionID == req.ID && project.Workflow.DefinitionVersion == req.Version {
			return WorkflowDefinitionRecord{}, fmt.Errorf("workflow definition %s@%d used by project %s", req.ID, req.Version, project.ID)
		}
	}
	delete(s.workflowDefinitions, key)
	return cloneWorkflowDefinitionRecord(record), nil
}

func (s *State) SetProjectWorkflowDefinition(req SetProjectWorkflowDefinition) (Project, error) {
	project, ok := s.projects[req.ProjectID]
	if !ok {
		return Project{}, fmt.Errorf("project %s not found", req.ProjectID)
	}
	record, ok := s.workflowDefinitions[workflowDefinitionKey{id: req.ID, version: req.Version}]
	if !ok {
		return Project{}, fmt.Errorf("workflow definition %s@%d not found", req.ID, req.Version)
	}
	project.Workflow.DefinitionID = record.ID
	project.Workflow.DefinitionVersion = record.Version
	project.Workflow.DefinitionHash = record.ContentHash
	project.Workflow.Stages = stagesForWorkflowDefinition(record.Definition)
	project.UpdatedAt = req.Now
	if err := validateProject(project); err != nil {
		return Project{}, err
	}
	s.projects[project.ID] = project
	return cloneProject(project), nil
}

func (s *State) PlanProjectWorkflowMigration(req PlanProjectWorkflowMigration) (WorkflowMigrationPlan, error) {
	project, ok := s.projects[req.ProjectID]
	if !ok {
		return WorkflowMigrationPlan{}, fmt.Errorf("project %s not found", req.ProjectID)
	}
	target, ok := s.workflowDefinitions[workflowDefinitionKey{id: req.ID, version: req.Version}]
	if !ok {
		return WorkflowMigrationPlan{}, fmt.Errorf("workflow definition %s@%d not found", req.ID, req.Version)
	}
	targetStages := map[string]struct{}{}
	for _, stage := range target.Definition.Stages {
		targetStages[stage] = struct{}{}
	}
	plan := WorkflowMigrationPlan{
		ProjectID:      project.ID,
		CurrentID:      project.Workflow.DefinitionID,
		CurrentVersion: project.Workflow.DefinitionVersion,
		TargetID:       target.ID,
		TargetVersion:  target.Version,
		Items:          []WorkflowMigrationItem{},
	}
	for _, item := range s.ListWorkItems(project.ID) {
		migrationItem := WorkflowMigrationItem{
			WorkItemID:             item.ID,
			Number:                 item.Number,
			Title:                  item.Title,
			CurrentWorkflowID:      item.WorkflowID,
			CurrentWorkflowVersion: item.WorkflowVersion,
			CurrentStageID:         item.StageID,
		}
		if _, ok := targetStages[item.StageID]; ok {
			migrationItem.Compatible = true
			migrationItem.TargetStageID = item.StageID
			plan.CompatibleItems++
		} else {
			migrationItem.Reason = fmt.Sprintf("stage %s not present in target workflow", item.StageID)
			plan.IncompatibleItems++
		}
		plan.ExistingItems++
		plan.ItemsPinnedToCurrentVersion++
		plan.Items = append(plan.Items, migrationItem)
	}
	return plan, nil
}

func (s *State) ListPromptTemplates() []PromptTemplate {
	out := make([]PromptTemplate, 0, len(s.promptTemplates))
	for _, prompt := range s.promptTemplates {
		out = append(out, prompt)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) ListWorkItems(projectID string) []WorkItem {
	out := make([]WorkItem, 0, len(s.items))
	for _, item := range s.items {
		if projectID == "" || item.ProjectID == projectID {
			out = append(out, cloneWorkItem(item))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ProjectID != out[j].ProjectID {
			return out[i].ProjectID < out[j].ProjectID
		}
		if out[i].Number != out[j].Number {
			return out[i].Number < out[j].Number
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) ListWorkflowActionAvailability(workItemID string) ([]WorkflowActionAvailability, error) {
	item, ok := s.items[workItemID]
	if !ok {
		return nil, fmt.Errorf("work item %s not found", workItemID)
	}
	definition, err := s.workflowDefinitionForItem(item)
	if err != nil {
		return nil, err
	}
	out := []WorkflowActionAvailability{}
	for _, action := range definition.Actions {
		if !workflowActionCanStartFrom(action, item.StageID) {
			continue
		}
		out = append(out, s.workflowActionAvailability(item, action))
	}
	return out, nil
}

func (s *State) ListWorkItemLinks(workItemID string) []WorkItemLink {
	out := make([]WorkItemLink, 0, len(s.links))
	for _, link := range s.links {
		if workItemID == "" || link.SourceWorkItemID == workItemID || link.TargetWorkItemID == workItemID {
			out = append(out, cloneWorkItemLink(link))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ProjectID != out[j].ProjectID {
			return out[i].ProjectID < out[j].ProjectID
		}
		if out[i].SourceWorkItemID != out[j].SourceWorkItemID {
			return out[i].SourceWorkItemID < out[j].SourceWorkItemID
		}
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		if out[i].TargetWorkItemID != out[j].TargetWorkItemID {
			return out[i].TargetWorkItemID < out[j].TargetWorkItemID
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func BuildReadyWorkExplanation(input ReadyWorkInput) ReadyWorkExplanation {
	projectsByID := map[string]Project{}
	for _, project := range input.Projects {
		projectsByID[project.ID] = project
	}
	itemsByID := map[string]WorkItem{}
	for _, item := range input.WorkItems {
		if input.ProjectID != "" && item.ProjectID != input.ProjectID {
			continue
		}
		itemsByID[item.ID] = cloneWorkItem(item)
	}

	outgoing := map[string][]WorkItemLink{}
	incoming := map[string][]WorkItemLink{}
	for _, link := range input.Links {
		if input.ProjectID != "" && link.ProjectID != input.ProjectID {
			continue
		}
		outgoing[link.SourceWorkItemID] = append(outgoing[link.SourceWorkItemID], cloneWorkItemLink(link))
		incoming[link.TargetWorkItemID] = append(incoming[link.TargetWorkItemID], cloneWorkItemLink(link))
	}

	explanation := ReadyWorkExplanation{}
	for _, item := range input.WorkItems {
		if input.ProjectID != "" && item.ProjectID != input.ProjectID {
			continue
		}
		if !readyWorkCandidate(item, projectsByID) {
			continue
		}

		var parentID *string
		var resolved []string
		var blockers []ReadyBlockerInfo
		for _, link := range outgoing[item.ID] {
			switch link.Type {
			case WorkItemLinkParentChild:
				parent := link.TargetWorkItemID
				parentID = &parent
			case WorkItemLinkBlocks:
				blocker, ok := itemsByID[link.TargetWorkItemID]
				if ok && workItemDone(blocker, projectsByID) {
					resolved = append(resolved, link.TargetWorkItemID)
					continue
				}
				blockers = append(blockers, readyBlockerInfo(link.TargetWorkItemID, blocker, ok))
			}
		}

		if len(blockers) > 0 {
			explanation.Blocked = append(explanation.Blocked, BlockedWorkItem{
				WorkItem:       cloneWorkItem(item),
				BlockedBy:      blockers,
				BlockedByCount: len(blockers),
			})
			continue
		}

		reason := "no blocking dependencies"
		if len(resolved) > 0 {
			reason = fmt.Sprintf("%d blocker(s) resolved", len(resolved))
		}
		explanation.Ready = append(explanation.Ready, ReadyWorkItem{
			WorkItem:         cloneWorkItem(item),
			Reason:           reason,
			ResolvedBlockers: resolved,
			DependencyCount:  len(outgoing[item.ID]),
			DependentCount:   len(incoming[item.ID]),
			ParentWorkItemID: parentID,
		})
	}

	sort.Slice(explanation.Ready, func(i, j int) bool {
		return workItemLess(explanation.Ready[i].WorkItem, explanation.Ready[j].WorkItem)
	})
	sort.Slice(explanation.Blocked, func(i, j int) bool {
		return workItemLess(explanation.Blocked[i].WorkItem, explanation.Blocked[j].WorkItem)
	})
	explanation.Summary = ReadyWorkSummary{
		TotalReady:   len(explanation.Ready),
		TotalBlocked: len(explanation.Blocked),
		CycleCount:   len(explanation.Cycles),
	}
	return explanation
}

func readyWorkCandidate(item WorkItem, projectsByID map[string]Project) bool {
	if kind, ok := stageKindForItem(item, projectsByID); ok {
		switch kind {
		case StageKindBacklog, StageKindReady:
		default:
			return false
		}
	} else {
		switch item.StageID {
		case StagePlanning, StageExecution, StageBlocked, StageReview, StageDone:
			return false
		}
	}
	switch item.RunState {
	case RunStateQueued, RunStateRunning, RunStateAwaitingInput:
		return false
	}
	return true
}

func workItemDone(item WorkItem, projectsByID map[string]Project) bool {
	if kind, ok := stageKindForItem(item, projectsByID); ok {
		return kind == StageKindDone
	}
	return item.StageID == StageDone
}

func stageKindForItem(item WorkItem, projectsByID map[string]Project) (string, bool) {
	project, ok := projectsByID[item.ProjectID]
	if !ok {
		return "", false
	}
	stage, ok := project.stage(item.StageID)
	if !ok {
		return "", false
	}
	return stage.Kind, true
}

func readyBlockerInfo(id string, blocker WorkItem, ok bool) ReadyBlockerInfo {
	info := ReadyBlockerInfo{ID: id}
	if !ok {
		return info
	}
	info.Number = blocker.Number
	info.Title = blocker.Title
	info.StageID = blocker.StageID
	info.RunState = blocker.RunState
	return info
}

func workItemLess(left WorkItem, right WorkItem) bool {
	if left.ProjectID != right.ProjectID {
		return left.ProjectID < right.ProjectID
	}
	if left.Number != right.Number {
		return left.Number < right.Number
	}
	return left.ID < right.ID
}

func (s *State) ListRuns(workItemID string) []WorkItemRun {
	out := make([]WorkItemRun, 0, len(s.runs))
	for _, run := range s.runs {
		if workItemID == "" || run.WorkItemID == workItemID {
			out = append(out, cloneRun(run))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].WorkItemID != out[j].WorkItemID {
			return out[i].WorkItemID < out[j].WorkItemID
		}
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) GetRun(id string) (WorkItemRun, bool) {
	run, ok := s.runs[id]
	return cloneRun(run), ok
}

func (s *State) ListArtifacts(workItemID string) []Artifact {
	out := make([]Artifact, 0, len(s.artifacts))
	for _, artifact := range s.artifacts {
		if workItemID == "" || artifact.WorkItemID == workItemID {
			out = append(out, cloneArtifact(artifact))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) ListQuestions(workItemID string) []Question {
	out := make([]Question, 0, len(s.questions))
	for _, question := range s.questions {
		if workItemID == "" || question.WorkItemID == workItemID {
			out = append(out, cloneQuestion(question))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) ListGateReports(workItemID string) []GateReport {
	out := make([]GateReport, 0, len(s.gateReports))
	for _, gate := range s.gateReports {
		if workItemID == "" || gate.WorkItemID == workItemID {
			out = append(out, cloneGateReport(gate))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) ListWorkflowEvents(workItemID string) []WorkflowEvent {
	out := make([]WorkflowEvent, 0, len(s.workflowEvents))
	for _, event := range s.workflowEvents {
		if workItemID == "" || event.WorkItemID == workItemID {
			out = append(out, event)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.Before(out[j].At)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) ListStatusEvents(filter ListStatusEvents) []StatusEvent {
	out := make([]StatusEvent, 0, len(s.statusEvents))
	for _, event := range s.statusEvents {
		if filter.ProjectID != "" && event.ProjectID != filter.ProjectID {
			continue
		}
		if filter.WorkItemID != "" && event.WorkItemID != filter.WorkItemID {
			continue
		}
		if filter.RunID != "" && event.RunID != filter.RunID {
			continue
		}
		if filter.SessionID != "" && event.SessionID != filter.SessionID {
			continue
		}
		if filter.PTYID != "" && event.PTYID != filter.PTYID {
			continue
		}
		if filter.UnreadOnly && event.ReadAt != nil {
			continue
		}
		out = append(out, cloneStatusEvent(event))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) GetProject(id string) (Project, bool) {
	project, ok := s.projects[id]
	return cloneProject(project), ok
}

func (s *State) GetWorkItem(id string) (WorkItem, bool) {
	item, ok := s.items[id]
	return cloneWorkItem(item), ok
}

func (s *State) CreateProject(req CreateProject) (Project, error) {
	if req.ID == "" {
		return Project{}, fmt.Errorf("project id required")
	}
	if _, exists := s.projects[req.ID]; exists {
		return Project{}, fmt.Errorf("project %s already exists", req.ID)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Project{}, fmt.Errorf("project name required")
	}
	rootDir, err := cleanAbsolutePath(req.RootDir, "root dir")
	if err != nil {
		return Project{}, err
	}
	slug := cleanSlug(req.Slug)
	if slug == "" {
		slug = cleanSlug(name)
	}
	if slug == "" {
		return Project{}, fmt.Errorf("project slug required")
	}
	workflowID := req.WorkflowID
	if workflowID == "" {
		workflowID = "default"
	}
	template, ok := s.templates[workflowID]
	if !ok {
		return Project{}, fmt.Errorf("workflow template %s not found", workflowID)
	}
	definition := DefaultWorkflowDefinitionRecord(time.Time{})
	if record, ok := s.workflowDefinitions[workflowDefinitionKey{id: WorkflowPlanExecuteReview, version: 1}]; ok {
		definition = record
	}
	projectWorkflowID := req.ProjectWorkflowID
	if projectWorkflowID == "" {
		projectWorkflowID = req.ID + "-workflow"
	}
	nextNumber := req.NextWorkItemNumber
	if nextNumber <= 0 {
		nextNumber = 1
	}
	project := Project{
		ID:          req.ID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Slug:        slug,
		RootDir:     rootDir,
		Workflow: ProjectWorkflow{
			ID:                projectWorkflowID,
			TemplateID:        template.ID,
			DefinitionID:      definition.ID,
			DefinitionVersion: definition.Version,
			DefinitionHash:    definition.ContentHash,
			Name:              template.Name,
			Stages:            cloneStages(template.Stages),
			TransitionRules:   cloneTransitionRules(template.TransitionRules),
		},
		Preferences:        defaultProjectPreferences(req.Preferences),
		Metadata:           map[string]MetadataValue{},
		NextWorkItemNumber: nextNumber,
		CreatedAt:          req.Now,
		UpdatedAt:          req.Now,
	}
	if err := validateProject(project); err != nil {
		return Project{}, err
	}
	s.projects[project.ID] = project
	return cloneProject(project), nil
}

func (s *State) UpdateProject(req UpdateProject) (Project, error) {
	project, ok := s.projects[req.ID]
	if !ok {
		return Project{}, fmt.Errorf("project %s not found", req.ID)
	}
	if req.Name != nil {
		project.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		project.Description = strings.TrimSpace(*req.Description)
	}
	if req.Slug != nil {
		project.Slug = cleanSlug(*req.Slug)
	}
	if req.UseInteractiveAgentShell != nil {
		project.Preferences.UseInteractiveAgentShell = *req.UseInteractiveAgentShell
	}
	if len(req.DefaultPhaseAgents) > 0 {
		merged := map[string]string{}
		for key, value := range project.Preferences.DefaultPhaseAgents {
			merged[key] = value
		}
		for key, value := range req.DefaultPhaseAgents {
			if strings.TrimSpace(value) == "" {
				delete(merged, key)
				continue
			}
			merged[key] = value
		}
		if len(merged) == 0 {
			merged = nil
		}
		project.Preferences.DefaultPhaseAgents = merged
	}
	project.UpdatedAt = req.Now
	if err := validateProject(project); err != nil {
		return Project{}, err
	}
	s.projects[project.ID] = project
	return cloneProject(project), nil
}

func (s *State) DeleteProject(req DeleteProject) (Project, error) {
	project, ok := s.projects[req.ID]
	if !ok {
		return Project{}, fmt.Errorf("project %s not found", req.ID)
	}
	delete(s.projects, req.ID)
	for id, item := range s.items {
		if item.ProjectID == req.ID {
			delete(s.items, id)
		}
	}
	for id, link := range s.links {
		if link.ProjectID == req.ID {
			delete(s.links, id)
		}
	}
	for id, run := range s.runs {
		if run.ProjectID == req.ID {
			delete(s.runs, id)
		}
	}
	for id, artifact := range s.artifacts {
		if artifact.ProjectID == req.ID {
			delete(s.artifacts, id)
		}
	}
	for id, question := range s.questions {
		if question.ProjectID == req.ID {
			delete(s.questions, id)
		}
	}
	for id, gate := range s.gateReports {
		if gate.ProjectID == req.ID {
			delete(s.gateReports, id)
		}
	}
	for id, event := range s.workflowEvents {
		if event.ProjectID == req.ID {
			delete(s.workflowEvents, id)
		}
	}
	for id, event := range s.statusEvents {
		if event.ProjectID == req.ID {
			delete(s.statusEvents, id)
		}
	}
	return cloneProject(project), nil
}

func (s *State) AddProjectAttachment(req AddProjectAttachment) (Project, error) {
	project, ok := s.projects[req.ProjectID]
	if !ok {
		return Project{}, fmt.Errorf("project %s not found", req.ProjectID)
	}
	attachment, err := validateAttachment(Attachment{
		ID:               req.ID,
		Kind:             req.Kind,
		Scope:            req.Scope,
		Title:            req.Title,
		Path:             req.Path,
		URL:              req.URL,
		Note:             req.Note,
		Provider:         req.Provider,
		Target:           req.Target,
		IncludeInContext: req.IncludeInContext,
		Meta:             req.Meta,
		CreatedAt:        req.Now,
	})
	if err != nil {
		return Project{}, err
	}
	project.Attachments = append(project.Attachments, attachment)
	project.UpdatedAt = req.Now
	s.projects[project.ID] = project
	return cloneProject(project), nil
}

func (s *State) UpdateProjectAttachment(req UpdateProjectAttachment) (Project, error) {
	project, ok := s.projects[req.ProjectID]
	if !ok {
		return Project{}, fmt.Errorf("project %s not found", req.ProjectID)
	}
	index := -1
	for i, attachment := range project.Attachments {
		if attachment.ID == req.ID {
			index = i
			break
		}
	}
	if index < 0 {
		return Project{}, fmt.Errorf("attachment %s not found", req.ID)
	}
	attachment := project.Attachments[index]
	if req.Title != nil {
		attachment.Title = *req.Title
	}
	if req.Path != nil {
		attachment.Path = *req.Path
	}
	if req.URL != nil {
		attachment.URL = *req.URL
	}
	if req.Note != nil {
		attachment.Note = *req.Note
	}
	if req.Provider != nil {
		attachment.Provider = *req.Provider
	}
	if req.Target != nil {
		attachment.Target = *req.Target
	}
	if req.IncludeInContext != nil {
		attachment.IncludeInContext = *req.IncludeInContext
	}
	if req.Meta != nil {
		attachment.Meta = req.Meta
	}
	validated, err := validateAttachment(attachment)
	if err != nil {
		return Project{}, err
	}
	project.Attachments[index] = validated
	project.UpdatedAt = req.Now
	s.projects[project.ID] = project
	return cloneProject(project), nil
}

func (s *State) DeleteProjectAttachment(req DeleteProjectAttachment) (Project, error) {
	project, ok := s.projects[req.ProjectID]
	if !ok {
		return Project{}, fmt.Errorf("project %s not found", req.ProjectID)
	}
	next := project.Attachments[:0]
	found := false
	for _, attachment := range project.Attachments {
		if attachment.ID == req.ID {
			found = true
			continue
		}
		next = append(next, attachment)
	}
	if !found {
		return Project{}, fmt.Errorf("attachment %s not found", req.ID)
	}
	project.Attachments = next
	project.UpdatedAt = req.Now
	s.projects[project.ID] = project
	return cloneProject(project), nil
}

func (s *State) CreateWorkItem(req CreateWorkItem) (WorkItem, error) {
	if req.ID == "" {
		return WorkItem{}, fmt.Errorf("work item id required")
	}
	if _, exists := s.items[req.ID]; exists {
		return WorkItem{}, fmt.Errorf("work item %s already exists", req.ID)
	}
	project, ok := s.projects[req.ProjectID]
	if !ok {
		return WorkItem{}, fmt.Errorf("project %s not found", req.ProjectID)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return WorkItem{}, fmt.Errorf("work item title required")
	}
	stageID := req.StageID
	if stageID == "" {
		stageID = project.Workflow.Stages[0].ID
	}
	if !project.hasStage(stageID) {
		return WorkItem{}, fmt.Errorf("stage %s not found", stageID)
	}
	stage, _ := project.stage(stageID)
	if stage.ProvisionWorktree {
		return WorkItem{}, fmt.Errorf("stage %s requires worktree", stageID)
	}
	workflowID := project.Workflow.DefinitionID
	if req.WorkflowID != "" {
		workflowID = req.WorkflowID
	}
	workflowVersion := project.Workflow.DefinitionVersion
	if workflowID == "" {
		workflowID = WorkflowPlanExecuteReview
	}
	if workflowVersion <= 0 {
		workflowVersion = 1
	}
	workflow, ok := s.workflowDefinitions[workflowDefinitionKey{id: workflowID, version: workflowVersion}]
	if !ok {
		return WorkItem{}, fmt.Errorf("workflow definition %s@%d not found", workflowID, workflowVersion)
	}
	number := project.NextWorkItemNumber
	project.NextWorkItemNumber++
	project.UpdatedAt = req.Now
	item := WorkItem{
		ID:              req.ID,
		ProjectID:       project.ID,
		WorkflowID:      workflow.ID,
		WorkflowVersion: workflow.Version,
		Number:          number,
		Title:           title,
		BodyMarkdown:    req.BodyMarkdown,
		StageID:         stageID,
		RunState:        RunStateIdle,
		CreatedAt:       req.Now,
		UpdatedAt:       req.Now,
		History: []HistoryEvent{{
			ID:      req.HistoryID,
			Type:    HistoryCreated,
			At:      req.Now,
			Actor:   req.Actor,
			Message: "created work item",
			StageID: stageID,
		}},
	}
	if item.History[0].ID == "" {
		return WorkItem{}, fmt.Errorf("history id required")
	}
	s.projects[project.ID] = project
	s.items[item.ID] = item
	return cloneWorkItem(item), nil
}

func (s *State) UpdateWorkItem(req UpdateWorkItem) (WorkItem, error) {
	item, ok := s.items[req.ID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.ID)
	}
	if req.Title == nil && req.BodyMarkdown == nil {
		return WorkItem{}, fmt.Errorf("work item update requires title or body")
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return WorkItem{}, fmt.Errorf("work item title required")
		}
		item.Title = title
	}
	if req.BodyMarkdown != nil {
		item.BodyMarkdown = *req.BodyMarkdown
	}
	item.UpdatedAt = req.Now
	if err := appendHistory(&item, HistoryEvent{
		ID:      req.HistoryID,
		Type:    HistoryUpdated,
		At:      req.Now,
		Actor:   req.Actor,
		Message: "updated work item",
		StageID: item.StageID,
	}); err != nil {
		return WorkItem{}, err
	}
	s.items[item.ID] = item
	return cloneWorkItem(item), nil
}

func (s *State) MoveWorkItem(req MoveWorkItem) (WorkItem, error) {
	item, ok := s.items[req.ID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.ID)
	}
	project := s.projects[item.ProjectID]
	stage, ok := project.stage(req.StageID)
	if !ok {
		return WorkItem{}, fmt.Errorf("stage %s not found", req.StageID)
	}
	if stage.ProvisionWorktree && item.Worktree == nil {
		return WorkItem{}, fmt.Errorf("stage %s requires worktree", req.StageID)
	}
	availability, ok, err := s.workflowActionAvailabilityForTransition(item, req.StageID)
	if err != nil {
		return WorkItem{}, err
	}
	if !ok {
		return WorkItem{}, fmt.Errorf("workflow action from %s to %s not found", item.StageID, req.StageID)
	}
	if !availability.Enabled {
		return WorkItem{}, fmt.Errorf("%s", availability.Reason)
	}
	item.StageID = req.StageID
	item.UpdatedAt = req.Now
	if err := appendHistory(&item, HistoryEvent{
		ID:      req.HistoryID,
		Type:    HistoryStageMoved,
		At:      req.Now,
		Actor:   req.Actor,
		Message: "moved work item",
		StageID: req.StageID,
	}); err != nil {
		return WorkItem{}, err
	}
	s.items[item.ID] = item
	return cloneWorkItem(item), nil
}

func (s *State) ApplyWorkflowAction(req ApplyWorkflowAction) (WorkItem, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	action, err := s.workflowActionForItem(item, req.ActionID)
	if err != nil {
		return WorkItem{}, err
	}
	if !workflowActionCanStartFrom(action, item.StageID) {
		return WorkItem{}, fmt.Errorf("workflow action %s cannot start from %s", action.ID, item.StageID)
	}
	availability := s.workflowActionAvailability(item, action)
	if !availability.Enabled {
		return WorkItem{}, fmt.Errorf("%s", availability.Reason)
	}
	if input := workflowActionInputKind(action); input != WorkflowActionInputNone {
		return WorkItem{}, fmt.Errorf("workflow action %s requires %s input", action.ID, input)
	}
	project := s.projects[item.ProjectID]
	target := workflowActionTargetStage(action, item)
	if target == "" {
		return WorkItem{}, fmt.Errorf("workflow action %s target stage required", action.ID)
	}
	stage, ok := project.stage(target)
	if !ok {
		return WorkItem{}, fmt.Errorf("stage %s not found", target)
	}
	if stage.ProvisionWorktree && item.Worktree == nil {
		return WorkItem{}, fmt.Errorf("stage %s requires worktree", target)
	}
	if action.SideStage && item.StageID != target {
		item.PreviousStageID = item.StageID
	}
	item.StageID = target
	if action.To == "$previousStage" || stage.Kind != StageKindBlocked {
		item.PreviousStageID = ""
	}
	if stage.Kind == StageKindBlocked {
		item.RunState = RunStateAwaitingInput
	}
	item.UpdatedAt = req.Now
	if req.RunID != "" {
		run, ok := s.runs[req.RunID]
		if !ok {
			return WorkItem{}, fmt.Errorf("work item run %s not found", req.RunID)
		}
		if run.WorkItemID != item.ID {
			return WorkItem{}, fmt.Errorf("work item run %s does not belong to work item %s", req.RunID, item.ID)
		}
		if stage.Kind == StageKindBlocked {
			run.Status = RunStateAwaitingInput
			run.UpdatedAt = req.Now
			s.runs[run.ID] = run
		}
	}
	s.items[item.ID] = item
	switch action.ID {
	case WorkflowActionReportBlocked:
		s.recordWorkflowEvent(WorkflowEventBlocked, item.ProjectID, item.ID, req.RunID, req.Actor, req.Reason, req.Now)
	case WorkflowActionUnblock:
		s.recordWorkflowEvent(WorkflowEventUnblocked, item.ProjectID, item.ID, "", req.Actor, "", req.Now)
	}
	return cloneWorkItem(item), nil
}

func (s *State) BindWorktree(req BindWorktree) (WorkItem, error) {
	item, ok := s.items[req.ID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.ID)
	}
	if item.Worktree != nil {
		return WorkItem{}, fmt.Errorf("work item %s already has worktree", req.ID)
	}
	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		return WorkItem{}, fmt.Errorf("branch required")
	}
	worktreePath, err := cleanAbsolutePath(req.WorktreePath, "worktree path")
	if err != nil {
		return WorkItem{}, err
	}
	item.Worktree = &WorktreeBinding{
		Branch:       branch,
		Base:         strings.TrimSpace(req.Base),
		WorktreePath: worktreePath,
		CreatedAt:    req.Now,
	}
	item.UpdatedAt = req.Now
	if err := appendHistory(&item, HistoryEvent{
		ID:           req.HistoryID,
		Type:         HistoryWorktreeBound,
		At:           req.Now,
		Actor:        req.Actor,
		Message:      "bound worktree",
		Branch:       branch,
		WorktreePath: worktreePath,
	}); err != nil {
		return WorkItem{}, err
	}
	s.items[item.ID] = item
	return cloneWorkItem(item), nil
}

func (s *State) AddAttachment(req AddAttachment) (WorkItem, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	attachment, err := validateAttachment(Attachment{
		ID:        req.ID,
		Kind:      req.Kind,
		Scope:     req.Scope,
		Path:      req.Path,
		URL:       req.URL,
		Note:      req.Note,
		CreatedAt: req.Now,
	})
	if err != nil {
		return WorkItem{}, err
	}
	item.Attachments = append(item.Attachments, attachment)
	item.UpdatedAt = req.Now
	if err := appendHistory(&item, HistoryEvent{
		ID:           req.HistoryID,
		Type:         HistoryAttachmentAdded,
		At:           req.Now,
		Actor:        req.Actor,
		Message:      "added attachment",
		AttachmentID: attachment.ID,
	}); err != nil {
		return WorkItem{}, err
	}
	s.items[item.ID] = item
	return cloneWorkItem(item), nil
}

func (s *State) DeleteWorkItem(req DeleteWorkItem) (WorkItem, error) {
	item, ok := s.items[req.ID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.ID)
	}
	if err := appendHistory(&item, HistoryEvent{
		ID:      req.HistoryID,
		Type:    HistoryDeleted,
		At:      req.Now,
		Actor:   req.Actor,
		Message: "deleted work item",
	}); err != nil {
		return WorkItem{}, err
	}
	delete(s.items, req.ID)
	for id, run := range s.runs {
		if run.WorkItemID == req.ID {
			delete(s.runs, id)
		}
	}
	for id, artifact := range s.artifacts {
		if artifact.WorkItemID == req.ID {
			delete(s.artifacts, id)
		}
	}
	for id, question := range s.questions {
		if question.WorkItemID == req.ID {
			delete(s.questions, id)
		}
	}
	for id, gate := range s.gateReports {
		if gate.WorkItemID == req.ID {
			delete(s.gateReports, id)
		}
	}
	for id, event := range s.workflowEvents {
		if event.WorkItemID == req.ID {
			delete(s.workflowEvents, id)
		}
	}
	for id, event := range s.statusEvents {
		if event.WorkItemID == req.ID {
			delete(s.statusEvents, id)
		}
	}
	for id, link := range s.links {
		if link.SourceWorkItemID == req.ID || link.TargetWorkItemID == req.ID {
			delete(s.links, id)
		}
	}
	return cloneWorkItem(item), nil
}

func (s *State) AddWorkItemLink(req AddWorkItemLink) (WorkItemLink, error) {
	source, ok := s.items[req.SourceWorkItemID]
	if !ok {
		return WorkItemLink{}, fmt.Errorf("work item %s not found", req.SourceWorkItemID)
	}
	link := WorkItemLink{
		ID:               req.ID,
		ProjectID:        source.ProjectID,
		SourceWorkItemID: req.SourceWorkItemID,
		TargetWorkItemID: req.TargetWorkItemID,
		Type:             strings.TrimSpace(req.Type),
		CreatedBy:        req.Actor,
		CreatedAt:        req.Now,
	}
	if err := s.validateWorkItemLink(link); err != nil {
		return WorkItemLink{}, err
	}
	if _, exists := s.links[link.ID]; exists {
		return WorkItemLink{}, fmt.Errorf("work item link %s already exists", link.ID)
	}
	for _, existing := range s.links {
		if existing.SourceWorkItemID == link.SourceWorkItemID && existing.TargetWorkItemID == link.TargetWorkItemID && existing.Type == link.Type {
			return WorkItemLink{}, fmt.Errorf("work item link already exists")
		}
	}
	if link.Type == WorkItemLinkBlocks && s.wouldCreateBlockingCycle(link.SourceWorkItemID, link.TargetWorkItemID) {
		return WorkItemLink{}, fmt.Errorf("work item link creates blocking cycle")
	}
	s.links[link.ID] = link
	return cloneWorkItemLink(link), nil
}

func (s *State) StartRun(req StartRun) (WorkItemRun, error) {
	if req.ID == "" {
		return WorkItemRun{}, fmt.Errorf("run id required")
	}
	if _, exists := s.runs[req.ID]; exists {
		return WorkItemRun{}, fmt.Errorf("work item run %s already exists", req.ID)
	}
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	project := s.projects[item.ProjectID]
	preset := strings.TrimSpace(req.Preset)
	if preset == "" {
		preset = defaultPresetForStage(project, item.StageID)
	}
	if !validPreset(preset) {
		return WorkItemRun{}, fmt.Errorf("unsupported run preset %s", preset)
	}
	templateID := strings.TrimSpace(req.PromptTemplateID)
	if templateID == "" {
		templateID = defaultPromptForStage(project, item.StageID)
	}
	template, ok := s.promptTemplates[templateID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("prompt template %s not found", templateID)
	}
	prompt := renderPrompt(template.Body, project, item)
	run := WorkItemRun{
		ID:               req.ID,
		WorkItemID:       item.ID,
		ProjectID:        item.ProjectID,
		Preset:           preset,
		PromptTemplateID: template.ID,
		PromptSnapshot:   prompt,
		SessionID:        strings.TrimSpace(req.SessionID),
		PTYID:            strings.TrimSpace(req.PTYID),
		Status:           RunStateQueued,
		CreatedAt:        req.Now,
		UpdatedAt:        req.Now,
		History: []RunEvent{{
			ID:      req.RunHistoryID,
			Type:    RunStateQueued,
			At:      req.Now,
			Actor:   req.Actor,
			Message: "started work item run",
		}},
	}
	if run.History[0].ID == "" {
		return WorkItemRun{}, fmt.Errorf("run history id required")
	}
	if err := appendHistory(&item, HistoryEvent{
		ID:      req.HistoryID,
		Type:    HistoryRunEventAppended,
		At:      req.Now,
		Actor:   req.Actor,
		Message: "started work item run",
	}); err != nil {
		return WorkItemRun{}, err
	}
	item.RunState = RunStateQueued
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	s.runs[run.ID] = run
	return cloneRun(run), nil
}

func (s *State) StartPlanning(req StartPlanning) (WorkItemRun, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	action, err := s.workflowActionForItem(item, WorkflowActionStartPlanning)
	if err != nil {
		return WorkItemRun{}, err
	}
	item.StageID = action.To
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	runEffect := action.CreatesRun
	if runEffect == nil {
		return WorkItemRun{}, fmt.Errorf("workflow action %s does not create a run", action.ID)
	}
	run, err := s.StartRun(StartRun{
		ID:               req.ID,
		HistoryID:        req.HistoryID,
		RunHistoryID:     req.RunHistoryID,
		WorkItemID:       req.WorkItemID,
		Preset:           runEffect.Preset,
		PromptTemplateID: runEffect.PromptTemplateID,
		SessionID:        req.SessionID,
		PTYID:            req.PTYID,
		Actor:            req.Actor,
		Now:              req.Now,
	})
	if err != nil {
		return WorkItemRun{}, err
	}
	s.recordWorkflowEvent(WorkflowEventPlanningStarted, item.ProjectID, item.ID, run.ID, req.Actor, "", req.Now)
	return run, nil
}

func (s *State) SubmitDraftPlan(req SubmitDraftPlan) (Artifact, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return Artifact{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return Artifact{}, fmt.Errorf("plan body required")
	}
	action, err := s.workflowActionForItem(item, WorkflowActionSubmitDraftPlan)
	if err != nil {
		return Artifact{}, err
	}
	effect := action.CreatesArtifact
	if effect == nil {
		return Artifact{}, fmt.Errorf("workflow action %s does not create an artifact", action.ID)
	}
	artifact := Artifact{
		ID:         req.ID,
		ProjectID:  item.ProjectID,
		WorkItemID: item.ID,
		RunID:      strings.TrimSpace(req.RunID),
		Kind:       effect.Kind,
		Status:     effect.Status,
		Title:      strings.TrimSpace(req.Title),
		Body:       body,
		CreatedAt:  req.Now,
		UpdatedAt:  req.Now,
	}
	if artifact.ID == "" {
		return Artifact{}, fmt.Errorf("artifact id required")
	}
	if _, exists := s.artifacts[artifact.ID]; exists {
		return Artifact{}, fmt.Errorf("artifact %s already exists", artifact.ID)
	}
	if err := s.validateArtifact(artifact); err != nil {
		return Artifact{}, err
	}
	s.artifacts[artifact.ID] = artifact
	s.recordWorkflowEvent(WorkflowEventDraftPlanSubmitted, item.ProjectID, item.ID, artifact.RunID, req.Actor, "", req.Now)
	return cloneArtifact(artifact), nil
}

func (s *State) ApprovePlan(req ApprovePlan) (WorkItem, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	artifact, ok := s.artifacts[req.ArtifactID]
	if !ok {
		return WorkItem{}, fmt.Errorf("artifact %s not found", req.ArtifactID)
	}
	if artifact.WorkItemID != item.ID || artifact.Kind != ArtifactKindPlan {
		return WorkItem{}, fmt.Errorf("plan artifact required")
	}
	action, err := s.workflowActionForItem(item, WorkflowActionApprovePlan)
	if err != nil {
		return WorkItem{}, err
	}
	if action.UpdatesArtifact == nil {
		return WorkItem{}, fmt.Errorf("workflow action %s does not update an artifact", action.ID)
	}
	artifact.Status = action.UpdatesArtifact.Status
	artifact.UpdatedAt = req.Now
	item.StageID = action.To
	item.UpdatedAt = req.Now
	s.artifacts[artifact.ID] = artifact
	s.items[item.ID] = item
	s.recordWorkflowEvent(WorkflowEventPlanApproved, item.ProjectID, item.ID, artifact.RunID, req.Actor, "", req.Now)
	return cloneWorkItem(item), nil
}

func (s *State) StartExecution(req StartExecution) (WorkItemRun, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	action, err := s.workflowActionForItem(item, WorkflowActionStartExecution)
	if err != nil {
		return WorkItemRun{}, err
	}
	if !s.hasRequiredArtifacts(item.ID, action.Requires) {
		return WorkItemRun{}, fmt.Errorf("approved plan required")
	}
	item.StageID = action.To
	item.PreviousStageID = ""
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	runEffect := action.CreatesRun
	if runEffect == nil {
		return WorkItemRun{}, fmt.Errorf("workflow action %s does not create a run", action.ID)
	}
	run, err := s.StartRun(StartRun{
		ID:               req.ID,
		HistoryID:        req.HistoryID,
		RunHistoryID:     req.RunHistoryID,
		WorkItemID:       req.WorkItemID,
		Preset:           runEffect.Preset,
		PromptTemplateID: runEffect.PromptTemplateID,
		SessionID:        req.SessionID,
		PTYID:            req.PTYID,
		Actor:            req.Actor,
		Now:              req.Now,
	})
	if err != nil {
		return WorkItemRun{}, err
	}
	s.recordWorkflowEvent(WorkflowEventExecutionStarted, item.ProjectID, item.ID, run.ID, req.Actor, "", req.Now)
	return run, nil
}

func (s *State) AskQuestion(req AskQuestion) (Question, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return Question{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return Question{}, fmt.Errorf("question prompt required")
	}
	question := Question{
		ID:         req.ID,
		ProjectID:  item.ProjectID,
		WorkItemID: item.ID,
		RunID:      strings.TrimSpace(req.RunID),
		SessionID:  strings.TrimSpace(req.SessionID),
		PTYID:      strings.TrimSpace(req.PTYID),
		Prompt:     prompt,
		Status:     QuestionStatusOpen,
		Actor:      req.Actor,
		CreatedAt:  req.Now,
		UpdatedAt:  req.Now,
	}
	if question.ID == "" {
		return Question{}, fmt.Errorf("question id required")
	}
	if _, exists := s.questions[question.ID]; exists {
		return Question{}, fmt.Errorf("question %s already exists", question.ID)
	}
	if question.RunID != "" {
		run, ok := s.runs[question.RunID]
		if !ok {
			return Question{}, fmt.Errorf("work item run %s not found", question.RunID)
		}
		run.Status = RunStateAwaitingInput
		run.UpdatedAt = req.Now
		if question.SessionID == "" {
			question.SessionID = run.SessionID
		}
		if question.PTYID == "" {
			question.PTYID = run.PTYID
		}
		s.runs[run.ID] = run
		item.RunState = RunStateAwaitingInput
		item.UpdatedAt = req.Now
		s.items[item.ID] = item
	}
	if err := s.validateQuestion(question); err != nil {
		return Question{}, err
	}
	s.questions[question.ID] = question
	s.recordWorkflowEvent(WorkflowEventQuestionAsked, item.ProjectID, item.ID, question.RunID, req.Actor, prompt, req.Now)
	return cloneQuestion(question), nil
}

func (s *State) AnswerQuestion(req AnswerQuestion) (Question, error) {
	question, ok := s.questions[req.ID]
	if !ok {
		return Question{}, fmt.Errorf("question %s not found", req.ID)
	}
	answer := strings.TrimSpace(req.Answer)
	if answer == "" {
		return Question{}, fmt.Errorf("question answer required")
	}
	question.Answer = answer
	question.Status = QuestionStatusAnswered
	question.AnsweredBy = req.Actor
	question.AnsweredAt = &req.Now
	question.UpdatedAt = req.Now
	s.questions[question.ID] = question
	if question.RunID != "" && !s.hasOpenQuestionForRun(question.RunID) {
		run := s.runs[question.RunID]
		if run.Status == RunStateAwaitingInput {
			run.Status = RunStateRunning
			run.UpdatedAt = req.Now
			s.runs[run.ID] = run
			item := s.items[run.WorkItemID]
			item.RunState = RunStateRunning
			item.UpdatedAt = req.Now
			s.items[item.ID] = item
		}
	}
	s.recordWorkflowEvent(WorkflowEventQuestionAnswered, question.ProjectID, question.WorkItemID, question.RunID, req.Actor, "", req.Now)
	return cloneQuestion(question), nil
}

func (s *State) ReportBlocked(req ReportBlocked) (WorkItem, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	action, err := s.workflowActionForItem(item, WorkflowActionReportBlocked)
	if err != nil {
		return WorkItem{}, err
	}
	if item.StageID != action.To {
		item.PreviousStageID = item.StageID
	}
	item.StageID = action.To
	item.RunState = RunStateAwaitingInput
	item.UpdatedAt = req.Now
	if req.RunID != "" {
		run := s.runs[req.RunID]
		run.Status = RunStateAwaitingInput
		run.UpdatedAt = req.Now
		s.runs[run.ID] = run
	}
	s.items[item.ID] = item
	s.recordWorkflowEvent(WorkflowEventBlocked, item.ProjectID, item.ID, req.RunID, req.Actor, req.Reason, req.Now)
	return cloneWorkItem(item), nil
}

func (s *State) Unblock(req Unblock) (WorkItem, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	action, err := s.workflowActionForItem(item, WorkflowActionUnblock)
	if err != nil {
		return WorkItem{}, err
	}
	blockedStage := StageBlocked
	for _, stage := range action.From {
		blockedStage = stage
		break
	}
	if item.StageID != blockedStage {
		return WorkItem{}, fmt.Errorf("work item %s is not blocked", req.WorkItemID)
	}
	next := action.To
	if next == "$previousStage" {
		next = item.PreviousStageID
	}
	if next == "" || next == "$previousStage" {
		next = StageExecution
	}
	item.StageID = next
	item.PreviousStageID = ""
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	s.recordWorkflowEvent(WorkflowEventUnblocked, item.ProjectID, item.ID, "", req.Actor, "", req.Now)
	return cloneWorkItem(item), nil
}

func (s *State) CompleteExecution(req CompleteExecution) (WorkItem, error) {
	run, ok := s.runs[req.RunID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item run %s not found", req.RunID)
	}
	item := s.items[run.WorkItemID]
	run.Status = RunStateCompleted
	run.CompletedAt = &req.Now
	run.UpdatedAt = req.Now
	action, err := s.workflowActionForItem(item, WorkflowActionCompleteExecution)
	if err != nil {
		return WorkItem{}, err
	}
	item.RunState = RunStateCompleted
	item.StageID = action.To
	item.UpdatedAt = req.Now
	s.runs[run.ID] = run
	s.items[item.ID] = item
	definition, err := s.workflowDefinitionForItem(item)
	if err != nil {
		return WorkItem{}, err
	}
	for _, gateID := range action.CreatesGates {
		if s.hasGateReport(item.ID, gateID) {
			continue
		}
		gateDefinition, ok := definition.Gate(gateID)
		if !ok {
			return WorkItem{}, fmt.Errorf("workflow gate %s not found", gateID)
		}
		gate := GateReport{
			ID:         fmt.Sprintf("gate_%d", len(s.gateReports)+1),
			ProjectID:  item.ProjectID,
			WorkItemID: item.ID,
			RunID:      run.ID,
			Name:       gateDefinition.ID,
			Blocking:   gateDefinition.Blocking,
			Status:     GateStatusPending,
			CreatedAt:  req.Now,
			UpdatedAt:  req.Now,
		}
		s.gateReports[gate.ID] = gate
	}
	s.recordWorkflowEvent(WorkflowEventExecutionCompleted, item.ProjectID, item.ID, run.ID, req.Actor, req.Message, req.Now)
	return cloneWorkItem(item), nil
}

func (s *State) SubmitReviewFeedback(req SubmitReviewFeedback) (Artifact, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return Artifact{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return Artifact{}, fmt.Errorf("feedback body required")
	}
	action, err := s.workflowActionForItem(item, WorkflowActionSubmitReviewFeedback)
	if err != nil {
		return Artifact{}, err
	}
	effect := action.CreatesArtifact
	if effect == nil {
		return Artifact{}, fmt.Errorf("workflow action %s does not create an artifact", action.ID)
	}
	artifact := Artifact{
		ID:         req.ID,
		ProjectID:  item.ProjectID,
		WorkItemID: item.ID,
		RunID:      strings.TrimSpace(req.RunID),
		Kind:       effect.Kind,
		Status:     effect.Status,
		Body:       body,
		CreatedAt:  req.Now,
		UpdatedAt:  req.Now,
	}
	var run WorkItemRun
	runOK := false
	if artifact.RunID != "" {
		var ok bool
		run, ok = s.runs[artifact.RunID]
		if !ok {
			return Artifact{}, fmt.Errorf("work item run %s not found", artifact.RunID)
		}
		if run.WorkItemID != item.ID {
			return Artifact{}, fmt.Errorf("work item run %s does not belong to work item %s", artifact.RunID, item.ID)
		}
		runOK = true
	}
	if artifact.ID == "" {
		return Artifact{}, fmt.Errorf("artifact id required")
	}
	if _, exists := s.artifacts[artifact.ID]; exists {
		return Artifact{}, fmt.Errorf("artifact %s already exists", artifact.ID)
	}
	if err := s.validateArtifact(artifact); err != nil {
		return Artifact{}, err
	}
	item.StageID = action.To
	item.RunState = RunStateRunning
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	if runOK {
		run.Status = RunStateRunning
		run.UpdatedAt = req.Now
		s.runs[run.ID] = run
	}
	s.artifacts[artifact.ID] = artifact
	s.recordWorkflowEvent(WorkflowEventReviewFeedbackAdded, item.ProjectID, item.ID, artifact.RunID, req.Actor, body, req.Now)
	return cloneArtifact(artifact), nil
}

func (s *State) CompleteGate(req CompleteGate) (GateReport, error) {
	gate, ok := s.gateReports[req.ID]
	if !ok {
		return GateReport{}, fmt.Errorf("gate report %s not found", req.ID)
	}
	switch req.Status {
	case GateStatusPassed, GateStatusFailed:
	case GateStatusOverridden:
		if strings.TrimSpace(req.OverrideReason) == "" {
			return GateReport{}, fmt.Errorf("override reason required")
		}
	default:
		return GateReport{}, fmt.Errorf("unsupported gate status %s", req.Status)
	}
	gate.Status = req.Status
	gate.OverrideReason = strings.TrimSpace(req.OverrideReason)
	gate.UpdatedAt = req.Now
	s.gateReports[gate.ID] = gate
	s.recordWorkflowEvent(WorkflowEventGateCompleted, gate.ProjectID, gate.WorkItemID, gate.RunID, req.Actor, gate.Status, req.Now)
	return cloneGateReport(gate), nil
}

func (s *State) ApproveDone(req ApproveDone) (WorkItem, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	action, err := s.workflowActionForItem(item, WorkflowActionApproveDone)
	if err != nil {
		return WorkItem{}, err
	}
	if action.RequiresPassingBlockingGates {
		for _, gate := range s.gateReports {
			if gate.WorkItemID == item.ID && gate.Blocking && gate.Status != GateStatusPassed && gate.Status != GateStatusOverridden {
				return WorkItem{}, fmt.Errorf("blocking gates must pass or be overridden")
			}
		}
	}
	item.StageID = action.To
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	s.recordWorkflowEvent(WorkflowEventDoneApproved, item.ProjectID, item.ID, "", req.Actor, req.Reason, req.Now)
	return cloneWorkItem(item), nil
}

func (s *State) SetMetadata(req SetMetadata) (MetadataValue, error) {
	key, err := metadataFullKey(req.Namespace, req.Key)
	if err != nil {
		return MetadataValue{}, err
	}
	if err := validateMetadataValue(req.Value); err != nil {
		return MetadataValue{}, err
	}
	switch req.OwnerType {
	case MetadataOwnerProject:
		project, ok := s.projects[req.OwnerID]
		if !ok {
			return MetadataValue{}, fmt.Errorf("project %s not found", req.OwnerID)
		}
		if project.Metadata == nil {
			project.Metadata = map[string]MetadataValue{}
		}
		project.Metadata[key] = req.Value
		project.UpdatedAt = req.Now
		s.projects[project.ID] = project
	case MetadataOwnerWorkItem:
		item, ok := s.items[req.OwnerID]
		if !ok {
			return MetadataValue{}, fmt.Errorf("work item %s not found", req.OwnerID)
		}
		if item.Metadata == nil {
			item.Metadata = map[string]MetadataValue{}
		}
		item.Metadata[key] = req.Value
		item.UpdatedAt = req.Now
		s.items[item.ID] = item
	case MetadataOwnerRun:
		run, ok := s.runs[req.OwnerID]
		if !ok {
			return MetadataValue{}, fmt.Errorf("work item run %s not found", req.OwnerID)
		}
		if run.Metadata == nil {
			run.Metadata = map[string]MetadataValue{}
		}
		run.Metadata[key] = req.Value
		run.UpdatedAt = req.Now
		s.runs[run.ID] = run
	case MetadataOwnerArtifact:
		artifact, ok := s.artifacts[req.OwnerID]
		if !ok {
			return MetadataValue{}, fmt.Errorf("artifact %s not found", req.OwnerID)
		}
		if artifact.Metadata == nil {
			artifact.Metadata = map[string]MetadataValue{}
		}
		artifact.Metadata[key] = req.Value
		artifact.UpdatedAt = req.Now
		s.artifacts[artifact.ID] = artifact
	default:
		return MetadataValue{}, fmt.Errorf("unsupported metadata owner %s", req.OwnerType)
	}
	return req.Value, nil
}

func (s *State) CancelRun(req CancelRun) (WorkItemRun, error) {
	run, ok := s.runs[req.ID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("work item run %s not found", req.ID)
	}
	if run.Status == RunStateCompleted || run.Status == RunStateFailed || run.Status == RunStateCancelled {
		return WorkItemRun{}, fmt.Errorf("work item run %s is already terminal", req.ID)
	}
	run.Status = RunStateCancelled
	run.UpdatedAt = req.Now
	run.CompletedAt = &req.Now
	if err := appendRunEvent(&run, RunEvent{
		ID:      req.RunHistoryID,
		Type:    RunStateCancelled,
		At:      req.Now,
		Actor:   req.Actor,
		Message: "cancelled work item run",
	}); err != nil {
		return WorkItemRun{}, err
	}
	item := s.items[run.WorkItemID]
	item.RunState = RunStateCancelled
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	s.runs[run.ID] = run
	s.recordWorkflowEvent(WorkflowEventRunCancelled, item.ProjectID, item.ID, run.ID, req.Actor, "cancelled work item run", req.Now)
	return cloneRun(run), nil
}

func (s *State) CompleteRun(req CompleteRun) (WorkItemRun, error) {
	run, ok := s.runs[req.ID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("work item run %s not found", req.ID)
	}
	if run.Status == RunStateCompleted || run.Status == RunStateFailed || run.Status == RunStateCancelled {
		return WorkItemRun{}, fmt.Errorf("work item run %s is already terminal", req.ID)
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = "completed work item run"
	}
	run.Status = RunStateCompleted
	run.UpdatedAt = req.Now
	run.CompletedAt = &req.Now
	if err := appendRunEvent(&run, RunEvent{
		ID:      req.RunHistoryID,
		Type:    RunStateCompleted,
		At:      req.Now,
		Actor:   req.Actor,
		Message: message,
	}); err != nil {
		return WorkItemRun{}, err
	}
	item := s.items[run.WorkItemID]
	item.RunState = RunStateCompleted
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	s.runs[run.ID] = run
	return cloneRun(run), nil
}

func (s *State) MarkRunRunning(req MarkRunRunning) (WorkItemRun, error) {
	run, ok := s.runs[req.ID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("work item run %s not found", req.ID)
	}
	if run.Status != RunStateQueued {
		return WorkItemRun{}, fmt.Errorf("work item run %s is %s, not queued", req.ID, run.Status)
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return WorkItemRun{}, fmt.Errorf("session id required")
	}
	ptyID := strings.TrimSpace(req.PTYID)
	if ptyID == "" {
		return WorkItemRun{}, fmt.Errorf("pty id required")
	}
	run.SessionID = sessionID
	run.PTYID = ptyID
	run.Status = RunStateRunning
	if req.DaemonOwned {
		if run.Metadata == nil {
			run.Metadata = map[string]MetadataValue{}
		}
		run.Metadata["whisk.daemon/owned_session"] = MetadataValue{Type: MetadataTypeBool, Bool: true}
	}
	run.UpdatedAt = req.Now
	if err := appendRunEvent(&run, RunEvent{
		ID:      req.RunHistoryID,
		Type:    RunStateRunning,
		At:      req.Now,
		Actor:   req.Actor,
		Message: "launched work item run",
	}); err != nil {
		return WorkItemRun{}, err
	}
	item := s.items[run.WorkItemID]
	item.RunState = RunStateRunning
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	s.runs[run.ID] = run
	return cloneRun(run), nil
}

func (s *State) FailRun(req FailRun) (WorkItemRun, error) {
	run, ok := s.runs[req.ID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("work item run %s not found", req.ID)
	}
	if run.Status == RunStateCompleted || run.Status == RunStateFailed || run.Status == RunStateCancelled {
		return WorkItemRun{}, fmt.Errorf("work item run %s is already terminal", req.ID)
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = "work item run failed"
	}
	run.Status = RunStateFailed
	run.UpdatedAt = req.Now
	run.CompletedAt = &req.Now
	if err := appendRunEvent(&run, RunEvent{
		ID:      req.RunHistoryID,
		Type:    RunStateFailed,
		At:      req.Now,
		Actor:   req.Actor,
		Message: message,
	}); err != nil {
		return WorkItemRun{}, err
	}
	item := s.items[run.WorkItemID]
	item.RunState = RunStateFailed
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	s.runs[run.ID] = run
	s.recordWorkflowEvent(WorkflowEventRunFailed, item.ProjectID, item.ID, run.ID, req.Actor, message, req.Now)
	return cloneRun(run), nil
}

func (s *State) ReportStatus(req ReportStatus) (StatusEvent, error) {
	kind := strings.TrimSpace(req.Kind)
	if !validStatusKind(kind) {
		return StatusEvent{}, fmt.Errorf("unsupported status kind %s", req.Kind)
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = kind
	}
	if req.ID == "" {
		return StatusEvent{}, fmt.Errorf("status event id required")
	}
	if _, exists := s.statusEvents[req.ID]; exists {
		return StatusEvent{}, fmt.Errorf("status event %s already exists", req.ID)
	}

	runID := strings.TrimSpace(req.RunID)
	workItemID := strings.TrimSpace(req.WorkItemID)
	projectID := strings.TrimSpace(req.ProjectID)
	sessionID := strings.TrimSpace(req.SessionID)
	paneID := strings.TrimSpace(req.PaneID)
	ptyID := strings.TrimSpace(req.PTYID)
	scope := StatusScopeSession

	if runID != "" {
		run, ok := s.runs[runID]
		if !ok {
			return StatusEvent{}, fmt.Errorf("work item run %s not found", runID)
		}
		if run.Status == RunStateCompleted || run.Status == RunStateFailed || run.Status == RunStateCancelled {
			return StatusEvent{}, fmt.Errorf("work item run %s is already terminal", runID)
		}
		workItemID = run.WorkItemID
		projectID = run.ProjectID
		if sessionID == "" {
			sessionID = run.SessionID
		}
		if ptyID == "" {
			ptyID = run.PTYID
		}
		scope = StatusScopeRun

		switch kind {
		case StatusKindQuestion, StatusKindBlocked:
			run.Status = RunStateAwaitingInput
			run.UpdatedAt = req.Now
			if err := appendRunEvent(&run, RunEvent{
				ID:      req.RunHistoryID,
				Type:    RunStateAwaitingInput,
				At:      req.Now,
				Actor:   req.Actor,
				Message: message,
			}); err != nil {
				return StatusEvent{}, err
			}
			item := s.items[run.WorkItemID]
			item.RunState = RunStateAwaitingInput
			item.UpdatedAt = req.Now
			s.items[item.ID] = item
		case StatusKindDone:
			run.Status = RunStateCompleted
			run.UpdatedAt = req.Now
			run.CompletedAt = &req.Now
			if err := appendRunEvent(&run, RunEvent{
				ID:      req.RunHistoryID,
				Type:    RunStateCompleted,
				At:      req.Now,
				Actor:   req.Actor,
				Message: message,
			}); err != nil {
				return StatusEvent{}, err
			}
			item := s.items[run.WorkItemID]
			item.RunState = RunStateCompleted
			project := s.projects[item.ProjectID]
			if reviewStageID := reviewStageID(project); reviewStageID != "" {
				item.StageID = reviewStageID
			} else if doneStageID := doneStageID(project); doneStageID != "" {
				item.StageID = doneStageID
			}
			item.UpdatedAt = req.Now
			s.items[item.ID] = item
		}
		s.runs[run.ID] = run
	} else if ptyID != "" {
		scope = StatusScopePTY
	} else if sessionID != "" {
		scope = StatusScopeSession
	} else {
		return StatusEvent{}, fmt.Errorf("status context required")
	}

	event := StatusEvent{
		ID:                   req.ID,
		Scope:                scope,
		Kind:                 kind,
		Message:              message,
		Actor:                req.Actor,
		ProjectID:            projectID,
		WorkItemID:           workItemID,
		RunID:                runID,
		SessionID:            sessionID,
		PaneID:               paneID,
		PTYID:                ptyID,
		RequiresAttention:    statusRequiresAttention(kind),
		NotificationKey:      statusNotificationKey(sessionID, paneID, ptyID, req.Actor, kind),
		NotificationSeverity: statusNotificationSeverity(kind),
		CreatedAt:            req.Now,
	}
	if err := s.validateStatusEvent(event); err != nil {
		return StatusEvent{}, err
	}
	s.statusEvents[event.ID] = event
	return cloneStatusEvent(event), nil
}

func (s *State) MarkStatusEventRead(req MarkStatusEventRead) (StatusEvent, error) {
	event, ok := s.statusEvents[req.ID]
	if !ok {
		return StatusEvent{}, fmt.Errorf("status event %s not found", req.ID)
	}
	event.ReadAt = &req.Now
	s.statusEvents[event.ID] = event
	return cloneStatusEvent(event), nil
}

func (p Project) hasStage(stageID string) bool {
	_, ok := p.stage(stageID)
	return ok
}

func reviewStageID(project Project) string {
	for _, stage := range project.Workflow.Stages {
		if stage.Kind == StageKindReview {
			return stage.ID
		}
	}
	return ""
}

func doneStageID(project Project) string {
	for _, stage := range project.Workflow.Stages {
		if stage.Kind == StageKindDone {
			return stage.ID
		}
	}
	return ""
}

func defaultPresetForStage(project Project, stageID string) string {
	if stage, ok := project.stage(stageID); ok && stage.DefaultRunPreset != "" {
		return stage.DefaultRunPreset
	}
	return RunPresetReader
}

func defaultPromptForStage(project Project, stageID string) string {
	if stage, ok := project.stage(stageID); ok && stage.DefaultPromptTemplateID != "" {
		return stage.DefaultPromptTemplateID
	}
	return PromptTemplatePlan
}

func (p Project) stage(stageID string) (WorkflowStage, bool) {
	for _, stage := range p.Workflow.Stages {
		if stage.ID == stageID {
			return stage, true
		}
	}
	return WorkflowStage{}, false
}

func (s *State) validateWorkItem(item WorkItem) error {
	if item.ID == "" {
		return fmt.Errorf("work item id required")
	}
	project, ok := s.projects[item.ProjectID]
	if !ok {
		return fmt.Errorf("project %s not found", item.ProjectID)
	}
	if item.Number <= 0 {
		return fmt.Errorf("work item number must be positive")
	}
	if strings.TrimSpace(item.Title) == "" {
		return fmt.Errorf("work item title required")
	}
	if item.WorkflowID == "" {
		return fmt.Errorf("work item workflow id required")
	}
	if item.WorkflowVersion <= 0 {
		return fmt.Errorf("work item workflow version must be positive")
	}
	if _, ok := s.workflowDefinitions[workflowDefinitionKey{id: item.WorkflowID, version: item.WorkflowVersion}]; !ok {
		return fmt.Errorf("workflow definition %s@%d not found", item.WorkflowID, item.WorkflowVersion)
	}
	if !project.hasStage(item.StageID) {
		return fmt.Errorf("stage %s not found", item.StageID)
	}
	if item.RunState == "" {
		return fmt.Errorf("work item run state required")
	}
	for _, attachment := range item.Attachments {
		if _, err := validateAttachment(attachment); err != nil {
			return err
		}
	}
	return validateMetadataMap(item.Metadata)
}

func (s *State) validateWorkItemLink(link WorkItemLink) error {
	if link.ID == "" {
		return fmt.Errorf("work item link id required")
	}
	if link.SourceWorkItemID == "" {
		return fmt.Errorf("source work item id required")
	}
	if link.TargetWorkItemID == "" {
		return fmt.Errorf("target work item id required")
	}
	if link.SourceWorkItemID == link.TargetWorkItemID {
		return fmt.Errorf("cannot link work item to itself")
	}
	source, ok := s.items[link.SourceWorkItemID]
	if !ok {
		return fmt.Errorf("work item %s not found", link.SourceWorkItemID)
	}
	target, ok := s.items[link.TargetWorkItemID]
	if !ok {
		return fmt.Errorf("work item %s not found", link.TargetWorkItemID)
	}
	if source.ProjectID != target.ProjectID {
		return fmt.Errorf("work item links must stay within the same project")
	}
	if link.ProjectID != "" && link.ProjectID != source.ProjectID {
		return fmt.Errorf("work item link project mismatch")
	}
	switch link.Type {
	case WorkItemLinkBlocks, WorkItemLinkParentChild, WorkItemLinkRelated, WorkItemLinkDuplicates, WorkItemLinkSupersedes:
		return nil
	default:
		return fmt.Errorf("unsupported link type %s", link.Type)
	}
}

func (s *State) wouldCreateBlockingCycle(sourceID string, targetID string) bool {
	graph := map[string][]string{}
	for _, link := range s.links {
		if link.Type == WorkItemLinkBlocks {
			graph[link.SourceWorkItemID] = append(graph[link.SourceWorkItemID], link.TargetWorkItemID)
		}
	}
	graph[sourceID] = append(graph[sourceID], targetID)
	seen := map[string]bool{}
	var hasPath func(string) bool
	hasPath = func(id string) bool {
		if id == sourceID {
			return true
		}
		if seen[id] {
			return false
		}
		seen[id] = true
		for _, next := range graph[id] {
			if hasPath(next) {
				return true
			}
		}
		return false
	}
	return hasPath(targetID)
}

func validateWorkflowTemplate(template WorkflowTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("workflow template id required")
	}
	if strings.TrimSpace(template.Name) == "" {
		return fmt.Errorf("workflow template name required")
	}
	if len(template.Stages) == 0 {
		return fmt.Errorf("workflow template stages required")
	}
	seen := map[string]struct{}{}
	for _, stage := range template.Stages {
		if stage.ID == "" {
			return fmt.Errorf("workflow stage id required")
		}
		if strings.TrimSpace(stage.Name) == "" {
			return fmt.Errorf("workflow stage name required")
		}
		if stage.Kind == "" {
			return fmt.Errorf("workflow stage kind required")
		}
		if _, exists := seen[stage.ID]; exists {
			return fmt.Errorf("workflow stage %s already exists", stage.ID)
		}
		seen[stage.ID] = struct{}{}
	}
	return nil
}

func validatePromptTemplate(template PromptTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("prompt template id required")
	}
	if strings.TrimSpace(template.Name) == "" {
		return fmt.Errorf("prompt template name required")
	}
	if strings.TrimSpace(template.Body) == "" {
		return fmt.Errorf("prompt template body required")
	}
	return nil
}

func validateWorkflowDefinitionRecord(record WorkflowDefinitionRecord) error {
	if record.ID == "" {
		return fmt.Errorf("workflow definition id required")
	}
	if record.Version <= 0 {
		return fmt.Errorf("workflow definition version must be positive")
	}
	if record.ContentHash == "" {
		return fmt.Errorf("workflow definition content hash required")
	}
	if record.ID != record.Definition.ID || record.Version != record.Definition.Version {
		return fmt.Errorf("workflow definition record identity mismatch")
	}
	if err := ValidateWorkflowDefinition(record.Definition); err != nil {
		return err
	}
	hash, err := WorkflowDefinitionHash(record.Definition)
	if err != nil {
		return err
	}
	if hash != record.ContentHash {
		return fmt.Errorf("workflow definition content hash mismatch")
	}
	return nil
}

func validateProject(project Project) error {
	if project.ID == "" {
		return fmt.Errorf("project id required")
	}
	if strings.TrimSpace(project.Name) == "" {
		return fmt.Errorf("project name required")
	}
	if cleanSlug(project.Slug) == "" {
		return fmt.Errorf("project slug required")
	}
	if _, err := cleanAbsolutePath(project.RootDir, "root dir"); err != nil {
		return err
	}
	if project.NextWorkItemNumber <= 0 {
		return fmt.Errorf("next work item number must be positive")
	}
	if err := validateMetadataMap(project.Metadata); err != nil {
		return err
	}
	for _, attachment := range project.Attachments {
		if _, err := validateAttachment(attachment); err != nil {
			return err
		}
	}
	return validateWorkflowTemplate(WorkflowTemplate{
		ID:     project.Workflow.ID,
		Name:   project.Workflow.Name,
		Stages: project.Workflow.Stages,
	})
}

func stagesForWorkflowDefinition(definition WorkflowDefinition) []WorkflowStage {
	defaults := map[string]WorkflowStage{}
	for _, stage := range DefaultWorkflowTemplate(time.Time{}).Stages {
		defaults[stage.ID] = stage
	}
	stages := make([]WorkflowStage, 0, len(definition.Stages))
	for index, id := range definition.Stages {
		if stage, ok := defaults[id]; ok {
			stages = append(stages, stage)
			continue
		}
		kind := StageKindActive
		switch {
		case index == len(definition.Stages)-1:
			kind = StageKindDone
		case index == 0:
			kind = StageKindBacklog
		}
		stages = append(stages, WorkflowStage{
			ID:   id,
			Name: stageNameFromID(id),
			Kind: kind,
		})
	}
	return stages
}

func stageNameFromID(id string) string {
	parts := strings.FieldsFunc(id, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	name := strings.Join(parts, " ")
	if name == "" {
		return id
	}
	return name
}

func defaultProjectPreferences(preferences ProjectPreferences) ProjectPreferences {
	if preferences.AutoRun == "" {
		preferences.AutoRun = AutoRunNever
	}
	if preferences.DefaultPhaseAgents != nil {
		copied := map[string]string{}
		for key, value := range preferences.DefaultPhaseAgents {
			copied[key] = value
		}
		preferences.DefaultPhaseAgents = copied
	}
	if preferences.Gates != nil {
		preferences.Gates = append([]GateConfig(nil), preferences.Gates...)
	}
	return preferences
}

func (s *State) validateArtifact(artifact Artifact) error {
	if artifact.ID == "" {
		return fmt.Errorf("artifact id required")
	}
	if _, ok := s.items[artifact.WorkItemID]; !ok {
		return fmt.Errorf("work item %s not found", artifact.WorkItemID)
	}
	switch artifact.Kind {
	case ArtifactKindPlan, ArtifactKindFeedback, ArtifactKindGateReport:
	default:
		return fmt.Errorf("unsupported artifact kind %s", artifact.Kind)
	}
	switch artifact.Status {
	case ArtifactStatusDraft, ArtifactStatusApproved:
	default:
		return fmt.Errorf("unsupported artifact status %s", artifact.Status)
	}
	return validateMetadataMap(artifact.Metadata)
}

func (s *State) validateQuestion(question Question) error {
	if question.ID == "" {
		return fmt.Errorf("question id required")
	}
	if _, ok := s.items[question.WorkItemID]; !ok {
		return fmt.Errorf("work item %s not found", question.WorkItemID)
	}
	if strings.TrimSpace(question.Prompt) == "" {
		return fmt.Errorf("question prompt required")
	}
	switch question.Status {
	case QuestionStatusOpen, QuestionStatusAnswered:
	default:
		return fmt.Errorf("unsupported question status %s", question.Status)
	}
	return nil
}

func (s *State) validateGateReport(gate GateReport) error {
	if gate.ID == "" {
		return fmt.Errorf("gate report id required")
	}
	if _, ok := s.items[gate.WorkItemID]; !ok {
		return fmt.Errorf("work item %s not found", gate.WorkItemID)
	}
	if strings.TrimSpace(gate.Name) == "" {
		return fmt.Errorf("gate report name required")
	}
	switch gate.Status {
	case GateStatusPending, GateStatusPassed, GateStatusFailed, GateStatusOverridden:
	default:
		return fmt.Errorf("unsupported gate status %s", gate.Status)
	}
	return nil
}

func (s *State) hasRequiredArtifacts(workItemID string, requirements []WorkflowArtifactRequirement) bool {
	return len(s.missingArtifactRequirements(workItemID, requirements)) == 0
}

func (s *State) missingArtifactRequirements(workItemID string, requirements []WorkflowArtifactRequirement) []WorkflowArtifactRequirement {
	missing := []WorkflowArtifactRequirement{}
	for _, requirement := range requirements {
		found := false
		for _, artifact := range s.artifacts {
			if artifact.WorkItemID == workItemID && artifact.Kind == requirement.Kind && artifact.Status == requirement.Status {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, requirement)
		}
	}
	return missing
}

func (s *State) hasOpenQuestionForRun(runID string) bool {
	for _, question := range s.questions {
		if question.RunID == runID && question.Status == QuestionStatusOpen {
			return true
		}
	}
	return false
}

func (s *State) hasGateReport(workItemID string, name string) bool {
	for _, gate := range s.gateReports {
		if gate.WorkItemID == workItemID && gate.Name == name {
			return true
		}
	}
	return false
}

func (s *State) workflowDefinitionForItem(item WorkItem) (WorkflowDefinition, error) {
	record, ok := s.workflowDefinitions[workflowDefinitionKey{id: item.WorkflowID, version: item.WorkflowVersion}]
	if !ok {
		return WorkflowDefinition{}, fmt.Errorf("workflow definition %s@%d not found", item.WorkflowID, item.WorkflowVersion)
	}
	return record.Definition, nil
}

func (s *State) workflowActionForItem(item WorkItem, id string) (WorkflowActionDefinition, error) {
	definition, err := s.workflowDefinitionForItem(item)
	if err != nil {
		return WorkflowActionDefinition{}, err
	}
	action, ok := definition.Action(id)
	if !ok {
		return WorkflowActionDefinition{}, fmt.Errorf("workflow action %s not found", id)
	}
	return action, nil
}

func (s *State) workflowActionAvailabilityForTransition(item WorkItem, stageID string) (WorkflowActionAvailability, bool, error) {
	definition, err := s.workflowDefinitionForItem(item)
	if err != nil {
		return WorkflowActionAvailability{}, false, err
	}
	for _, action := range definition.Actions {
		if !workflowActionCanStartFrom(action, item.StageID) {
			continue
		}
		if workflowActionTargetStage(action, item) != stageID {
			continue
		}
		return s.workflowActionAvailability(item, action), true, nil
	}
	return WorkflowActionAvailability{}, false, nil
}

func (s *State) workflowActionAvailability(item WorkItem, action WorkflowActionDefinition) WorkflowActionAvailability {
	availability := WorkflowActionAvailability{
		Action:    cloneWorkflowActionDefinition(action),
		Enabled:   true,
		InputKind: workflowActionInputKind(action),
	}
	if missing := s.missingArtifactRequirements(item.ID, action.Requires); len(missing) > 0 {
		availability.Enabled = false
		availability.Reason = workflowArtifactRequirementReason(missing[0])
	}
	if availability.Enabled && action.RequiresPassingBlockingGates {
		for _, gate := range s.gateReports {
			if gate.WorkItemID == item.ID && gate.Blocking && gate.Status != GateStatusPassed && gate.Status != GateStatusOverridden {
				availability.Enabled = false
				availability.Reason = "blocking gates must pass or be overridden"
				break
			}
		}
	}
	return availability
}

func workflowActionTargetStage(action WorkflowActionDefinition, item WorkItem) string {
	if action.To == "$previousStage" {
		if item.PreviousStageID != "" {
			return item.PreviousStageID
		}
		return StageExecution
	}
	return action.To
}

func workflowActionCanStartFrom(action WorkflowActionDefinition, stageID string) bool {
	for _, stage := range action.From {
		if stage == stageID {
			return true
		}
	}
	return false
}

func workflowActionInputKind(action WorkflowActionDefinition) string {
	switch {
	case action.CreatesRun != nil || action.CompletesRun:
		return WorkflowActionInputRun
	case action.CreatesArtifact != nil:
		return WorkflowActionInputArtifact
	case action.UpdatesArtifact != nil:
		return WorkflowActionInputArtifactSelection
	case action.RequiresPassingBlockingGates:
		return WorkflowActionInputGate
	default:
		return WorkflowActionInputNone
	}
}

func workflowArtifactRequirementReason(requirement WorkflowArtifactRequirement) string {
	if requirement.Kind == ArtifactKindPlan && requirement.Status == ArtifactStatusDraft {
		return "plan draft required"
	}
	if requirement.Kind == ArtifactKindPlan && requirement.Status == ArtifactStatusApproved {
		return "approved plan required"
	}
	return fmt.Sprintf("%s %s artifact required", requirement.Status, requirement.Kind)
}

func (s *State) backfillProjectWorkflowDefinition(project Project) Project {
	if project.Workflow.DefinitionID != "" && project.Workflow.DefinitionVersion > 0 && project.Workflow.DefinitionHash != "" {
		return project
	}
	record, ok := s.workflowDefinitions[workflowDefinitionKey{id: WorkflowPlanExecuteReview, version: 1}]
	if !ok {
		record = DefaultWorkflowDefinitionRecord(time.Time{})
	}
	project.Workflow.DefinitionID = record.ID
	project.Workflow.DefinitionVersion = record.Version
	project.Workflow.DefinitionHash = record.ContentHash
	if len(project.Workflow.Stages) == 0 {
		project.Workflow.Stages = stagesForWorkflowDefinition(record.Definition)
	}
	return project
}

func (s *State) recordWorkflowEvent(kind, projectID, workItemID, runID, actor, message string, at time.Time) {
	id := fmt.Sprintf("workflow_event_%d", len(s.workflowEvents)+1)
	s.workflowEvents[id] = WorkflowEvent{
		ID:         id,
		ProjectID:  projectID,
		WorkItemID: workItemID,
		RunID:      runID,
		Type:       kind,
		Actor:      actor,
		Message:    message,
		At:         at,
	}
}

func mustDefaultWorkflowAction(id string) WorkflowActionDefinition {
	action, ok := DefaultWorkflowDefinition().Action(id)
	if !ok {
		panic(fmt.Sprintf("default workflow action %s not found", id))
	}
	return action
}

func metadataFullKey(namespace, key string) (string, error) {
	namespace = strings.TrimSpace(namespace)
	key = strings.TrimSpace(key)
	if !validMetadataToken(namespace) {
		return "", fmt.Errorf("invalid metadata namespace")
	}
	if !validMetadataToken(key) {
		return "", fmt.Errorf("invalid metadata key")
	}
	return namespace + "/" + key, nil
}

func validMetadataToken(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

func validateMetadataMap(metadata map[string]MetadataValue) error {
	for key, value := range metadata {
		parts := strings.Split(key, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid metadata key %s", key)
		}
		if _, err := metadataFullKey(parts[0], parts[1]); err != nil {
			return err
		}
		if err := validateMetadataValue(value); err != nil {
			return err
		}
	}
	return nil
}

func validateMetadataValue(value MetadataValue) error {
	switch value.Type {
	case MetadataTypeString, MetadataTypeNumber, MetadataTypeBool:
		return nil
	case MetadataTypeJSON:
		if len(value.JSON) == 0 || !json.Valid(value.JSON) {
			return fmt.Errorf("metadata json value must be valid json")
		}
		return nil
	default:
		return fmt.Errorf("unsupported metadata type %s", value.Type)
	}
}

func validateAttachment(attachment Attachment) (Attachment, error) {
	if attachment.ID == "" {
		return Attachment{}, fmt.Errorf("attachment id required")
	}
	if attachment.Kind == "" {
		return Attachment{}, fmt.Errorf("attachment kind required")
	}
	if attachment.Scope == "" {
		attachment.Scope = AttachmentScopeProject
	}
	switch attachment.Kind {
	case AttachmentKindFile:
		if attachment.Path == "" {
			return Attachment{}, fmt.Errorf("attachment path required")
		}
		if filepath.IsAbs(attachment.Path) && attachment.Scope != AttachmentScopeExternal {
			return Attachment{}, fmt.Errorf("absolute attachment path requires external scope")
		}
		attachment.Title = strings.TrimSpace(attachment.Title)
		attachment.Path = filepath.Clean(attachment.Path)
	case AttachmentKindURL:
		if strings.TrimSpace(attachment.URL) == "" {
			return Attachment{}, fmt.Errorf("attachment url required")
		}
		attachment.Title = strings.TrimSpace(attachment.Title)
		attachment.URL = strings.TrimSpace(attachment.URL)
	case AttachmentKindNote:
		if strings.TrimSpace(attachment.Note) == "" {
			return Attachment{}, fmt.Errorf("attachment note required")
		}
		attachment.Title = strings.TrimSpace(attachment.Title)
		attachment.Note = strings.TrimSpace(attachment.Note)
	case AttachmentKindExternal:
		attachment.Provider = strings.TrimSpace(attachment.Provider)
		attachment.Target = strings.TrimSpace(attachment.Target)
		attachment.Title = strings.TrimSpace(attachment.Title)
		attachment.URL = strings.TrimSpace(attachment.URL)
		if attachment.Provider == "" {
			return Attachment{}, fmt.Errorf("attachment provider required")
		}
		if attachment.Target == "" {
			return Attachment{}, fmt.Errorf("attachment target required")
		}
	default:
		return Attachment{}, fmt.Errorf("unsupported attachment kind %s", attachment.Kind)
	}
	if err := validateMetadataMap(attachment.Meta); err != nil {
		return Attachment{}, err
	}
	return attachment, nil
}

func appendHistory(item *WorkItem, event HistoryEvent) error {
	if event.ID == "" {
		return fmt.Errorf("history id required")
	}
	if event.Type == "" {
		return fmt.Errorf("history event type required")
	}
	item.History = append(item.History, event)
	return nil
}

func appendRunEvent(run *WorkItemRun, event RunEvent) error {
	if event.ID == "" {
		return fmt.Errorf("run history id required")
	}
	if event.Type == "" {
		return fmt.Errorf("run event type required")
	}
	run.History = append(run.History, event)
	return nil
}

func (s *State) validateRun(run WorkItemRun) error {
	if run.ID == "" {
		return fmt.Errorf("work item run id required")
	}
	item, ok := s.items[run.WorkItemID]
	if !ok {
		return fmt.Errorf("work item %s not found", run.WorkItemID)
	}
	if run.ProjectID != item.ProjectID {
		return fmt.Errorf("work item run project mismatch")
	}
	if !validPreset(run.Preset) {
		return fmt.Errorf("unsupported run preset %s", run.Preset)
	}
	if _, ok := s.promptTemplates[run.PromptTemplateID]; !ok {
		return fmt.Errorf("prompt template %s not found", run.PromptTemplateID)
	}
	if strings.TrimSpace(run.PromptSnapshot) == "" {
		return fmt.Errorf("prompt snapshot required")
	}
	if run.Status == "" {
		return fmt.Errorf("run status required")
	}
	return validateMetadataMap(run.Metadata)
}

func (s *State) validateStatusEvent(event StatusEvent) error {
	if event.ID == "" {
		return fmt.Errorf("status event id required")
	}
	if !validStatusKind(event.Kind) {
		return fmt.Errorf("unsupported status kind %s", event.Kind)
	}
	if strings.TrimSpace(event.Message) == "" {
		return fmt.Errorf("status message required")
	}
	if event.NotificationSeverity != "" && !validStatusNotificationSeverity(event.NotificationSeverity) {
		return fmt.Errorf("unsupported status notification severity %s", event.NotificationSeverity)
	}
	switch event.Scope {
	case StatusScopeRun:
		if event.RunID == "" {
			return fmt.Errorf("run status event requires run id")
		}
		if _, ok := s.runs[event.RunID]; !ok {
			return fmt.Errorf("work item run %s not found", event.RunID)
		}
	case StatusScopePTY:
		if event.PTYID == "" {
			return fmt.Errorf("pty status event requires pty id")
		}
	case StatusScopeSession:
		if event.SessionID == "" {
			return fmt.Errorf("session status event requires session id")
		}
	default:
		return fmt.Errorf("unsupported status scope %s", event.Scope)
	}
	return nil
}

func validStatusKind(kind string) bool {
	switch kind {
	case StatusKindQuestion, StatusKindDone, StatusKindBlocked:
		return true
	default:
		return false
	}
}

func statusRequiresAttention(kind string) bool {
	return kind == StatusKindQuestion || kind == StatusKindBlocked
}

func statusNotificationSeverity(kind string) string {
	switch kind {
	case StatusKindQuestion:
		return StatusNotificationSeverityAttention
	case StatusKindBlocked:
		return StatusNotificationSeverityWarning
	case StatusKindDone:
		return StatusNotificationSeverityInfo
	default:
		return ""
	}
}

func validStatusNotificationSeverity(severity string) bool {
	switch severity {
	case StatusNotificationSeverityAttention, StatusNotificationSeverityWarning, StatusNotificationSeverityInfo:
		return true
	default:
		return false
	}
}

func statusNotificationKey(sessionID string, paneID string, ptyID string, actor string, kind string) string {
	sessionPart := strings.TrimSpace(sessionID)
	if sessionPart == "" {
		sessionPart = "none"
	}
	targetPart := "session"
	if pane := strings.TrimSpace(paneID); pane != "" {
		targetPart = "pane:" + pane
	} else if pty := strings.TrimSpace(ptyID); pty != "" {
		targetPart = "pty:" + pty
	}
	actorPart := strings.TrimSpace(actor)
	if actorPart == "" {
		actorPart = "unknown"
	}
	kindPart := strings.TrimSpace(kind)
	if kindPart == "" {
		kindPart = "status"
	}
	return strings.Join([]string{"status", "session:" + sessionPart, targetPart, "actor:" + actorPart, "kind:" + kindPart}, "|")
}

func validPreset(preset string) bool {
	switch preset {
	case RunPresetReader, RunPresetWriter, RunPresetReviewer, RunPresetManager:
		return true
	default:
		return false
	}
}

func renderPrompt(template string, project Project, item WorkItem) string {
	replacements := map[string]string{
		"{{project.name}}":           project.Name,
		"{{project.rootDir}}":        project.RootDir,
		"{{work_item.id}}":           item.ID,
		"{{work_item.number}}":       fmt.Sprintf("%d", item.Number),
		"{{work_item.title}}":        item.Title,
		"{{work_item.bodyMarkdown}}": item.BodyMarkdown,
		"{{work_item.stageId}}":      item.StageID,
		"{{attachments}}":            renderAttachments(item.Attachments),
		"{{worktree.branch}}":        "",
		"{{worktree.path}}":          "",
	}
	if item.Worktree != nil {
		replacements["{{worktree.branch}}"] = item.Worktree.Branch
		replacements["{{worktree.path}}"] = item.Worktree.WorktreePath
	}
	out := template
	for old, replacement := range replacements {
		out = strings.ReplaceAll(out, old, replacement)
	}
	return strings.TrimSpace(out)
}

func renderAttachments(attachments []Attachment) string {
	if len(attachments) == 0 {
		return "- none"
	}
	lines := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		value := attachment.Path
		if value == "" {
			value = attachment.URL
		}
		if value == "" {
			value = attachment.Note
		}
		lines = append(lines, "- "+value)
	}
	return strings.Join(lines, "\n")
}

func cleanAbsolutePath(path string, label string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s required", label)
	}
	cleaned := filepath.Clean(path)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("%s must be absolute", label)
	}
	return cleaned, nil
}

func cleanSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func cloneProject(project Project) Project {
	project.Workflow.Stages = cloneStages(project.Workflow.Stages)
	project.Workflow.TransitionRules = cloneTransitionRules(project.Workflow.TransitionRules)
	project.Preferences = defaultProjectPreferences(project.Preferences)
	project.Attachments = append([]Attachment(nil), project.Attachments...)
	project.Metadata = cloneMetadata(project.Metadata)
	return project
}

func cloneTemplate(template WorkflowTemplate) WorkflowTemplate {
	template.Stages = cloneStages(template.Stages)
	template.TransitionRules = cloneTransitionRules(template.TransitionRules)
	return template
}

func cloneWorkflowDefinitionRecord(record WorkflowDefinitionRecord) WorkflowDefinitionRecord {
	record.Definition = cloneWorkflowDefinition(record.Definition)
	return record
}

func cloneWorkflowDefinition(definition WorkflowDefinition) WorkflowDefinition {
	definition.Stages = append([]string(nil), definition.Stages...)
	definition.Actions = cloneWorkflowActionDefinitions(definition.Actions)
	definition.Gates = append([]WorkflowGateDefinition(nil), definition.Gates...)
	return definition
}

func cloneWorkflowActionDefinitions(actions []WorkflowActionDefinition) []WorkflowActionDefinition {
	out := make([]WorkflowActionDefinition, 0, len(actions))
	for _, action := range actions {
		out = append(out, cloneWorkflowActionDefinition(action))
	}
	return out
}

func cloneWorkflowActionDefinition(action WorkflowActionDefinition) WorkflowActionDefinition {
	action.From = append([]string(nil), action.From...)
	action.Requires = append([]WorkflowArtifactRequirement(nil), action.Requires...)
	if action.CreatesArtifact != nil {
		effect := *action.CreatesArtifact
		action.CreatesArtifact = &effect
	}
	if action.UpdatesArtifact != nil {
		effect := *action.UpdatesArtifact
		action.UpdatesArtifact = &effect
	}
	if action.CreatesRun != nil {
		effect := *action.CreatesRun
		action.CreatesRun = &effect
	}
	action.CreatesGates = append([]string(nil), action.CreatesGates...)
	return action
}

func cloneWorkItem(item WorkItem) WorkItem {
	if item.Worktree != nil {
		worktree := *item.Worktree
		item.Worktree = &worktree
	}
	item.Attachments = append([]Attachment(nil), item.Attachments...)
	item.Metadata = cloneMetadata(item.Metadata)
	item.History = append([]HistoryEvent(nil), item.History...)
	return item
}

func cloneWorkItemLink(link WorkItemLink) WorkItemLink {
	return link
}

func cloneRun(run WorkItemRun) WorkItemRun {
	if run.CompletedAt != nil {
		completedAt := *run.CompletedAt
		run.CompletedAt = &completedAt
	}
	run.Metadata = cloneMetadata(run.Metadata)
	run.History = append([]RunEvent(nil), run.History...)
	return run
}

func cloneArtifact(artifact Artifact) Artifact {
	artifact.Metadata = cloneMetadata(artifact.Metadata)
	return artifact
}

func cloneQuestion(question Question) Question {
	if question.AnsweredAt != nil {
		answeredAt := *question.AnsweredAt
		question.AnsweredAt = &answeredAt
	}
	return question
}

func cloneGateReport(gate GateReport) GateReport {
	return gate
}

func cloneStatusEvent(event StatusEvent) StatusEvent {
	if event.ReadAt != nil {
		readAt := *event.ReadAt
		event.ReadAt = &readAt
	}
	return event
}

func cloneStages(stages []WorkflowStage) []WorkflowStage {
	return append([]WorkflowStage(nil), stages...)
}

func cloneTransitionRules(rules []TransitionRule) []TransitionRule {
	return append([]TransitionRule(nil), rules...)
}

func cloneMetadata(metadata map[string]MetadataValue) map[string]MetadataValue {
	if metadata == nil {
		return nil
	}
	out := map[string]MetadataValue{}
	for key, value := range metadata {
		if value.JSON != nil {
			value.JSON = append(json.RawMessage(nil), value.JSON...)
		}
		out[key] = value
	}
	return out
}
