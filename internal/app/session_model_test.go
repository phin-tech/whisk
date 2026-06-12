package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/ptybookmark"
	"github.com/phin-tech/whisk/internal/domain/session"
)

func TestRuntimeCreateSessionWithoutInitialPTYCreatesEmptyPane(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	runtime := app.NewRuntime(app.RuntimeConfig{})

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:    "Whisk",
		RootDir: rootDir,
	})

	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.PTYID != nil {
		t.Fatalf("expected no initial pty, got %#v", created.PTYID)
	}
	if created.WindowID == "" || created.PaneID == "" {
		t.Fatalf("created missing window/pane ids: %#v", created)
	}
	pane := created.Session.Panes[created.PaneID]
	if pane.CurrentPTYID != nil {
		t.Fatalf("expected empty pane, got %#v", pane.CurrentPTYID)
	}
	if pane.WorkingDir != rootDir {
		t.Fatalf("pane working dir = %q, want %q", pane.WorkingDir, rootDir)
	}
}

func TestRuntimeStartPanePTYSpawnsIntoExistingEmptyPane(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	ctx := context.Background()
	rootDir := t.TempDir()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:    "Whisk",
		RootDir: rootDir,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	started, err := runtime.StartPanePTY(ctx, app.StartPanePTYRequest{
		SessionID: created.Session.ID,
		PaneID:    created.PaneID,
		Options:   app.StartPTYOptions{Cols: 90, Rows: 30},
	})

	if err != nil {
		t.Fatalf("start pane pty: %v", err)
	}
	if started.PTYID == "" {
		t.Fatalf("missing pty id: %#v", started)
	}
	pane := started.Session.Panes[created.PaneID]
	if pane.CurrentPTYID == nil || *pane.CurrentPTYID != started.PTYID {
		t.Fatalf("pane current pty = %#v, started = %#v", pane.CurrentPTYID, started)
	}
	ptys, err := runtime.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys: %v", err)
	}
	if len(ptys) != 1 || ptys[0].ID != started.PTYID || ptys[0].WorkingDir != rootDir {
		t.Fatalf("ptys = %#v", ptys)
	}
}

func TestRuntimeInjectsWhiskContextIntoSessionPTYs(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	rootDir := t.TempDir()
	ptyBackend := newMemoryPTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: ptyBackend,
		DaemonURL:  "http://127.0.0.1:8787",
		CLIPath:    "/usr/local/bin/whisk",
	})

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    rootDir,
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if got := ptyBackend.spawns[0].Env; got["WHISKD_URL"] != "http://127.0.0.1:8787" ||
		got["WHISK_CLI"] != "/usr/local/bin/whisk" ||
		got["PATH"] != "/usr/local/bin:/usr/bin:/bin" ||
		got["WHISK_SESSION_ID"] != created.Session.ID ||
		got["WHISK_PTY_ID"] != created.MainPtyID {
		t.Fatalf("create session env = %#v", got)
	}

	split, err := runtime.SplitPane(ctx, app.SplitPaneRequest{
		SessionID:    created.Session.ID,
		WindowID:     created.WindowID,
		TargetPaneID: created.PaneID,
		Direction:    session.SplitHorizontal,
		InitialPTY:   &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("split pane: %v", err)
	}
	if got := ptyBackend.spawns[1].Env; got["WHISK_SESSION_ID"] != created.Session.ID || got["WHISK_PTY_ID"] != split.PtyID {
		t.Fatalf("split env = %#v", got)
	}

	empty, err := runtime.CreateSession(ctx, app.CreateSessionRequest{Name: "Empty", RootDir: rootDir})
	if err != nil {
		t.Fatalf("create empty session: %v", err)
	}
	started, err := runtime.StartPanePTY(ctx, app.StartPanePTYRequest{
		SessionID: empty.Session.ID,
		PaneID:    empty.PaneID,
		Options:   app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("start pane pty: %v", err)
	}
	if got := ptyBackend.spawns[2].Env; got["WHISK_SESSION_ID"] != empty.Session.ID || got["WHISK_PTY_ID"] != started.PTYID {
		t.Fatalf("start env = %#v", got)
	}

	if _, err := runtime.KillPTY(ctx, app.KillPTYRequest{PTYID: started.PTYID}); err != nil {
		t.Fatalf("kill pty: %v", err)
	}
	restarted, err := runtime.RestartPanePTY(ctx, app.RestartPanePTYRequest{
		SessionID: empty.Session.ID,
		PaneID:    empty.PaneID,
		Options:   app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("restart pane pty: %v", err)
	}
	if got := ptyBackend.spawns[3].Env; got["WHISK_SESSION_ID"] != empty.Session.ID || got["WHISK_PTY_ID"] != restarted.PTYID {
		t.Fatalf("restart env = %#v", got)
	}
}

func TestRuntimePrependsWhiskCLIToExistingPTYPath(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	ptyBackend := newMemoryPTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: ptyBackend,
		CLIPath:    "/opt/whisk/bin/whisk",
	})

	_, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:    "Whisk",
		RootDir: rootDir,
		InitialPTY: &app.StartPTYOptions{
			Cols: 80,
			Rows: 24,
			Env:  map[string]string{"PATH": "/usr/bin:/bin"},
		},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if got := ptyBackend.spawns[0].Env["PATH"]; got != "/opt/whisk/bin:/usr/bin:/bin" {
		t.Fatalf("PATH = %q", got)
	}
}

func TestRuntimeSetRootDirRejectsRunningPTYAndValidatesFilesystem(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	ctx := context.Background()
	rootDir := t.TempDir()
	nextRoot := t.TempDir()
	filePath := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    rootDir,
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := runtime.SetSessionRootDir(ctx, app.SetSessionRootDirRequest{
		SessionID: created.Session.ID,
		RootDir:   nextRoot,
	}); err == nil {
		t.Fatalf("expected running pty root change error")
	}

	empty, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:    "Empty",
		RootDir: rootDir,
	})
	if err != nil {
		t.Fatalf("create empty session: %v", err)
	}
	if _, err := runtime.SetSessionRootDir(ctx, app.SetSessionRootDirRequest{
		SessionID: empty.Session.ID,
		RootDir:   filePath,
	}); err == nil {
		t.Fatalf("expected file root error")
	}
	updated, err := runtime.SetSessionRootDir(ctx, app.SetSessionRootDirRequest{
		SessionID: empty.Session.ID,
		RootDir:   nextRoot,
	})
	if err != nil {
		t.Fatalf("set root dir: %v", err)
	}
	if updated.RootDir != nextRoot || updated.Panes[empty.PaneID].WorkingDir != nextRoot {
		t.Fatalf("updated = %#v", updated)
	}
}

func TestRuntimeSetPaneWorkingDirRejectsRunningPTYAndValidatesFilesystem(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	ctx := context.Background()
	rootDir := t.TempDir()
	nextDir := t.TempDir()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    rootDir,
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := runtime.SetPaneWorkingDir(ctx, app.SetPaneWorkingDirRequest{
		SessionID:  created.Session.ID,
		PaneID:     created.PaneID,
		WorkingDir: nextDir,
	}); err == nil {
		t.Fatalf("expected running pty working dir error")
	}

	empty, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:    "Empty",
		RootDir: rootDir,
	})
	if err != nil {
		t.Fatalf("create empty session: %v", err)
	}
	updated, err := runtime.SetPaneWorkingDir(ctx, app.SetPaneWorkingDirRequest{
		SessionID:  empty.Session.ID,
		PaneID:     empty.PaneID,
		WorkingDir: nextDir,
	})
	if err != nil {
		t.Fatalf("set pane working dir: %v", err)
	}
	if updated.Panes[empty.PaneID].WorkingDir != nextDir {
		t.Fatalf("pane = %#v", updated.Panes[empty.PaneID])
	}
}

func TestRuntimeClosePaneRejectsCurrentPTYAndClosesEmptySplitPane(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	ctx := context.Background()
	rootDir := t.TempDir()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    rootDir,
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	split, err := runtime.SplitPane(ctx, app.SplitPaneRequest{
		SessionID:    created.Session.ID,
		WindowID:     created.WindowID,
		TargetPaneID: created.PaneID,
		Direction:    session.SplitHorizontal,
	})
	if err != nil {
		t.Fatalf("split pane: %v", err)
	}
	if _, err := runtime.ClosePane(ctx, app.ClosePaneRequest{
		SessionID: created.Session.ID,
		WindowID:  created.WindowID,
		PaneID:    created.PaneID,
	}); err == nil {
		t.Fatalf("expected close current pty error")
	}
	updated, err := runtime.ClosePane(ctx, app.ClosePaneRequest{
		SessionID: created.Session.ID,
		WindowID:  created.WindowID,
		PaneID:    split.PaneID,
	})
	if err != nil {
		t.Fatalf("close empty pane: %v", err)
	}
	if _, ok := updated.Panes[split.PaneID]; ok {
		t.Fatalf("closed pane still present: %#v", updated.Panes)
	}
}

func TestRuntimeCloseSessionDeletesSessionKillsPTYsAndPersists(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	ctx := context.Background()
	store := &memorySessionStore{}
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend:   native.NewBackend(),
		SessionStore: store,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Close me",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	remaining, err := runtime.CloseSession(ctx, app.CloseSessionRequest{SessionID: created.Session.ID})
	if err != nil {
		t.Fatalf("close session: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining = %#v", remaining)
	}
	listed, err := runtime.ListSessions(ctx)
	if err != nil || len(listed) != 0 {
		t.Fatalf("sessions = %#v, %v", listed, err)
	}
	ptys, err := runtime.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys: %v", err)
	}
	if len(ptys) != 1 || ptys[0].ID != created.MainPtyID || ptys[0].Status != app.PTYStatusKilled || ptys[0].Running {
		t.Fatalf("ptys = %#v", ptys)
	}
	if len(store.saved) < 2 || len(store.saved[len(store.saved)-1]) != 0 {
		t.Fatalf("saved sessions = %#v", store.saved)
	}
}

func TestRuntimeDetachPanePTYKeepsPTYSessionOwnership(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	ctx := context.Background()
	rootDir := t.TempDir()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    rootDir,
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	detached, err := runtime.DetachPanePTY(ctx, app.DetachPanePTYRequest{
		SessionID: created.Session.ID,
		PaneID:    created.PaneID,
	})

	if err != nil {
		t.Fatalf("detach pane pty: %v", err)
	}
	if detached.PTYID != created.MainPtyID {
		t.Fatalf("detached = %#v, created = %#v", detached, created)
	}
	if detached.Session.Panes[created.PaneID].CurrentPTYID != nil {
		t.Fatalf("pane current pty = %#v", detached.Session.Panes[created.PaneID].CurrentPTYID)
	}
	ptys, err := runtime.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys: %v", err)
	}
	if len(ptys) != 1 {
		t.Fatalf("ptys = %#v", ptys)
	}
	if ptys[0].ID != created.MainPtyID || ptys[0].SessionID != created.Session.ID || ptys[0].PaneID != "" || ptys[0].WindowID != "" || ptys[0].OriginPaneID != created.PaneID {
		t.Fatalf("detached pty info = %#v", ptys[0])
	}
	if ptys[0].Status != app.PTYStatusRunning {
		t.Fatalf("pty status = %q", ptys[0].Status)
	}
}

func TestRuntimeKillPTYMarksStatusAndAllowsPaneRestart(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	ctx := context.Background()
	rootDir := t.TempDir()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    rootDir,
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	killed, err := runtime.KillPTY(ctx, app.KillPTYRequest{PTYID: created.MainPtyID})
	if err != nil {
		t.Fatalf("kill pty: %v", err)
	}
	if killed.Status != app.PTYStatusKilled || killed.Running {
		t.Fatalf("killed = %#v", killed)
	}

	restarted, err := runtime.RestartPanePTY(ctx, app.RestartPanePTYRequest{
		SessionID: created.Session.ID,
		PaneID:    created.PaneID,
		Options:   app.StartPTYOptions{Cols: 100, Rows: 40},
	})
	if err != nil {
		t.Fatalf("restart pane pty: %v", err)
	}
	if restarted.PTYID == "" || restarted.PTYID == created.MainPtyID {
		t.Fatalf("restarted = %#v", restarted)
	}
	pane := restarted.Session.Panes[created.PaneID]
	if pane.CurrentPTYID == nil || *pane.CurrentPTYID != restarted.PTYID {
		t.Fatalf("pane current pty = %#v", pane.CurrentPTYID)
	}
	ptys, err := runtime.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys: %v", err)
	}
	byID := map[string]app.PTYInfo{}
	for _, pty := range ptys {
		byID[pty.ID] = pty
	}
	if byID[created.MainPtyID].Status != app.PTYStatusKilled || byID[created.MainPtyID].PaneID != "" || byID[created.MainPtyID].OriginPaneID != created.PaneID {
		t.Fatalf("old pty = %#v", byID[created.MainPtyID])
	}
	if byID[restarted.PTYID].Status != app.PTYStatusRunning || byID[restarted.PTYID].PaneID != created.PaneID {
		t.Fatalf("new pty = %#v", byID[restarted.PTYID])
	}
}

func TestRuntimeRestartPanePTYRejectsRunningCurrentPTY(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	ctx := context.Background()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := runtime.RestartPanePTY(ctx, app.RestartPanePTYRequest{
		SessionID: created.Session.ID,
		PaneID:    created.PaneID,
		Options:   app.StartPTYOptions{Cols: 80, Rows: 24},
	}); err == nil {
		t.Fatalf("expected running pty restart error")
	}
}

func TestRuntimeCreateSessionWithInitialPTYSpawnsIntoDefaultPane(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	ctx := context.Background()
	rootDir := t.TempDir()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:    "Whisk",
		RootDir: rootDir,
		InitialPTY: &app.StartPTYOptions{
			Cols: 80,
			Rows: 24,
		},
	})

	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.PTYID == nil || *created.PTYID == "" {
		t.Fatalf("created missing pty id: %#v", created)
	}
	pane := created.Session.Panes[created.PaneID]
	if pane.CurrentPTYID == nil || *pane.CurrentPTYID != *created.PTYID {
		t.Fatalf("pane pty = %#v, created pty = %#v", pane.CurrentPTYID, created.PTYID)
	}
	ptys, err := runtime.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys: %v", err)
	}
	if len(ptys) != 1 || ptys[0].ID != *created.PTYID || ptys[0].SessionID != created.Session.ID || ptys[0].PaneID != created.PaneID || ptys[0].WindowID != created.WindowID {
		t.Fatalf("ptys = %#v", ptys)
	}
}

func TestRuntimeCreateSessionRejectsMissingOrInvalidRootDir(t *testing.T) {
	ctx := context.Background()
	runtime := app.NewRuntime(app.RuntimeConfig{})

	tests := []struct {
		name    string
		rootDir string
	}{
		{name: "missing", rootDir: ""},
		{name: "relative", rootDir: "repo"},
		{name: "not exists", rootDir: "/definitely/not/a/real/whisk/root"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := runtime.CreateSession(ctx, app.CreateSessionRequest{RootDir: test.rootDir}); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestRuntimeLoadsPersistedSessionsWithoutLivePTYReferences(t *testing.T) {
	ptyID := "pty_01"
	store := &memorySessionStore{
		loaded: []session.Session{
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
		},
	}
	runtime, err := app.NewRuntimeWithError(app.RuntimeConfig{SessionStore: store})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	sessions, err := runtime.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "sess_01" {
		t.Fatalf("sessions = %#v", sessions)
	}
	if sessions[0].Panes["pane_01"].CurrentPTYID != nil {
		t.Fatalf("restored live pty reference = %#v", sessions[0].Panes["pane_01"].CurrentPTYID)
	}
}

func TestRuntimePersistsSessionMutations(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	store := &memorySessionStore{}
	runtime, err := app.NewRuntimeWithError(app.RuntimeConfig{SessionStore: store})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{Name: "Persist", RootDir: rootDir})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if len(store.saved) != 1 || store.saved[0][0].ID != created.Session.ID {
		t.Fatalf("saved after create = %#v", store.saved)
	}

	_, err = runtime.SplitPane(ctx, app.SplitPaneRequest{
		SessionID:    created.Session.ID,
		WindowID:     created.WindowID,
		TargetPaneID: created.PaneID,
		Direction:    session.SplitHorizontal,
	})
	if err != nil {
		t.Fatalf("split pane: %v", err)
	}
	if len(store.saved) != 2 || len(store.saved[1][0].Panes) != 2 {
		t.Fatalf("saved after split = %#v", store.saved)
	}
}

func TestRuntimeCapturesPTYTranscriptOutput(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	transcripts := &memoryTranscriptStore{}
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend:      native.NewBackend(),
		TranscriptStore: transcripts,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Transcript",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if len(transcripts.registered) != 1 || transcripts.registered[0].PTYID != created.MainPtyID {
		t.Fatalf("registered transcripts = %#v", transcripts.registered)
	}
	if err := runtime.WritePTY(ctx, created.MainPtyID, []byte("printf 'transcript-ok\\n'\n")); err != nil {
		t.Fatalf("write pty: %v", err)
	}
	for !strings.Contains(transcripts.outputString(created.MainPtyID), "transcript-ok") {
		select {
		case <-time.After(20 * time.Millisecond):
		case <-ctx.Done():
			t.Fatalf("timed out waiting for transcript output; got %q", transcripts.outputString(created.MainPtyID))
		}
	}
}

func TestRuntimeAddsListsPersistsAndRemovesPTYBookmarks(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	ctx := context.Background()
	store := &memoryBookmarkStore{}
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend:    native.NewBackend(),
		BookmarkStore: store,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Bookmarks",
		RootDir:    t.TempDir(),
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	bookmark, err := runtime.AddPTYBookmark(ctx, app.AddPTYBookmarkRequest{
		PTYID:  created.MainPtyID,
		Offset: 42,
		Kind:   "prompt",
		Label:  "Last prompt",
	})
	if err != nil {
		t.Fatalf("add bookmark: %v", err)
	}
	if bookmark.PTYID != created.MainPtyID || bookmark.SessionID != created.Session.ID || bookmark.PaneID != created.PaneID || bookmark.Offset != 42 {
		t.Fatalf("bookmark = %#v", bookmark)
	}
	if len(store.saved) != 1 || store.saved[0][0].ID != bookmark.ID {
		t.Fatalf("saved bookmarks = %#v", store.saved)
	}
	listed, err := runtime.ListPTYBookmarks(ctx, created.MainPtyID)
	if err != nil || len(listed) != 1 || listed[0].ID != bookmark.ID {
		t.Fatalf("listed = %#v, %v", listed, err)
	}
	if err := runtime.RemovePTYBookmark(ctx, app.RemovePTYBookmarkRequest{BookmarkID: bookmark.ID}); err != nil {
		t.Fatalf("remove bookmark: %v", err)
	}
	listed, err = runtime.ListPTYBookmarks(ctx, created.MainPtyID)
	if err != nil || len(listed) != 0 {
		t.Fatalf("listed after remove = %#v, %v", listed, err)
	}
	if len(store.saved) != 2 || len(store.saved[1]) != 0 {
		t.Fatalf("saved after remove = %#v", store.saved)
	}
}

func TestRuntimeLoadsPersistedPTYBookmarks(t *testing.T) {
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	store := &memoryBookmarkStore{
		loaded: []ptybookmark.Bookmark{
			{ID: "bm_01", PTYID: "pty_01", Offset: 7, CreatedAt: createdAt},
		},
	}
	runtime, err := app.NewRuntimeWithError(app.RuntimeConfig{BookmarkStore: store})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	bookmarks, err := runtime.ListPTYBookmarks(context.Background(), "pty_01")
	if err != nil || len(bookmarks) != 1 || bookmarks[0].ID != "bm_01" {
		t.Fatalf("bookmarks = %#v, %v", bookmarks, err)
	}
}

type memorySessionStore struct {
	loaded []session.Session
	saved  [][]session.Session
}

func (s *memorySessionStore) LoadSessions(context.Context) ([]session.Session, error) {
	return cloneTestSessions(s.loaded), nil
}

func (s *memorySessionStore) SaveSessions(_ context.Context, sessions []session.Session) error {
	s.saved = append(s.saved, cloneTestSessions(sessions))
	return nil
}

func cloneTestSessions(in []session.Session) []session.Session {
	out := make([]session.Session, len(in))
	for i, current := range in {
		current.Windows = cloneTestWindows(current.Windows)
		current.Panes = cloneTestPanes(current.Panes)
		out[i] = current
	}
	return out
}

func cloneTestWindows(in map[string]session.SessionWindow) map[string]session.SessionWindow {
	out := make(map[string]session.SessionWindow, len(in))
	for id, window := range in {
		out[id] = window
	}
	return out
}

func cloneTestPanes(in map[string]session.Pane) map[string]session.Pane {
	out := make(map[string]session.Pane, len(in))
	for id, pane := range in {
		if pane.CurrentPTYID != nil {
			ptyID := *pane.CurrentPTYID
			pane.CurrentPTYID = &ptyID
		}
		out[id] = pane
	}
	return out
}

type memoryBookmarkStore struct {
	loaded []ptybookmark.Bookmark
	saved  [][]ptybookmark.Bookmark
}

func (s *memoryBookmarkStore) LoadBookmarks(context.Context) ([]ptybookmark.Bookmark, error) {
	return append([]ptybookmark.Bookmark(nil), s.loaded...), nil
}

func (s *memoryBookmarkStore) SaveBookmarks(_ context.Context, bookmarks []ptybookmark.Bookmark) error {
	s.saved = append(s.saved, append([]ptybookmark.Bookmark(nil), bookmarks...))
	return nil
}

type memoryTranscriptStore struct {
	mu         sync.Mutex
	registered []app.PTYTranscriptMeta
	outputs    map[string][]byte
	exits      []app.PTYTranscriptExit
}

func (s *memoryTranscriptStore) RegisterPTY(_ context.Context, meta app.PTYTranscriptMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registered = append(s.registered, meta)
	return nil
}

func (s *memoryTranscriptStore) AppendPTYOutput(_ context.Context, event app.PTYTranscriptOutput) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.outputs == nil {
		s.outputs = map[string][]byte{}
	}
	s.outputs[event.PTYID] = append(s.outputs[event.PTYID], event.Bytes...)
	return nil
}

func (s *memoryTranscriptStore) MarkPTYExit(_ context.Context, event app.PTYTranscriptExit) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exits = append(s.exits, event)
	return nil
}

func (s *memoryTranscriptStore) outputString(ptyID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return string(s.outputs[ptyID])
}
