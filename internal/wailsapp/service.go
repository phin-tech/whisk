package wailsapp

import (
	"context"
	"fmt"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/session"
)

type Service struct {
	runtime *app.Runtime
}

type CreateSessionRequest struct {
	Name       string `json:"name"`
	WorkingDir string `json:"workingDir"`
	Cols       int    `json:"cols"`
	Rows       int    `json:"rows"`
}

type SplitPaneRequest struct {
	SessionID    string `json:"sessionId"`
	TargetPaneID string `json:"targetPaneId"`
	Direction    string `json:"direction"`
	Cols         int    `json:"cols"`
	Rows         int    `json:"rows"`
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

func NewService(runtime *app.Runtime) *Service {
	return &Service{runtime: runtime}
}

func (s *Service) ListSessions(ctx context.Context) ([]session.Session, error) {
	return s.runtime.ListSessions(ctx)
}

func (s *Service) CreateSession(ctx context.Context, req CreateSessionRequest) (app.CreatedSession, error) {
	return s.runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       req.Name,
		WorkingDir: req.WorkingDir,
		Cols:       req.Cols,
		Rows:       req.Rows,
	})
}

func (s *Service) SplitPane(ctx context.Context, req SplitPaneRequest) (app.SplitPaneResult, error) {
	direction, err := parseDirection(req.Direction)
	if err != nil {
		return app.SplitPaneResult{}, err
	}
	return s.runtime.SplitPane(ctx, app.SplitPaneRequest{
		SessionID:    req.SessionID,
		TargetPaneID: req.TargetPaneID,
		Direction:    direction,
		Cols:         req.Cols,
		Rows:         req.Rows,
	})
}

func (s *Service) WritePTY(ctx context.Context, req WritePTYRequest) error {
	return s.runtime.WritePTY(ctx, req.PtyID, []byte(req.Data))
}

func (s *Service) ResizePTY(ctx context.Context, req ResizePTYRequest) error {
	return s.runtime.ResizePTY(ctx, req.PtyID, app.PTYSize{
		Cols: req.Cols,
		Rows: req.Rows,
	})
}

func (s *Service) Output(ctx context.Context, req OutputRequest) (OutputSnapshot, error) {
	snapshot, err := s.runtime.PTYOutput(ctx, req.PtyID, req.FromOffset)
	if err != nil {
		return OutputSnapshot{}, err
	}
	return OutputSnapshot{
		PtyID:  snapshot.Record.ID,
		Offset: snapshot.Offset + uint64(len(snapshot.OutputBytes)),
		Output: string(snapshot.OutputBytes),
	}, nil
}

func parseDirection(value string) (session.SplitDirection, error) {
	switch value {
	case "", "horizontal":
		return session.SplitHorizontal, nil
	case "vertical":
		return session.SplitVertical, nil
	default:
		return "", fmt.Errorf("unknown split direction %q", value)
	}
}
