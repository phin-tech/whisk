package session_test

import (
	"testing"

	"github.com/phin-tech/whisk/internal/domain/session"
)

func TestCreateSessionCreatesDefaultWindowAndPane(t *testing.T) {
	state := session.NewState()
	initialPTYID := "pty_01"

	created, err := state.CreateSession(session.CreateSession{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		PaneID:       "pane_01",
		InitialPTYID: &initialPTYID,
		Name:         "Whisk",
		RootDir:      "/repo",
	})

	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.ID != "sess_01" || created.RootDir != "/repo" {
		t.Fatalf("created session = %#v", created)
	}
	if _, ok := created.Windows["win_01"]; !ok {
		t.Fatalf("default window missing: %#v", created.Windows)
	}
	window := created.Windows["win_01"]
	if window.Name != "Main" {
		t.Fatalf("window name = %q", window.Name)
	}
	if window.Layout.Kind != session.LayoutLeaf || window.Layout.PaneID != "pane_01" {
		t.Fatalf("window layout = %#v", window.Layout)
	}
	pane := created.Panes["pane_01"]
	if pane.WindowID != "win_01" {
		t.Fatalf("pane window = %q", pane.WindowID)
	}
	if pane.WorkingDir != "/repo" {
		t.Fatalf("pane working dir = %q", pane.WorkingDir)
	}
	if pane.CurrentPTYID == nil || *pane.CurrentPTYID != "pty_01" {
		t.Fatalf("pane current pty = %#v", pane.CurrentPTYID)
	}
}

func TestCreateSessionCanCreateEmptyDefaultPane(t *testing.T) {
	state := session.NewState()

	created, err := state.CreateSession(session.CreateSession{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
		RootDir:   "/repo",
	})

	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	pane := created.Panes["pane_01"]
	if pane.CurrentPTYID != nil {
		t.Fatalf("expected empty pane, got %#v", pane.CurrentPTYID)
	}
	if pane.WorkingDir != "/repo" {
		t.Fatalf("pane working dir = %q", pane.WorkingDir)
	}
}

func TestSplitPaneWithinWindowInheritsWorkingDirAndMayBeEmpty(t *testing.T) {
	state := session.NewState()
	initialPTYID := "pty_01"
	_, err := state.CreateSession(session.CreateSession{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		PaneID:       "pane_01",
		InitialPTYID: &initialPTYID,
		RootDir:      "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	updated, err := state.SplitPane(session.SplitPane{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		TargetPaneID: "pane_01",
		NewPaneID:    "pane_02",
		Direction:    session.SplitHorizontal,
	})

	if err != nil {
		t.Fatalf("split pane: %v", err)
	}
	window := updated.Windows["win_01"]
	if window.Layout.Kind != session.LayoutSplit {
		t.Fatalf("layout kind = %q", window.Layout.Kind)
	}
	if len(window.Layout.Children) != 2 {
		t.Fatalf("children = %#v", window.Layout.Children)
	}
	if updated.Panes["pane_02"].CurrentPTYID != nil {
		t.Fatalf("new pane should be empty: %#v", updated.Panes["pane_02"])
	}
	if updated.Panes["pane_02"].WorkingDir != "/repo" {
		t.Fatalf("new pane working dir = %q", updated.Panes["pane_02"].WorkingDir)
	}
}

func TestSplitPaneWithinWindowCanAttachInitialPTY(t *testing.T) {
	state := session.NewState()
	_, err := state.CreateSession(session.CreateSession{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
		RootDir:   "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	newPTYID := "pty_02"

	updated, err := state.SplitPane(session.SplitPane{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		TargetPaneID: "pane_01",
		NewPaneID:    "pane_02",
		NewPTYID:     &newPTYID,
		Direction:    session.SplitVertical,
	})

	if err != nil {
		t.Fatalf("split pane: %v", err)
	}
	pane := updated.Panes["pane_02"]
	if pane.CurrentPTYID == nil || *pane.CurrentPTYID != "pty_02" {
		t.Fatalf("pane current pty = %#v", pane.CurrentPTYID)
	}
}

func TestSetSessionRootDirUpdatesOnlyDefaultPaneWorkingDirs(t *testing.T) {
	state := session.NewState()
	_, err := state.CreateSession(session.CreateSession{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
		RootDir:   "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := state.SetPaneWorkingDir(session.SetPaneWorkingDir{
		SessionID:  "sess_01",
		PaneID:     "pane_01",
		WorkingDir: "/repo/frontend",
	}); err != nil {
		t.Fatalf("set pane working dir: %v", err)
	}

	updated, err := state.SetSessionRootDir(session.SetSessionRootDir{
		SessionID: "sess_01",
		RootDir:   "/other",
	})

	if err != nil {
		t.Fatalf("set session root: %v", err)
	}
	if updated.RootDir != "/other" {
		t.Fatalf("root dir = %q", updated.RootDir)
	}
	if updated.Panes["pane_01"].WorkingDir != "/repo/frontend" {
		t.Fatalf("drifted pane working dir was overwritten: %#v", updated.Panes["pane_01"])
	}
}

func TestSetSessionRootDirUpdatesEmptyAndDefaultPaneWorkingDirs(t *testing.T) {
	state := session.NewState()
	initialPTYID := "pty_01"
	_, err := state.CreateSession(session.CreateSession{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		PaneID:       "pane_01",
		InitialPTYID: &initialPTYID,
		RootDir:      "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := state.SplitPane(session.SplitPane{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		TargetPaneID: "pane_01",
		NewPaneID:    "pane_02",
	}); err != nil {
		t.Fatalf("split pane: %v", err)
	}

	updated, err := state.SetSessionRootDir(session.SetSessionRootDir{
		SessionID: "sess_01",
		RootDir:   "/other",
	})

	if err != nil {
		t.Fatalf("set root dir: %v", err)
	}
	if updated.Panes["pane_01"].WorkingDir != "/other" || updated.Panes["pane_02"].WorkingDir != "/other" {
		t.Fatalf("pane working dirs = %#v", updated.Panes)
	}
}

func TestSetPaneWorkingDirRejectsInvalidInput(t *testing.T) {
	state := session.NewState()
	_, err := state.CreateSession(session.CreateSession{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
		RootDir:   "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := state.SetPaneWorkingDir(session.SetPaneWorkingDir{
		SessionID:  "sess_01",
		PaneID:     "pane_01",
		WorkingDir: "relative",
	}); err == nil {
		t.Fatalf("expected relative working dir error")
	}
	if _, err := state.SetPaneWorkingDir(session.SetPaneWorkingDir{
		SessionID:  "sess_01",
		PaneID:     "missing",
		WorkingDir: "/repo",
	}); err == nil {
		t.Fatalf("expected missing pane error")
	}
}

func TestStartPanePTYAttachesPTYToEmptyPane(t *testing.T) {
	state := session.NewState()
	_, err := state.CreateSession(session.CreateSession{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
		RootDir:   "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	updated, err := state.StartPanePTY(session.StartPanePTY{
		SessionID: "sess_01",
		PaneID:    "pane_01",
		PTYID:     "pty_01",
	})

	if err != nil {
		t.Fatalf("start pane pty: %v", err)
	}
	if updated.Panes["pane_01"].CurrentPTYID == nil || *updated.Panes["pane_01"].CurrentPTYID != "pty_01" {
		t.Fatalf("pane pty = %#v", updated.Panes["pane_01"].CurrentPTYID)
	}
}

func TestStartPanePTYRejectsNonEmptyPane(t *testing.T) {
	state := session.NewState()
	ptyID := "pty_01"
	_, err := state.CreateSession(session.CreateSession{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		PaneID:       "pane_01",
		InitialPTYID: &ptyID,
		RootDir:      "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := state.StartPanePTY(session.StartPanePTY{
		SessionID: "sess_01",
		PaneID:    "pane_01",
		PTYID:     "pty_02",
	}); err == nil {
		t.Fatalf("expected non-empty pane error")
	}
}

func TestDetachPanePTYClearsCurrentPTY(t *testing.T) {
	state := session.NewState()
	ptyID := "pty_01"
	_, err := state.CreateSession(session.CreateSession{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		PaneID:       "pane_01",
		InitialPTYID: &ptyID,
		RootDir:      "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	updated, detached, err := state.DetachPanePTY(session.DetachPanePTY{
		SessionID: "sess_01",
		PaneID:    "pane_01",
	})

	if err != nil {
		t.Fatalf("detach pane pty: %v", err)
	}
	if detached != "pty_01" {
		t.Fatalf("detached = %q", detached)
	}
	if updated.Panes["pane_01"].CurrentPTYID != nil {
		t.Fatalf("pane current pty = %#v", updated.Panes["pane_01"].CurrentPTYID)
	}
}

func TestDetachPanePTYRejectsEmptyPane(t *testing.T) {
	state := session.NewState()
	_, err := state.CreateSession(session.CreateSession{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
		RootDir:   "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, _, err := state.DetachPanePTY(session.DetachPanePTY{
		SessionID: "sess_01",
		PaneID:    "pane_01",
	}); err == nil {
		t.Fatalf("expected empty pane detach error")
	}
}

func TestRestartPanePTYReplacesCurrentPTY(t *testing.T) {
	state := session.NewState()
	ptyID := "pty_01"
	_, err := state.CreateSession(session.CreateSession{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		PaneID:       "pane_01",
		InitialPTYID: &ptyID,
		RootDir:      "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	updated, oldPTY, err := state.RestartPanePTY(session.RestartPanePTY{
		SessionID: "sess_01",
		PaneID:    "pane_01",
		NewPTYID:  "pty_02",
	})

	if err != nil {
		t.Fatalf("restart pane pty: %v", err)
	}
	if oldPTY != "pty_01" {
		t.Fatalf("old pty = %q", oldPTY)
	}
	if updated.Panes["pane_01"].CurrentPTYID == nil || *updated.Panes["pane_01"].CurrentPTYID != "pty_02" {
		t.Fatalf("pane pty = %#v", updated.Panes["pane_01"].CurrentPTYID)
	}
}

func TestRestartPanePTYRejectsEmptyPane(t *testing.T) {
	state := session.NewState()
	_, err := state.CreateSession(session.CreateSession{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
		RootDir:   "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, _, err := state.RestartPanePTY(session.RestartPanePTY{
		SessionID: "sess_01",
		PaneID:    "pane_01",
		NewPTYID:  "pty_02",
	}); err == nil {
		t.Fatalf("expected empty pane restart error")
	}
}

func TestClosePaneRemovesPaneAndLayoutLeaf(t *testing.T) {
	state := session.NewState()
	_, err := state.CreateSession(session.CreateSession{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
		RootDir:   "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := state.SplitPane(session.SplitPane{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		TargetPaneID: "pane_01",
		NewPaneID:    "pane_02",
	}); err != nil {
		t.Fatalf("split pane: %v", err)
	}

	updated, err := state.ClosePane(session.ClosePane{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_02",
	})

	if err != nil {
		t.Fatalf("close pane: %v", err)
	}
	if _, ok := updated.Panes["pane_02"]; ok {
		t.Fatalf("closed pane still present: %#v", updated.Panes)
	}
	if updated.Windows["win_01"].Layout.Kind != session.LayoutLeaf || updated.Windows["win_01"].Layout.PaneID != "pane_01" {
		t.Fatalf("layout = %#v", updated.Windows["win_01"].Layout)
	}
}

func TestClosePaneRejectsLastPaneAndPaneWithCurrentPTY(t *testing.T) {
	state := session.NewState()
	ptyID := "pty_01"
	_, err := state.CreateSession(session.CreateSession{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		PaneID:       "pane_01",
		InitialPTYID: &ptyID,
		RootDir:      "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := state.ClosePane(session.ClosePane{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
	}); err == nil {
		t.Fatalf("expected current pty close error")
	}
	if _, err := state.SplitPane(session.SplitPane{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		TargetPaneID: "pane_01",
		NewPaneID:    "pane_02",
	}); err != nil {
		t.Fatalf("split pane: %v", err)
	}
	if _, err := state.ClosePane(session.ClosePane{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_02",
	}); err != nil {
		t.Fatalf("close empty pane: %v", err)
	}
	if _, err := state.ClosePane(session.ClosePane{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
	}); err == nil {
		t.Fatalf("expected last pane/current pty close error")
	}
}

func TestRemoveSessionDeletesSession(t *testing.T) {
	state := session.NewState()
	_, err := state.CreateSession(session.CreateSession{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
		RootDir:   "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	removed, err := state.RemoveSession(session.RemoveSession{SessionID: "sess_01"})
	if err != nil {
		t.Fatalf("remove session: %v", err)
	}
	if removed.ID != "sess_01" {
		t.Fatalf("removed = %#v", removed)
	}
	if _, ok := state.Get("sess_01"); ok {
		t.Fatalf("session still present")
	}
	if len(state.List()) != 0 {
		t.Fatalf("sessions = %#v", state.List())
	}
}

func TestRemoveSessionRejectsMissingSession(t *testing.T) {
	if _, err := session.NewState().RemoveSession(session.RemoveSession{SessionID: "missing"}); err == nil {
		t.Fatalf("expected missing session error")
	}
}

func TestGetAndListReturnClones(t *testing.T) {
	state := session.NewState()
	created, err := state.CreateSession(session.CreateSession{
		SessionID: "sess_01",
		WindowID:  "win_01",
		PaneID:    "pane_01",
		RootDir:   "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	created.Name = "mutated"
	created.Panes["pane_01"] = session.Pane{ID: "pane_01", WindowID: "win_01", WorkingDir: "mutated"}
	created.Windows["win_01"] = session.SessionWindow{ID: "win_01", SessionID: "sess_01", Name: "mutated"}

	got, ok := state.Get("sess_01")
	if !ok {
		t.Fatalf("session not found")
	}
	if got.Name == "mutated" || got.Panes["pane_01"].WorkingDir == "mutated" || got.Windows["win_01"].Name == "mutated" {
		t.Fatalf("get leaked mutable state: %#v", got)
	}

	listed := state.List()
	listed[0].Name = "listed-mutated"
	listed[0].Panes["pane_01"] = session.Pane{ID: "pane_01", WindowID: "win_01", WorkingDir: "listed-mutated"}
	listed[0].Windows["win_01"] = session.SessionWindow{ID: "win_01", SessionID: "sess_01", Name: "listed-mutated"}

	got, ok = state.Get("sess_01")
	if !ok {
		t.Fatalf("session not found after list")
	}
	if got.Name == "listed-mutated" || got.Panes["pane_01"].WorkingDir == "listed-mutated" || got.Windows["win_01"].Name == "listed-mutated" {
		t.Fatalf("list leaked mutable state: %#v", got)
	}
}

func TestPTYOwnersMapsCurrentPTYsToSessionWindowPanes(t *testing.T) {
	state := session.NewState()
	initialPTYID := "pty_01"
	created, err := state.CreateSession(session.CreateSession{
		SessionID:    "sess_01",
		WindowID:     "win_01",
		PaneID:       "pane_01",
		InitialPTYID: &initialPTYID,
		Name:         "Whisk",
		RootDir:      "/repo",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	secondPTYID := "pty_02"
	if _, err := state.SplitPane(session.SplitPane{
		SessionID:    created.ID,
		WindowID:     "win_01",
		TargetPaneID: "pane_01",
		NewPaneID:    "pane_02",
		NewPTYID:     &secondPTYID,
		Direction:    session.SplitHorizontal,
	}); err != nil {
		t.Fatalf("split pane: %v", err)
	}

	owners := state.PTYOwners()

	if owners["pty_01"].SessionID != "sess_01" || owners["pty_01"].WindowID != "win_01" || owners["pty_01"].PaneID != "pane_01" {
		t.Fatalf("pty_01 owner = %#v", owners["pty_01"])
	}
	if owners["pty_02"].SessionID != "sess_01" || owners["pty_02"].WindowID != "win_01" || owners["pty_02"].PaneID != "pane_02" {
		t.Fatalf("pty_02 owner = %#v", owners["pty_02"])
	}
}

func TestNewStateFromSessionsRestoresValidatedSessions(t *testing.T) {
	ptyID := "pty_01"
	state, err := session.NewStateFromSessions([]session.Session{
		{
			ID:      "sess_01",
			Name:    "Whisk",
			RootDir: "/repo/.",
			Windows: map[string]session.SessionWindow{
				"win_01": {
					ID:        "win_01",
					SessionID: "sess_01",
					Name:      "Main",
					Layout:    session.LayoutNode{Kind: session.LayoutLeaf, PaneID: "pane_01"},
				},
			},
			Panes: map[string]session.Pane{
				"pane_01": {ID: "pane_01", WindowID: "win_01", CurrentPTYID: &ptyID, WorkingDir: "/repo/."},
			},
		},
	})
	if err != nil {
		t.Fatalf("restore sessions: %v", err)
	}

	restored, ok := state.Get("sess_01")
	if !ok {
		t.Fatalf("session not restored")
	}
	if restored.RootDir != "/repo" || restored.Panes["pane_01"].WorkingDir != "/repo" {
		t.Fatalf("paths not cleaned: %#v", restored)
	}
	if restored.Panes["pane_01"].CurrentPTYID == nil || *restored.Panes["pane_01"].CurrentPTYID != "pty_01" {
		t.Fatalf("pty id not restored: %#v", restored.Panes["pane_01"])
	}
}

func TestNewStateFromSessionsRejectsMalformedLayout(t *testing.T) {
	_, err := session.NewStateFromSessions([]session.Session{
		{
			ID:      "sess_01",
			Name:    "Whisk",
			RootDir: "/repo",
			Windows: map[string]session.SessionWindow{
				"win_01": {
					ID:        "win_01",
					SessionID: "sess_01",
					Name:      "Main",
					Layout:    session.LayoutNode{Kind: session.LayoutLeaf, PaneID: "missing"},
				},
			},
			Panes: map[string]session.Pane{
				"pane_01": {ID: "pane_01", WindowID: "win_01", WorkingDir: "/repo"},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected malformed layout error")
	}
}

func TestCreateSessionRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		req  session.CreateSession
	}{
		{name: "missing session", req: session.CreateSession{WindowID: "win", PaneID: "pane", RootDir: "/repo"}},
		{name: "missing window", req: session.CreateSession{SessionID: "session", PaneID: "pane", RootDir: "/repo"}},
		{name: "missing pane", req: session.CreateSession{SessionID: "session", WindowID: "win", RootDir: "/repo"}},
		{name: "missing root", req: session.CreateSession{SessionID: "session", WindowID: "win", PaneID: "pane"}},
		{name: "relative root", req: session.CreateSession{SessionID: "session", WindowID: "win", PaneID: "pane", RootDir: "repo"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := session.NewState().CreateSession(test.req); err == nil {
				t.Fatalf("expected error")
			}
		})
	}

	state := session.NewState()
	_, err := state.CreateSession(session.CreateSession{SessionID: "session", WindowID: "win", PaneID: "pane", RootDir: "/repo"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := state.CreateSession(session.CreateSession{SessionID: "session", WindowID: "win_2", PaneID: "pane_2", RootDir: "/repo"}); err == nil {
		t.Fatalf("expected duplicate session error")
	}
}

func TestSplitPaneRejectsInvalidInput(t *testing.T) {
	state := session.NewState()
	if _, err := state.SplitPane(session.SplitPane{SessionID: "missing"}); err == nil {
		t.Fatalf("expected missing session error")
	}
	_, err := state.CreateSession(session.CreateSession{SessionID: "session", WindowID: "win", PaneID: "pane", RootDir: "/repo"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	tests := []struct {
		name string
		req  session.SplitPane
	}{
		{name: "missing window", req: session.SplitPane{SessionID: "session", TargetPaneID: "pane", NewPaneID: "pane_2"}},
		{name: "missing target", req: session.SplitPane{SessionID: "session", WindowID: "win", TargetPaneID: "missing", NewPaneID: "pane_2"}},
		{name: "missing new pane", req: session.SplitPane{SessionID: "session", WindowID: "win", TargetPaneID: "pane"}},
		{name: "duplicate pane", req: session.SplitPane{SessionID: "session", WindowID: "win", TargetPaneID: "pane", NewPaneID: "pane"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := state.SplitPane(test.req); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}
