package native

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
)

func TestBackendSpawnWriteOutputAttachAndResize(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	backend := NewBackend()
	t.Cleanup(func() { _ = backend.Shutdown(context.Background()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record, err := backend.Spawn(ctx, app.SpawnPTYRequest{
		ID:         "pty_01",
		WorkingDir: ".",
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if record.ID != "pty_01" || record.Cols != 80 || record.Rows != 24 || !record.Running {
		t.Fatalf("record = %#v", record)
	}

	attach, err := backend.Attach(ctx, app.AttachPTYRequest{PtyID: "pty_01"})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	defer attach.Close()

	if err := backend.Resize(ctx, "pty_01", app.PTYSize{Cols: 70, Rows: 18}); err != nil {
		t.Fatalf("resize: %v", err)
	}
	if err := backend.Write(ctx, "pty_01", []byte("printf 'native-ok\\n'\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	var output strings.Builder
	for !strings.Contains(output.String(), "native-ok") {
		select {
		case event := <-attach.Events:
			if event.Kind == app.PTYOutput {
				output.Write(event.Bytes)
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for output; got %q", output.String())
		}
	}

	snapshot, err := backend.Output(ctx, "pty_01", 0)
	if err != nil {
		t.Fatalf("output: %v", err)
	}
	if snapshot.Record.Cols != 70 || snapshot.Record.Rows != 18 {
		t.Fatalf("snapshot size = %#v", snapshot.Record)
	}
	if !strings.Contains(string(snapshot.OutputBytes), "native-ok") {
		t.Fatalf("snapshot output = %q", string(snapshot.OutputBytes))
	}
}

func TestBackendSpawnInjectsEnvironment(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	backend := NewBackend()
	t.Cleanup(func() { _ = backend.Shutdown(context.Background()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := backend.Spawn(ctx, app.SpawnPTYRequest{
		ID:         "pty_env",
		WorkingDir: ".",
		Cols:       80,
		Rows:       24,
		Env:        map[string]string{"WHISK_TEST_CONTEXT": "context-ok"},
	}); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	attach, err := backend.Attach(ctx, app.AttachPTYRequest{PtyID: "pty_env"})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	defer attach.Close()
	if err := backend.Write(ctx, "pty_env", []byte("printf \"$WHISK_TEST_CONTEXT\\n\"\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	var output strings.Builder
	for !strings.Contains(output.String(), "context-ok") {
		select {
		case event := <-attach.Events:
			if event.Kind == app.PTYOutput {
				output.Write(event.Bytes)
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for env output; got %q", output.String())
		}
	}
}

func TestBackendRejectsInvalidOperations(t *testing.T) {
	backend := NewBackend()
	t.Cleanup(func() { _ = backend.Shutdown(context.Background()) })
	ctx := context.Background()

	if _, err := backend.Spawn(ctx, app.SpawnPTYRequest{}); err == nil {
		t.Fatalf("expected missing id error")
	}
	if _, err := backend.Spawn(ctx, app.SpawnPTYRequest{ID: "pty_file", WorkingDir: "go.mod"}); err == nil {
		t.Fatalf("expected file working dir error")
	}
	if err := backend.Write(ctx, "missing", []byte("x")); err == nil {
		t.Fatalf("expected missing pty write error")
	}
	if err := backend.Resize(ctx, "missing", app.PTYSize{Cols: 80, Rows: 24}); err == nil {
		t.Fatalf("expected missing pty resize error")
	}
	if err := backend.Resize(ctx, "missing", app.PTYSize{Cols: 0, Rows: 24}); err == nil {
		t.Fatalf("expected invalid cols error")
	}
	if _, err := backend.Attach(ctx, app.AttachPTYRequest{PtyID: "missing"}); err == nil {
		t.Fatalf("expected missing pty attach error")
	}
	if _, err := backend.Output(ctx, "missing", 0); err == nil {
		t.Fatalf("expected missing pty output error")
	}
}

func TestOutputBufferClampsOffsetsAndKeepsSnapshotImmutable(t *testing.T) {
	buffer := newOutputBuffer(5)
	buffer.append([]byte("hello"))
	buffer.append([]byte("world"))

	offset, snapshot := buffer.snapshotFrom(0)
	if offset != 5 || string(snapshot) != "world" {
		t.Fatalf("snapshot = offset %d bytes %q", offset, string(snapshot))
	}

	snapshot[0] = 'W'
	_, second := buffer.snapshotFrom(offset)
	if string(second) != "world" {
		t.Fatalf("snapshot mutated buffer: %q", string(second))
	}

	offset, snapshot = buffer.snapshotFrom(100)
	if offset != 10 || len(snapshot) != 0 {
		t.Fatalf("future snapshot = offset %d bytes %q", offset, string(snapshot))
	}
}
