package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/agenthooklog"
	"github.com/phin-tech/whisk/internal/adapters/agenthooks"
	"github.com/phin-tech/whisk/internal/domain/agentbridge"
	"github.com/phin-tech/whisk/internal/domain/httpforward"
	"github.com/phin-tech/whisk/internal/domain/ptybookmark"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/domain/workitem"
)

type PTYEventKind string

const (
	PTYOutput PTYEventKind = "output"
	PTYExit   PTYEventKind = "exit"
)

type PTYStatus string

const (
	PTYStatusStarting PTYStatus = "starting"
	PTYStatusRunning  PTYStatus = "running"
	PTYStatusExited   PTYStatus = "exited"
	PTYStatusKilled   PTYStatus = "killed"
	PTYStatusFailed   PTYStatus = "failed"
	PTYStatusLost     PTYStatus = "lost"
)

type PTYRecord struct {
	ID         string `json:"id"`
	WorkingDir string `json:"workingDir"`
	Cols       int    `json:"cols"`
	Rows       int    `json:"rows"`
	Running    bool   `json:"running"`
}

type PTYInfo struct {
	ID             string    `json:"id"`
	WorkingDir     string    `json:"workingDir"`
	Cols           int       `json:"cols"`
	Rows           int       `json:"rows"`
	Running        bool      `json:"running"`
	Status         PTYStatus `json:"status"`
	ExitCode       *int      `json:"exitCode,omitempty"`
	SessionID      string    `json:"sessionId"`
	WindowID       string    `json:"windowId"`
	PaneID         string    `json:"paneId"`
	OriginWindowID string    `json:"originWindowId"`
	OriginPaneID   string    `json:"originPaneId"`
}

type ClearDaemonResult struct {
	SessionsCleared  int
	PTYsCleared      int
	BookmarksCleared int
	ProjectsCleared  int
	WorkItemsCleared int
	ForwardsCleared  int
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
	Kill(ctx context.Context, ptyID string) (PTYRecord, error)
	Attach(ctx context.Context, req AttachPTYRequest) (*PTYAttach, error)
	Output(ctx context.Context, ptyID string, fromOffset uint64) (PTYOutputSnapshot, error)
	List(ctx context.Context) ([]PTYRecord, error)
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
	Env        map[string]string
	Command    string
	Args       []string
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
	PTYBackend                 PTYBackend
	Worktrees                  WorktreeBackend
	IDGenerator                func() string
	EventSink                  EventSink
	SessionStore               SessionStore
	TranscriptStore            TranscriptStore
	BookmarkStore              BookmarkStore
	WorkItemStore              WorkItemStore
	DaemonURL                  string
	CLIPath                    string
	AgentHookPaths             *agenthooks.Paths
	AgentHookLogPaths          *agenthooklog.Paths
	AgentBridgeApprovalTimeout time.Duration
}

type Runtime struct {
	mu                         sync.Mutex
	ids                        func() string
	ptys                       PTYBackend
	worktrees                  WorktreeBackend
	state                      *session.State
	bookmarks                  *ptybookmark.State
	sessionStore               SessionStore
	transcriptStore            TranscriptStore
	bookmarkStore              BookmarkStore
	workItemStore              WorkItemStore
	daemonURL                  string
	cliPath                    string
	agentHookPaths             *agenthooks.Paths
	agentHookLogPaths          *agenthooklog.Paths
	agentHookLogEnabled        bool
	clearHookLogAfterSession   bool
	agentHookEvents            []agentbridge.Event
	ptyMeta                    map[string]ptyMetadata
	forwards                   *httpforward.State
	workItems                  *workitem.State
	agentBridges               agentbridge.State
	agentBridgeApprovalWaiters map[string]chan agentbridge.EvaluationDecision
	agentBridgeApprovalTimeout time.Duration
	nextID                     int
	eventSink                  EventSink
	watchCtx                   context.Context
	watchCancel                context.CancelFunc
}

type ptyMetadata struct {
	SessionID      string
	WindowID       string
	PaneID         string
	OriginWindowID string
	OriginPaneID   string
	Status         PTYStatus
	ExitCode       *int
}

type RuntimeEventType string

const (
	EventSessionChanged              RuntimeEventType = "session.changed"
	EventPTYChanged                  RuntimeEventType = "pty.changed"
	EventPTYOutput                   RuntimeEventType = "pty.output"
	EventWorkItemsChanged            RuntimeEventType = "workitems.changed"
	EventStatusChanged               RuntimeEventType = "status.changed"
	EventAgentBridgeApprovalsChanged RuntimeEventType = "agent_bridge_approvals.changed"
	EventAgentHookEventsChanged      RuntimeEventType = "agent_hook_events.changed"
)

type RuntimeEvent struct {
	Type   RuntimeEventType `json:"type"`
	PtyID  string           `json:"ptyId,omitempty"`
	Offset uint64           `json:"offset,omitempty"`
}

type EventSink interface {
	Publish(ctx context.Context, event RuntimeEvent) error
}

type EventSource interface {
	Next(ctx context.Context) (RuntimeEvent, error)
}

type SessionStore interface {
	LoadSessions(ctx context.Context) ([]session.Session, error)
	SaveSessions(ctx context.Context, sessions []session.Session) error
}

type TranscriptStore interface {
	RegisterPTY(ctx context.Context, meta PTYTranscriptMeta) error
	AppendPTYOutput(ctx context.Context, event PTYTranscriptOutput) error
	MarkPTYExit(ctx context.Context, event PTYTranscriptExit) error
}

type BookmarkStore interface {
	LoadBookmarks(ctx context.Context) ([]ptybookmark.Bookmark, error)
	SaveBookmarks(ctx context.Context, bookmarks []ptybookmark.Bookmark) error
}

type WorkItemStore interface {
	LoadWorkItems(ctx context.Context) (workitem.Snapshot, error)
	SaveWorkItems(ctx context.Context, snapshot workitem.Snapshot) error
}

type PTYTranscriptMeta struct {
	PTYID      string
	SessionID  string
	WindowID   string
	PaneID     string
	WorkingDir string
	Cols       int
	Rows       int
}

type PTYTranscriptOutput struct {
	PTYID  string
	Offset uint64
	Bytes  []byte
}

type PTYTranscriptExit struct {
	PTYID string
	Code  *int
}

type CreateSessionRequest struct {
	Name       string
	RootDir    string
	InitialPTY *StartPTYOptions
}

type CreatedSession struct {
	Session   session.Session
	WindowID  string
	PaneID    string
	PTYID     *string
	MainPtyID string
}

type SplitPaneRequest struct {
	SessionID    string
	WindowID     string
	TargetPaneID string
	Direction    session.SplitDirection
	InitialPTY   *StartPTYOptions
}

type StartPTYOptions struct {
	Cols        int
	Rows        int
	Command     string
	Env         map[string]string
	Args        []string
	Exec        bool
	AgentBridge *StartPTYAgentBridgeOptions
}

type StartPTYAgentBridgeOptions struct {
	Enabled  bool
	Provider string
}

type SplitPaneResult struct {
	Session session.Session
	PaneID  string
	PTYID   *string
	PtyID   string
}

type SetSessionRootDirRequest struct {
	SessionID string
	RootDir   string
}

type SetPaneWorkingDirRequest struct {
	SessionID  string
	PaneID     string
	WorkingDir string
}

type StartPanePTYRequest struct {
	SessionID string
	PaneID    string
	Options   StartPTYOptions
}

type StartedPanePTY struct {
	Session session.Session
	PTYID   string
}

type DetachedPanePTY struct {
	Session session.Session
	PTYID   string
}

type ClosePaneRequest struct {
	SessionID string
	WindowID  string
	PaneID    string
}

type CloseSessionRequest struct {
	SessionID string
}

type DetachPanePTYRequest struct {
	SessionID string
	PaneID    string
}

type RestartPanePTYRequest struct {
	SessionID string
	PaneID    string
	Options   StartPTYOptions
}

type RestartedPanePTY struct {
	Session  session.Session
	PTYID    string
	OldPTYID string
}

type KillPTYRequest struct {
	PTYID string
}

type AddPTYBookmarkRequest struct {
	PTYID  string
	Offset uint64
	Kind   string
	Label  string
}

type RemovePTYBookmarkRequest struct {
	BookmarkID string
}

func NewRuntime(config RuntimeConfig) *Runtime {
	runtime, err := NewRuntimeWithError(config)
	if err != nil {
		panic(err)
	}
	return runtime
}

func NewRuntimeWithError(config RuntimeConfig) (*Runtime, error) {
	ids := config.IDGenerator
	watchCtx, watchCancel := context.WithCancel(context.Background())
	state := session.NewState()
	nextID := 0
	if config.SessionStore != nil {
		sessions, err := config.SessionStore.LoadSessions(context.Background())
		if err != nil {
			watchCancel()
			return nil, err
		}
		nextID = maxWhiskIDFromSessions(sessions)
		state, err = session.NewStateFromSessions(clearRestoredCurrentPTYs(sessions))
		if err != nil {
			watchCancel()
			return nil, err
		}
	}
	bookmarks := ptybookmark.NewState()
	if config.BookmarkStore != nil {
		loaded, err := config.BookmarkStore.LoadBookmarks(context.Background())
		if err != nil {
			watchCancel()
			return nil, err
		}
		bookmarks, err = ptybookmark.NewStateFromBookmarks(loaded)
		if err != nil {
			watchCancel()
			return nil, err
		}
	}
	workItems := workitem.NewState()
	if config.WorkItemStore != nil {
		snapshot, err := config.WorkItemStore.LoadWorkItems(context.Background())
		if err != nil {
			watchCancel()
			return nil, err
		}
		workItems, err = workitem.NewStateFromSnapshot(snapshot)
		if err != nil {
			watchCancel()
			return nil, err
		}
	}
	agentBridges, err := agentbridge.NewState()
	if err != nil {
		watchCancel()
		return nil, err
	}
	approvalTimeout := config.AgentBridgeApprovalTimeout
	if approvalTimeout <= 0 {
		approvalTimeout = 25 * time.Second
	}
	r := &Runtime{
		ids:                        ids,
		ptys:                       config.PTYBackend,
		worktrees:                  config.Worktrees,
		state:                      state,
		bookmarks:                  bookmarks,
		sessionStore:               config.SessionStore,
		transcriptStore:            config.TranscriptStore,
		bookmarkStore:              config.BookmarkStore,
		workItemStore:              config.WorkItemStore,
		daemonURL:                  config.DaemonURL,
		cliPath:                    config.CLIPath,
		agentHookPaths:             config.AgentHookPaths,
		agentHookLogPaths:          config.AgentHookLogPaths,
		agentHookLogEnabled:        true,
		ptyMeta:                    map[string]ptyMetadata{},
		forwards:                   httpforward.NewState(),
		workItems:                  workItems,
		agentBridges:               agentBridges,
		agentBridgeApprovalWaiters: map[string]chan agentbridge.EvaluationDecision{},
		agentBridgeApprovalTimeout: approvalTimeout,
		eventSink:                  config.EventSink,
		watchCtx:                   watchCtx,
		watchCancel:                watchCancel,
		nextID:                     nextID,
	}
	if r.ids == nil {
		r.ids = r.generatedID
	}
	if err := r.reconcileLoadedWorkItemRuns(context.Background()); err != nil {
		watchCancel()
		return nil, err
	}
	return r, nil
}

func (r *Runtime) CreateSession(ctx context.Context, req CreateSessionRequest) (CreatedSession, error) {
	rootDir, err := validateExistingRootDir(req.RootDir)
	if err != nil {
		return CreatedSession{}, err
	}
	sessionID := r.ids()
	windowID := r.ids()
	paneID := r.ids()
	var ptyID *string
	var mainPtyID string
	if req.InitialPTY != nil {
		if r.ptys == nil {
			return CreatedSession{}, fmt.Errorf("pty backend required")
		}
		nextPTYID := r.ids()
		bridgeLaunch, err := r.preparePTYAgentBridge(req.InitialPTY, nextPTYID, rootDir)
		if err != nil {
			return CreatedSession{}, err
		}
		record, err := r.ptys.Spawn(ctx, SpawnPTYRequest{
			ID:         nextPTYID,
			WorkingDir: rootDir,
			Cols:       req.InitialPTY.Cols,
			Rows:       req.InitialPTY.Rows,
			Env:        r.ptyContextEnv(sessionID, nextPTYID, mergedAgentBridgeEnv(req.InitialPTY.Env, bridgeLaunch)),
			Command:    directPTYCommand(req.InitialPTY),
			Args:       directPTYArgs(req.InitialPTY),
		})
		if err != nil {
			return CreatedSession{}, err
		}
		if err := r.registerPTYTranscript(ctx, record, sessionID, windowID, paneID); err != nil {
			_, _ = r.ptys.Kill(ctx, record.ID)
			return CreatedSession{}, err
		}
		ptyID = &record.ID
		mainPtyID = record.ID
		r.registerPTY(record.ID, ptyMetadata{
			SessionID:      sessionID,
			WindowID:       windowID,
			PaneID:         paneID,
			OriginWindowID: windowID,
			OriginPaneID:   paneID,
			Status:         PTYStatusRunning,
		})
		r.watchPTYOutput(record.ID)
		if bridgeLaunch != nil {
			bridge := bridgeLaunch.Bridge
			bridge.SessionID = sessionID
			bridge.PTYID = record.ID
			if err := r.registerAgentBridge(bridge); err != nil {
				_, _ = r.ptys.Kill(ctx, record.ID)
				return CreatedSession{}, err
			}
		}
		if !req.InitialPTY.Exec {
			if err := r.writeInitialCommand(ctx, record.ID, req.InitialPTY.Command); err != nil {
				_, _ = r.ptys.Kill(ctx, record.ID)
				return CreatedSession{}, err
			}
		}
	}
	created, err := r.state.CreateSession(session.CreateSession{
		SessionID:    sessionID,
		WindowID:     windowID,
		PaneID:       paneID,
		InitialPTYID: ptyID,
		Name:         req.Name,
		RootDir:      rootDir,
	})
	if err != nil {
		return CreatedSession{}, err
	}
	if err := r.persistSessions(ctx); err != nil {
		return CreatedSession{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	if mainPtyID != "" {
		r.publish(ctx, RuntimeEvent{Type: EventPTYChanged, PtyID: mainPtyID})
	}
	return CreatedSession{Session: created, WindowID: windowID, PaneID: paneID, PTYID: ptyID, MainPtyID: mainPtyID}, nil
}

func directPTYCommand(options *StartPTYOptions) string {
	if options == nil || !options.Exec {
		return ""
	}
	return options.Command
}

func directPTYArgs(options *StartPTYOptions) []string {
	if options == nil || !options.Exec || len(options.Args) == 0 {
		return nil
	}
	return append([]string(nil), options.Args...)
}

func (r *Runtime) preparePTYAgentBridge(options *StartPTYOptions, ownerID string, workingDir string) (*agentBridgeLaunch, error) {
	if options == nil || options.AgentBridge == nil || !options.AgentBridge.Enabled {
		return nil, nil
	}
	if strings.TrimSpace(r.daemonURL) == "" {
		return nil, fmt.Errorf("daemon URL required for agent bridge terminal")
	}
	provider := strings.TrimSpace(options.AgentBridge.Provider)
	if provider == "" {
		provider = string(agentbridge.ProviderClaude)
	}
	bridgeProvider, ok := agentBridgeProviderFromString(provider)
	if !ok {
		return nil, fmt.Errorf("unsupported agent bridge provider %q", options.AgentBridge.Provider)
	}
	return prepareAgentBridgeLaunchForProvider(r, bridgeProvider, ownerID, workingDir)
}

func mergedAgentBridgeEnv(base map[string]string, launch *agentBridgeLaunch) map[string]string {
	if launch == nil {
		return base
	}
	merged := map[string]string{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range launch.Env {
		merged[key] = value
	}
	return merged
}

func (r *Runtime) ListSessions(_ context.Context) ([]session.Session, error) {
	return r.state.List(), nil
}

func (r *Runtime) SplitPane(ctx context.Context, req SplitPaneRequest) (SplitPaneResult, error) {
	current, ok := r.state.Get(req.SessionID)
	if !ok {
		return SplitPaneResult{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	if _, ok := current.Windows[req.WindowID]; !ok {
		return SplitPaneResult{}, fmt.Errorf("window %s not found", req.WindowID)
	}
	target, ok := current.Panes[req.TargetPaneID]
	if !ok || target.WindowID != req.WindowID {
		return SplitPaneResult{}, fmt.Errorf("pane %s not found", req.TargetPaneID)
	}
	newPaneID := r.ids()
	var ptyID *string
	var ptyIDValue string
	if req.InitialPTY != nil {
		if r.ptys == nil {
			return SplitPaneResult{}, fmt.Errorf("pty backend required")
		}
		newPtyID := r.ids()
		bridgeLaunch, err := r.preparePTYAgentBridge(req.InitialPTY, newPtyID, target.WorkingDir)
		if err != nil {
			return SplitPaneResult{}, err
		}
		record, err := r.ptys.Spawn(ctx, SpawnPTYRequest{
			ID:         newPtyID,
			WorkingDir: target.WorkingDir,
			Cols:       req.InitialPTY.Cols,
			Rows:       req.InitialPTY.Rows,
			Env:        r.ptyContextEnv(req.SessionID, newPtyID, mergedAgentBridgeEnv(req.InitialPTY.Env, bridgeLaunch)),
			Command:    directPTYCommand(req.InitialPTY),
			Args:       directPTYArgs(req.InitialPTY),
		})
		if err != nil {
			return SplitPaneResult{}, err
		}
		if err := r.registerPTYTranscript(ctx, record, req.SessionID, req.WindowID, newPaneID); err != nil {
			_, _ = r.ptys.Kill(ctx, record.ID)
			return SplitPaneResult{}, err
		}
		ptyID = &record.ID
		ptyIDValue = record.ID
		r.registerPTY(record.ID, ptyMetadata{
			SessionID:      req.SessionID,
			WindowID:       req.WindowID,
			PaneID:         newPaneID,
			OriginWindowID: req.WindowID,
			OriginPaneID:   newPaneID,
			Status:         PTYStatusRunning,
		})
		r.watchPTYOutput(record.ID)
		if bridgeLaunch != nil {
			bridge := bridgeLaunch.Bridge
			bridge.SessionID = req.SessionID
			bridge.PTYID = record.ID
			if err := r.registerAgentBridge(bridge); err != nil {
				_, _ = r.ptys.Kill(ctx, record.ID)
				return SplitPaneResult{}, err
			}
		}
		if !req.InitialPTY.Exec {
			if err := r.writeInitialCommand(ctx, record.ID, req.InitialPTY.Command); err != nil {
				_, _ = r.ptys.Kill(ctx, record.ID)
				return SplitPaneResult{}, err
			}
		}
	}
	updated, err := r.state.SplitPane(session.SplitPane{
		SessionID:    req.SessionID,
		WindowID:     req.WindowID,
		TargetPaneID: req.TargetPaneID,
		NewPaneID:    newPaneID,
		NewPTYID:     ptyID,
		Direction:    req.Direction,
	})
	if err != nil {
		return SplitPaneResult{}, err
	}
	if err := r.persistSessions(ctx); err != nil {
		return SplitPaneResult{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	if ptyIDValue != "" {
		r.publish(ctx, RuntimeEvent{Type: EventPTYChanged, PtyID: ptyIDValue})
	}
	return SplitPaneResult{Session: updated, PaneID: newPaneID, PTYID: ptyID, PtyID: ptyIDValue}, nil
}

func (r *Runtime) SetSessionRootDir(ctx context.Context, req SetSessionRootDirRequest) (session.Session, error) {
	rootDir, err := validateExistingRootDir(req.RootDir)
	if err != nil {
		return session.Session{}, err
	}
	current, ok := r.state.Get(req.SessionID)
	if !ok {
		return session.Session{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	running, err := r.sessionHasRunningCurrentPTY(ctx, current)
	if err != nil {
		return session.Session{}, err
	}
	if running {
		return session.Session{}, fmt.Errorf("cannot change root dir while session has running ptys")
	}
	updated, err := r.state.SetSessionRootDir(session.SetSessionRootDir{
		SessionID: req.SessionID,
		RootDir:   rootDir,
	})
	if err != nil {
		return session.Session{}, err
	}
	if err := r.persistSessions(ctx); err != nil {
		return session.Session{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	return updated, nil
}

func (r *Runtime) SetPaneWorkingDir(ctx context.Context, req SetPaneWorkingDirRequest) (session.Session, error) {
	workingDir, err := validateExistingRootDir(req.WorkingDir)
	if err != nil {
		return session.Session{}, err
	}
	current, ok := r.state.Get(req.SessionID)
	if !ok {
		return session.Session{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	pane, ok := current.Panes[req.PaneID]
	if !ok {
		return session.Session{}, fmt.Errorf("pane %s not found", req.PaneID)
	}
	running, err := r.ptyIDIsRunning(ctx, pane.CurrentPTYID)
	if err != nil {
		return session.Session{}, err
	}
	if running {
		return session.Session{}, fmt.Errorf("cannot change pane working dir while pty is running")
	}
	updated, err := r.state.SetPaneWorkingDir(session.SetPaneWorkingDir{
		SessionID:  req.SessionID,
		PaneID:     req.PaneID,
		WorkingDir: workingDir,
	})
	if err != nil {
		return session.Session{}, err
	}
	if err := r.persistSessions(ctx); err != nil {
		return session.Session{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	return updated, nil
}

func (r *Runtime) StartPanePTY(ctx context.Context, req StartPanePTYRequest) (StartedPanePTY, error) {
	if r.ptys == nil {
		return StartedPanePTY{}, fmt.Errorf("pty backend required")
	}
	current, ok := r.state.Get(req.SessionID)
	if !ok {
		return StartedPanePTY{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	pane, ok := current.Panes[req.PaneID]
	if !ok {
		return StartedPanePTY{}, fmt.Errorf("pane %s not found", req.PaneID)
	}
	if pane.CurrentPTYID != nil {
		return StartedPanePTY{}, fmt.Errorf("pane %s already has current pty", req.PaneID)
	}
	ptyID := r.ids()
	bridgeLaunch, err := r.preparePTYAgentBridge(&req.Options, ptyID, pane.WorkingDir)
	if err != nil {
		return StartedPanePTY{}, err
	}
	record, err := r.ptys.Spawn(ctx, SpawnPTYRequest{
		ID:         ptyID,
		WorkingDir: pane.WorkingDir,
		Cols:       req.Options.Cols,
		Rows:       req.Options.Rows,
		Env:        r.ptyContextEnv(req.SessionID, ptyID, mergedAgentBridgeEnv(req.Options.Env, bridgeLaunch)),
		Command:    directPTYCommand(&req.Options),
		Args:       directPTYArgs(&req.Options),
	})
	if err != nil {
		return StartedPanePTY{}, err
	}
	if err := r.registerPTYTranscript(ctx, record, req.SessionID, pane.WindowID, req.PaneID); err != nil {
		_, _ = r.ptys.Kill(ctx, record.ID)
		return StartedPanePTY{}, err
	}
	r.registerPTY(record.ID, ptyMetadata{
		SessionID:      req.SessionID,
		WindowID:       pane.WindowID,
		PaneID:         req.PaneID,
		OriginWindowID: pane.WindowID,
		OriginPaneID:   req.PaneID,
		Status:         PTYStatusRunning,
	})
	r.watchPTYOutput(record.ID)
	if bridgeLaunch != nil {
		bridge := bridgeLaunch.Bridge
		bridge.SessionID = req.SessionID
		bridge.PTYID = record.ID
		if err := r.registerAgentBridge(bridge); err != nil {
			_, _ = r.ptys.Kill(ctx, record.ID)
			return StartedPanePTY{}, err
		}
	}
	if !req.Options.Exec {
		if err := r.writeInitialCommand(ctx, record.ID, req.Options.Command); err != nil {
			_, _ = r.ptys.Kill(ctx, record.ID)
			return StartedPanePTY{}, err
		}
	}
	updated, err := r.state.StartPanePTY(session.StartPanePTY{
		SessionID: req.SessionID,
		PaneID:    req.PaneID,
		PTYID:     record.ID,
	})
	if err != nil {
		return StartedPanePTY{}, err
	}
	if err := r.persistSessions(ctx); err != nil {
		return StartedPanePTY{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	r.publish(ctx, RuntimeEvent{Type: EventPTYChanged, PtyID: record.ID})
	return StartedPanePTY{Session: updated, PTYID: record.ID}, nil
}

func (r *Runtime) DetachPanePTY(ctx context.Context, req DetachPanePTYRequest) (DetachedPanePTY, error) {
	current, ok := r.state.Get(req.SessionID)
	if !ok {
		return DetachedPanePTY{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	pane, ok := current.Panes[req.PaneID]
	if !ok {
		return DetachedPanePTY{}, fmt.Errorf("pane %s not found", req.PaneID)
	}
	updated, detached, err := r.state.DetachPanePTY(session.DetachPanePTY{
		SessionID: req.SessionID,
		PaneID:    req.PaneID,
	})
	if err != nil {
		return DetachedPanePTY{}, err
	}
	if err := r.persistSessions(ctx); err != nil {
		return DetachedPanePTY{}, err
	}
	r.detachPTY(detached, req.SessionID, pane.WindowID, req.PaneID)
	r.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	r.publish(ctx, RuntimeEvent{Type: EventPTYChanged, PtyID: detached})
	return DetachedPanePTY{Session: updated, PTYID: detached}, nil
}

func (r *Runtime) RestartPanePTY(ctx context.Context, req RestartPanePTYRequest) (RestartedPanePTY, error) {
	if r.ptys == nil {
		return RestartedPanePTY{}, fmt.Errorf("pty backend required")
	}
	current, ok := r.state.Get(req.SessionID)
	if !ok {
		return RestartedPanePTY{}, fmt.Errorf("session %s not found", req.SessionID)
	}
	pane, ok := current.Panes[req.PaneID]
	if !ok {
		return RestartedPanePTY{}, fmt.Errorf("pane %s not found", req.PaneID)
	}
	if pane.CurrentPTYID == nil {
		return RestartedPanePTY{}, fmt.Errorf("pane %s has no current pty", req.PaneID)
	}
	running, err := r.ptyIDIsRunning(ctx, pane.CurrentPTYID)
	if err != nil {
		return RestartedPanePTY{}, err
	}
	if running {
		return RestartedPanePTY{}, fmt.Errorf("cannot restart pane while current pty is running")
	}
	newPTYID := r.ids()
	bridgeLaunch, err := r.preparePTYAgentBridge(&req.Options, newPTYID, pane.WorkingDir)
	if err != nil {
		return RestartedPanePTY{}, err
	}
	record, err := r.ptys.Spawn(ctx, SpawnPTYRequest{
		ID:         newPTYID,
		WorkingDir: pane.WorkingDir,
		Cols:       req.Options.Cols,
		Rows:       req.Options.Rows,
		Env:        r.ptyContextEnv(req.SessionID, newPTYID, mergedAgentBridgeEnv(req.Options.Env, bridgeLaunch)),
		Command:    directPTYCommand(&req.Options),
		Args:       directPTYArgs(&req.Options),
	})
	if err != nil {
		return RestartedPanePTY{}, err
	}
	if err := r.registerPTYTranscript(ctx, record, req.SessionID, pane.WindowID, req.PaneID); err != nil {
		_, _ = r.ptys.Kill(ctx, record.ID)
		return RestartedPanePTY{}, err
	}
	r.registerPTY(record.ID, ptyMetadata{
		SessionID:      req.SessionID,
		WindowID:       pane.WindowID,
		PaneID:         req.PaneID,
		OriginWindowID: pane.WindowID,
		OriginPaneID:   req.PaneID,
		Status:         PTYStatusRunning,
	})
	r.watchPTYOutput(record.ID)
	if bridgeLaunch != nil {
		bridge := bridgeLaunch.Bridge
		bridge.SessionID = req.SessionID
		bridge.PTYID = record.ID
		if err := r.registerAgentBridge(bridge); err != nil {
			_, _ = r.ptys.Kill(ctx, record.ID)
			return RestartedPanePTY{}, err
		}
	}
	if !req.Options.Exec {
		if err := r.writeInitialCommand(ctx, record.ID, req.Options.Command); err != nil {
			_, _ = r.ptys.Kill(ctx, record.ID)
			return RestartedPanePTY{}, err
		}
	}
	updated, oldPTYID, err := r.state.RestartPanePTY(session.RestartPanePTY{
		SessionID: req.SessionID,
		PaneID:    req.PaneID,
		NewPTYID:  record.ID,
	})
	if err != nil {
		return RestartedPanePTY{}, err
	}
	if err := r.persistSessions(ctx); err != nil {
		return RestartedPanePTY{}, err
	}
	r.detachPTY(oldPTYID, req.SessionID, pane.WindowID, req.PaneID)
	r.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	r.publish(ctx, RuntimeEvent{Type: EventPTYChanged, PtyID: oldPTYID})
	r.publish(ctx, RuntimeEvent{Type: EventPTYChanged, PtyID: record.ID})
	return RestartedPanePTY{Session: updated, PTYID: record.ID, OldPTYID: oldPTYID}, nil
}

func (r *Runtime) KillPTY(ctx context.Context, req KillPTYRequest) (PTYInfo, error) {
	if r.ptys == nil {
		return PTYInfo{}, fmt.Errorf("pty backend required")
	}
	record, err := r.ptys.Kill(ctx, req.PTYID)
	if err != nil {
		return PTYInfo{}, err
	}
	r.markPTYKilled(req.PTYID)
	if err := r.cancelRunsLinkedToClosedTerminals(ctx, nil, []string{req.PTYID}); err != nil {
		return PTYInfo{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventPTYChanged, PtyID: req.PTYID})
	return r.ptyInfoForRecord(record), nil
}

func (r *Runtime) AddPTYBookmark(ctx context.Context, req AddPTYBookmarkRequest) (ptybookmark.Bookmark, error) {
	record, err := r.findPTYRecord(ctx, req.PTYID)
	if err != nil {
		return ptybookmark.Bookmark{}, err
	}
	meta := r.ptyMetadataForRecord(record)
	bookmark, err := r.bookmarks.Add(ptybookmark.AddBookmark{
		ID:        r.ids(),
		PTYID:     record.ID,
		SessionID: meta.SessionID,
		WindowID:  meta.WindowID,
		PaneID:    meta.PaneID,
		Offset:    req.Offset,
		Kind:      req.Kind,
		Label:     req.Label,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return ptybookmark.Bookmark{}, err
	}
	if err := r.persistBookmarks(ctx); err != nil {
		return ptybookmark.Bookmark{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventPTYChanged, PtyID: record.ID})
	return bookmark, nil
}

func (r *Runtime) ListPTYBookmarks(_ context.Context, ptyID string) ([]ptybookmark.Bookmark, error) {
	return r.bookmarks.List(ptyID), nil
}

func (r *Runtime) RemovePTYBookmark(ctx context.Context, req RemovePTYBookmarkRequest) error {
	removed, err := r.bookmarks.Remove(req.BookmarkID)
	if err != nil {
		return err
	}
	if err := r.persistBookmarks(ctx); err != nil {
		return err
	}
	r.publish(ctx, RuntimeEvent{Type: EventPTYChanged, PtyID: removed.PTYID})
	return nil
}

func (r *Runtime) CloseSession(ctx context.Context, req CloseSessionRequest) ([]session.Session, error) {
	current, ok := r.state.Get(req.SessionID)
	if !ok {
		return nil, fmt.Errorf("session %s not found", req.SessionID)
	}
	ptyIDs, err := r.killSessionPTYs(ctx, current.ID)
	if err != nil {
		return nil, err
	}
	if _, err := r.state.RemoveSession(session.RemoveSession{SessionID: current.ID}); err != nil {
		return nil, err
	}
	if err := r.persistSessions(ctx); err != nil {
		return nil, err
	}
	if err := r.cancelRunsLinkedToClosedTerminals(ctx, []string{current.ID}, ptyIDs); err != nil {
		return nil, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	for _, ptyID := range ptyIDs {
		r.publish(ctx, RuntimeEvent{Type: EventPTYChanged, PtyID: ptyID})
	}
	return r.state.List(), nil
}

func (r *Runtime) cancelRunsLinkedToClosedTerminals(ctx context.Context, sessionIDs []string, ptyIDs []string) error {
	sessionSet := map[string]struct{}{}
	for _, id := range sessionIDs {
		if id != "" {
			sessionSet[id] = struct{}{}
		}
	}
	ptySet := map[string]struct{}{}
	for _, id := range ptyIDs {
		if id != "" {
			ptySet[id] = struct{}{}
		}
	}
	if len(sessionSet) == 0 && len(ptySet) == 0 {
		return nil
	}
	changed := false
	now := time.Now().UTC()
	for _, run := range r.workItems.ListRuns("") {
		if terminalRunStatus(run.Status) {
			continue
		}
		_, sessionMatch := sessionSet[run.SessionID]
		_, ptyMatch := ptySet[run.PTYID]
		if !sessionMatch && !ptyMatch {
			continue
		}
		if _, err := r.workItems.CancelRun(workitem.CancelRun{
			ID:           run.ID,
			RunHistoryID: r.ids(),
			Actor:        "runtime",
			Now:          now,
		}); err != nil {
			return err
		}
		changed = true
	}
	if !changed {
		return nil
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return nil
}

func (r *Runtime) reconcileLoadedWorkItemRuns(ctx context.Context) error {
	sessionIDs := map[string]struct{}{}
	for _, sess := range r.state.List() {
		sessionIDs[sess.ID] = struct{}{}
	}
	ptyIDs := map[string]struct{}{}
	if r.ptys != nil {
		records, err := r.ptys.List(ctx)
		if err != nil {
			return err
		}
		for _, record := range records {
			if record.Running {
				ptyIDs[record.ID] = struct{}{}
			}
		}
	}
	changed := false
	now := time.Now().UTC()
	for _, run := range r.workItems.ListRuns("") {
		if run.Status != workitem.RunStateRunning && run.Status != workitem.RunStateAwaitingInput {
			continue
		}
		missingSession := false
		if run.SessionID != "" {
			_, ok := sessionIDs[run.SessionID]
			missingSession = !ok
		}
		missingPTY := false
		if run.PTYID != "" {
			_, ok := ptyIDs[run.PTYID]
			missingPTY = !ok
		}
		if !missingSession && !missingPTY {
			continue
		}
		if _, err := r.workItems.CancelRun(workitem.CancelRun{
			ID:           run.ID,
			RunHistoryID: r.ids(),
			Actor:        "runtime",
			Now:          now,
		}); err != nil {
			return err
		}
		changed = true
	}
	if !changed {
		return nil
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return err
	}
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	return nil
}

func (r *Runtime) ClosePane(ctx context.Context, req ClosePaneRequest) (session.Session, error) {
	updated, err := r.state.ClosePane(session.ClosePane{
		SessionID: req.SessionID,
		WindowID:  req.WindowID,
		PaneID:    req.PaneID,
	})
	if err != nil {
		return session.Session{}, err
	}
	if err := r.persistSessions(ctx); err != nil {
		return session.Session{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	return updated, nil
}

func (r *Runtime) ListPTYs(ctx context.Context) ([]PTYInfo, error) {
	if r.ptys == nil {
		return nil, fmt.Errorf("pty backend required")
	}
	records, err := r.ptys.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PTYInfo, 0, len(records))
	for _, record := range records {
		out = append(out, r.ptyInfoForRecord(record))
	}
	return out, nil
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

func (r *Runtime) NextEvent(ctx context.Context) (RuntimeEvent, error) {
	source, ok := r.eventSink.(EventSource)
	if !ok || source == nil {
		return RuntimeEvent{}, fmt.Errorf("runtime event source unavailable")
	}
	return source.Next(ctx)
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r.watchCancel != nil {
		r.watchCancel()
	}
	r.mu.Lock()
	clearHookLog := r.clearHookLogAfterSession
	r.mu.Unlock()
	if clearHookLog {
		_, _ = r.ClearAgentHookLog(ctx)
	}
	if r.ptys == nil {
		return nil
	}
	return r.ptys.Shutdown(ctx)
}

func (r *Runtime) ClearDaemon(ctx context.Context) (ClearDaemonResult, error) {
	r.mu.Lock()
	forwardsCleared := len(r.forwards.List())
	r.forwards = httpforward.NewState()
	r.ptyMeta = map[string]ptyMetadata{}
	r.mu.Unlock()

	result := ClearDaemonResult{
		SessionsCleared:  len(r.state.List()),
		BookmarksCleared: len(r.bookmarks.List("")),
		ProjectsCleared:  len(r.workItems.ListProjects()),
		WorkItemsCleared: len(r.workItems.ListWorkItems("")),
		ForwardsCleared:  forwardsCleared,
	}
	if r.ptys != nil {
		ptys, err := r.ptys.List(ctx)
		if err != nil {
			return ClearDaemonResult{}, err
		}
		result.PTYsCleared = len(ptys)
		if err := r.ptys.Shutdown(ctx); err != nil {
			return ClearDaemonResult{}, err
		}
	}

	r.state = session.NewState()
	r.bookmarks = ptybookmark.NewState()
	r.workItems = workitem.NewState()

	if err := r.persistSessions(ctx); err != nil {
		return ClearDaemonResult{}, err
	}
	if err := r.persistBookmarks(ctx); err != nil {
		return ClearDaemonResult{}, err
	}
	if err := r.persistWorkItems(ctx); err != nil {
		return ClearDaemonResult{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	r.publish(ctx, RuntimeEvent{Type: EventWorkItemsChanged})
	r.publish(ctx, RuntimeEvent{Type: EventPTYChanged})
	return result, nil
}

func (r *Runtime) generatedID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	return fmt.Sprintf("whisk_%06d", r.nextID)
}

func (r *Runtime) publish(ctx context.Context, event RuntimeEvent) {
	if r.eventSink == nil {
		return
	}
	_ = r.eventSink.Publish(ctx, event)
}

func validateExistingRootDir(rootDir string) (string, error) {
	if rootDir == "" {
		return "", fmt.Errorf("root dir required")
	}
	if !filepath.IsAbs(rootDir) {
		return "", fmt.Errorf("root dir must be absolute")
	}
	cleaned := filepath.Clean(rootDir)
	info, err := os.Stat(cleaned)
	if err != nil {
		return "", fmt.Errorf("root dir invalid: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("root dir must be a directory")
	}
	return cleaned, nil
}

func (r *Runtime) persistSessions(ctx context.Context) error {
	if r.sessionStore == nil {
		return nil
	}
	return r.sessionStore.SaveSessions(ctx, r.state.List())
}

func (r *Runtime) persistBookmarks(ctx context.Context) error {
	if r.bookmarkStore == nil {
		return nil
	}
	return r.bookmarkStore.SaveBookmarks(ctx, r.bookmarks.List(""))
}

func (r *Runtime) persistWorkItems(ctx context.Context) error {
	if r.workItemStore == nil {
		return nil
	}
	return r.workItemStore.SaveWorkItems(ctx, r.workItems.Snapshot())
}

func (r *Runtime) registerPTYTranscript(ctx context.Context, record PTYRecord, sessionID string, windowID string, paneID string) error {
	if r.transcriptStore == nil {
		return nil
	}
	return r.transcriptStore.RegisterPTY(ctx, PTYTranscriptMeta{
		PTYID:      record.ID,
		SessionID:  sessionID,
		WindowID:   windowID,
		PaneID:     paneID,
		WorkingDir: record.WorkingDir,
		Cols:       record.Cols,
		Rows:       record.Rows,
	})
}

func (r *Runtime) writeInitialCommand(ctx context.Context, ptyID string, command string) error {
	if command == "" {
		return nil
	}
	return r.ptys.Write(ctx, ptyID, []byte(command+"\n"))
}

func (r *Runtime) ptyContextEnv(sessionID string, ptyID string, extra map[string]string) map[string]string {
	env := map[string]string{}
	for key, value := range extra {
		env[key] = value
	}
	if r.daemonURL != "" {
		env["WHISKD_URL"] = r.daemonURL
	}
	if r.cliPath != "" {
		env["WHISK_CLI"] = r.cliPath
		cliDir := filepath.Dir(r.cliPath)
		if cliDir != "." && cliDir != string(filepath.Separator) {
			pathValue := env["PATH"]
			if pathValue == "" {
				pathValue = os.Getenv("PATH")
			}
			if pathValue == "" {
				env["PATH"] = cliDir
			} else {
				env["PATH"] = cliDir + string(os.PathListSeparator) + pathValue
			}
		}
	}
	if sessionID != "" {
		env["WHISK_SESSION"] = "1"
		env["WHISK_SESSION_ID"] = sessionID
	}
	if ptyID != "" {
		env["WHISK_PTY_ID"] = ptyID
	}
	if len(env) == 0 {
		return nil
	}
	return env
}

func (r *Runtime) appendPTYTranscriptOutput(ctx context.Context, ptyID string, offset uint64, data []byte) {
	if r.transcriptStore == nil || len(data) == 0 {
		return
	}
	_ = r.transcriptStore.AppendPTYOutput(ctx, PTYTranscriptOutput{
		PTYID:  ptyID,
		Offset: offset,
		Bytes:  append([]byte(nil), data...),
	})
}

func (r *Runtime) markPTYTranscriptExit(ctx context.Context, ptyID string, code *int) {
	if r.transcriptStore == nil {
		return
	}
	_ = r.transcriptStore.MarkPTYExit(ctx, PTYTranscriptExit{
		PTYID: ptyID,
		Code:  cloneIntPtr(code),
	})
}

func clearRestoredCurrentPTYs(sessions []session.Session) []session.Session {
	out := make([]session.Session, len(sessions))
	for i, restored := range sessions {
		restored.Panes = clonePanesWithoutCurrentPTY(restored.Panes)
		out[i] = restored
	}
	return out
}

func maxWhiskIDFromSessions(sessions []session.Session) int {
	maxID := 0
	for _, current := range sessions {
		maxID = max(maxID, whiskIDNumber(current.ID))
		for _, window := range current.Windows {
			maxID = max(maxID, whiskIDNumber(window.ID))
		}
		for _, pane := range current.Panes {
			maxID = max(maxID, whiskIDNumber(pane.ID))
			if pane.CurrentPTYID != nil {
				maxID = max(maxID, whiskIDNumber(*pane.CurrentPTYID))
			}
		}
	}
	return maxID
}

func whiskIDNumber(id string) int {
	suffix, ok := strings.CutPrefix(id, "whisk_")
	if !ok {
		return 0
	}
	number, err := strconv.Atoi(suffix)
	if err != nil {
		return 0
	}
	return number
}

func clonePanesWithoutCurrentPTY(panes map[string]session.Pane) map[string]session.Pane {
	out := make(map[string]session.Pane, len(panes))
	for id, pane := range panes {
		pane.CurrentPTYID = nil
		out[id] = pane
	}
	return out
}

func (r *Runtime) sessionHasRunningCurrentPTY(ctx context.Context, current session.Session) (bool, error) {
	for _, pane := range current.Panes {
		running, err := r.ptyIDIsRunning(ctx, pane.CurrentPTYID)
		if err != nil {
			return false, err
		}
		if running {
			return true, nil
		}
	}
	return false, nil
}

func (r *Runtime) ptyIDIsRunning(ctx context.Context, ptyID *string) (bool, error) {
	if ptyID == nil || *ptyID == "" || r.ptys == nil {
		return false, nil
	}
	records, err := r.ptys.List(ctx)
	if err != nil {
		return false, err
	}
	for _, record := range records {
		if record.ID == *ptyID {
			return record.Running, nil
		}
	}
	return false, nil
}

func (r *Runtime) findPTYRecord(ctx context.Context, ptyID string) (PTYRecord, error) {
	if ptyID == "" {
		return PTYRecord{}, fmt.Errorf("pty id required")
	}
	if r.ptys == nil {
		return PTYRecord{}, fmt.Errorf("pty backend required")
	}
	records, err := r.ptys.List(ctx)
	if err != nil {
		return PTYRecord{}, err
	}
	for _, record := range records {
		if record.ID == ptyID {
			return record, nil
		}
	}
	return PTYRecord{}, fmt.Errorf("pty %s not found", ptyID)
}

func (r *Runtime) killSessionPTYs(ctx context.Context, sessionID string) ([]string, error) {
	if r.ptys == nil {
		return nil, nil
	}
	records, err := r.ptys.List(ctx)
	if err != nil {
		return nil, err
	}
	killed := make([]string, 0)
	for _, record := range records {
		meta := r.ptyMetadataForRecord(record)
		if meta.SessionID != sessionID {
			continue
		}
		if record.Running {
			if _, err := r.ptys.Kill(ctx, record.ID); err != nil {
				return nil, err
			}
		}
		r.markPTYKilled(record.ID)
		killed = append(killed, record.ID)
	}
	return killed, nil
}

func (r *Runtime) watchPTYOutput(ptyID string) {
	if r.ptys == nil {
		return
	}
	attach, err := r.ptys.Attach(r.watchCtx, AttachPTYRequest{PtyID: ptyID})
	if err != nil {
		return
	}
	go func() {
		defer attach.Close()
		if len(attach.ReplayBytes) > 0 {
			r.appendPTYTranscriptOutput(r.watchCtx, ptyID, attach.ReplayOffset, attach.ReplayBytes)
		}
		for {
			select {
			case event, ok := <-attach.Events:
				if !ok {
					return
				}
				switch event.Kind {
				case PTYOutput:
					r.appendPTYTranscriptOutput(r.watchCtx, ptyID, event.Offset, event.Bytes)
					r.publish(r.watchCtx, RuntimeEvent{Type: EventPTYOutput, PtyID: ptyID, Offset: event.Offset + uint64(len(event.Bytes))})
				case PTYExit:
					r.markPTYTranscriptExit(r.watchCtx, ptyID, event.Code)
					r.markPTYExited(ptyID, event.Code)
					_ = r.cancelRunsLinkedToClosedTerminals(r.watchCtx, nil, []string{ptyID})
					r.publish(r.watchCtx, RuntimeEvent{Type: EventPTYChanged, PtyID: ptyID})
				}
			case <-r.watchCtx.Done():
				return
			}
		}
	}()
}

func (r *Runtime) registerPTY(ptyID string, meta ptyMetadata) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ptyMeta[ptyID] = meta
}

func (r *Runtime) detachPTY(ptyID string, sessionID string, originWindowID string, originPaneID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	meta := r.ptyMeta[ptyID]
	if meta.SessionID == "" {
		meta.SessionID = sessionID
	}
	if meta.OriginWindowID == "" {
		meta.OriginWindowID = originWindowID
	}
	if meta.OriginPaneID == "" {
		meta.OriginPaneID = originPaneID
	}
	if meta.Status == "" {
		meta.Status = PTYStatusRunning
	}
	meta.WindowID = ""
	meta.PaneID = ""
	r.ptyMeta[ptyID] = meta
}

func (r *Runtime) markPTYExited(ptyID string, code *int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	meta := r.ptyMeta[ptyID]
	if meta.Status == PTYStatusKilled {
		return
	}
	meta.Status = PTYStatusExited
	meta.ExitCode = cloneIntPtr(code)
	r.ptyMeta[ptyID] = meta
}

func (r *Runtime) markPTYKilled(ptyID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	meta := r.ptyMeta[ptyID]
	meta.Status = PTYStatusKilled
	meta.ExitCode = nil
	r.ptyMeta[ptyID] = meta
}

func (r *Runtime) ptyInfoForRecord(record PTYRecord) PTYInfo {
	meta := r.ptyMetadataForRecord(record)
	return PTYInfo{
		ID:             record.ID,
		WorkingDir:     record.WorkingDir,
		Cols:           record.Cols,
		Rows:           record.Rows,
		Running:        record.Running,
		Status:         meta.Status,
		ExitCode:       cloneIntPtr(meta.ExitCode),
		SessionID:      meta.SessionID,
		WindowID:       meta.WindowID,
		PaneID:         meta.PaneID,
		OriginWindowID: meta.OriginWindowID,
		OriginPaneID:   meta.OriginPaneID,
	}
}

func (r *Runtime) ptyMetadataForRecord(record PTYRecord) ptyMetadata {
	r.mu.Lock()
	meta := r.ptyMeta[record.ID]
	r.mu.Unlock()
	if meta.Status == "" {
		if record.Running {
			meta.Status = PTYStatusRunning
		} else {
			meta.Status = PTYStatusExited
		}
	}
	if !record.Running && meta.Status == PTYStatusRunning {
		meta.Status = PTYStatusExited
	}
	return meta
}

func cloneIntPtr(in *int) *int {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
