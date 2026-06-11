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

	created, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:       "Whisk",
		WorkingDir: ".",
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.Session.ID == "" || created.MainPtyID == "" {
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

	event, err := daemon.NextEvent(ctx, protocol.NextEventRequest{TimeoutMs: 10})
	if err != nil {
		t.Fatalf("next event: %v", err)
	}
	if event.Type != "session.changed" {
		t.Fatalf("event = %#v", event)
	}

	split, err := daemon.SplitPane(ctx, protocol.SplitPaneRequest{
		SessionID:    created.Session.ID,
		TargetPaneID: created.Session.FocusedPaneID,
		Direction:    "horizontal",
		Cols:         80,
		Rows:         24,
	})
	if err != nil {
		t.Fatalf("split pane: %v", err)
	}
	if split.PaneID == "" || split.PtyID == "" || split.Session.FocusedPaneID != split.PaneID {
		t.Fatalf("split result = %#v", split)
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
