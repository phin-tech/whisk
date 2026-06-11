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

type ErrorResponse struct {
	Error string `json:"error"`
}
