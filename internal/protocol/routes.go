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
	apiSessionList  = []session.Session(nil)
	apiPTYList      = []PTYInfo(nil)
	apiBookmarkList = []ptybookmark.Bookmark(nil)
	apiWorktreeList = []Worktree(nil)
	apiForwardList  = []HTTPForward(nil)
	apiProjectList  = []Project(nil)
	apiWorkflowList = []WorkflowTemplate(nil)
	apiPromptList   = []PromptTemplate(nil)
	apiWorkItemList = []WorkItem(nil)
	apiRunList      = []WorkItemRun(nil)
	apiStatusList   = []StatusEvent(nil)
)

var APIRoutes = []APIRoute{
	{Method: "GET", Path: "/v1/compat", OperationID: "getCompatibility", Tag: "system", Summary: "Daemon API version and git SHA", Response: CompatibilityResponse{}},

	{Method: "GET", Path: "/v1/sessions", OperationID: "listSessions", Tag: "sessions", Response: apiSessionList},
	{Method: "POST", Path: "/v1/sessions", OperationID: "createSession", Tag: "sessions", Request: CreateSessionRequest{}, Response: CreatedSession{}, Status: 201},
	{Method: "DELETE", Path: "/v1/sessions/{sessionID}", OperationID: "closeSession", Tag: "sessions", Response: apiSessionList},
	{Method: "POST", Path: "/v1/sessions/{sessionID}/split", OperationID: "splitPane", Tag: "sessions", Request: SplitPaneRequest{}, Response: SplitPaneResult{}},
	{Method: "POST", Path: "/v1/sessions/{sessionID}/set-root-dir", OperationID: "setSessionRootDir", Tag: "sessions", Request: SetSessionRootDirRequest{}, Response: session.Session{}},
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
	{Method: "GET", Path: "/v1/workflow-templates", OperationID: "listWorkflowTemplates", Tag: "workitems", Response: apiWorkflowList},
	{Method: "GET", Path: "/v1/prompt-templates", OperationID: "listPromptTemplates", Tag: "workitems", Response: apiPromptList},
	{Method: "GET", Path: "/v1/work-items", OperationID: "listWorkItems", Tag: "workitems", Response: apiWorkItemList, Query: []APIQueryParam{{Name: "projectId", Type: "string"}}},
	{Method: "POST", Path: "/v1/work-items", OperationID: "createWorkItem", Tag: "workitems", Request: CreateWorkItemRequest{}, Response: WorkItem{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/move", OperationID: "moveWorkItem", Tag: "workitems", Request: MoveWorkItemRequest{}, Response: WorkItem{}},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/start-planning", OperationID: "startPlanning", Tag: "workitems", Request: StartPlanningRequest{}, Response: WorkItemRun{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/plan-drafts", OperationID: "submitDraftPlan", Tag: "workitems", Request: SubmitDraftPlanRequest{}, Response: Artifact{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/approve-plan", OperationID: "approvePlan", Tag: "workitems", Request: ApprovePlanRequest{}, Response: WorkItem{}},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/start-execution", OperationID: "startExecution", Tag: "workitems", Request: StartExecutionRequest{}, Response: WorkItemRun{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/complete-execution", OperationID: "completeExecutionForWorkItem", Tag: "workitems", Request: CompleteExecutionRequest{}, Response: WorkItem{}},
	{Method: "POST", Path: "/v1/work-item-runs/{runID}/complete-execution", OperationID: "completeExecution", Tag: "workitems", Request: CompleteExecutionRequest{}, Response: WorkItem{}},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/review-feedback", OperationID: "submitReviewFeedback", Tag: "workitems", Request: SubmitReviewFeedbackRequest{}, Response: Artifact{}, Status: 201},
	{Method: "POST", Path: "/v1/questions", OperationID: "askQuestion", Tag: "workitems", Request: AskQuestionRequest{}, Response: Question{}, Status: 201},
	{Method: "POST", Path: "/v1/questions/{questionID}/answer", OperationID: "answerQuestion", Tag: "workitems", Request: AnswerQuestionRequest{}, Response: Question{}},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/bind-worktree", OperationID: "bindWorkItemWorktree", Tag: "workitems", Request: BindWorkItemWorktreeRequest{}, Response: WorkItem{}},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/attachments", OperationID: "addWorkItemAttachment", Tag: "workitems", Request: AddWorkItemAttachmentRequest{}, Response: WorkItem{}, Status: 201},
	{Method: "POST", Path: "/v1/work-items/{workItemID}/delete", OperationID: "deleteWorkItem", Tag: "workitems", Request: DeleteWorkItemRequest{}, Response: WorkItem{}},

	{Method: "GET", Path: "/v1/work-item-runs", OperationID: "listWorkItemRuns", Tag: "workitems", Response: apiRunList, Query: []APIQueryParam{{Name: "workItemId", Type: "string"}}},
	{Method: "POST", Path: "/v1/work-item-runs", OperationID: "startWorkItemRun", Tag: "workitems", Request: StartWorkItemRunRequest{}, Response: WorkItemRun{}, Status: 201},
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
