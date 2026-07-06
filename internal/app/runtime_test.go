package app_test

import (
	"context"
	"encoding/json"
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
	"github.com/phin-tech/whisk/internal/domain/terminal"
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

func TestRuntimeWritePTYTracksRecentInput(t *testing.T) {
	ptyBackend := newMemoryPTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: ptyBackend})

	ctx := context.Background()
	if err := runtime.WritePTY(ctx, "missing", []byte("x")); err == nil {
		t.Fatalf("expected missing pty write error")
	}
	if runtime.PTYInputRecent("missing", time.Hour) {
		t.Fatalf("failed pty write should not be recent")
	}

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if runtime.PTYInputRecent(created.MainPtyID, time.Hour) {
		t.Fatalf("pty input should not be recent before write")
	}
	if err := runtime.WritePTY(ctx, created.MainPtyID, []byte("x")); err != nil {
		t.Fatalf("write pty: %v", err)
	}
	if !runtime.PTYInputRecent(created.MainPtyID, time.Hour) {
		t.Fatalf("pty input should be recent after write")
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

	sessionEvent := sink.waitFor(t, ctx, app.EventSessionChanged, "")
	ptyEvent := sink.waitFor(t, ctx, app.EventPTYChanged, created.MainPtyID)
	if sessionEvent.Seq == 0 || ptyEvent.Seq <= sessionEvent.Seq {
		t.Fatalf("events not sequenced: session=%#v pty=%#v", sessionEvent, ptyEvent)
	}

	if err := runtime.WritePTY(ctx, created.MainPtyID, []byte("printf 'event-output-ok\\n'\n")); err != nil {
		t.Fatalf("write pty: %v", err)
	}
	event := sink.waitFor(t, ctx, app.EventPTYOutput, created.MainPtyID)
	if event.Offset == 0 {
		t.Fatalf("output event missing offset: %#v", event)
	}
	if event.Seq <= ptyEvent.Seq {
		t.Fatalf("output event not sequenced after pty event: %#v", event)
	}
}

func TestRuntimeTeesPTYOutputAndExitToTerminalHistory(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ptyBackend := newAttachableMemoryPTYBackend()
	history := newMemoryTerminalHistoryStore()
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend:           ptyBackend,
		TerminalHistoryStore: history,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "History",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 90, Rows: 30},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	registered := history.registeredPTYs()
	if len(registered) != 1 ||
		registered[0].PTYID != created.MainPtyID ||
		registered[0].SessionID != created.Session.ID ||
		registered[0].WindowID != created.WindowID ||
		registered[0].PaneID != created.PaneID ||
		registered[0].OriginWindowID != created.WindowID ||
		registered[0].OriginPaneID != created.PaneID ||
		registered[0].Cols != 90 ||
		registered[0].Rows != 30 ||
		registered[0].Status != app.TerminalHistoryStatusRunning {
		t.Fatalf("registered history = %#v", registered)
	}

	ptyBackend.output(created.MainPtyID, 0, []byte("restore"))
	first := history.waitForOutput(t, ctx)
	if first.PTYID != created.MainPtyID || first.Offset != 0 || string(first.Bytes) != "restore" {
		t.Fatalf("first output = %#v", first)
	}
	ptyBackend.output(created.MainPtyID, 7, []byte("-bytes"))
	second := history.waitForOutput(t, ctx)
	if second.PTYID != created.MainPtyID || second.Offset != 7 || string(second.Bytes) != "-bytes" {
		t.Fatalf("second output = %#v", second)
	}

	ptyBackend.exit(created.MainPtyID, 42)
	exited := history.waitForExit(t, ctx)
	if exited.PTYID != created.MainPtyID || exited.Code == nil || *exited.Code != 42 {
		t.Fatalf("exit = %#v", exited)
	}
}

func TestRuntimeTracksTerminalSnapshotFromPTYOutput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ptyBackend := newAttachableMemoryPTYBackend()
	history := newMemoryTerminalHistoryStore()
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend:           ptyBackend,
		TerminalHistoryStore: history,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Snapshot",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 40, Rows: 10},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	ptyBackend.output(created.MainPtyID, 0, []byte("hello snapshot\r\n"))
	history.waitForOutput(t, ctx)

	snapshot, err := runtime.TerminalSnapshot(ctx, created.MainPtyID)
	if err != nil {
		t.Fatalf("terminal snapshot: %v", err)
	}
	if snapshot.Offset != uint64(len("hello snapshot\r\n")) ||
		snapshot.Cols != 40 ||
		snapshot.Rows != 10 ||
		!strings.Contains(snapshot.ViewportAnsi, "hello snapshot") {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestRuntimeListPTYsIncludesTerminalMetadataAndAdvisoryStatus(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ptyBackend := newAttachableMemoryPTYBackend()
	sink := newRecordingEventSink()
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: ptyBackend,
		EventSink:  sink,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Status",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	sink.waitFor(t, ctx, app.EventSessionChanged, "")
	sink.waitFor(t, ctx, app.EventPTYChanged, created.MainPtyID)

	title := "Codex waiting for approval"
	cwd := "file://localhost/tmp/whisk-status"
	data := []byte("\x1b]0;" + title + "\x07\x1b]7;" + cwd + "\x07")
	ptyBackend.output(created.MainPtyID, 0, data)
	sink.waitFor(t, ctx, app.EventPTYChanged, created.MainPtyID)

	ptys, err := runtime.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys: %v", err)
	}
	if len(ptys) != 1 {
		t.Fatalf("ptys = %#v", ptys)
	}
	pty := ptys[0]
	if pty.Title != title || pty.TerminalWorkingDirectory != cwd {
		t.Fatalf("terminal metadata = title %q cwd %q", pty.Title, pty.TerminalWorkingDirectory)
	}
	if pty.AgentStatus == nil {
		t.Fatalf("missing advisory agent status: %#v", pty)
	}
	if pty.AgentStatus.Agent != "codex" ||
		pty.AgentStatus.Label != "Codex" ||
		pty.AgentStatus.State != "waiting" ||
		pty.AgentStatus.Source != "osc-title" ||
		pty.AgentStatus.Confidence != "fallback" ||
		!pty.AgentStatus.Advisory {
		t.Fatalf("agent status = %#v", pty.AgentStatus)
	}
}

func TestRuntimeResizesTrackedTerminalSnapshot(t *testing.T) {
	ctx := context.Background()
	ptyBackend := newAttachableMemoryPTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: ptyBackend,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Snapshot resize",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 40, Rows: 10},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := runtime.ResizePTY(ctx, created.MainPtyID, app.PTYSize{Cols: 100, Rows: 32}); err != nil {
		t.Fatalf("resize pty: %v", err)
	}

	snapshot, err := runtime.TerminalSnapshot(ctx, created.MainPtyID)
	if err != nil {
		t.Fatalf("terminal snapshot: %v", err)
	}
	if snapshot.Cols != 100 || snapshot.Rows != 32 {
		t.Fatalf("snapshot size = %dx%d", snapshot.Cols, snapshot.Rows)
	}
}

func TestRuntimeWritesTerminalCheckpointWhenHistoryRequestsIt(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ptyBackend := newAttachableMemoryPTYBackend()
	history := newMemoryTerminalHistoryStore()
	history.setAppendResult(app.TerminalHistoryAppendResult{
		AppendedOffset:   0,
		AppendedBytes:    len("checkpoint me"),
		LogPayloadBytes:  5 * 1024 * 1024,
		CheckpointNeeded: true,
	})
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend:           ptyBackend,
		TerminalHistoryStore: history,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Checkpoint",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	ptyBackend.output(created.MainPtyID, 0, []byte("checkpoint me"))
	checkpoint := history.waitForCheckpoint(t, ctx)

	if checkpoint.PTYID != created.MainPtyID ||
		checkpoint.Offset != uint64(len("checkpoint me")) ||
		checkpoint.Cols != 80 ||
		checkpoint.Rows != 24 ||
		len(checkpoint.TerminalSnapshot) == 0 {
		t.Fatalf("checkpoint = %#v", checkpoint)
	}
	var snapshot terminal.Snapshot
	if err := json.Unmarshal(checkpoint.TerminalSnapshot, &snapshot); err != nil {
		t.Fatalf("unmarshal terminal snapshot: %v", err)
	}
	if snapshot.Offset != checkpoint.Offset || !strings.Contains(snapshot.ViewportAnsi, "checkpoint me") {
		t.Fatalf("checkpoint snapshot = %#v", snapshot)
	}
}

func TestRuntimePrunesTerminalHistoryOnPTYDeleteAndDaemonClear(t *testing.T) {
	ctx := context.Background()
	ptyBackend := newAttachableMemoryPTYBackend()
	history := newMemoryTerminalHistoryStore()
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend:           ptyBackend,
		TerminalHistoryStore: history,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Delete history",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := runtime.KillPTY(ctx, app.KillPTYRequest{PTYID: created.MainPtyID}); err != nil {
		t.Fatalf("kill pty: %v", err)
	}
	if err := runtime.DeletePTY(ctx, app.DeletePTYRequest{PTYID: created.MainPtyID}); err != nil {
		t.Fatalf("delete pty: %v", err)
	}
	if got := history.deletedPTYs(); len(got) != 1 || got[0] != created.MainPtyID {
		t.Fatalf("deleted histories = %#v", got)
	}

	if _, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Clear history",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	}); err != nil {
		t.Fatalf("create session before clear: %v", err)
	}
	if _, err := runtime.ClearDaemon(ctx); err != nil {
		t.Fatalf("clear daemon: %v", err)
	}
	if !history.cleared() {
		t.Fatalf("terminal history was not cleared")
	}
}

func TestRuntimeNextEventRequiresEventSource(t *testing.T) {
	ctx := context.Background()
	runtime := app.NewRuntime(app.RuntimeConfig{})
	if _, err := runtime.NextEvent(ctx, 0); err == nil {
		t.Fatalf("expected missing event source error")
	}

	sink := newRecordingEventSink()
	runtime = app.NewRuntime(app.RuntimeConfig{EventSink: sink})
	want := app.RuntimeEvent{Type: app.EventWorkItemsChanged}
	if err := sink.Publish(ctx, want); err != nil {
		t.Fatalf("publish: %v", err)
	}
	got, err := runtime.NextEvent(ctx, 7)
	if err != nil {
		t.Fatalf("next event: %v", err)
	}
	if got.Event.Type != want.Type || sink.afterSeq != 7 {
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
	mu       sync.Mutex
	events   []app.RuntimeEvent
	ch       chan app.RuntimeEvent
	afterSeq uint64
}

type memoryTerminalHistoryStore struct {
	mu           sync.Mutex
	registered   []app.TerminalHistoryPTYMeta
	outputs      []app.TerminalHistoryOutput
	exits        []app.TerminalHistoryExit
	checkpoints  []app.TerminalHistoryCheckpoint
	deleted      []string
	clear        bool
	appendResult app.TerminalHistoryAppendResult
	outputCh     chan app.TerminalHistoryOutput
	exitCh       chan app.TerminalHistoryExit
	checkpointCh chan app.TerminalHistoryCheckpoint
}

func newMemoryTerminalHistoryStore() *memoryTerminalHistoryStore {
	return &memoryTerminalHistoryStore{
		outputCh:     make(chan app.TerminalHistoryOutput, 8),
		exitCh:       make(chan app.TerminalHistoryExit, 8),
		checkpointCh: make(chan app.TerminalHistoryCheckpoint, 8),
	}
}

func (s *memoryTerminalHistoryStore) RegisterPTY(_ context.Context, meta app.TerminalHistoryPTYMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta.ExitCode = cloneTestIntPtr(meta.ExitCode)
	s.registered = append(s.registered, meta)
	return nil
}

func (s *memoryTerminalHistoryStore) AppendPTYOutput(_ context.Context, event app.TerminalHistoryOutput) (app.TerminalHistoryAppendResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event.Bytes = append([]byte(nil), event.Bytes...)
	s.outputs = append(s.outputs, event)
	s.outputCh <- event
	if s.appendResult.CheckpointNeeded || s.appendResult.AppendedBytes != 0 || s.appendResult.LogPayloadBytes != 0 {
		return s.appendResult, nil
	}
	return app.TerminalHistoryAppendResult{
		AppendedOffset:  event.Offset,
		AppendedBytes:   len(event.Bytes),
		LogPayloadBytes: uint64(len(event.Bytes)),
	}, nil
}

func (s *memoryTerminalHistoryStore) WriteCheckpoint(_ context.Context, checkpoint app.TerminalHistoryCheckpoint) (app.TerminalHistoryCheckpoint, error) {
	checkpoint.TerminalSnapshot = append(checkpoint.TerminalSnapshot[:0:0], checkpoint.TerminalSnapshot...)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkpoints = append(s.checkpoints, checkpoint)
	s.checkpointCh <- checkpoint
	return checkpoint, nil
}

func (s *memoryTerminalHistoryStore) ReadCheckpoint(context.Context, string) (app.TerminalHistoryCheckpoint, error) {
	return app.TerminalHistoryCheckpoint{}, nil
}

func (s *memoryTerminalHistoryStore) MarkPTYExit(_ context.Context, event app.TerminalHistoryExit) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	event.Code = cloneTestIntPtr(event.Code)
	s.exits = append(s.exits, event)
	s.exitCh <- event
	return nil
}

func (s *memoryTerminalHistoryStore) ListRestorable(context.Context) ([]app.TerminalHistoryRestoredPTY, error) {
	return nil, nil
}

func (s *memoryTerminalHistoryStore) DeletePTY(_ context.Context, ptyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, ptyID)
	return nil
}

func (s *memoryTerminalHistoryStore) Clear(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clear = true
	return nil
}

func (s *memoryTerminalHistoryStore) registeredPTYs() []app.TerminalHistoryPTYMeta {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]app.TerminalHistoryPTYMeta, len(s.registered))
	copy(out, s.registered)
	return out
}

func (s *memoryTerminalHistoryStore) setAppendResult(result app.TerminalHistoryAppendResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appendResult = result
}

func (s *memoryTerminalHistoryStore) deletedPTYs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.deleted...)
}

func (s *memoryTerminalHistoryStore) cleared() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clear
}

func (s *memoryTerminalHistoryStore) waitForOutput(t *testing.T, ctx context.Context) app.TerminalHistoryOutput {
	t.Helper()
	select {
	case output := <-s.outputCh:
		return output
	case <-ctx.Done():
		t.Fatalf("timed out waiting for terminal history output")
		return app.TerminalHistoryOutput{}
	}
}

func (s *memoryTerminalHistoryStore) waitForExit(t *testing.T, ctx context.Context) app.TerminalHistoryExit {
	t.Helper()
	select {
	case exit := <-s.exitCh:
		return exit
	case <-ctx.Done():
		t.Fatalf("timed out waiting for terminal history exit")
		return app.TerminalHistoryExit{}
	}
}

func (s *memoryTerminalHistoryStore) waitForCheckpoint(t *testing.T, ctx context.Context) app.TerminalHistoryCheckpoint {
	t.Helper()
	select {
	case checkpoint := <-s.checkpointCh:
		return checkpoint
	case <-ctx.Done():
		t.Fatalf("timed out waiting for terminal history checkpoint")
		return app.TerminalHistoryCheckpoint{}
	}
}

func cloneTestIntPtr(in *int) *int {
	if in == nil {
		return nil
	}
	out := *in
	return &out
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

func (s *recordingEventSink) Next(ctx context.Context, afterSeq uint64) (app.NextRuntimeEventResult, error) {
	s.afterSeq = afterSeq
	select {
	case event := <-s.ch:
		return app.NextRuntimeEventResult{Event: event}, nil
	case <-ctx.Done():
		return app.NextRuntimeEventResult{}, ctx.Err()
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
