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
	PtyID  string `json:"ptyId"`
	Offset uint64 `json:"offset"`
	Output string `json:"output"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
