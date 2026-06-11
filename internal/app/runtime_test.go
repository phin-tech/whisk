package app_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/app"
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
