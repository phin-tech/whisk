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
}

type runtimeClientFake struct {
	sessions []session.Session
	created  protocol.CreatedSession
	split    protocol.SplitPaneResult
	output   protocol.OutputSnapshot

	createReq protocol.CreateSessionRequest
	splitReq  protocol.SplitPaneRequest
	writeReq  protocol.WritePTYRequest
	resizeReq protocol.ResizePTYRequest
	outputReq protocol.OutputRequest
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
