package protocol

import "github.com/phin-tech/whisk/internal/domain/session"

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
	apiPTYHistoryList           = []PTYHistorySummary(nil)
	apiWorktreeList             = []Worktree(nil)
	apiForwardList              = []HTTPForward(nil)
	apiProjectList              = []Project(nil)
	apiWorkflowDefinitionList   = []WorkflowDefinitionRecord(nil)
	apiWorkflowActionList       = []WorkflowActionAvailability(nil)
	apiWorkflowTemplateList     = []WorkflowTemplate(nil)
	apiPromptList               = []PromptTemplate(nil)
	apiAgentProfileList         = []AgentProfile(nil)
	apiDetectedAgentList        = []DetectedAgent(nil)
	apiSkillCatalog             = SkillCatalog{}
	apiWorkItemList             = []WorkItem(nil)
	apiWorkItemLinkList         = []WorkItemLink(nil)
	apiRunList                  = []WorkItemRun(nil)
	apiArtifactList             = []Artifact(nil)
	apiQuestionList             = []Question(nil)
	apiGateList                 = []GateReport(nil)
	apiWorkflowEventList        = []WorkflowEvent(nil)
	apiStatusList               = []StatusEvent(nil)
	apiMailList                 = []MailMessage(nil)
	apiBrowserResourceList      = []BrowserResource(nil)
	apiBrowserTargetList        = []BrowserTarget(nil)
	apiAgentBridgeApprovalList  = []AgentBridgeApproval(nil)
	apiAgentBridgeEventList     = []AgentBridgeEvent(nil)
	apiAgentPromptList          = []AgentPrompt(nil)
	apiAgentHookIntegrationList = []AgentHookIntegration(nil)
	apiPluginList               = []PluginStatus(nil)
	apiUsageResolverList        = []UsageResolverReadModel(nil)
	apiRegistryPluginList       = []RegistryPlugin(nil)
	apiUIContributionsResponse  = UIContributionsResponse{}
)

var APIRoutes = []APIRoute{
	{Method: "GET", Path: "/v1/compat", OperationID: "getCompatibility", Tag: "system", Summary: "Daemon protocol compatibility and build metadata", Response: CompatibilityResponse{}},
	{Method: "POST", Path: "/v1/daemon/clear", OperationID: "clearDaemon", Tag: "system", Summary: "Clear daemon-owned runtime state", Request: ClearDaemonRequest{}, Response: ClearDaemonResponse{}},
	{Method: "GET", Path: "/v1/onboarding", OperationID: "getOnboarding", Tag: "system", Summary: "Get local onboarding status", Response: OnboardingStatus{}},
	{Method: "POST", Path: "/v1/onboarding/apply", OperationID: "applyOnboarding", Tag: "system", Summary: "Apply selected local onboarding items", Request: OnboardingApplyRequest{}, Response: OnboardingStatus{}},
	{Method: "GET", Path: "/v1/skills", OperationID: "listSkills", Tag: "skills", Summary: "List daemon-discovered agent skills", Response: apiSkillCatalog, Query: []APIQueryParam{{Name: "projectId", Type: "string"}, {Name: "sessionId", Type: "string"}}},
	{Method: "POST", Path: "/v1/skills/rescan", OperationID: "rescanSkills", Tag: "skills", Summary: "Rescan daemon-discovered agent skills", Request: ListSkillsRequest{}, Response: apiSkillCatalog},
	{Method: "POST", Path: "/v1/agent-bridges/{bridgeID}/hooks", OperationID: "agentBridgeHook", Tag: "agent-bridges", Summary: "Handle provider hook callback for a daemon-owned agent bridge", Request: AgentBridgeHookRequest{}, Response: AgentBridgeHookResponse{}},
	{Method: "POST", Path: "/v1/agent-hook-events", OperationID: "recordAgentHookEvent", Tag: "agent-bridges", Summary: "Record a passive provider hook event", Request: AgentBridgeHookRequest{}, Response: AgentBridgeEvent{}, Status: 201},
	{Method: "GET", Path: "/v1/agent-bridge-approvals", OperationID: "listAgentBridgeApprovals", Tag: "agent-bridges", Summary: "List pending or resolved daemon-owned agent bridge approvals", Response: apiAgentBridgeApprovalList, Query: []APIQueryParam{{Name: "status", Type: "string"}}},
	{Method: "POST", Path: "/v1/agent-bridge-approvals/{approvalID}/resolve", OperationID: "resolveAgentBridgeApproval", Tag: "agent-bridges", Summary: "Resolve a pending daemon-owned agent bridge approval", Request: ResolveAgentBridgeApprovalRequest{}, Response: AgentBridgeApproval{}},
	{Method: "GET", Path: "/v1/agent-prompts", OperationID: "listAgentPrompts", Tag: "agent-bridges", Summary: "List pending or resolved daemon-owned agent prompts", Response: apiAgentPromptList, Query: []APIQueryParam{{Name: "status", Type: "string"}}},
	{Method: "POST", Path: "/v1/agent-prompts/{promptID}/resolve", OperationID: "resolveAgentPrompt", Tag: "agent-bridges", Summary: "Resolve a pending daemon-owned agent prompt", Request: ResolveAgentPromptRequest{}, Response: AgentPrompt{}},
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
	{Method: "GET", Path: "/v1/plugin-registry", OperationID: "listRegistryPlugins", Tag: "plugins", Summary: "List installable plugins from the configured registries", Response: apiRegistryPluginList},
	{Method: "POST", Path: "/v1/plugin-registry/install", OperationID: "installRegistryPlugin", Tag: "plugins", Summary: "Install a plugin from a configured registry (untrusted)", Request: InstallRegistryPluginRequest{}, Response: PluginStatus{}, Status: 201},
	{Method: "POST", Path: "/v1/plugins/rescan", OperationID: "rescanPlugins", Tag: "plugins", Summary: "Rescan plugin directories", Response: apiPluginList},
	{Method: "POST", Path: "/v1/plugins/{pluginID}/trust", OperationID: "trustPlugin", Tag: "plugins", Summary: "Trust a discovered plugin", Response: PluginStatus{}},
	{Method: "POST", Path: "/v1/plugins/{pluginID}/untrust", OperationID: "untrustPlugin", Tag: "plugins", Summary: "Untrust a discovered plugin", Response: PluginStatus{}},
	{Method: "POST", Path: "/v1/plugins/{pluginID}/project-attachment-templates/{templateID}", OperationID: "runPluginProjectAttachmentTemplate", Tag: "plugins", Summary: "Run a trusted plugin project attachment template", Request: RunPluginProjectAttachmentTemplateRequest{}, Response: Project{}, Status: 201},
	{Method: "GET", Path: "/v1/usage-resolvers", OperationID: "listUsageResolvers", Tag: "plugins", Summary: "List daemon-owned usage resolver results", Response: apiUsageResolverList},
	{Method: "POST", Path: "/v1/plugins/{pluginID}/usage-resolvers/{resolverID}/refresh", OperationID: "refreshUsageResolver", Tag: "plugins", Summary: "Refresh one trusted plugin usage resolver", Request: RefreshUsageResolverRequest{}, Response: UsageResolverReadModel{}},
	{Method: "GET", Path: "/v1/ui-contributions", OperationID: "listUIContributions", Tag: "plugins", Summary: "Get aggregated UI contributions scoped to an entity", Response: apiUIContributionsResponse, Query: []APIQueryParam{
		{Name: "projectId", Type: "string"},
		{Name: "workItemId", Type: "string"},
		{Name: "runId", Type: "string"},
		{Name: "sessionId", Type: "string"},
		{Name: "paneId", Type: "string"},
		{Name: "ptyId", Type: "string"},
		{Name: "gateReportId", Type: "string"},
		{Name: "phase", Type: "string"},
	}},

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
	{Method: "GET", Path: "/v1/pty-history", OperationID: "listPTYHistory", Tag: "ptys", Summary: "List persisted PTY transcripts", Response: apiPTYHistoryList},
	{Method: "GET", Path: "/v1/pty-history/{ptyID}", OperationID: "readPTYHistory", Tag: "ptys", Summary: "Read one persisted PTY transcript", Response: PTYHistory{}},
	{Method: "POST", Path: "/v1/ptys/{ptyID}/write", OperationID: "writePTY", Tag: "ptys", Request: WritePTYRequest{}, Status: 204},
	{Method: "POST", Path: "/v1/ptys/{ptyID}/resize", OperationID: "resizePTY", Tag: "ptys", Request: ResizePTYRequest{}, Status: 204},
	{Method: "POST", Path: "/v1/ptys/{ptyID}/kill", OperationID: "killPTY", Tag: "ptys", Request: KillPTYRequest{}, Response: PTYInfo{}},
	{Method: "DELETE", Path: "/v1/ptys/{ptyID}", OperationID: "deletePTY", Tag: "ptys", Status: 204},
	{Method: "GET", Path: "/v1/ptys/{ptyID}/output", OperationID: "getPTYOutput", Tag: "ptys", Response: OutputSnapshot{}, Query: []APIQueryParam{{Name: "from", Type: "integer"}, {Name: "snapshot", Type: "boolean"}}},

	{Method: "GET", Path: "/v1/events/next", OperationID: "nextEvent", Tag: "events", Response: NextEventResponse{}, Query: []APIQueryParam{{Name: "timeoutMs", Type: "integer"}, {Name: "afterSeq", Type: "integer"}}},

	{Method: "GET", Path: "/v1/browser-resources", OperationID: "listBrowserResources", Tag: "browser", Summary: "List daemon-owned browser resources", Response: apiBrowserResourceList},
	{Method: "POST", Path: "/v1/browser-resources", OperationID: "connectBrowserResource", Tag: "browser", Summary: "Register an existing loopback Chrome CDP endpoint", Request: ConnectBrowserResourceRequest{}, Response: BrowserResource{}, Status: 201},
	{Method: "DELETE", Path: "/v1/browser-resources/{resourceID}", OperationID: "disconnectBrowserResource", Tag: "browser", Summary: "Detach a daemon-owned browser resource", Status: 204},
	{Method: "GET", Path: "/v1/browser-resources/{resourceID}/targets", OperationID: "listBrowserTargets", Tag: "browser", Summary: "List targets for a daemon-owned browser resource", Response: apiBrowserTargetList},

	{Method: "POST", Path: "/v1/mail", OperationID: "sendMail", Tag: "mail", Summary: "Send a daemon-owned mailbox message", Request: SendMailRequest{}, Response: MailMessage{}, Status: 201},
	{Method: "GET", Path: "/v1/mail", OperationID: "listMail", Tag: "mail", Summary: "List daemon-owned mailbox messages", Response: apiMailList, Query: []APIQueryParam{
		{Name: "to", Type: "string"},
		{Name: "unread", Type: "boolean"},
		{Name: "types", Type: "string"},
		{Name: "projectId", Type: "string"},
		{Name: "workItemId", Type: "string"},
		{Name: "runId", Type: "string"},
		{Name: "threadId", Type: "string"},
		{Name: "limit", Type: "integer"},
	}},
	{Method: "GET", Path: "/v1/mail/next", OperationID: "nextMail", Tag: "mail", Summary: "Get the next unread mailbox message, optionally waiting", Response: NextMailResponse{}, Query: []APIQueryParam{
		{Name: "to", Type: "string"},
		{Name: "types", Type: "string"},
		{Name: "timeoutMs", Type: "integer"},
		{Name: "projectId", Type: "string"},
	}},
	{Method: "POST", Path: "/v1/mail/{mailID}/read", OperationID: "markMailRead", Tag: "mail", Summary: "Mark a mailbox message read", Request: MarkMailReadRequest{}, Response: MailMessage{}},
	{Method: "POST", Path: "/v1/mail/{mailID}/reply", OperationID: "replyMail", Tag: "mail", Summary: "Reply to a mailbox message", Request: ReplyMailRequest{}, Response: MailMessage{}, Status: 201},

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
	{Method: "POST", Path: "/v1/projects/{projectID}/delete", OperationID: "deleteProject", Tag: "workitems", Request: DeleteProjectRequest{}, Response: Project{}},
	{Method: "GET", Path: "/v1/projects/{projectID}/detail", OperationID: "getProjectDetail", Tag: "workitems", Response: ProjectDetail{}},
	{Method: "POST", Path: "/v1/projects/{projectID}/workflow-definition", OperationID: "setProjectWorkflowDefinition", Tag: "workitems", Request: SetProjectWorkflowDefinitionRequest{}, Response: Project{}},
	{Method: "POST", Path: "/v1/projects/{projectID}/workflow-migration-plan", OperationID: "planProjectWorkflowMigration", Tag: "workitems", Request: PlanProjectWorkflowMigrationRequest{}, Response: WorkflowMigrationPlan{}},
	{Method: "POST", Path: "/v1/projects/{projectID}/attachments", OperationID: "addProjectAttachment", Tag: "workitems", Request: AddProjectAttachmentRequest{}, Response: Project{}, Status: 201},
	{Method: "POST", Path: "/v1/project-attachments/{attachmentID}/update", OperationID: "updateProjectAttachment", Tag: "workitems", Request: UpdateProjectAttachmentRequest{}, Response: Project{}},
	{Method: "POST", Path: "/v1/project-attachments/{attachmentID}/delete", OperationID: "deleteProjectAttachment", Tag: "workitems", Request: DeleteProjectAttachmentRequest{}, Response: Project{}},
	{Method: "GET", Path: "/v1/projects/{projectID}/context", OperationID: "getProjectContext", Tag: "workitems", Response: ProjectContext{}},
	{Method: "GET", Path: "/v1/workflow-definitions", OperationID: "listWorkflowDefinitions", Tag: "workitems", Response: apiWorkflowDefinitionList},
	{Method: "POST", Path: "/v1/workflow-definitions/validate", OperationID: "validateWorkflowDefinition", Tag: "workitems", Request: ValidateWorkflowDefinitionRequest{}, Response: WorkflowValidationReport{}},
	{Method: "POST", Path: "/v1/workflow-definitions/validate-file", OperationID: "validateWorkflowDefinitionFile", Tag: "workitems", Request: ValidateWorkflowDefinitionFileRequest{}, Response: WorkflowValidationReport{}},
	{Method: "POST", Path: "/v1/workflow-definitions/import", OperationID: "importWorkflowDefinition", Tag: "workitems", Request: ImportWorkflowDefinitionRequest{}, Response: WorkflowDefinitionRecord{}, Status: 201},
	{Method: "POST", Path: "/v1/workflow-definitions/import-file", OperationID: "importWorkflowDefinitionFile", Tag: "workitems", Request: ImportWorkflowDefinitionFileRequest{}, Response: WorkflowDefinitionRecord{}, Status: 201},
	{Method: "POST", Path: "/v1/workflow-definitions/export-file", OperationID: "exportWorkflowDefinitionFile", Tag: "workitems", Request: ExportWorkflowDefinitionFileRequest{}, Status: 204},
	{Method: "POST", Path: "/v1/workflow-definitions/{workflowID}/{version}/delete", OperationID: "deleteWorkflowDefinition", Tag: "workitems", Response: WorkflowDefinitionRecord{}},
	{Method: "GET", Path: "/v1/workflow-templates", OperationID: "listWorkflowTemplates", Tag: "workitems", Response: apiWorkflowTemplateList},
	{Method: "GET", Path: "/v1/prompt-templates", OperationID: "listPromptTemplates", Tag: "workitems", Response: apiPromptList},
	{Method: "GET", Path: "/v1/agent-profiles", OperationID: "listAgentProfiles", Tag: "agents", Summary: "List daemon agent profiles", Response: apiAgentProfileList},
	{Method: "GET", Path: "/v1/agents/detected", OperationID: "listDetectedAgents", Tag: "agents", Summary: "List builtin agent profiles detected on PATH", Response: apiDetectedAgentList},
	{Method: "GET", Path: "/v1/work-items", OperationID: "listWorkItems", Tag: "workitems", Response: apiWorkItemList, Query: []APIQueryParam{{Name: "projectId", Type: "string"}}},
	{Method: "POST", Path: "/v1/work-items", OperationID: "createWorkItem", Tag: "workitems", Request: CreateWorkItemRequest{}, Response: WorkItem{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/update", OperationID: "updateWorkItem", Tag: "workitems", Request: UpdateWorkItemRequest{}, Response: WorkItem{}},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/move", OperationID: "moveWorkItem", Tag: "workitems", Request: MoveWorkItemRequest{}, Response: WorkItem{}},
	{Method: "GET", Path: "/v1/work-items/{workItemID}/workflow-actions", OperationID: "listWorkItemWorkflowActions", Tag: "workitems", Response: apiWorkflowActionList},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/actions/{actionID}", OperationID: "runWorkItemWorkflowAction", Tag: "workitems", Request: RunWorkItemWorkflowActionRequest{}, Response: WorkItem{}},
	{Method: "GET", Path: "/v1/work-item-links", OperationID: "listWorkItemLinks", Tag: "workitems", Response: apiWorkItemLinkList, Query: []APIQueryParam{{Name: "workItemId", Type: "string"}}},
	{Method: "POST", Path: "/v1/work-item-links", OperationID: "addWorkItemLink", Tag: "workitems", Request: AddWorkItemLinkRequest{}, Response: WorkItemLink{}, Status: 201},
	{Method: "GET", Path: "/v1/ready-work", OperationID: "readyWork", Tag: "workitems", Response: ReadyWorkExplanation{}, Query: []APIQueryParam{{Name: "projectId", Type: "string"}}},
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
