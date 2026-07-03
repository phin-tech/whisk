package native

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	records, err := backend.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(records) != 1 || records[0].ID != "pty_01" || !records[0].Running {
		t.Fatalf("records = %#v", records)
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
	killed, err := backend.Kill(ctx, "pty_01")
	if err != nil {
		t.Fatalf("kill: %v", err)
	}
	if killed.ID != "pty_01" || killed.Running {
		t.Fatalf("killed = %#v", killed)
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

func TestBackendSpawnResolvesCommandWithRequestPATH(t *testing.T) {
	binDir := t.TempDir()
	command := "whisk-path-agent"
	commandPath := filepath.Join(binDir, command)
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nprintf 'path-ok\\n'\n"), 0o755); err != nil {
		t.Fatalf("write command: %v", err)
	}
	t.Setenv("PATH", "/usr/bin:/bin")

	backend := NewBackend()
	t.Cleanup(func() { _ = backend.Shutdown(context.Background()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := backend.Spawn(ctx, app.SpawnPTYRequest{
		ID:         "pty_path",
		WorkingDir: ".",
		Command:    command,
		Cols:       80,
		Rows:       24,
		Env:        map[string]string{"PATH": binDir + string(os.PathListSeparator) + "/usr/bin:/bin"},
	}); err != nil {
		t.Fatalf("spawn: %v", err)
	}

	for {
		snapshot, err := backend.Output(ctx, "pty_path", 0)
		if err != nil {
			t.Fatalf("output: %v", err)
		}
		if strings.Contains(string(snapshot.OutputBytes), "path-ok") {
			return
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-ctx.Done():
			t.Fatalf("timed out waiting for PATH command output; got %q", string(snapshot.OutputBytes))
		}
	}
}

func TestBackendAttachSlowConsumerCatchesUpFromBuffer(t *testing.T) {
	backend := NewBackend()
	backend.ptys["pty_slow"] = &proc{
		record: app.PTYRecord{ID: "pty_slow", WorkingDir: ".", Cols: 80, Rows: 24, Running: true},
		buffer: newOutputBuffer(16 * 1024),
		subs:   map[*subscriber]struct{}{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	attach, err := backend.Attach(ctx, app.AttachPTYRequest{PtyID: "pty_slow"})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	defer attach.Close()

	var expected strings.Builder
	for i := range 128 {
		chunk := fmt.Sprintf("chunk-%03d\n", i)
		expected.WriteString(chunk)
		backend.broadcastOutput("pty_slow", []byte(chunk))
	}

	want := expected.String()
	var got strings.Builder
	nextOffset := attach.ReplayOffset + uint64(len(attach.ReplayBytes))
	for got.Len() < len(want) {
		select {
		case event, ok := <-attach.Events:
			if !ok {
				t.Fatalf("attach closed after %d bytes, want %d", got.Len(), len(want))
			}
			if event.Kind != app.PTYOutput {
				continue
			}
			if event.Offset != nextOffset {
				t.Fatalf("output offset gap: got %d, want %d", event.Offset, nextOffset)
			}
			got.Write(event.Bytes)
			nextOffset += uint64(len(event.Bytes))
		case <-ctx.Done():
			t.Fatalf("timed out waiting for slow consumer catch-up; got %d bytes, want %d", got.Len(), len(want))
		}
	}
	if got.String() != want {
		t.Fatalf("output mismatch:\n got %q\nwant %q", got.String(), want)
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
	if _, err := backend.Kill(ctx, "missing"); err == nil {
		t.Fatalf("expected missing pty kill error")
	}
	if records, err := backend.List(ctx); err != nil || len(records) != 0 {
		t.Fatalf("records = %#v, err = %v", records, err)
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
