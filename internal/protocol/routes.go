package protocol

import (
	"github.com/phin-tech/whisk/internal/domain/ptybookmark"
	"github.com/phin-tech/whisk/internal/domain/session"
)

// APIRoute maps a daemon HTTP endpoint to its protocol request/response types.
// SDK/OpenAPI generation consumes this catalog; parity tests keep it aligned
// with the real HTTP router.
type APIRoute struct {
	Method      string
	Path        string
	OperationID string
	Tag         string
	Summary     string
	Request     any
	Response    any
	Status      int
	Query       []APIQueryParam
}

type APIQueryParam struct {
	Name     string
	Type     string
	Required bool
}

var (
	apiSessionList              = []session.Session(nil)
	apiPTYList                  = []PTYInfo(nil)
	apiBookmarkList             = []ptybookmark.Bookmark(nil)
	apiWorktreeList             = []Worktree(nil)
	apiForwardList              = []HTTPForward(nil)
	apiProjectList              = []Project(nil)
	apiWorkflowTemplateList     = []WorkflowTemplate(nil)
	apiPromptList               = []PromptTemplate(nil)
	apiWorkItemList             = []WorkItem(nil)
	apiRunList                  = []WorkItemRun(nil)
	apiArtifactList             = []Artifact(nil)
	apiQuestionList             = []Question(nil)
	apiGateList                 = []GateReport(nil)
	apiWorkflowEventList        = []WorkflowEvent(nil)
	apiStatusList               = []StatusEvent(nil)
	apiAgentBridgeApprovalList  = []AgentBridgeApproval(nil)
	apiAgentBridgeEventList     = []AgentBridgeEvent(nil)
	apiAgentHookIntegrationList = []AgentHookIntegration(nil)
	apiPluginList               = []PluginStatus(nil)
)

var APIRoutes = []APIRoute{
	{Method: "GET", Path: "/v1/compat", OperationID: "getCompatibility", Tag: "system", Summary: "Daemon API version and git SHA", Response: CompatibilityResponse{}},
	{Method: "POST", Path: "/v1/daemon/clear", OperationID: "clearDaemon", Tag: "system", Summary: "Clear daemon-owned runtime state", Request: ClearDaemonRequest{}, Response: ClearDaemonResponse{}},
	{Method: "GET", Path: "/v1/onboarding", OperationID: "getOnboarding", Tag: "system", Summary: "Get local onboarding status", Response: OnboardingStatus{}},
	{Method: "POST", Path: "/v1/onboarding/apply", OperationID: "applyOnboarding", Tag: "system", Summary: "Apply selected local onboarding items", Request: OnboardingApplyRequest{}, Response: OnboardingStatus{}},
	{Method: "POST", Path: "/v1/agent-bridges/{bridgeID}/hooks", OperationID: "agentBridgeHook", Tag: "agent-bridges", Summary: "Handle provider hook callback for a daemon-owned agent bridge", Request: AgentBridgeHookRequest{}, Response: AgentBridgeHookResponse{}},
	{Method: "POST", Path: "/v1/agent-hook-events", OperationID: "recordAgentHookEvent", Tag: "agent-bridges", Summary: "Record a passive provider hook event", Request: AgentBridgeHookRequest{}, Response: AgentBridgeEvent{}, Status: 201},
	{Method: "GET", Path: "/v1/agent-bridge-approvals", OperationID: "listAgentBridgeApprovals", Tag: "agent-bridges", Summary: "List pending or resolved daemon-owned agent bridge approvals", Response: apiAgentBridgeApprovalList, Query: []APIQueryParam{{Name: "status", Type: "string"}}},
	{Method: "POST", Path: "/v1/agent-bridge-approvals/{approvalID}/resolve", OperationID: "resolveAgentBridgeApproval", Tag: "agent-bridges", Summary: "Resolve a pending daemon-owned agent bridge approval", Request: ResolveAgentBridgeApprovalRequest{}, Response: AgentBridgeApproval{}},
	{Method: "GET", Path: "/v1/agent-bridge-events", OperationID: "listAgentBridgeEvents", Tag: "agent-bridges", Summary: "List passive provider hook events", Response: apiAgentBridgeEventList, Query: []APIQueryParam{{Name: "status", Type: "string"}}},
	{Method: "POST", Path: "/v1/agent-bridge-events/{eventID}/read", OperationID: "markAgentBridgeEventRead", Tag: "agent-bridges", Summary: "Mark a passive provider hook event read", Request: MarkAgentBridgeEventReadRequest{}, Response: AgentBridgeEvent{}},
	{Method: "GET", Path: "/v1/agent-hook-integrations", OperationID: "listAgentHookIntegrations", Tag: "agent-bridges", Summary: "List globally installed provider hook integrations", Response: apiAgentHookIntegrationList},
	{Method: "POST", Path: "/v1/agent-hook-integrations/check", OperationID: "checkAgentHookIntegration", Tag: "agent-bridges", Summary: "Check one global provider hook integration", Request: AgentHookIntegrationRequest{}, Response: AgentHookIntegration{}},
	{Method: "POST", Path: "/v1/agent-hook-integrations/install", OperationID: "installAgentHookIntegration", Tag: "agent-bridges", Summary: "Install or update one global provider hook integration", Request: AgentHookIntegrationRequest{}, Response: AgentHookIntegration{}},
	{Method: "POST", Path: "/v1/agent-hook-integrations/remove", OperationID: "removeAgentHookIntegration", Tag: "agent-bridges", Summary: "Remove one global provider hook integration", Request: AgentHookIntegrationRequest{}, Response: AgentHookIntegration{}},
	{Method: "GET", Path: "/v1/agent-hook-log", OperationID: "getAgentHookLogStatus", Tag: "agent-bridges", Summary: "Get hook log status and path", Response: AgentHookLogStatus{}},
	{Method: "POST", Path: "/v1/agent-hook-log/settings", OperationID: "setAgentHookLogSettings", Tag: "agent-bridges", Summary: "Update hook log settings", Request: SetAgentHookLogSettingsRequest{}, Response: AgentHookLogStatus{}},
	{Method: "POST", Path: "/v1/agent-hook-log/clear", OperationID: "clearAgentHookLog", Tag: "agent-bridges", Summary: "Clear hook log files", Response: AgentHookLogStatus{}},
	{Method: "POST", Path: "/v1/agent-hook-log/open", OperationID: "openAgentHookLog", Tag: "agent-bridges", Summary: "Open hook log in the platform editor", Response: AgentHookLogStatus{}},
	{Method: "GET", Path: "/v1/plugins", OperationID: "listPlugins", Tag: "plugins", Summary: "List discovered plugins", Response: apiPluginList},
	{Method: "POST", Path: "/v1/plugins/rescan", OperationID: "rescanPlugins", Tag: "plugins", Summary: "Rescan plugin directories", Response: apiPluginList},
	{Method: "POST", Path: "/v1/plugins/{pluginID}/trust", OperationID: "trustPlugin", Tag: "plugins", Summary: "Trust a discovered plugin", Response: PluginStatus{}},
	{Method: "POST", Path: "/v1/plugins/{pluginID}/untrust", OperationID: "untrustPlugin", Tag: "plugins", Summary: "Untrust a discovered plugin", Response: PluginStatus{}},
	{Method: "POST", Path: "/v1/plugins/{pluginID}/project-attachment-templates/{templateID}", OperationID: "runPluginProjectAttachmentTemplate", Tag: "plugins", Summary: "Run a trusted plugin project attachment template", Request: RunPluginProjectAttachmentTemplateRequest{}, Response: Project{}, Status: 201},

	{Method: "GET", Path: "/v1/sessions", OperationID: "listSessions", Tag: "sessions", Response: apiSessionList},
	{Method: "POST", Path: "/v1/sessions", OperationID: "createSession", Tag: "sessions", Request: CreateSessionRequest{}, Response: CreatedSession{}, Status: 201},
	{Method: "DELETE", Path: "/v1/sessions/{sessionID}", OperationID: "closeSession", Tag: "sessions", Response: apiSessionList},
	{Method: "POST", Path: "/v1/sessions/{sessionID}/split", OperationID: "splitPane", Tag: "sessions", Request: SplitPaneRequest{}, Response: SplitPaneResult{}},
	{Method: "POST", Path: "/v1/sessions/{sessionID}/set-root-dir", OperationID: "setSessionRootDir", Tag: "sessions", Request: SetSessionRootDirRequest{}, Response: session.Session{}},
	{Method: "POST", Path: "/v1/sessions/{sessionID}/set-project", OperationID: "setSessionProject", Tag: "sessions", Request: SetSessionProjectRequest{}, Response: session.Session{}},
	{Method: "POST", Path: "/v1/sessions/{sessionID}/panes/{paneID}/set-working-dir", OperationID: "setPaneWorkingDir", Tag: "sessions", Request: SetPaneWorkingDirRequest{}, Response: session.Session{}},
	{Method: "POST", Path: "/v1/sessions/{sessionID}/panes/{paneID}/start-pty", OperationID: "startPanePTY", Tag: "sessions", Request: StartPanePTYRequest{}, Response: StartedPanePTY{}, Status: 201},
	{Method: "POST", Path: "/v1/sessions/{sessionID}/panes/{paneID}/restart-pty", OperationID: "restartPanePTY", Tag: "sessions", Request: RestartPanePTYRequest{}, Response: RestartedPanePTY{}, Status: 201},
	{Method: "POST", Path: "/v1/sessions/{sessionID}/panes/{paneID}/detach-pty", OperationID: "detachPanePTY", Tag: "sessions", Request: DetachPanePTYRequest{}, Response: DetachedPanePTY{}},
	{Method: "POST", Path: "/v1/sessions/{sessionID}/windows/{windowID}/panes/{paneID}/close", OperationID: "closePane", Tag: "sessions", Request: ClosePaneRequest{}, Response: session.Session{}},

	{Method: "GET", Path: "/v1/ptys", OperationID: "listPTYs", Tag: "ptys", Response: apiPTYList},
	{Method: "POST", Path: "/v1/ptys/{ptyID}/write", OperationID: "writePTY", Tag: "ptys", Request: WritePTYRequest{}, Status: 204},
	{Method: "POST", Path: "/v1/ptys/{ptyID}/resize", OperationID: "resizePTY", Tag: "ptys", Request: ResizePTYRequest{}, Status: 204},
	{Method: "POST", Path: "/v1/ptys/{ptyID}/kill", OperationID: "killPTY", Tag: "ptys", Request: KillPTYRequest{}, Response: PTYInfo{}},
	{Method: "GET", Path: "/v1/ptys/{ptyID}/output", OperationID: "getPTYOutput", Tag: "ptys", Response: OutputSnapshot{}, Query: []APIQueryParam{{Name: "from", Type: "integer"}}},
	{Method: "POST", Path: "/v1/ptys/{ptyID}/bookmarks", OperationID: "addPTYBookmark", Tag: "ptys", Request: AddPTYBookmarkRequest{}, Response: ptybookmark.Bookmark{}, Status: 201},
	{Method: "GET", Path: "/v1/ptys/{ptyID}/bookmarks", OperationID: "listPTYBookmarks", Tag: "ptys", Response: apiBookmarkList},
	{Method: "DELETE", Path: "/v1/pty-bookmarks/{bookmarkID}", OperationID: "removePTYBookmark", Tag: "ptys", Status: 204},

	{Method: "GET", Path: "/v1/events/next", OperationID: "nextEvent", Tag: "events", Response: RuntimeEvent{}, Query: []APIQueryParam{{Name: "timeoutMs", Type: "integer"}}},

	{Method: "POST", Path: "/v1/worktrunk/detect", OperationID: "detectWorktrunk", Tag: "worktrees", Request: DetectWorktrunkRequest{}, Response: WorktrunkStatus{}},
	{Method: "POST", Path: "/v1/worktrees/list", OperationID: "listWorktrees", Tag: "worktrees", Request: ListWorktreesRequest{}, Response: apiWorktreeList},
	{Method: "POST", Path: "/v1/worktrees/create", OperationID: "createWorktree", Tag: "worktrees", Request: CreateWorktreeRequest{}, Response: CreatedWorktree{}, Status: 201},
	{Method: "POST", Path: "/v1/worktrees/remove", OperationID: "removeWorktree", Tag: "worktrees", Request: RemoveWorktreeRequest{}, Status: 204},

	{Method: "POST", Path: "/v1/http-forwards", OperationID: "createHTTPForward", Tag: "forwards", Request: CreateHTTPForwardRequest{}, Response: HTTPForward{}, Status: 201},
	{Method: "GET", Path: "/v1/http-forwards", OperationID: "listHTTPForwards", Tag: "forwards", Response: apiForwardList},
	{Method: "DELETE", Path: "/v1/http-forwards/{forwardID}", OperationID: "deleteHTTPForward", Tag: "forwards", Status: 204},

	{Method: "GET", Path: "/v1/projects", OperationID: "listProjects", Tag: "workitems", Response: apiProjectList},
	{Method: "POST", Path: "/v1/projects", OperationID: "createProject", Tag: "workitems", Request: CreateProjectRequest{}, Response: Project{}, Status: 201},
	{Method: "POST", Path: "/v1/projects/{projectID}/update", OperationID: "updateProject", Tag: "workitems", Request: UpdateProjectRequest{}, Response: Project{}},
	{Method: "GET", Path: "/v1/projects/{projectID}/detail", OperationID: "getProjectDetail", Tag: "workitems", Response: ProjectDetail{}},
	{Method: "POST", Path: "/v1/projects/{projectID}/attachments", OperationID: "addProjectAttachment", Tag: "workitems", Request: AddProjectAttachmentRequest{}, Response: Project{}, Status: 201},
	{Method: "POST", Path: "/v1/project-attachments/{attachmentID}/update", OperationID: "updateProjectAttachment", Tag: "workitems", Request: UpdateProjectAttachmentRequest{}, Response: Project{}},
	{Method: "POST", Path: "/v1/project-attachments/{attachmentID}/delete", OperationID: "deleteProjectAttachment", Tag: "workitems", Request: DeleteProjectAttachmentRequest{}, Response: Project{}},
	{Method: "GET", Path: "/v1/projects/{projectID}/context", OperationID: "getProjectContext", Tag: "workitems", Response: ProjectContext{}},
	{Method: "GET", Path: "/v1/workflow-templates", OperationID: "listWorkflowTemplates", Tag: "workitems", Response: apiWorkflowTemplateList},
	{Method: "GET", Path: "/v1/prompt-templates", OperationID: "listPromptTemplates", Tag: "workitems", Response: apiPromptList},
	{Method: "GET", Path: "/v1/work-items", OperationID: "listWorkItems", Tag: "workitems", Response: apiWorkItemList, Query: []APIQueryParam{{Name: "projectId", Type: "string"}}},
	{Method: "POST", Path: "/v1/work-items", OperationID: "createWorkItem", Tag: "workitems", Request: CreateWorkItemRequest{}, Response: WorkItem{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/move", OperationID: "moveWorkItem", Tag: "workitems", Request: MoveWorkItemRequest{}, Response: WorkItem{}},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/start-planning", OperationID: "startPlanning", Tag: "workitems", Request: StartPlanningRequest{}, Response: WorkItemRun{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/plan-drafts", OperationID: "submitDraftPlan", Tag: "workitems", Request: SubmitDraftPlanRequest{}, Response: Artifact{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/approve-plan", OperationID: "approvePlan", Tag: "workitems", Request: ApprovePlanRequest{}, Response: WorkItem{}},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/start-execution", OperationID: "startExecution", Tag: "workitems", Request: StartExecutionRequest{}, Response: WorkItemRun{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/queue-execution", OperationID: "queueExecution", Tag: "workitems", Request: QueueExecutionRequest{}, Response: WorkItemRun{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/launch-execution", OperationID: "launchExecution", Tag: "workitems", Request: LaunchExecutionRequest{}, Response: WorkItemRun{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/complete-execution", OperationID: "completeExecutionForWorkItem", Tag: "workitems", Request: CompleteExecutionRequest{}, Response: WorkItem{}},
	{Method: "POST", Path: "/v1/work-item-runs/{runID}/complete-execution", OperationID: "completeExecution", Tag: "workitems", Request: CompleteExecutionRequest{}, Response: WorkItem{}},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/review-feedback", OperationID: "submitReviewFeedback", Tag: "workitems", Request: SubmitReviewFeedbackRequest{}, Response: Artifact{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/approve-done", OperationID: "approveDone", Tag: "workitems", Request: ApproveDoneRequest{}, Response: WorkItem{}},
	{Method: "GET", Path: "/v1/artifacts", OperationID: "listArtifacts", Tag: "workitems", Response: apiArtifactList, Query: []APIQueryParam{{Name: "workItemId", Type: "string"}}},
	{Method: "GET", Path: "/v1/questions", OperationID: "listQuestions", Tag: "workitems", Response: apiQuestionList, Query: []APIQueryParam{{Name: "workItemId", Type: "string"}}},
	{Method: "POST", Path: "/v1/questions", OperationID: "askQuestion", Tag: "workitems", Request: AskQuestionRequest{}, Response: Question{}, Status: 201},
	{Method: "POST", Path: "/v1/questions/{questionID}/answer", OperationID: "answerQuestion", Tag: "workitems", Request: AnswerQuestionRequest{}, Response: Question{}},
	{Method: "GET", Path: "/v1/gate-reports", OperationID: "listGateReports", Tag: "workitems", Response: apiGateList, Query: []APIQueryParam{{Name: "workItemId", Type: "string"}}},
	{Method: "POST", Path: "/v1/gate-reports/{gateReportID}/complete", OperationID: "completeGate", Tag: "workitems", Request: CompleteGateRequest{}, Response: GateReport{}},
	{Method: "GET", Path: "/v1/workflow-events", OperationID: "listWorkflowEvents", Tag: "workitems", Response: apiWorkflowEventList, Query: []APIQueryParam{{Name: "workItemId", Type: "string"}}},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/bind-worktree", OperationID: "bindWorkItemWorktree", Tag: "workitems", Request: BindWorkItemWorktreeRequest{}, Response: WorkItem{}},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/attachments", OperationID: "addWorkItemAttachment", Tag: "workitems", Request: AddWorkItemAttachmentRequest{}, Response: WorkItem{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/delete", OperationID: "deleteWorkItem", Tag: "workitems", Request: DeleteWorkItemRequest{}, Response: WorkItem{}},

	{Method: "GET", Path: "/v1/work-item-runs", OperationID: "listWorkItemRuns", Tag: "workitems", Response: apiRunList, Query: []APIQueryParam{{Name: "workItemId", Type: "string"}}},
	{Method: "POST", Path: "/v1/work-item-runs", OperationID: "startWorkItemRun", Tag: "workitems", Request: StartWorkItemRunRequest{}, Response: WorkItemRun{}, Status: 201},
	{Method: "POST", Path: "/v1/work-item-runs/{runID}/launch", OperationID: "launchWorkItemRun", Tag: "workitems", Request: LaunchWorkItemRunRequest{}, Response: WorkItemRun{}},
	{Method: "POST", Path: "/v1/work-item-runs/{runID}/cancel", OperationID: "cancelWorkItemRun", Tag: "workitems", Request: CancelWorkItemRunRequest{}, Response: WorkItemRun{}},
	{Method: "POST", Path: "/v1/status", OperationID: "reportStatus", Tag: "workitems", Request: ReportStatusRequest{}, Response: ReportStatusResponse{}, Status: 201},
	{Method: "GET", Path: "/v1/status-events", OperationID: "listStatusEvents", Tag: "workitems", Response: apiStatusList, Query: []APIQueryParam{
		{Name: "projectId", Type: "string"},
		{Name: "workItemId", Type: "string"},
		{Name: "runId", Type: "string"},
		{Name: "sessionId", Type: "string"},
		{Name: "ptyId", Type: "string"},
		{Name: "unreadOnly", Type: "boolean"},
	}},
	{Method: "POST", Path: "/v1/status-events/{statusEventID}/read", OperationID: "markStatusEventRead", Tag: "workitems", Request: MarkStatusEventReadRequest{}, Response: StatusEvent{}},
}
