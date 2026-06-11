package app_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/session"
)

func TestRuntimeCreateSessionAttachAndReplayOutput(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: native.NewBackend(),
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		WorkingDir: ".",
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	sessions, err := runtime.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != created.Session.ID {
		t.Fatalf("sessions = %#v", sessions)
	}

	attach, err := runtime.AttachPTY(ctx, app.AttachPTYRequest{
		PtyID:            created.MainPtyID,
		ReplayFromOffset: 0,
	})
	if err != nil {
		t.Fatalf("attach pty: %v", err)
	}
	defer attach.Close()

	if err := runtime.WritePTY(ctx, created.MainPtyID, []byte("printf 'hello-whisk\\n'\n")); err != nil {
		t.Fatalf("write pty: %v", err)
	}

	var output strings.Builder
	for output.Len() == 0 || !strings.Contains(output.String(), "hello-whisk") {
		select {
		case event := <-attach.Events:
			if event.Kind == app.PTYOutput {
				output.Write(event.Bytes)
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for output; got %q", output.String())
		}
	}

	secondAttach, err := runtime.AttachPTY(ctx, app.AttachPTYRequest{
		PtyID:            created.MainPtyID,
		ReplayFromOffset: 0,
	})
	if err != nil {
		t.Fatalf("reattach pty: %v", err)
	}
	defer secondAttach.Close()

	if !strings.Contains(string(secondAttach.ReplayBytes), "hello-whisk") {
		t.Fatalf("replay bytes missing output: %q", string(secondAttach.ReplayBytes))
	}

	snapshot, err := runtime.PTYOutput(ctx, created.MainPtyID, 0)
	if err != nil {
		t.Fatalf("pty output: %v", err)
	}
	if !strings.Contains(string(snapshot.OutputBytes), "hello-whisk") {
		t.Fatalf("snapshot missing output: %q", string(snapshot.OutputBytes))
	}
}

func TestRuntimeSplitPaneCreatesNewPTY(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: native.NewBackend(),
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		WorkingDir: ".",
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	split, err := runtime.SplitPane(ctx, app.SplitPaneRequest{
		SessionID:    created.Session.ID,
		TargetPaneID: created.Session.FocusedPaneID,
		Direction:    session.SplitHorizontal,
		Cols:         80,
		Rows:         24,
	})
	if err != nil {
		t.Fatalf("split pane: %v", err)
	}
	if split.PaneID == "" || split.PtyID == "" {
		t.Fatalf("split missing ids: %#v", split)
	}
	if split.Session.FocusedPaneID != split.PaneID {
		t.Fatalf("focused pane = %q, split pane = %q", split.Session.FocusedPaneID, split.PaneID)
	}
}

func TestRuntimeListPTYsIncludesSessionOwnership(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: native.NewBackend(),
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		WorkingDir: ".",
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	split, err := runtime.SplitPane(ctx, app.SplitPaneRequest{
		SessionID:    created.Session.ID,
		TargetPaneID: created.Session.FocusedPaneID,
		Direction:    session.SplitVertical,
		Cols:         100,
		Rows:         30,
	})
	if err != nil {
		t.Fatalf("split pane: %v", err)
	}

	ptys, err := runtime.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys: %v", err)
	}

	byID := map[string]app.PTYInfo{}
	for _, pty := range ptys {
		byID[pty.ID] = pty
	}
	if byID[created.MainPtyID].SessionID != created.Session.ID || byID[created.MainPtyID].PaneID != created.Session.FocusedPaneID {
		t.Fatalf("main pty info = %#v", byID[created.MainPtyID])
	}
	if byID[split.PtyID].SessionID != created.Session.ID || byID[split.PtyID].PaneID != split.PaneID {
		t.Fatalf("split pty info = %#v", byID[split.PtyID])
	}
	if !byID[split.PtyID].Running || byID[split.PtyID].Cols != 100 || byID[split.PtyID].Rows != 30 {
		t.Fatalf("split pty process info = %#v", byID[split.PtyID])
	}
}

func TestRuntimePublishesSessionPTYAndOutputEvents(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	sink := newRecordingEventSink()
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: native.NewBackend(),
		EventSink:  sink,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		WorkingDir: ".",
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	sink.waitFor(t, ctx, app.EventSessionChanged, "")
	sink.waitFor(t, ctx, app.EventPTYChanged, created.MainPtyID)

	if err := runtime.WritePTY(ctx, created.MainPtyID, []byte("printf 'event-output-ok\\n'\n")); err != nil {
		t.Fatalf("write pty: %v", err)
	}
	event := sink.waitFor(t, ctx, app.EventPTYOutput, created.MainPtyID)
	if event.Offset == 0 {
		t.Fatalf("output event missing offset: %#v", event)
	}
}

func TestRuntimeRejectsInvalidRequests(t *testing.T) {
	ctx := context.Background()
	runtime := app.NewRuntime(app.RuntimeConfig{})
	if _, err := runtime.CreateSession(ctx, app.CreateSessionRequest{}); err == nil {
		t.Fatalf("expected missing backend create error")
	}
	if _, err := runtime.SplitPane(ctx, app.SplitPaneRequest{}); err == nil {
		t.Fatalf("expected missing backend split error")
	}
	if err := runtime.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown without backend: %v", err)
	}

	runtime = app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	if err := runtime.ResizePTY(ctx, "missing", app.PTYSize{Cols: 0, Rows: 1}); err == nil {
		t.Fatalf("expected invalid cols error")
	}
	if err := runtime.ResizePTY(ctx, "missing", app.PTYSize{Cols: 1, Rows: 0}); err == nil {
		t.Fatalf("expected invalid rows error")
	}
	if _, err := runtime.SplitPane(ctx, app.SplitPaneRequest{SessionID: "missing"}); err == nil {
		t.Fatalf("expected missing session split error")
	}
}

func TestRuntimeResizePTYChangesShellGrid(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: native.NewBackend(),
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		WorkingDir: ".",
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := runtime.ResizePTY(ctx, created.MainPtyID, app.PTYSize{Cols: 73, Rows: 17}); err != nil {
		t.Fatalf("resize pty: %v", err)
	}

	attach, err := runtime.AttachPTY(ctx, app.AttachPTYRequest{
		PtyID:            created.MainPtyID,
		ReplayFromOffset: 0,
	})
	if err != nil {
		t.Fatalf("attach pty: %v", err)
	}
	defer attach.Close()

	if err := runtime.WritePTY(ctx, created.MainPtyID, []byte("stty size\n")); err != nil {
		t.Fatalf("write pty: %v", err)
	}

	var output strings.Builder
	for !strings.Contains(output.String(), "17 73") {
		select {
		case event := <-attach.Events:
			if event.Kind == app.PTYOutput {
				output.Write(event.Bytes)
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for resized grid; got %q", output.String())
		}
	}
}

type recordingEventSink struct {
	mu     sync.Mutex
	events []app.RuntimeEvent
	ch     chan app.RuntimeEvent
}

func newRecordingEventSink() *recordingEventSink {
	return &recordingEventSink{ch: make(chan app.RuntimeEvent, 64)}
}

func (s *recordingEventSink) Publish(_ context.Context, event app.RuntimeEvent) error {
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
	s.ch <- event
	return nil
}

func (s *recordingEventSink) waitFor(t *testing.T, ctx context.Context, eventType app.RuntimeEventType, ptyID string) app.RuntimeEvent {
	t.Helper()
	for {
		select {
		case event := <-s.ch:
			if event.Type == eventType && (ptyID == "" || event.PtyID == ptyID) {
				return event
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s %s", eventType, ptyID)
		}
	}
}
