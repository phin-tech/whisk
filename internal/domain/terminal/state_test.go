package terminal_test

import (
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/domain/terminal"
)

func TestStateWritesSnapshotsAndResizes(t *testing.T) {
	state := terminal.New(4, 2, terminal.Options{MaxScrollbackLines: 4})

	if err := state.Write(0, []byte("abcd\nef")); err != nil {
		t.Fatalf("write: %v", err)
	}
	state.Resize(6, 3)
	snapshot := state.Snapshot()

	if snapshot.Offset != uint64(len("abcd\nef")) {
		t.Fatalf("offset = %d", snapshot.Offset)
	}
	if snapshot.Cols != 6 || snapshot.Rows != 3 {
		t.Fatalf("size = %dx%d", snapshot.Cols, snapshot.Rows)
	}
	if !strings.Contains(snapshot.ViewportAnsi, "e") || !strings.Contains(snapshot.ViewportAnsi, "f") {
		t.Fatalf("viewport ansi = %q", snapshot.ViewportAnsi)
	}
	if snapshot.Modes.CursorVisible != true {
		t.Fatalf("cursor visible = %v", snapshot.Modes.CursorVisible)
	}
}

func TestStateRejectsOffsetGaps(t *testing.T) {
	state := terminal.New(10, 2, terminal.Options{})

	if err := state.Write(1, []byte("x")); err == nil {
		t.Fatal("expected offset mismatch error")
	}
}

func TestSnapshotSanitizesAndBoundsFields(t *testing.T) {
	state := terminal.New(5, 2, terminal.Options{
		MaxScrollbackLines:       10,
		MaxSnapshotFieldBytes:    12,
		MaxTitleBytes:            8,
		MaxWorkingDirectoryBytes: 12,
	})
	input := "\x1b]0;hello\x07bad\nname\x07" +
		"\x1b]7;file://host/tmp/project\x07" +
		"line1\nline2\nline3\nline4"
	if err := state.Write(0, []byte(input)); err != nil {
		t.Fatalf("write: %v", err)
	}

	snapshot := state.Snapshot()
	if strings.ContainsAny(snapshot.Title, "\a\n") {
		t.Fatalf("title was not sanitized: %q", snapshot.Title)
	}
	if len(snapshot.Title) > 8 {
		t.Fatalf("title length = %d, want <= 8: %q", len(snapshot.Title), snapshot.Title)
	}
	if strings.ContainsAny(snapshot.WorkingDirectory, "\a\n") {
		t.Fatalf("working directory was not sanitized: %q", snapshot.WorkingDirectory)
	}
	if len(snapshot.WorkingDirectory) > 12 {
		t.Fatalf("working directory length = %d, want <= 12: %q", len(snapshot.WorkingDirectory), snapshot.WorkingDirectory)
	}
	if len(snapshot.ScrollbackAnsi) > 12 {
		t.Fatalf("scrollback ansi length = %d, want <= 12: %q", len(snapshot.ScrollbackAnsi), snapshot.ScrollbackAnsi)
	}
	if len(snapshot.ViewportAnsi) > 12 {
		t.Fatalf("viewport ansi length = %d, want <= 12: %q", len(snapshot.ViewportAnsi), snapshot.ViewportAnsi)
	}
	if !snapshot.Truncated {
		t.Fatal("expected snapshot to report truncation")
	}
}
