package client_test

import (
	"context"
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

	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
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
