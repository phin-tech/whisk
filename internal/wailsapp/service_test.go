package wailsapp_test

import (
	"context"
	"testing"

	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/wailsapp"
)

func TestServiceDelegatesToRuntimeClient(t *testing.T) {
	fake := &runtimeClientFake{
		sessions: []session.Session{{ID: "sess_01"}},
		created: protocol.CreatedSession{
			Session:   session.Session{ID: "sess_02"},
			MainPtyID: "pty_01",
		},
		split:  protocol.SplitPaneResult{Session: session.Session{ID: "sess_02"}, PaneID: "pane_02", PtyID: "pty_02"},
		output: protocol.OutputSnapshot{PtyID: "pty_01", Offset: 12, Output: "hello"},
		ptys:   []protocol.PTYInfo{{ID: "pty_01", SessionID: "sess_01", PaneID: "pane_01"}},
		event:  protocol.RuntimeEvent{Type: "pty.changed", PtyID: "pty_01"},
		worktrunk: protocol.WorktrunkStatus{
			Available:   true,
			ConfigFound: true,
			Binary:      protocol.WorktrunkBinary{Path: "/bin/wt", Version: "0.44.0"},
		},
		worktrees: []protocol.Worktree{{Branch: "feature", Path: "/repo/.worktrees/feature"}},
		createdWorktree: protocol.CreatedWorktree{
			Path: "/repo/.worktrees/created",
		},
		httpForwards: []protocol.HTTPForward{{ID: "fwd_01", TargetURL: "http://127.0.0.1:4966"}},
	}
	service := wailsapp.NewService(fake)
	ctx := context.Background()

	sessions, err := service.ListSessions(ctx)
	if err != nil || sessions[0].ID != "sess_01" {
		t.Fatalf("list sessions = %#v, %v", sessions, err)
	}
	created, err := service.CreateSession(ctx, protocol.CreateSessionRequest{Name: "created"})
	if err != nil || created.MainPtyID != "pty_01" || fake.createReq.Name != "created" {
		t.Fatalf("create = %#v, req = %#v, err = %v", created, fake.createReq, err)
	}
	split, err := service.SplitPane(ctx, protocol.SplitPaneRequest{SessionID: "sess_02"})
	if err != nil || split.PaneID != "pane_02" || fake.splitReq.SessionID != "sess_02" {
		t.Fatalf("split = %#v, req = %#v, err = %v", split, fake.splitReq, err)
	}
	if err := service.WritePTY(ctx, protocol.WritePTYRequest{PtyID: "pty_01", Data: "x"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if fake.writeReq.Data != "x" {
		t.Fatalf("write req = %#v", fake.writeReq)
	}
	if err := service.ResizePTY(ctx, protocol.ResizePTYRequest{PtyID: "pty_01", Cols: 80, Rows: 24}); err != nil {
		t.Fatalf("resize: %v", err)
	}
	if fake.resizeReq.Cols != 80 || fake.resizeReq.Rows != 24 {
		t.Fatalf("resize req = %#v", fake.resizeReq)
	}
	output, err := service.Output(ctx, protocol.OutputRequest{PtyID: "pty_01", FromOffset: 7})
	if err != nil || output.Offset != 12 || fake.outputReq.FromOffset != 7 {
		t.Fatalf("output = %#v, req = %#v, err = %v", output, fake.outputReq, err)
	}
	ptys, err := service.ListPTYs(ctx)
	if err != nil || ptys[0].ID != "pty_01" {
		t.Fatalf("ptys = %#v, err = %v", ptys, err)
	}
	event, err := service.NextEvent(ctx, protocol.NextEventRequest{TimeoutMs: 25})
	if err != nil || event.Type != "pty.changed" || fake.nextEventReq.TimeoutMs != 25 {
		t.Fatalf("event = %#v, req = %#v, err = %v", event, fake.nextEventReq, err)
	}

	worktrunk, err := service.DetectWorktrunk(ctx, protocol.DetectWorktrunkRequest{RepoPath: "/repo"})
	if err != nil || !worktrunk.Available || fake.detectWorktrunkReq.RepoPath != "/repo" {
		t.Fatalf("detect worktrunk = %#v, req = %#v, err = %v", worktrunk, fake.detectWorktrunkReq, err)
	}
	worktrees, err := service.ListWorktrees(ctx, protocol.ListWorktreesRequest{RepoPath: "/repo"})
	if err != nil || len(worktrees) != 1 || worktrees[0].Branch != "feature" || fake.listWorktreesReq.RepoPath != "/repo" {
		t.Fatalf("list worktrees = %#v, req = %#v, err = %v", worktrees, fake.listWorktreesReq, err)
	}
	createdWorktree, err := service.CreateWorktree(ctx, protocol.CreateWorktreeRequest{
		RepoPath: "/repo",
		Branch:   "created",
		Base:     "main",
	})
	if err != nil || createdWorktree.Path != "/repo/.worktrees/created" || fake.createWorktreeReq.Base != "main" {
		t.Fatalf("create worktree = %#v, req = %#v, err = %v", createdWorktree, fake.createWorktreeReq, err)
	}
	if err := service.RemoveWorktree(ctx, protocol.RemoveWorktreeRequest{RepoPath: "/repo", WorktreePath: "/repo/.worktrees/created"}); err != nil {
		t.Fatalf("remove worktree: %v", err)
	}
	if fake.removeWorktreeReq.WorktreePath != "/repo/.worktrees/created" || fake.removeWorktreeReq.AlsoBranch {
		t.Fatalf("remove worktree req = %#v", fake.removeWorktreeReq)
	}
	httpForwards, err := service.ListHTTPForwards(ctx)
	if err != nil || len(httpForwards) != 1 || httpForwards[0].ID != "fwd_01" {
		t.Fatalf("list http forwards = %#v, err = %v", httpForwards, err)
	}
	if _, err := service.StartHTTPForward(ctx, protocol.StartHTTPForwardRequest{TargetURL: "http://127.0.0.1:4966"}); err == nil {
		t.Fatalf("expected start error without HTTP client")
	}
	if err := service.StopHTTPForward(ctx, "fwd_01"); err == nil {
		t.Fatalf("expected stop error without HTTP client")
	}
}

type runtimeClientFake struct {
	sessions        []session.Session
	created         protocol.CreatedSession
	split           protocol.SplitPaneResult
	output          protocol.OutputSnapshot
	ptys            []protocol.PTYInfo
	event           protocol.RuntimeEvent
	worktrunk       protocol.WorktrunkStatus
	worktrees       []protocol.Worktree
	createdWorktree protocol.CreatedWorktree
	httpForwards    []protocol.HTTPForward

	createReq    protocol.CreateSessionRequest
	splitReq     protocol.SplitPaneRequest
	writeReq     protocol.WritePTYRequest
	resizeReq    protocol.ResizePTYRequest
	outputReq    protocol.OutputRequest
	nextEventReq protocol.NextEventRequest

	detectWorktrunkReq protocol.DetectWorktrunkRequest
	listWorktreesReq   protocol.ListWorktreesRequest
	createWorktreeReq  protocol.CreateWorktreeRequest
	removeWorktreeReq  protocol.RemoveWorktreeRequest
	createForwardReq   protocol.CreateHTTPForwardRequest
	deleteForwardID    string
}

func (f *runtimeClientFake) ListSessions(context.Context) ([]session.Session, error) {
	return f.sessions, nil
}

func (f *runtimeClientFake) CreateSession(_ context.Context, req protocol.CreateSessionRequest) (protocol.CreatedSession, error) {
	f.createReq = req
	return f.created, nil
}

func (f *runtimeClientFake) SplitPane(_ context.Context, req protocol.SplitPaneRequest) (protocol.SplitPaneResult, error) {
	f.splitReq = req
	return f.split, nil
}

func (f *runtimeClientFake) WritePTY(_ context.Context, req protocol.WritePTYRequest) error {
	f.writeReq = req
	return nil
}

func (f *runtimeClientFake) ResizePTY(_ context.Context, req protocol.ResizePTYRequest) error {
	f.resizeReq = req
	return nil
}

func (f *runtimeClientFake) Output(_ context.Context, req protocol.OutputRequest) (protocol.OutputSnapshot, error) {
	f.outputReq = req
	return f.output, nil
}

func (f *runtimeClientFake) ListPTYs(context.Context) ([]protocol.PTYInfo, error) {
	return f.ptys, nil
}

func (f *runtimeClientFake) NextEvent(_ context.Context, req protocol.NextEventRequest) (protocol.RuntimeEvent, error) {
	f.nextEventReq = req
	return f.event, nil
}

func (f *runtimeClientFake) DetectWorktrunk(_ context.Context, req protocol.DetectWorktrunkRequest) (protocol.WorktrunkStatus, error) {
	f.detectWorktrunkReq = req
	return f.worktrunk, nil
}

func (f *runtimeClientFake) ListWorktrees(_ context.Context, req protocol.ListWorktreesRequest) ([]protocol.Worktree, error) {
	f.listWorktreesReq = req
	return f.worktrees, nil
}

func (f *runtimeClientFake) CreateWorktree(_ context.Context, req protocol.CreateWorktreeRequest) (protocol.CreatedWorktree, error) {
	f.createWorktreeReq = req
	return f.createdWorktree, nil
}

func (f *runtimeClientFake) RemoveWorktree(_ context.Context, req protocol.RemoveWorktreeRequest) error {
	f.removeWorktreeReq = req
	return nil
}

func (f *runtimeClientFake) CreateHTTPForward(_ context.Context, req protocol.CreateHTTPForwardRequest) (protocol.HTTPForward, error) {
	f.createForwardReq = req
	return protocol.HTTPForward{ID: "fwd_02", TargetURL: req.TargetURL}, nil
}

func (f *runtimeClientFake) ListHTTPForwards(context.Context) ([]protocol.HTTPForward, error) {
	return f.httpForwards, nil
}

func (f *runtimeClientFake) DeleteHTTPForward(_ context.Context, id string) error {
	f.deleteForwardID = id
	return nil
}
