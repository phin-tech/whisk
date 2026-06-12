package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPClientDrivesDaemonRuntime(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")

	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend(), EventSink: newFakeEventBus()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := daemon.Health(ctx); err != nil {
		t.Fatalf("health: %v", err)
	}
	compatibility, err := daemon.Compatibility(ctx)
	if err != nil {
		t.Fatalf("compatibility: %v", err)
	}
	if compatibility.APIVersion != protocol.DaemonAPIVersion || compatibility.GitSHA == "" {
		t.Fatalf("compatibility = %#v", compatibility)
	}

	created, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.Session.ID == "" || created.WindowID == "" || created.PaneID == "" || created.PTYID == nil || created.MainPtyID == "" {
		t.Fatalf("created session missing ids: %#v", created)
	}

	sessions, err := daemon.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != created.Session.ID {
		t.Fatalf("sessions = %#v", sessions)
	}

	ptys, err := daemon.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys: %v", err)
	}
	if len(ptys) != 1 || ptys[0].ID != created.MainPtyID || ptys[0].SessionID != created.Session.ID {
		t.Fatalf("ptys = %#v", ptys)
	}
	bookmark, err := daemon.AddPTYBookmark(ctx, protocol.AddPTYBookmarkRequest{
		PTYID:  created.MainPtyID,
		Offset: 2,
		Kind:   "prompt",
		Label:  "Prompt",
	})
	if err != nil || bookmark.PTYID != created.MainPtyID || bookmark.Offset != 2 {
		t.Fatalf("add bookmark = %#v, %v", bookmark, err)
	}
	bookmarks, err := daemon.ListPTYBookmarks(ctx, created.MainPtyID)
	if err != nil || len(bookmarks) != 1 || bookmarks[0].ID != bookmark.ID {
		t.Fatalf("list bookmarks = %#v, %v", bookmarks, err)
	}
	if err := daemon.RemovePTYBookmark(ctx, protocol.RemovePTYBookmarkRequest{BookmarkID: bookmark.ID}); err != nil {
		t.Fatalf("remove bookmark: %v", err)
	}
	closeViaClient, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:       "Close via client",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create close via client session: %v", err)
	}
	remaining, err := daemon.CloseSession(ctx, protocol.CloseSessionRequest{SessionID: closeViaClient.Session.ID})
	if err != nil {
		t.Fatalf("close session: %v", err)
	}
	for _, session := range remaining {
		if session.ID == closeViaClient.Session.ID {
			t.Fatalf("closed session still present: %#v", remaining)
		}
	}

	event, err := daemon.NextEvent(ctx, protocol.NextEventRequest{TimeoutMs: 10})
	if err != nil {
		t.Fatalf("next event: %v", err)
	}
	if event.Type != "session.changed" {
		t.Fatalf("event = %#v", event)
	}

	split, err := daemon.SplitPane(ctx, protocol.SplitPaneRequest{
		SessionID:    created.Session.ID,
		WindowID:     created.WindowID,
		TargetPaneID: created.PaneID,
		Direction:    "horizontal",
		InitialPTY:   &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("split pane: %v", err)
	}
	if split.PaneID == "" || split.PtyID == "" || split.PTYID == nil {
		t.Fatalf("split result = %#v", split)
	}

	emptyRoot := t.TempDir()
	empty, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:    "Empty",
		RootDir: emptyRoot,
	})
	if err != nil {
		t.Fatalf("create empty session: %v", err)
	}
	nextRoot := t.TempDir()
	updated, err := daemon.SetSessionRootDir(ctx, protocol.SetSessionRootDirRequest{
		SessionID: empty.Session.ID,
		RootDir:   nextRoot,
	})
	if err != nil || updated.RootDir != nextRoot {
		t.Fatalf("set root = %#v, %v", updated, err)
	}
	paneDir := t.TempDir()
	updated, err = daemon.SetPaneWorkingDir(ctx, protocol.SetPaneWorkingDirRequest{
		SessionID:  empty.Session.ID,
		PaneID:     empty.PaneID,
		WorkingDir: paneDir,
	})
	if err != nil || updated.Panes[empty.PaneID].WorkingDir != paneDir {
		t.Fatalf("set pane dir = %#v, %v", updated.Panes[empty.PaneID], err)
	}
	started, err := daemon.StartPanePTY(ctx, protocol.StartPanePTYRequest{
		SessionID: empty.Session.ID,
		PaneID:    empty.PaneID,
		Options:   protocol.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil || started.PTYID == "" {
		t.Fatalf("start pane pty = %#v, %v", started, err)
	}
	detached, err := daemon.DetachPanePTY(ctx, protocol.DetachPanePTYRequest{
		SessionID: empty.Session.ID,
		PaneID:    empty.PaneID,
	})
	if err != nil || detached.PTYID != started.PTYID {
		t.Fatalf("detach pane pty = %#v, %v", detached, err)
	}
	ptys, err = daemon.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys after detach: %v", err)
	}
	var detachedInfo protocol.PTYInfo
	for _, pty := range ptys {
		if pty.ID == detached.PTYID {
			detachedInfo = pty
		}
	}
	if detachedInfo.SessionID != empty.Session.ID || detachedInfo.PaneID != "" || detachedInfo.OriginPaneID != empty.PaneID || detachedInfo.Status != "running" {
		t.Fatalf("detached pty info = %#v", detachedInfo)
	}
	restartSession, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:       "Restart",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create restart session: %v", err)
	}
	killed, err := daemon.KillPTY(ctx, protocol.KillPTYRequest{PTYID: restartSession.MainPtyID})
	if err != nil || killed.Status != "killed" || killed.Running {
		t.Fatalf("kill pty = %#v, %v", killed, err)
	}
	restarted, err := daemon.RestartPanePTY(ctx, protocol.RestartPanePTYRequest{
		SessionID: restartSession.Session.ID,
		PaneID:    restartSession.PaneID,
		Options:   protocol.StartPTYOptions{Cols: 100, Rows: 40},
	})
	if err != nil || restarted.OldPTYID != restartSession.MainPtyID || restarted.PTYID == "" {
		t.Fatalf("restart pane pty = %#v, %v", restarted, err)
	}
	restartedPane := restarted.Session.Panes[restartSession.PaneID]
	if restartedPane.CurrentPTYID == nil || *restartedPane.CurrentPTYID != restarted.PTYID {
		t.Fatalf("restarted pane = %#v", restartedPane)
	}
	closeSession, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:    "Close",
		RootDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create close session: %v", err)
	}
	emptySplit, err := daemon.SplitPane(ctx, protocol.SplitPaneRequest{
		SessionID:    closeSession.Session.ID,
		WindowID:     closeSession.WindowID,
		TargetPaneID: closeSession.PaneID,
		Direction:    "horizontal",
	})
	if err != nil {
		t.Fatalf("split empty pane: %v", err)
	}
	closed, err := daemon.ClosePane(ctx, protocol.ClosePaneRequest{
		SessionID: closeSession.Session.ID,
		WindowID:  closeSession.WindowID,
		PaneID:    emptySplit.PaneID,
	})
	if err != nil {
		t.Fatalf("close pane: %v", err)
	}
	if _, ok := closed.Panes[emptySplit.PaneID]; ok {
		t.Fatalf("closed pane still present: %#v", closed.Panes)
	}

	if err := daemon.ResizePTY(ctx, protocol.ResizePTYRequest{PtyID: created.MainPtyID, Cols: 73, Rows: 17}); err != nil {
		t.Fatalf("resize pty: %v", err)
	}
	if err := daemon.WritePTY(ctx, protocol.WritePTYRequest{PtyID: created.MainPtyID, Data: "printf 'daemon-http-ok\\n'\n"}); err != nil {
		t.Fatalf("write pty: %v", err)
	}

	var offset uint64
	var output strings.Builder
	for !strings.Contains(output.String(), "daemon-http-ok") {
		snapshot, err := daemon.Output(ctx, protocol.OutputRequest{PtyID: created.MainPtyID, FromOffset: offset})
		if err != nil {
			t.Fatalf("output: %v", err)
		}
		offset = snapshot.Offset
		output.WriteString(snapshot.Output)
		if strings.Contains(output.String(), "daemon-http-ok") {
			break
		}
		select {
		case <-time.After(20 * time.Millisecond):
		case <-ctx.Done():
			t.Fatalf("timed out waiting for output; got %q", output.String())
		}
	}
}

func TestHTTPClientNextEventTimeoutReturnsNoop(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend(), EventSink: newFakeEventBus()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	event, err := daemon.NextEvent(ctx, protocol.NextEventRequest{TimeoutMs: 1})
	if err != nil {
		t.Fatalf("next event: %v", err)
	}
	if event.Type != protocol.RuntimeEventNone {
		t.Fatalf("event = %#v", event)
	}
}

type fakeEventBus struct {
	ch chan app.RuntimeEvent
}

func newFakeEventBus() *fakeEventBus {
	return &fakeEventBus{ch: make(chan app.RuntimeEvent, 64)}
}

func (b *fakeEventBus) Publish(_ context.Context, event app.RuntimeEvent) error {
	b.ch <- event
	return nil
}

func (b *fakeEventBus) Next(ctx context.Context) (app.RuntimeEvent, error) {
	select {
	case event := <-b.ch:
		return event, nil
	case <-ctx.Done():
		return app.RuntimeEvent{}, ctx.Err()
	}
}

func TestHTTPClientReportsDaemonErrors(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":false}`))
		case "/v1/sessions":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad session"}`))
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`plain failure`))
		}
	}))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx := context.Background()

	if err := daemon.Health(ctx); err == nil || !strings.Contains(err.Error(), "health") {
		t.Fatalf("health error = %v", err)
	}
	if _, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{}); err == nil || !strings.Contains(err.Error(), "bad session") {
		t.Fatalf("create error = %v", err)
	}
	if _, err := daemon.Output(ctx, protocol.OutputRequest{PtyID: "missing"}); err == nil || !strings.Contains(err.Error(), "plain failure") {
		t.Fatalf("output error = %v", err)
	}
}
