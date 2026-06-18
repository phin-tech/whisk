package protocol

import (
	"time"

	"github.com/phin-tech/whisk/internal/domain/onboarding"
	"github.com/phin-tech/whisk/internal/domain/ptybookmark"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/domain/workitem"
)

const DaemonAPIVersion = 18

type CompatibilityResponse struct {
	APIVersion int    `json:"apiVersion"`
	GitSHA     string `json:"gitSha"`
}

type OnboardingItem = onboarding.Item

type OnboardingStatus struct {
	Items       []OnboardingItem `json:"items"`
	ShouldShow  bool             `json:"shouldShow"`
	LocalDaemon bool             `json:"localDaemon"`
	StatePath   string           `json:"statePath"`
}

type OnboardingApplyRequest struct {
	ItemIDs []string `json:"itemIds"`
}

type ClearDaemonRequest struct{}

type ClearDaemonResponse struct {
	SessionsCleared  int `json:"sessionsCleared"`
	PTYsCleared      int `json:"ptysCleared"`
	BookmarksCleared int `json:"bookmarksCleared"`
	ProjectsCleared  int `json:"projectsCleared"`
	WorkItemsCleared int `json:"workItemsCleared"`
	ForwardsCleared  int `json:"forwardsCleared"`
}

type CreateSessionRequest struct {
	Name       string           `json:"name"`
	RootDir    string           `json:"rootDir"`
	WorkingDir string           `json:"workingDir,omitempty"`
	ProjectID  string           `json:"projectId,omitempty"`
	InitialPTY *StartPTYOptions `json:"initialPty,omitempty"`
}

type CreatedSession struct {
	Session   session.Session `json:"session"`
	WindowID  string          `json:"windowId"`
	PaneID    string          `json:"paneId"`
	PTYID     *string         `json:"ptyId,omitempty"`
	MainPtyID string          `json:"mainPtyId,omitempty"`
}

type SplitPaneRequest struct {
	SessionID    string           `json:"sessionId"`
	WindowID     string           `json:"windowId"`
	TargetPaneID string           `json:"targetPaneId"`
	Direction    string           `json:"direction"`
	InitialPTY   *StartPTYOptions `json:"initialPty,omitempty"`
}

type StartPTYOptions struct {
	Cols        int                         `json:"cols"`
	Rows        int                         `json:"rows"`
	Command     string                      `json:"command,omitempty"`
	Env         map[string]string           `json:"env,omitempty"`
	Args        []string                    `json:"args,omitempty"`
	Exec        bool                        `json:"exec,omitempty"`
	AgentBridge *StartPTYAgentBridgeOptions `json:"agentBridge,omitempty"`
}

type StartPTYAgentBridgeOptions struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider,omitempty"`
}

type SplitPaneResult struct {
	Session session.Session `json:"session"`
	PaneID  string          `json:"paneId"`
	PTYID   *string         `json:"ptyId,omitempty"`
	PtyID   string          `json:"legacyPtyId,omitempty"`
}

type SetSessionRootDirRequest struct {
	SessionID string `json:"sessionId"`
	RootDir   string `json:"rootDir"`
}

type SetSessionProjectRequest struct {
	SessionID string `json:"sessionId"`
	ProjectID string `json:"projectId,omitempty"`
}

type SetPaneWorkingDirRequest struct {
	SessionID  string `json:"sessionId"`
	PaneID     string `json:"paneId"`
	WorkingDir string `json:"workingDir"`
}

type StartPanePTYRequest struct {
	SessionID string          `json:"sessionId"`
	PaneID    string          `json:"paneId"`
	Options   StartPTYOptions `json:"options"`
}

type StartedPanePTY struct {
	Session session.Session `json:"session"`
	PTYID   string          `json:"ptyId"`
}

type RestartPanePTYRequest struct {
	SessionID string          `json:"sessionId"`
	PaneID    string          `json:"paneId"`
	Options   StartPTYOptions `json:"options"`
}

type RestartedPanePTY struct {
	Session  session.Session `json:"session"`
	PTYID    string          `json:"ptyId"`
	OldPTYID string          `json:"oldPtyId"`
}

type ClosePaneRequest struct {
	SessionID string `json:"sessionId"`
	WindowID  string `json:"windowId"`
	PaneID    string `json:"paneId"`
}

type CloseSessionRequest struct {
	SessionID string `json:"sessionId"`
}

type DetachPanePTYRequest struct {
	SessionID string `json:"sessionId"`
	PaneID    string `json:"paneId"`
}

type DetachedPanePTY struct {
	Session session.Session `json:"session"`
	PTYID   string          `json:"ptyId"`
}

type KillPTYRequest struct {
	PTYID string `json:"ptyId"`
}

type AddPTYBookmarkRequest struct {
	PTYID  string `json:"ptyId"`
	Offset uint64 `json:"offset"`
	Kind   string `json:"kind"`
	Label  string `json:"label"`
}

type RemovePTYBookmarkRequest struct {
	BookmarkID string `json:"bookmarkId"`
}

type PTYBookmark = ptybookmark.Bookmark

type WritePTYRequest struct {
	PtyID string `json:"ptyId"`
	Data  string `json:"data"`
}

type ResizePTYRequest struct {
	PtyID string `json:"ptyId"`
	Cols  int    `json:"cols"`
	Rows  int    `json:"rows"`
}

type OutputRequest struct {
	PtyID      string `json:"ptyId"`
	FromOffset uint64 `json:"fromOffset"`
}

type OutputSnapshot struct {
	PtyID        string `json:"ptyId"`
	Offset       uint64 `json:"offset"`
	Output       string `json:"output"`
	OutputBase64 string `json:"outputBase64"`
}

type PTYStreamFrame struct {
	Type         string `json:"type"`
	PtyID        string `json:"ptyId"`
	Offset       uint64 `json:"offset,omitempty"`
	OutputBase64 string `json:"outputBase64,omitempty"`
	Code         *int   `json:"code,omitempty"`
	Message      string `json:"message,omitempty"`
}

type PTYInfo struct {
	ID             string `json:"id"`
	WorkingDir     string `json:"workingDir"`
	Cols           int    `json:"cols"`
	Rows           int    `json:"rows"`
	Running        bool   `json:"running"`
	Status         string `json:"status"`
	ExitCode       *int   `json:"exitCode,omitempty"`
	SessionID      string `json:"sessionId"`
	WindowID       string `json:"windowId"`
	PaneID         string `json:"paneId"`
	OriginWindowID string `json:"originWindowId"`
	OriginPaneID   string `json:"originPaneId"`
}

type NextEventRequest struct {
	TimeoutMs int `json:"timeoutMs"`
}

const RuntimeEventNone = "none"

type RuntimeEvent struct {
	Type   string `json:"type"`
	PtyID  string `json:"ptyId,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
}

type AgentBridgeHookDecision struct {
	Action string `json:"action,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type AgentBridgeHookRequest struct {
	Token            string                  `json:"token"`
	Provider         string                  `json:"provider"`
	EventName        string                  `json:"eventName"`
	ToolName         string                  `json:"toolName,omitempty"`
	ToolInput        map[string]any          `json:"toolInput,omitempty"`
	ToolOutput       string                  `json:"toolOutput,omitempty"`
	Message          string                  `json:"message,omitempty"`
	NotificationType string                  `json:"notificationType,omitempty"`
	ElicitationID    string                  `json:"elicitationId,omitempty"`
	Action           string                  `json:"action,omitempty"`
	SessionID        string                  `json:"sessionId,omitempty"`
	PTYID            string                  `json:"ptyId,omitempty"`
	RawPayload       map[string]any          `json:"rawPayload,omitempty"`
	Decision         AgentBridgeHookDecision `json:"decision,omitempty"`
}

type AgentBridgeHookResponse struct {
	Output map[string]any `json:"output,omitempty"`
}

type AgentBridgeApproval struct {
	ID         string                  `json:"id"`
	BridgeID   string                  `json:"bridgeId"`
	SessionID  string                  `json:"sessionId,omitempty"`
	PTYID      string                  `json:"ptyId,omitempty"`
	RunID      string                  `json:"runId,omitempty"`
	Provider   string                  `json:"provider"`
	EventName  string                  `json:"eventName"`
	ToolName   string                  `json:"toolName"`
	ToolInput  map[string]any          `json:"toolInput,omitempty"`
	Status     string                  `json:"status"`
	Decision   AgentBridgeHookDecision `json:"decision,omitempty"`
	CreatedAt  time.Time               `json:"createdAt"`
	ResolvedAt *time.Time              `json:"resolvedAt,omitempty"`
}

type AgentBridgeEvent struct {
	ID               string         `json:"id"`
	BridgeID         string         `json:"bridgeId,omitempty"`
	SessionID        string         `json:"sessionId,omitempty"`
	PTYID            string         `json:"ptyId,omitempty"`
	Provider         string         `json:"provider"`
	EventName        string         `json:"eventName"`
	ToolName         string         `json:"toolName,omitempty"`
	Message          string         `json:"message,omitempty"`
	NotificationType string         `json:"notificationType,omitempty"`
	ElicitationID    string         `json:"elicitationId,omitempty"`
	Action           string         `json:"action,omitempty"`
	Result           string         `json:"result,omitempty"`
	Status           string         `json:"status"`
	CreatedAt        time.Time      `json:"createdAt"`
	Raw              map[string]any `json:"raw,omitempty"`
}

type ListAgentBridgeEventsRequest struct {
	Status string `json:"status,omitempty"`
}

type MarkAgentBridgeEventReadRequest struct {
	ID string `json:"id"`
}

type ListAgentBridgeApprovalsRequest struct {
	Status string `json:"status,omitempty"`
}

type ResolveAgentBridgeApprovalRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
}

type AgentHookIntegration struct {
	Provider         string `json:"provider"`
	Status           string `json:"status"`
	InstalledVersion string `json:"installedVersion,omitempty"`
	LatestVersion    string `json:"latestVersion"`
	HelperPath       string `json:"helperPath"`
	ConfigPath       string `json:"configPath"`
	ManifestPath     string `json:"manifestPath"`
	Detail           string `json:"detail,omitempty"`
}

type AgentHookIntegrationRequest struct {
	Provider string `json:"provider"`
}

type PluginStatus struct {
	ID                         string                      `json:"id"`
	Name                       string                      `json:"name"`
	Version                    string                      `json:"version"`
	Dir                        string                      `json:"dir"`
	ManifestPath               string                      `json:"manifestPath"`
	Trusted                    bool                        `json:"trusted"`
	Valid                      bool                        `json:"valid"`
	Error                      string                      `json:"error,omitempty"`
	Resolvers                  []PluginResolver            `json:"resolvers,omitempty"`
	ProjectAttachmentTemplates []ProjectAttachmentTemplate `json:"projectAttachmentTemplates,omitempty"`
}

type PluginResolver struct {
	Provider string   `json:"provider"`
	Kinds    []string `json:"kinds,omitempty"`
}

type ProjectAttachmentTemplate struct {
	ID       string                `json:"id"`
	Label    string                `json:"label"`
	Provider string                `json:"provider"`
	Kind     string                `json:"kind"`
	Fields   []PluginTemplateField `json:"fields,omitempty"`
}

type PluginTemplateField struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Placeholder string   `json:"placeholder,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Options     []string `json:"options,omitempty"`
}

type RunPluginProjectAttachmentTemplateRequest struct {
	ProjectID string            `json:"projectId"`
	Values    map[string]string `json:"values,omitempty"`
}

type AgentHookLogStatus struct {
	Enabled           bool   `json:"enabled"`
	ClearAfterSession bool   `json:"clearAfterSession"`
	Path              string `json:"path"`
	SizeBytes         int64  `json:"sizeBytes"`
}

type SetAgentHookLogSettingsRequest struct {
	Enabled           *bool `json:"enabled,omitempty"`
	ClearAfterSession *bool `json:"clearAfterSession,omitempty"`
}

type Project = workitem.Project
type WorkflowTemplate = workitem.WorkflowTemplate
type PromptTemplate = workitem.PromptTemplate
type WorkItem = workitem.WorkItem
type WorkItemRun = workitem.WorkItemRun
type StatusEvent = workitem.StatusEvent
type WorktreeBinding = workitem.WorktreeBinding
type Attachment = workitem.Attachment
type ProjectPreferences = workitem.ProjectPreferences
type MetadataValue = workitem.MetadataValue
type Artifact = workitem.Artifact
type Question = workitem.Question
type GateReport = workitem.GateReport
type WorkflowEvent = workitem.WorkflowEvent

type ProjectDetail struct {
	Project   Project           `json:"project"`
	WorkItems []WorkItem        `json:"workItems"`
	Sessions  []session.Session `json:"sessions"`
	Runs      []WorkItemRun     `json:"runs"`
}

type ProjectContext struct {
	ProjectID string               `json:"projectId"`
	Items     []ProjectContextItem `json:"items"`
}

type ProjectContextItem struct {
	AttachmentID string `json:"attachmentId"`
	Kind         string `json:"kind"`
	Provider     string `json:"provider,omitempty"`
	Target       string `json:"target,omitempty"`
	Title        string `json:"title,omitempty"`
	Delivery     string `json:"delivery"`
	ContentType  string `json:"contentType,omitempty"`
	Content      string `json:"content,omitempty"`
	SourceURL    string `json:"sourceUrl,omitempty"`
	Error        string `json:"error,omitempty"`
}

type CreateProjectRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Slug        string             `json:"slug,omitempty"`
	RootDir     string             `json:"rootDir"`
	WorkflowID  string             `json:"workflowId,omitempty"`
	Preferences ProjectPreferences `json:"preferences,omitempty"`
}

type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Slug        *string `json:"slug,omitempty"`
}

type AddProjectAttachmentRequest struct {
	ProjectID        string                   `json:"projectId"`
	Kind             string                   `json:"kind"`
	Scope            string                   `json:"scope,omitempty"`
	Title            string                   `json:"title,omitempty"`
	Path             string                   `json:"path,omitempty"`
	URL              string                   `json:"url,omitempty"`
	Note             string                   `json:"note,omitempty"`
	Provider         string                   `json:"provider,omitempty"`
	Target           string                   `json:"target,omitempty"`
	IncludeInContext bool                     `json:"includeInContext,omitempty"`
	Meta             map[string]MetadataValue `json:"meta,omitempty"`
}

type UpdateProjectAttachmentRequest struct {
	ProjectID        string                   `json:"projectId"`
	Title            *string                  `json:"title,omitempty"`
	Path             *string                  `json:"path,omitempty"`
	URL              *string                  `json:"url,omitempty"`
	Note             *string                  `json:"note,omitempty"`
	Provider         *string                  `json:"provider,omitempty"`
	Target           *string                  `json:"target,omitempty"`
	IncludeInContext *bool                    `json:"includeInContext,omitempty"`
	Meta             map[string]MetadataValue `json:"meta,omitempty"`
}

type DeleteProjectAttachmentRequest struct {
	ProjectID string `json:"projectId"`
}

type CreateWorkItemRequest struct {
	ProjectID    string `json:"projectId"`
	WorkflowID   string `json:"workflowId,omitempty"`
	Title        string `json:"title"`
	BodyMarkdown string `json:"bodyMarkdown,omitempty"`
	StageID      string `json:"stageId,omitempty"`
	Actor        string `json:"actor,omitempty"`
}

type MoveWorkItemRequest struct {
	ID      string `json:"id"`
	StageID string `json:"stageId"`
	Actor   string `json:"actor,omitempty"`
}

type BindWorkItemWorktreeRequest struct {
	ID           string `json:"id"`
	Branch       string `json:"branch"`
	Base         string `json:"base,omitempty"`
	WorktreePath string `json:"worktreePath"`
	Actor        string `json:"actor,omitempty"`
}

type AddWorkItemAttachmentRequest struct {
	WorkItemID string `json:"workItemId"`
	Kind       string `json:"kind"`
	Scope      string `json:"scope,omitempty"`
	Path       string `json:"path,omitempty"`
	URL        string `json:"url,omitempty"`
	Note       string `json:"note,omitempty"`
	Actor      string `json:"actor,omitempty"`
}

type DeleteWorkItemRequest struct {
	ID    string `json:"id"`
	Actor string `json:"actor,omitempty"`
}

type StartWorkItemRunRequest struct {
	WorkItemID       string `json:"workItemId"`
	Preset           string `json:"preset,omitempty"`
	PromptTemplateID string `json:"promptTemplateId,omitempty"`
	SessionID        string `json:"sessionId,omitempty"`
	PTYID            string `json:"ptyId,omitempty"`
	Launch           bool   `json:"launch,omitempty"`
	AgentProfileID   string `json:"agentProfileId,omitempty"`
	SystemPrompt     string `json:"systemPrompt,omitempty"`
	Actor            string `json:"actor,omitempty"`
}

type LaunchWorkItemRunRequest struct {
	ID             string `json:"id"`
	AgentProfileID string `json:"agentProfileId,omitempty"`
	SystemPrompt   string `json:"systemPrompt,omitempty"`
	Actor          string `json:"actor,omitempty"`
}

type QueueExecutionRequest struct {
	WorkItemID string `json:"workItemId"`
	Actor      string `json:"actor,omitempty"`
}

type LaunchExecutionRequest struct {
	WorkItemID     string `json:"workItemId"`
	AgentProfileID string `json:"agentProfileId,omitempty"`
	SystemPrompt   string `json:"systemPrompt,omitempty"`
	Actor          string `json:"actor,omitempty"`
}

type CancelWorkItemRunRequest struct {
	ID    string `json:"id"`
	Actor string `json:"actor,omitempty"`
}

type StartPlanningRequest struct {
	WorkItemID     string `json:"workItemId"`
	SessionID      string `json:"sessionId,omitempty"`
	PTYID          string `json:"ptyId,omitempty"`
	Launch         bool   `json:"launch,omitempty"`
	AgentProfileID string `json:"agentProfileId,omitempty"`
	SystemPrompt   string `json:"systemPrompt,omitempty"`
	Actor          string `json:"actor,omitempty"`
}

type SubmitDraftPlanRequest struct {
	WorkItemID string `json:"workItemId"`
	RunID      string `json:"runId,omitempty"`
	Title      string `json:"title,omitempty"`
	Body       string `json:"body"`
	Actor      string `json:"actor,omitempty"`
}

type ApprovePlanRequest struct {
	ArtifactID string `json:"artifactId"`
	WorkItemID string `json:"workItemId"`
	Actor      string `json:"actor,omitempty"`
}

type StartExecutionRequest struct {
	WorkItemID     string `json:"workItemId"`
	SessionID      string `json:"sessionId,omitempty"`
	PTYID          string `json:"ptyId,omitempty"`
	Launch         bool   `json:"launch,omitempty"`
	AgentProfileID string `json:"agentProfileId,omitempty"`
	SystemPrompt   string `json:"systemPrompt,omitempty"`
	Actor          string `json:"actor,omitempty"`
}

type AskQuestionRequest struct {
	WorkItemID string `json:"workItemId,omitempty"`
	RunID      string `json:"runId,omitempty"`
	SessionID  string `json:"sessionId,omitempty"`
	PTYID      string `json:"ptyId,omitempty"`
	Prompt     string `json:"prompt"`
	Actor      string `json:"actor,omitempty"`
}

type AnswerQuestionRequest struct {
	ID     string `json:"id"`
	Answer string `json:"answer"`
	Actor  string `json:"actor,omitempty"`
}

type CompleteExecutionRequest struct {
	WorkItemID string `json:"workItemId,omitempty"`
	RunID      string `json:"runId"`
	Message    string `json:"message,omitempty"`
	Actor      string `json:"actor,omitempty"`
}

type SubmitReviewFeedbackRequest struct {
	WorkItemID string `json:"workItemId"`
	RunID      string `json:"runId,omitempty"`
	Body       string `json:"body"`
	Actor      string `json:"actor,omitempty"`
}

type ApproveDoneRequest struct {
	WorkItemID string `json:"workItemId"`
	Reason     string `json:"reason,omitempty"`
	Actor      string `json:"actor,omitempty"`
}

type CompleteGateRequest struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	OverrideReason string `json:"overrideReason,omitempty"`
	Actor          string `json:"actor,omitempty"`
}

type ReportStatusRequest struct {
	Kind       string `json:"kind"`
	Message    string `json:"message"`
	Actor      string `json:"actor,omitempty"`
	ProjectID  string `json:"projectId,omitempty"`
	WorkItemID string `json:"workItemId,omitempty"`
	RunID      string `json:"runId,omitempty"`
	SessionID  string `json:"sessionId,omitempty"`
	PTYID      string `json:"ptyId,omitempty"`
}

type ReportStatusResponse struct {
	Event    StatusEvent  `json:"event"`
	Run      *WorkItemRun `json:"run,omitempty"`
	WorkItem *WorkItem    `json:"workItem,omitempty"`
}

type ListStatusEventsRequest struct {
	ProjectID  string `json:"projectId,omitempty"`
	WorkItemID string `json:"workItemId,omitempty"`
	RunID      string `json:"runId,omitempty"`
	SessionID  string `json:"sessionId,omitempty"`
	PTYID      string `json:"ptyId,omitempty"`
	UnreadOnly bool   `json:"unreadOnly,omitempty"`
}

type MarkStatusEventReadRequest struct {
	ID string `json:"id"`
}

type DetectWorktrunkRequest struct {
	RepoPath     string `json:"repoPath"`
	OverridePath string `json:"overridePath"`
}

type WorktrunkBinary struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

type WorktrunkStatus struct {
	Available   bool            `json:"available"`
	ConfigFound bool            `json:"configFound"`
	Binary      WorktrunkBinary `json:"binary"`
}

type ListWorktreesRequest struct {
	RepoPath string `json:"repoPath"`
}

type Worktree struct {
	Branch    string `json:"branch"`
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	IsMain    bool   `json:"isMain"`
	IsCurrent bool   `json:"isCurrent"`
	Dirty     bool   `json:"dirty"`
	Locked    bool   `json:"locked"`
}

type CreateWorktreeRequest struct {
	RepoPath string `json:"repoPath"`
	Branch   string `json:"branch"`
	Base     string `json:"base"`
}

type CreatedWorktree struct {
	Path string `json:"path"`
}

type RemoveWorktreeRequest struct {
	RepoPath     string `json:"repoPath"`
	WorktreePath string `json:"worktreePath"`
	AlsoBranch   bool   `json:"alsoBranch"`
	Force        bool   `json:"force"`
}

type HTTPForward struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	TargetURL string `json:"targetUrl"`
	SessionID string `json:"sessionId"`
}

type CreateHTTPForwardRequest struct {
	Name      string `json:"name"`
	TargetURL string `json:"targetUrl"`
	SessionID string `json:"sessionId"`
}

type StartHTTPForwardRequest struct {
	Name      string `json:"name"`
	TargetURL string `json:"targetUrl"`
	SessionID string `json:"sessionId"`
}

type StartedHTTPForward struct {
	ID       string      `json:"id"`
	LocalURL string      `json:"localUrl"`
	Forward  HTTPForward `json:"forward"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
