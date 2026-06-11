package app_test

import (
	"context"
	"strings"
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
