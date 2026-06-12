package main

import (
	"github.com/phin-tech/whisk/internal/domain/ptybookmark"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

// route maps one HTTP endpoint to its request/response Go types. Paths and
// methods mirror internal/server/http.go exactly; keep them in sync. The
// CI diff-check (make openapi) fails if this drifts from the structs.
type route struct {
	method  string
	path    string
	op      string // operationId -> client function name
	tag     string
	summary string
	req     any // zero-value request struct, or nil for no body
	resp    any // zero-value (or typed-nil slice) response, or nil for 204
	status  int // success status; 0 => 200
	query   []queryParam
}

type queryParam struct {
	name     string
	typ      string // "string" | "integer" | "boolean"
	required bool
}

// typed-nil list helpers so reflect.TypeOf yields the slice type.
var (
	sessionList  = []session.Session(nil)
	ptyList      = []protocol.PTYInfo(nil)
	bookmarkList = []ptybookmark.Bookmark(nil)
	worktreeList = []protocol.Worktree(nil)
	forwardList  = []protocol.HTTPForward(nil)
	projectList  = []protocol.Project(nil)
	workflowList = []protocol.WorkflowTemplate(nil)
	promptList   = []protocol.PromptTemplate(nil)
	workItemList = []protocol.WorkItem(nil)
	runList      = []protocol.WorkItemRun(nil)
	statusList   = []protocol.StatusEvent(nil)
)

var routes = []route{
	{method: "GET", path: "/v1/compat", op: "getCompatibility", tag: "system", summary: "Daemon API version and git SHA", resp: protocol.CompatibilityResponse{}},

	// Sessions & panes
	{method: "GET", path: "/v1/sessions", op: "listSessions", tag: "sessions", resp: sessionList},
	{method: "POST", path: "/v1/sessions", op: "createSession", tag: "sessions", req: protocol.CreateSessionRequest{}, resp: protocol.CreatedSession{}, status: 201},
	{method: "DELETE", path: "/v1/sessions/{sessionID}", op: "closeSession", tag: "sessions", resp: sessionList},
	{method: "POST", path: "/v1/sessions/{sessionID}/split", op: "splitPane", tag: "sessions", req: protocol.SplitPaneRequest{}, resp: protocol.SplitPaneResult{}},
	{method: "POST", path: "/v1/sessions/{sessionID}/set-root-dir", op: "setSessionRootDir", tag: "sessions", req: protocol.SetSessionRootDirRequest{}, resp: session.Session{}},
	{method: "POST", path: "/v1/sessions/{sessionID}/panes/{paneID}/set-working-dir", op: "setPaneWorkingDir", tag: "sessions", req: protocol.SetPaneWorkingDirRequest{}, resp: session.Session{}},
	{method: "POST", path: "/v1/sessions/{sessionID}/panes/{paneID}/start-pty", op: "startPanePTY", tag: "sessions", req: protocol.StartPanePTYRequest{}, resp: protocol.StartedPanePTY{}, status: 201},
	{method: "POST", path: "/v1/sessions/{sessionID}/panes/{paneID}/restart-pty", op: "restartPanePTY", tag: "sessions", req: protocol.RestartPanePTYRequest{}, resp: protocol.RestartedPanePTY{}, status: 201},
	{method: "POST", path: "/v1/sessions/{sessionID}/panes/{paneID}/detach-pty", op: "detachPanePTY", tag: "sessions", req: protocol.DetachPanePTYRequest{}, resp: protocol.DetachedPanePTY{}},
	{method: "POST", path: "/v1/sessions/{sessionID}/windows/{windowID}/panes/{paneID}/close", op: "closePane", tag: "sessions", req: protocol.ClosePaneRequest{}, resp: session.Session{}},

	// PTYs
	{method: "GET", path: "/v1/ptys", op: "listPTYs", tag: "ptys", resp: ptyList},
	{method: "POST", path: "/v1/ptys/{ptyID}/write", op: "writePTY", tag: "ptys", req: protocol.WritePTYRequest{}, status: 204},
	{method: "POST", path: "/v1/ptys/{ptyID}/resize", op: "resizePTY", tag: "ptys", req: protocol.ResizePTYRequest{}, status: 204},
	{method: "POST", path: "/v1/ptys/{ptyID}/kill", op: "killPTY", tag: "ptys", req: protocol.KillPTYRequest{}, resp: protocol.PTYInfo{}},
	{method: "GET", path: "/v1/ptys/{ptyID}/output", op: "getPTYOutput", tag: "ptys", resp: protocol.OutputSnapshot{}, query: []queryParam{{name: "from", typ: "integer"}}},

	// PTY bookmarks
	{method: "POST", path: "/v1/ptys/{ptyID}/bookmarks", op: "addPTYBookmark", tag: "ptys", req: protocol.AddPTYBookmarkRequest{}, resp: ptybookmark.Bookmark{}, status: 201},
	{method: "GET", path: "/v1/ptys/{ptyID}/bookmarks", op: "listPTYBookmarks", tag: "ptys", resp: bookmarkList},
	{method: "DELETE", path: "/v1/pty-bookmarks/{bookmarkID}", op: "removePTYBookmark", tag: "ptys", status: 204},

	// Events
	{method: "GET", path: "/v1/events/next", op: "nextEvent", tag: "events", resp: protocol.RuntimeEvent{}, query: []queryParam{{name: "timeoutMs", typ: "integer"}}},

	// Worktrees
	{method: "POST", path: "/v1/worktrunk/detect", op: "detectWorktrunk", tag: "worktrees", req: protocol.DetectWorktrunkRequest{}, resp: protocol.WorktrunkStatus{}},
	{method: "POST", path: "/v1/worktrees/list", op: "listWorktrees", tag: "worktrees", req: protocol.ListWorktreesRequest{}, resp: worktreeList},
	{method: "POST", path: "/v1/worktrees/create", op: "createWorktree", tag: "worktrees", req: protocol.CreateWorktreeRequest{}, resp: protocol.CreatedWorktree{}, status: 201},
	{method: "POST", path: "/v1/worktrees/remove", op: "removeWorktree", tag: "worktrees", req: protocol.RemoveWorktreeRequest{}, status: 204},

	// HTTP forwards
	{method: "POST", path: "/v1/http-forwards", op: "createHTTPForward", tag: "forwards", req: protocol.CreateHTTPForwardRequest{}, resp: protocol.HTTPForward{}, status: 201},
	{method: "GET", path: "/v1/http-forwards", op: "listHTTPForwards", tag: "forwards", resp: forwardList},
	{method: "DELETE", path: "/v1/http-forwards/{forwardID}", op: "deleteHTTPForward", tag: "forwards", status: 204},

	// Projects, templates, work items
	{method: "GET", path: "/v1/projects", op: "listProjects", tag: "workitems", resp: projectList},
	{method: "POST", path: "/v1/projects", op: "createProject", tag: "workitems", req: protocol.CreateProjectRequest{}, resp: protocol.Project{}, status: 201},
	{method: "GET", path: "/v1/workflow-templates", op: "listWorkflowTemplates", tag: "workitems", resp: workflowList},
	{method: "GET", path: "/v1/prompt-templates", op: "listPromptTemplates", tag: "workitems", resp: promptList},
	{method: "GET", path: "/v1/work-items", op: "listWorkItems", tag: "workitems", resp: workItemList, query: []queryParam{{name: "projectId", typ: "string"}}},
	{method: "POST", path: "/v1/work-items", op: "createWorkItem", tag: "workitems", req: protocol.CreateWorkItemRequest{}, resp: protocol.WorkItem{}, status: 201},
	{method: "POST", path: "/v1/work-items/{workItemID}/move", op: "moveWorkItem", tag: "workitems", req: protocol.MoveWorkItemRequest{}, resp: protocol.WorkItem{}},
	{method: "POST", path: "/v1/work-items/{workItemID}/bind-worktree", op: "bindWorkItemWorktree", tag: "workitems", req: protocol.BindWorkItemWorktreeRequest{}, resp: protocol.WorkItem{}},
	{method: "POST", path: "/v1/work-items/{workItemID}/attachments", op: "addWorkItemAttachment", tag: "workitems", req: protocol.AddWorkItemAttachmentRequest{}, resp: protocol.WorkItem{}, status: 201},
	{method: "POST", path: "/v1/work-items/{workItemID}/delete", op: "deleteWorkItem", tag: "workitems", req: protocol.DeleteWorkItemRequest{}, resp: protocol.WorkItem{}},

	// Work item runs & status
	{method: "GET", path: "/v1/work-item-runs", op: "listWorkItemRuns", tag: "workitems", resp: runList, query: []queryParam{{name: "workItemId", typ: "string"}}},
	{method: "POST", path: "/v1/work-item-runs", op: "startWorkItemRun", tag: "workitems", req: protocol.StartWorkItemRunRequest{}, resp: protocol.WorkItemRun{}, status: 201},
	{method: "POST", path: "/v1/work-item-runs/{runID}/cancel", op: "cancelWorkItemRun", tag: "workitems", req: protocol.CancelWorkItemRunRequest{}, resp: protocol.WorkItemRun{}},
	{method: "POST", path: "/v1/status", op: "reportStatus", tag: "workitems", req: protocol.ReportStatusRequest{}, resp: protocol.ReportStatusResponse{}, status: 201},
	{method: "GET", path: "/v1/status-events", op: "listStatusEvents", tag: "workitems", resp: statusList, query: []queryParam{
		{name: "projectId", typ: "string"},
		{name: "workItemId", typ: "string"},
		{name: "runId", typ: "string"},
		{name: "sessionId", typ: "string"},
		{name: "ptyId", typ: "string"},
		{name: "unreadOnly", typ: "boolean"},
	}},
	{method: "POST", path: "/v1/status-events/{statusEventID}/read", op: "markStatusEventRead", tag: "workitems", req: protocol.MarkStatusEventReadRequest{}, resp: protocol.StatusEvent{}},
}
