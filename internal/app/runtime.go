package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/phin-tech/whisk/internal/domain/session"
)

type PTYEventKind string

const (
	PTYOutput PTYEventKind = "output"
	PTYExit   PTYEventKind = "exit"
)

type PTYRecord struct {
	ID         string `json:"id"`
	WorkingDir string `json:"workingDir"`
	Cols       int    `json:"cols"`
	Rows       int    `json:"rows"`
	Running    bool   `json:"running"`
}

type PTYEvent struct {
	Kind   PTYEventKind
	Offset uint64
	Bytes  []byte
	Code   *int
}

type PTYAttach struct {
	Record       PTYRecord
	ReplayBytes  []byte
	ReplayOffset uint64
	Events       <-chan PTYEvent
	CloseFunc    func()
}

func (a *PTYAttach) Close() {
	if a.CloseFunc != nil {
		a.CloseFunc()
	}
}

type PTYBackend interface {
	Spawn(ctx context.Context, req SpawnPTYRequest) (PTYRecord, error)
	Write(ctx context.Context, ptyID string, data []byte) error
	Resize(ctx context.Context, ptyID string, size PTYSize) error
	Attach(ctx context.Context, req AttachPTYRequest) (*PTYAttach, error)
	Output(ctx context.Context, ptyID string, fromOffset uint64) (PTYOutputSnapshot, error)
	Shutdown(ctx context.Context) error
}

type PTYOutputSnapshot struct {
	Record      PTYRecord `json:"record"`
	Offset      uint64    `json:"offset"`
	OutputBytes []byte    `json:"outputBytes"`
}

type SpawnPTYRequest struct {
	ID         string
	WorkingDir string
	Cols       int
	Rows       int
}

type PTYSize struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

type AttachPTYRequest struct {
	PtyID            string
	ReplayFromOffset uint64
}

type RuntimeConfig struct {
	PTYBackend  PTYBackend
	IDGenerator func() string
}

type Runtime struct {
	mu     sync.Mutex
	ids    func() string
	ptys   PTYBackend
	state  *session.State
	nextID int
}

type CreateSessionRequest struct {
	Name       string
	WorkingDir string
	Cols       int
	Rows       int
}

type CreatedSession struct {
	Session   session.Session
	MainPtyID string
}

type SplitPaneRequest struct {
	SessionID    string
	TargetPaneID string
	Direction    session.SplitDirection
	Cols         int
	Rows         int
}

type SplitPaneResult struct {
	Session session.Session
	PaneID  string
	PtyID   string
}

func NewRuntime(config RuntimeConfig) *Runtime {
	ids := config.IDGenerator
	r := &Runtime{
		ids:   ids,
		ptys:  config.PTYBackend,
		state: session.NewState(),
	}
	if r.ids == nil {
		r.ids = r.generatedID
	}
	return r
}

func (r *Runtime) CreateSession(ctx context.Context, req CreateSessionRequest) (CreatedSession, error) {
	if r.ptys == nil {
		return CreatedSession{}, fmt.Errorf("pty backend required")
	}
	sessionID := r.ids()
	paneID := r.ids()
	ptyID := r.ids()
	record, err := r.ptys.Spawn(ctx, SpawnPTYRequest{
		ID:         ptyID,
		WorkingDir: req.WorkingDir,
		Cols:       req.Cols,
		Rows:       req.Rows,
	})
	if err != nil {
		return CreatedSession{}, err
	}
	created, err := r.state.CreateSession(session.CreateSession{
		SessionID:  sessionID,
		PaneID:     paneID,
		PtyID:      record.ID,
		Name:       req.Name,
		WorkingDir: req.WorkingDir,
	})
	if err != nil {
		return CreatedSession{}, err
	}
	return CreatedSession{Session: created, MainPtyID: record.ID}, nil
}

func (r *Runtime) ListSessions(_ context.Context) ([]session.Session, error) {
	return r.state.List(), nil
}

func (r *Runtime) SplitPane(ctx context.Context, req SplitPaneRequest) (SplitPaneResult, error) {
	if r.ptys == nil {
		return SplitPaneResult{}, fmt.Errorf("pty backend required")
	}
	current, ok := r.state.Get(req.SessionID)
	if !ok {
		return SplitPaneResult{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	target, ok := current.Panes[req.TargetPaneID]
	if !ok {
		return SplitPaneResult{}, fmt.Errorf("pane %s not found", req.TargetPaneID)
	}
	newPaneID := r.ids()
	newPtyID := r.ids()
	record, err := r.ptys.Spawn(ctx, SpawnPTYRequest{
		ID:         newPtyID,
		WorkingDir: current.WorkingDir,
		Cols:       req.Cols,
		Rows:       req.Rows,
	})
	if err != nil {
		return SplitPaneResult{}, err
	}
	_ = target
	updated, err := r.state.SplitPane(session.SplitPane{
		SessionID:    req.SessionID,
		TargetPaneID: req.TargetPaneID,
		NewPaneID:    newPaneID,
		NewPtyID:     record.ID,
		Direction:    req.Direction,
	})
	if err != nil {
		return SplitPaneResult{}, err
	}
	return SplitPaneResult{Session: updated, PaneID: newPaneID, PtyID: record.ID}, nil
}

func (r *Runtime) WritePTY(ctx context.Context, ptyID string, data []byte) error {
	return r.ptys.Write(ctx, ptyID, data)
}

func (r *Runtime) ResizePTY(ctx context.Context, ptyID string, size PTYSize) error {
	if size.Cols <= 0 {
		return fmt.Errorf("pty cols must be positive")
	}
	if size.Rows <= 0 {
		return fmt.Errorf("pty rows must be positive")
	}
	return r.ptys.Resize(ctx, ptyID, size)
}

func (r *Runtime) AttachPTY(ctx context.Context, req AttachPTYRequest) (*PTYAttach, error) {
	return r.ptys.Attach(ctx, req)
}

func (r *Runtime) PTYOutput(ctx context.Context, ptyID string, fromOffset uint64) (PTYOutputSnapshot, error) {
	return r.ptys.Output(ctx, ptyID, fromOffset)
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r.ptys == nil {
		return nil
	}
	return r.ptys.Shutdown(ctx)
}

func (r *Runtime) generatedID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	return fmt.Sprintf("whisk_%06d", r.nextID)
}
