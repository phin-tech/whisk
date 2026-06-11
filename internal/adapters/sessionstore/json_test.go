package sessionstore_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/sessionstore"
	"github.com/phin-tech/whisk/internal/domain/session"
)

func TestDefaultPathUsesXDGConfigHomeOrHomeDotConfig(t *testing.T) {
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path, err := sessionstore.DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	if path != filepath.Join(xdg, "whisk", "runtime-state.json") {
		t.Fatalf("path = %q", path)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err = sessionstore.DefaultPath()
	if err != nil {
		t.Fatalf("default path with home: %v", err)
	}
	if path != filepath.Join(home, ".config", "whisk", "runtime-state.json") {
		t.Fatalf("path = %q", path)
	}
}

func TestJSONStoreLoadMissingFileReturnsEmpty(t *testing.T) {
	store, err := sessionstore.NewJSONStore(filepath.Join(t.TempDir(), "missing", "runtime-state.json"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sessions, err := store.LoadSessions(context.Background())
	if err != nil {
		t.Fatalf("load sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions = %#v", sessions)
	}
}

func TestJSONStoreRoundTripsSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "whisk", "runtime-state.json")
	store, err := sessionstore.NewJSONStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	ptyID := "pty_01"
	input := []session.Session{
		{
			ID:      "sess_01",
			Name:    "Persisted",
			RootDir: "/repo",
			Windows: map[string]session.SessionWindow{
				"win_01": {
					ID:        "win_01",
					SessionID: "sess_01",
					Name:      "Main",
					Layout:    session.LayoutNode{Kind: session.LayoutLeaf, PaneID: "pane_01"},
				},
			},
			Panes: map[string]session.Pane{
				"pane_01": {ID: "pane_01", WindowID: "win_01", CurrentPTYID: &ptyID, WorkingDir: "/repo"},
			},
		},
	}

	if err := store.SaveSessions(context.Background(), input); err != nil {
		t.Fatalf("save sessions: %v", err)
	}
	loaded, err := store.LoadSessions(context.Background())
	if err != nil {
		t.Fatalf("load sessions: %v", err)
	}
	if len(loaded) != 1 || loaded[0].ID != "sess_01" {
		t.Fatalf("loaded = %#v", loaded)
	}
	if loaded[0].Panes["pane_01"].CurrentPTYID == nil || *loaded[0].Panes["pane_01"].CurrentPTYID != "pty_01" {
		t.Fatalf("loaded pane = %#v", loaded[0].Panes["pane_01"])
	}
}
