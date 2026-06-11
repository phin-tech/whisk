package session_test

import (
	"testing"

	"github.com/phin-tech/whisk/internal/domain/session"
)

func TestCreateSessionCreatesFocusedMainPane(t *testing.T) {
	state := session.NewState()
	created, err := state.CreateSession(session.CreateSession{
		SessionID:  "sess_01",
		PaneID:     "pane_01",
		PtyID:      "pty_01",
		Name:       "Whisk",
		WorkingDir: "/repo",
	})

	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if created.ID != "sess_01" {
		t.Fatalf("session id = %q", created.ID)
	}
	if created.FocusedPaneID != "pane_01" {
		t.Fatalf("focused pane = %q", created.FocusedPaneID)
	}
	if created.Layout.Kind != session.LayoutLeaf {
		t.Fatalf("layout kind = %q", created.Layout.Kind)
	}
	if created.Layout.PaneID != "pane_01" {
		t.Fatalf("layout pane = %q", created.Layout.PaneID)
	}
	if created.Panes["pane_01"].PtyID != "pty_01" {
		t.Fatalf("pane pty = %q", created.Panes["pane_01"].PtyID)
	}
}

func TestSplitFocusedPaneCreatesNewFocusedPane(t *testing.T) {
	state := session.NewState()
	_, err := state.CreateSession(session.CreateSession{
		SessionID:  "sess_01",
		PaneID:     "pane_01",
		PtyID:      "pty_01",
		WorkingDir: "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	updated, err := state.SplitPane(session.SplitPane{
		SessionID:    "sess_01",
		TargetPaneID: "pane_01",
		NewPaneID:    "pane_02",
		NewPtyID:     "pty_02",
		Direction:    session.SplitHorizontal,
	})

	if err != nil {
		t.Fatalf("split pane: %v", err)
	}
	if updated.FocusedPaneID != "pane_02" {
		t.Fatalf("focused pane = %q", updated.FocusedPaneID)
	}
	if updated.Layout.Kind != session.LayoutSplit {
		t.Fatalf("layout kind = %q", updated.Layout.Kind)
	}
	if updated.Layout.Direction != session.SplitHorizontal {
		t.Fatalf("split direction = %q", updated.Layout.Direction)
	}
	if len(updated.Layout.Children) != 2 {
		t.Fatalf("children = %d", len(updated.Layout.Children))
	}
	if updated.Layout.Children[0].PaneID != "pane_01" || updated.Layout.Children[1].PaneID != "pane_02" {
		t.Fatalf("children = %#v", updated.Layout.Children)
	}
}
