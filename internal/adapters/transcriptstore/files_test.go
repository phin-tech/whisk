package transcriptstore_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/transcriptstore"
	"github.com/phin-tech/whisk/internal/app"
)

func TestDefaultRootUsesXDGConfigHomeOrHomeDotConfig(t *testing.T) {
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	root, err := transcriptstore.DefaultRoot()
	if err != nil {
		t.Fatalf("default root: %v", err)
	}
	if root != filepath.Join(xdg, "whisk", "transcripts") {
		t.Fatalf("root = %q", root)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	root, err = transcriptstore.DefaultRoot()
	if err != nil {
		t.Fatalf("default root with home: %v", err)
	}
	if root != filepath.Join(home, ".config", "whisk", "transcripts") {
		t.Fatalf("root = %q", root)
	}
}

func TestFileStoreWritesMetadataRawOutputAndEvents(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := transcriptstore.NewFileStore(root)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	err = store.RegisterPTY(ctx, app.PTYTranscriptMeta{
		PTYID:      "pty_01",
		SessionID:  "sess_01",
		WindowID:   "win_01",
		PaneID:     "pane_01",
		WorkingDir: "/repo",
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("register pty: %v", err)
	}
	if err := store.AppendPTYOutput(ctx, app.PTYTranscriptOutput{PTYID: "pty_01", Offset: 0, Bytes: []byte("hello")}); err != nil {
		t.Fatalf("append output: %v", err)
	}
	if err := store.AppendPTYOutput(ctx, app.PTYTranscriptOutput{PTYID: "pty_01", Offset: 3, Bytes: []byte("lo world")}); err != nil {
		t.Fatalf("append overlap: %v", err)
	}
	code := 0
	if err := store.MarkPTYExit(ctx, app.PTYTranscriptExit{PTYID: "pty_01", Code: &code}); err != nil {
		t.Fatalf("mark exit: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(root, "ptys", "pty_01.raw"))
	if err != nil {
		t.Fatalf("read raw: %v", err)
	}
	if string(raw) != "hello world" {
		t.Fatalf("raw = %q", string(raw))
	}
	meta, err := os.ReadFile(filepath.Join(root, "ptys", "pty_01.json"))
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	if !strings.Contains(string(meta), `"sessionId": "sess_01"`) {
		t.Fatalf("meta = %s", string(meta))
	}
	events, err := os.ReadFile(filepath.Join(root, "events.jsonl"))
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if strings.Count(string(events), "pty.output") != 2 || !strings.Contains(string(events), "pty.exit") {
		t.Fatalf("events = %s", string(events))
	}
}

func TestFileStoreRejectsOutputGaps(t *testing.T) {
	store, err := transcriptstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	err = store.AppendPTYOutput(context.Background(), app.PTYTranscriptOutput{PTYID: "pty_01", Offset: 4, Bytes: []byte("gap")})
	if err == nil {
		t.Fatalf("expected gap error")
	}
}
