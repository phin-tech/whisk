package protocol

import (
	"github.com/phin-tech/whisk/internal/domain/ptybookmark"
	"github.com/phin-tech/whisk/internal/domain/session"
)

const DaemonAPIVersion = 1

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
