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

func TestSplitNestedPanePreservesExistingLayout(t *testing.T) {
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
	_, err = state.SplitPane(session.SplitPane{
		SessionID:    "sess_01",
		TargetPaneID: "pane_01",
		NewPaneID:    "pane_02",
		NewPtyID:     "pty_02",
		Direction:    session.SplitHorizontal,
	})
	if err != nil {
		t.Fatalf("split pane: %v", err)
	}

	updated, err := state.SplitPane(session.SplitPane{
		SessionID:    "sess_01",
		TargetPaneID: "pane_01",
		NewPaneID:    "pane_03",
		NewPtyID:     "pty_03",
		Direction:    session.SplitVertical,
	})
	if err != nil {
		t.Fatalf("nested split pane: %v", err)
	}

	if updated.Layout.Kind != session.LayoutSplit || updated.Layout.Direction != session.SplitHorizontal {
		t.Fatalf("root layout = %#v", updated.Layout)
	}
	if len(updated.Layout.Children) != 2 {
		t.Fatalf("root children = %d", len(updated.Layout.Children))
	}
	left := updated.Layout.Children[0]
	right := updated.Layout.Children[1]
	if right.PaneID != "pane_02" {
		t.Fatalf("right pane = %#v", right)
	}
	if left.Kind != session.LayoutSplit || left.Direction != session.SplitVertical {
		t.Fatalf("left nested layout = %#v", left)
	}
	if len(left.Children) != 2 || left.Children[0].PaneID != "pane_01" || left.Children[1].PaneID != "pane_03" {
		t.Fatalf("left nested children = %#v", left.Children)
	}
	if updated.FocusedPaneID != "pane_03" {
		t.Fatalf("focused pane = %q", updated.FocusedPaneID)
	}
}

func TestGetAndListReturnClones(t *testing.T) {
	state := session.NewState()
	created, err := state.CreateSession(session.CreateSession{
		SessionID:  "sess_01",
		PaneID:     "pane_01",
		PtyID:      "pty_01",
		WorkingDir: "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	created.Name = "mutated"
	created.Panes["pane_01"] = session.Pane{ID: "pane_01", PtyID: "mutated"}

	got, ok := state.Get("sess_01")
	if !ok {
		t.Fatalf("session not found")
	}
	if got.Name == "mutated" || got.Panes["pane_01"].PtyID == "mutated" {
		t.Fatalf("get leaked mutable state: %#v", got)
	}

	listed := state.List()
	listed[0].Name = "listed-mutated"
	listed[0].Panes["pane_01"] = session.Pane{ID: "pane_01", PtyID: "listed-mutated"}

	got, ok = state.Get("sess_01")
	if !ok {
		t.Fatalf("session not found after list")
	}
	if got.Name == "listed-mutated" || got.Panes["pane_01"].PtyID == "listed-mutated" {
		t.Fatalf("list leaked mutable state: %#v", got)
	}
}

func TestPTYOwnersMapsPTYsToSessionPanes(t *testing.T) {
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
	if _, err := state.SplitPane(session.SplitPane{
		SessionID:    created.ID,
		TargetPaneID: created.FocusedPaneID,
		NewPaneID:    "pane_02",
		NewPtyID:     "pty_02",
		Direction:    session.SplitHorizontal,
	}); err != nil {
		t.Fatalf("split pane: %v", err)
	}

	owners := state.PTYOwners()

	if owners["pty_01"].SessionID != "sess_01" || owners["pty_01"].PaneID != "pane_01" {
		t.Fatalf("pty_01 owner = %#v", owners["pty_01"])
	}
	if owners["pty_02"].SessionID != "sess_01" || owners["pty_02"].PaneID != "pane_02" {
		t.Fatalf("pty_02 owner = %#v", owners["pty_02"])
	}
}

func TestCreateSessionRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		req  session.CreateSession
	}{
		{name: "missing session", req: session.CreateSession{PaneID: "pane", PtyID: "pty"}},
		{name: "missing pane", req: session.CreateSession{SessionID: "session", PtyID: "pty"}},
		{name: "missing pty", req: session.CreateSession{SessionID: "session", PaneID: "pane"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := session.NewState().CreateSession(test.req); err == nil {
				t.Fatalf("expected error")
			}
		})
	}

	state := session.NewState()
	_, err := state.CreateSession(session.CreateSession{SessionID: "session", PaneID: "pane", PtyID: "pty"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := state.CreateSession(session.CreateSession{SessionID: "session", PaneID: "pane_2", PtyID: "pty_2"}); err == nil {
		t.Fatalf("expected duplicate session error")
	}
}

func TestSplitPaneRejectsInvalidInput(t *testing.T) {
	state := session.NewState()
	if _, err := state.SplitPane(session.SplitPane{SessionID: "missing"}); err == nil {
		t.Fatalf("expected missing session error")
	}
	_, err := state.CreateSession(session.CreateSession{SessionID: "session", PaneID: "pane", PtyID: "pty"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	tests := []struct {
		name string
		req  session.SplitPane
	}{
		{name: "missing target", req: session.SplitPane{SessionID: "session", TargetPaneID: "missing", NewPaneID: "pane_2", NewPtyID: "pty_2"}},
		{name: "missing new pane", req: session.SplitPane{SessionID: "session", TargetPaneID: "pane", NewPtyID: "pty_2"}},
		{name: "missing new pty", req: session.SplitPane{SessionID: "session", TargetPaneID: "pane", NewPaneID: "pane_2"}},
		{name: "duplicate pane", req: session.SplitPane{SessionID: "session", TargetPaneID: "pane", NewPaneID: "pane", NewPtyID: "pty_2"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := state.SplitPane(test.req); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}
