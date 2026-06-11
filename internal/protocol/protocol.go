package protocol

import "github.com/phin-tech/whisk/internal/domain/session"

type CreateSessionRequest struct {
	Name       string `json:"name"`
	WorkingDir string `json:"workingDir"`
	Cols       int    `json:"cols"`
	Rows       int    `json:"rows"`
}

type CreatedSession struct {
	Session   session.Session `json:"session"`
	MainPtyID string          `json:"mainPtyId"`
}

type SplitPaneRequest struct {
	SessionID    string `json:"sessionId"`
	TargetPaneID string `json:"targetPaneId"`
	Direction    string `json:"direction"`
	Cols         int    `json:"cols"`
	Rows         int    `json:"rows"`
}

type SplitPaneResult struct {
	Session session.Session `json:"session"`
	PaneID  string          `json:"paneId"`
	PtyID   string          `json:"ptyId"`
}

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
	ID         string `json:"id"`
	WorkingDir string `json:"workingDir"`
	Cols       int    `json:"cols"`
	Rows       int    `json:"rows"`
	Running    bool   `json:"running"`
	SessionID  string `json:"sessionId"`
	PaneID     string `json:"paneId"`
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
