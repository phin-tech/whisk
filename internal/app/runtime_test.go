package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
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

	created := createRuntimeSession(t, runtime, ctx, 80, 24)
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

func TestRuntimeWritePTYNormalizesTrailingLineFeed(t *testing.T) {
	ptyBackend := newMemoryPTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: ptyBackend})

	ctx := context.Background()
	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := runtime.WritePTY(ctx, created.MainPtyID, []byte("one\ntwo\n")); err != nil {
		t.Fatalf("write pty: %v", err)
	}
	if err := runtime.WritePTY(ctx, created.MainPtyID, []byte("three\r\n")); err != nil {
		t.Fatalf("write pty crlf: %v", err)
	}

	writes := ptyBackend.writes[created.MainPtyID]
	if len(writes) != 2 || string(writes[0]) != "one\ntwo\r" || string(writes[1]) != "three\r" {
		t.Fatalf("writes = %#v", writes)
	}
}

func TestRuntimeCreateSessionRunsInitialCommand(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: native.NewBackend(),
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rootDir := t.TempDir()
	markerPath := filepath.Join(rootDir, "initial-command-marker")
	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:    "Whisk",
		RootDir: rootDir,
		InitialPTY: &app.StartPTYOptions{
			Command: "printf initial-command-ok > " + strconv.Quote(markerPath),
		},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	attach, err := runtime.AttachPTY(ctx, app.AttachPTYRequest{
		PtyID:            created.MainPtyID,
		ReplayFromOffset: 0,
	})
	if err != nil {
		t.Fatalf("attach pty: %v", err)
	}
	defer attach.Close()

	var output strings.Builder
	output.Write(attach.ReplayBytes)
	for !strings.Contains(output.String(), "initial-command-ok") {
		select {
		case event := <-attach.Events:
			if event.Kind == app.PTYOutput {
				output.Write(event.Bytes)
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for initial command; got %q", output.String())
		}
	}

	for {
		bytes, err := os.ReadFile(markerPath)
		if err == nil && string(bytes) == "initial-command-ok" {
			return
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-ctx.Done():
			t.Fatalf("timed out waiting for initial command marker; content=%q err=%v", string(bytes), err)
		}
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

	created := createRuntimeSession(t, runtime, ctx, 80, 24)

	split, err := runtime.SplitPane(ctx, app.SplitPaneRequest{
		SessionID:    created.Session.ID,
		WindowID:     created.WindowID,
		TargetPaneID: created.PaneID,
		Direction:    session.SplitHorizontal,
		InitialPTY:   &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("split pane: %v", err)
	}
	if split.PaneID == "" || split.PtyID == "" {
		t.Fatalf("split missing ids: %#v", split)
	}
	if split.PTYID == nil || *split.PTYID != split.PtyID {
		t.Fatalf("split pty mismatch: %#v", split)
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

	created := createRuntimeSession(t, runtime, ctx, 80, 24)
	split, err := runtime.SplitPane(ctx, app.SplitPaneRequest{
		SessionID:    created.Session.ID,
		WindowID:     created.WindowID,
		TargetPaneID: created.PaneID,
		Direction:    session.SplitVertical,
		InitialPTY:   &app.StartPTYOptions{Cols: 100, Rows: 30},
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
	if byID[created.MainPtyID].SessionID != created.Session.ID || byID[created.MainPtyID].WindowID != created.WindowID || byID[created.MainPtyID].PaneID != created.PaneID {
		t.Fatalf("main pty info = %#v", byID[created.MainPtyID])
	}
	if byID[split.PtyID].SessionID != created.Session.ID || byID[split.PtyID].WindowID != created.WindowID || byID[split.PtyID].PaneID != split.PaneID {
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

	created := createRuntimeSession(t, runtime, ctx, 80, 24)

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

func TestRuntimeNextEventRequiresEventSource(t *testing.T) {
	ctx := context.Background()
	runtime := app.NewRuntime(app.RuntimeConfig{})
	if _, err := runtime.NextEvent(ctx); err == nil {
		t.Fatalf("expected missing event source error")
	}

	sink := newRecordingEventSink()
	runtime = app.NewRuntime(app.RuntimeConfig{EventSink: sink})
	want := app.RuntimeEvent{Type: app.EventWorkItemsChanged}
	if err := sink.Publish(ctx, want); err != nil {
		t.Fatalf("publish: %v", err)
	}
	got, err := runtime.NextEvent(ctx)
	if err != nil {
		t.Fatalf("next event: %v", err)
	}
	if got.Type != want.Type {
		t.Fatalf("event = %#v", got)
	}
}

func TestRuntimeRejectsInvalidRequests(t *testing.T) {
	ctx := context.Background()
	runtime := app.NewRuntime(app.RuntimeConfig{})
	if _, err := runtime.CreateSession(ctx, app.CreateSessionRequest{}); err == nil {
		t.Fatalf("expected missing root create error")
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

	created := createRuntimeSession(t, runtime, ctx, 80, 24)
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

func createRuntimeSession(t *testing.T, runtime *app.Runtime, ctx context.Context, cols int, rows int) app.CreatedSession {
	t.Helper()
	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:    "Whisk",
		RootDir: t.TempDir(),
		InitialPTY: &app.StartPTYOptions{
			Cols: cols,
			Rows: rows,
		},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.PTYID == nil || created.MainPtyID == "" || *created.PTYID != created.MainPtyID {
		t.Fatalf("created pty mismatch: %#v", created)
	}
	return created
}

func (s *recordingEventSink) Publish(_ context.Context, event app.RuntimeEvent) error {
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
	s.ch <- event
	return nil
}

func (s *recordingEventSink) Next(ctx context.Context) (app.RuntimeEvent, error) {
	select {
	case event := <-s.ch:
		return event, nil
	case <-ctx.Done():
		return app.RuntimeEvent{}, ctx.Err()
	}
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
