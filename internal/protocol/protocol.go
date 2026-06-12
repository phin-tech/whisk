package protocol

import (
	"github.com/phin-tech/whisk/internal/domain/ptybookmark"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/domain/workitem"
)

const DaemonAPIVersion = 5

type CompatibilityResponse struct {
	APIVersion int `json:"apiVersion"`
}

type CreateSessionRequest struct {
	Name       string           `json:"name"`
	RootDir    string           `json:"rootDir"`
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
	Cols    int    `json:"cols"`
	Rows    int    `json:"rows"`
	Command string `json:"command,omitempty"`
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

type Project = workitem.Project
type WorkflowTemplate = workitem.WorkflowTemplate
type PromptTemplate = workitem.PromptTemplate
type WorkItem = workitem.WorkItem
type WorkItemRun = workitem.WorkItemRun
type StatusEvent = workitem.StatusEvent
type WorktreeBinding = workitem.WorktreeBinding
type Attachment = workitem.Attachment

type CreateProjectRequest struct {
	Name       string `json:"name"`
	Slug       string `json:"slug,omitempty"`
	RootDir    string `json:"rootDir"`
	WorkflowID string `json:"workflowId,omitempty"`
}

type CreateWorkItemRequest struct {
	ProjectID    string `json:"projectId"`
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

type CancelWorkItemRunRequest struct {
	ID    string `json:"id"`
	Actor string `json:"actor,omitempty"`
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
