package client_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/agenthooklog"
	"github.com/phin-tech/whisk/internal/adapters/agenthooks"
	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/adapters/transcriptstore"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPClientDrivesDaemonRuntime(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")

	transcripts, err := transcriptstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("new transcript store: %v", err)
	}
	if err := transcripts.RegisterPTY(context.Background(), app.PTYTranscriptMeta{
		PTYID:      "pty_history",
		SessionID:  "sess_history",
		WindowID:   "win_history",
		PaneID:     "pane_history",
		WorkingDir: "/repo",
		Cols:       80,
		Rows:       24,
	}); err != nil {
		t.Fatalf("register transcript: %v", err)
	}
	if err := transcripts.AppendPTYOutput(context.Background(), app.PTYTranscriptOutput{PTYID: "pty_history", Bytes: []byte("client saved output")}); err != nil {
		t.Fatalf("append transcript: %v", err)
	}
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend(), EventSink: newFakeEventBus(), TranscriptStore: transcripts})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := daemon.Health(ctx); err != nil {
		t.Fatalf("health: %v", err)
	}
	compatibility, err := daemon.Compatibility(ctx)
	if err != nil {
		t.Fatalf("compatibility: %v", err)
	}
	if compatibility.APIVersion != protocol.DaemonAPIVersion || compatibility.GitSHA == "" {
		t.Fatalf("compatibility = %#v", compatibility)
	}
	if compatibility.ProtocolVersion != protocol.ProtocolVersion {
		t.Fatalf("compatibility missing protocol version: %#v", compatibility)
	}
	if compatibility.Version == "" {
		t.Fatalf("compatibility missing version: %#v", compatibility)
	}
	agentEvent, err := daemon.RecordAgentHookEvent(ctx, protocol.AgentBridgeHookRequest{
		Provider:  "claude",
		EventName: "Notification",
		Message:   "Need input.",
	})
	if err != nil {
		t.Fatalf("record agent hook event: %v", err)
	}
	readAgentEvent, err := daemon.MarkAgentBridgeEventRead(ctx, protocol.MarkAgentBridgeEventReadRequest{ID: agentEvent.ID})
	if err != nil || readAgentEvent.Status != "read" {
		t.Fatalf("mark agent event read = %#v, err = %v", readAgentEvent, err)
	}
	pendingAgentEvents, err := daemon.ListAgentBridgeEvents(ctx, protocol.ListAgentBridgeEventsRequest{Status: "pending"})
	if err != nil || len(pendingAgentEvents) != 0 {
		t.Fatalf("pending agent events = %#v, err = %v", pendingAgentEvents, err)
	}
	for range 2 {
		event, err := daemon.NextEvent(ctx, protocol.NextEventRequest{TimeoutMs: 10})
		if err != nil || event.Event.Type != "agent_hook_events.changed" || event.Event.Seq == 0 {
			t.Fatalf("agent hook event = %#v, err = %v", event, err)
		}
	}

	created, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.Session.ID == "" || created.WindowID == "" || created.PaneID == "" || created.PTYID == nil || created.MainPtyID == "" {
		t.Fatalf("created session missing ids: %#v", created)
	}

	sessions, err := daemon.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != created.Session.ID {
		t.Fatalf("sessions = %#v", sessions)
	}

	ptys, err := daemon.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys: %v", err)
	}
	if len(ptys) != 1 || ptys[0].ID != created.MainPtyID || ptys[0].SessionID != created.Session.ID {
		t.Fatalf("ptys = %#v", ptys)
	}
	history, err := daemon.ListPTYHistory(ctx)
	if err != nil || len(history) == 0 {
		t.Fatalf("pty history = %#v, err = %v", history, err)
	}
	selectedHistory, err := daemon.ReadPTYHistory(ctx, "pty_history")
	if err != nil || selectedHistory.Output != "client saved output" || selectedHistory.SessionID != "sess_history" {
		t.Fatalf("selected pty history = %#v, err = %v", selectedHistory, err)
	}
	sessionRoot := t.TempDir()
	sessionWorkingDir := t.TempDir()
	separateDir, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:       "Separate",
		RootDir:    sessionRoot,
		WorkingDir: sessionWorkingDir,
	})
	if err != nil {
		t.Fatalf("create session with working dir: %v", err)
	}
	if separateDir.Session.RootDir != sessionRoot || separateDir.Session.Panes[separateDir.PaneID].WorkingDir != sessionWorkingDir {
		t.Fatalf("separate dir session = %#v", separateDir.Session)
	}
	closeViaClient, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:       "Close via client",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create close via client session: %v", err)
	}
	remaining, err := daemon.CloseSession(ctx, protocol.CloseSessionRequest{SessionID: closeViaClient.Session.ID})
	if err != nil {
		t.Fatalf("close session: %v", err)
	}
	for _, session := range remaining {
		if session.ID == closeViaClient.Session.ID {
			t.Fatalf("closed session still present: %#v", remaining)
		}
	}

	event, err := daemon.NextEvent(ctx, protocol.NextEventRequest{TimeoutMs: 10})
	if err != nil {
		t.Fatalf("next event: %v", err)
	}
	if event.Event.Type != "session.changed" || event.Event.Seq == 0 {
		t.Fatalf("event = %#v", event)
	}

	split, err := daemon.SplitPane(ctx, protocol.SplitPaneRequest{
		SessionID:    created.Session.ID,
		WindowID:     created.WindowID,
		TargetPaneID: created.PaneID,
		Direction:    "horizontal",
		InitialPTY:   &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("split pane: %v", err)
	}
	if split.PaneID == "" || split.PtyID == "" || split.PTYID == nil {
		t.Fatalf("split result = %#v", split)
	}

	emptyRoot := t.TempDir()
	empty, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:    "Empty",
		RootDir: emptyRoot,
	})
	if err != nil {
		t.Fatalf("create empty session: %v", err)
	}
	nextRoot := t.TempDir()
	updated, err := daemon.SetSessionRootDir(ctx, protocol.SetSessionRootDirRequest{
		SessionID: empty.Session.ID,
		RootDir:   nextRoot,
	})
	if err != nil || updated.RootDir != nextRoot {
		t.Fatalf("set root = %#v, %v", updated, err)
	}
	paneDir := t.TempDir()
	updated, err = daemon.SetPaneWorkingDir(ctx, protocol.SetPaneWorkingDirRequest{
		SessionID:  empty.Session.ID,
		PaneID:     empty.PaneID,
		WorkingDir: paneDir,
	})
	if err != nil || updated.Panes[empty.PaneID].WorkingDir != paneDir {
		t.Fatalf("set pane dir = %#v, %v", updated.Panes[empty.PaneID], err)
	}
	started, err := daemon.StartPanePTY(ctx, protocol.StartPanePTYRequest{
		SessionID: empty.Session.ID,
		PaneID:    empty.PaneID,
		Options:   protocol.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil || started.PTYID == "" {
		t.Fatalf("start pane pty = %#v, %v", started, err)
	}
	detached, err := daemon.DetachPanePTY(ctx, protocol.DetachPanePTYRequest{
		SessionID: empty.Session.ID,
		PaneID:    empty.PaneID,
	})
	if err != nil || detached.PTYID != started.PTYID {
		t.Fatalf("detach pane pty = %#v, %v", detached, err)
	}
	ptys, err = daemon.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys after detach: %v", err)
	}
	var detachedInfo protocol.PTYInfo
	for _, pty := range ptys {
		if pty.ID == detached.PTYID {
			detachedInfo = pty
		}
	}
	if detachedInfo.SessionID != empty.Session.ID || detachedInfo.PaneID != "" || detachedInfo.OriginPaneID != empty.PaneID || detachedInfo.Status != "running" {
		t.Fatalf("detached pty info = %#v", detachedInfo)
	}
	restartSession, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:       "Restart",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create restart session: %v", err)
	}
	killed, err := daemon.KillPTY(ctx, protocol.KillPTYRequest{PTYID: restartSession.MainPtyID})
	if err != nil || killed.Status != "killed" || killed.Running {
		t.Fatalf("kill pty = %#v, %v", killed, err)
	}
	restarted, err := daemon.RestartPanePTY(ctx, protocol.RestartPanePTYRequest{
		SessionID: restartSession.Session.ID,
		PaneID:    restartSession.PaneID,
		Options:   protocol.StartPTYOptions{Cols: 100, Rows: 40},
	})
	if err != nil || restarted.OldPTYID != restartSession.MainPtyID || restarted.PTYID == "" {
		t.Fatalf("restart pane pty = %#v, %v", restarted, err)
	}
	restartedPane := restarted.Session.Panes[restartSession.PaneID]
	if restartedPane.CurrentPTYID == nil || *restartedPane.CurrentPTYID != restarted.PTYID {
		t.Fatalf("restarted pane = %#v", restartedPane)
	}
	closeSession, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:    "Close",
		RootDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create close session: %v", err)
	}
	emptySplit, err := daemon.SplitPane(ctx, protocol.SplitPaneRequest{
		SessionID:    closeSession.Session.ID,
		WindowID:     closeSession.WindowID,
		TargetPaneID: closeSession.PaneID,
		Direction:    "horizontal",
	})
	if err != nil {
		t.Fatalf("split empty pane: %v", err)
	}
	closed, err := daemon.ClosePane(ctx, protocol.ClosePaneRequest{
		SessionID: closeSession.Session.ID,
		WindowID:  closeSession.WindowID,
		PaneID:    emptySplit.PaneID,
	})
	if err != nil {
		t.Fatalf("close pane: %v", err)
	}
	if _, ok := closed.Panes[emptySplit.PaneID]; ok {
		t.Fatalf("closed pane still present: %#v", closed.Panes)
	}

	if err := daemon.ResizePTY(ctx, protocol.ResizePTYRequest{PtyID: created.MainPtyID, Cols: 73, Rows: 17}); err != nil {
		t.Fatalf("resize pty: %v", err)
	}
	if err := daemon.WritePTY(ctx, protocol.WritePTYRequest{PtyID: created.MainPtyID, Data: "printf 'daemon-http-ok\\n'\n"}); err != nil {
		t.Fatalf("write pty: %v", err)
	}

	var offset uint64
	var output strings.Builder
	for !strings.Contains(output.String(), "daemon-http-ok") {
		snapshot, err := daemon.Output(ctx, protocol.OutputRequest{PtyID: created.MainPtyID, FromOffset: offset})
		if err != nil {
			t.Fatalf("output: %v", err)
		}
		offset = snapshot.Offset
		output.WriteString(snapshot.Output)
		if strings.Contains(output.String(), "daemon-http-ok") {
			break
		}
		select {
		case <-time.After(20 * time.Millisecond):
		case <-ctx.Done():
			t.Fatalf("timed out waiting for output; got %q", output.String())
		}
	}

	cleared, err := daemon.ClearDaemon(ctx, protocol.ClearDaemonRequest{})
	if err != nil {
		t.Fatalf("clear daemon: %v", err)
	}
	if cleared.SessionsCleared == 0 || cleared.PTYsCleared == 0 {
		t.Fatalf("cleared = %#v", cleared)
	}
	sessions, err = daemon.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions after clear: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions after clear = %#v", sessions)
	}
	ptys, err = daemon.ListPTYs(ctx)
	if err != nil {
		t.Fatalf("list ptys after clear: %v", err)
	}
	if len(ptys) != 0 {
		t.Fatalf("ptys after clear = %#v", ptys)
	}

	afterClear, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{
		Name:    "After clear",
		RootDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create after clear: %v", err)
	}
	if afterClear.Session.ID == "" {
		t.Fatalf("after clear session = %#v", afterClear)
	}
}

func TestHTTPClientDrivesPluginAPI(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{Plugins: &pluginRegistryFake{statuses: []app.PluginStatus{{ID: "github", Name: "GitHub", Valid: true}}}})
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	plugins, err := daemon.ListPlugins(ctx)
	if err != nil || len(plugins) != 1 || plugins[0].ID != "github" {
		t.Fatalf("plugins = %#v, err = %v", plugins, err)
	}
	registry, err := daemon.ListRegistryPlugins(ctx)
	if err != nil || len(registry) != 1 || registry[0].Registry != "phin-tech" || registry[0].SourceType != "path" {
		t.Fatalf("registry = %#v, err = %v", registry, err)
	}
	installed, err := daemon.InstallPlugin(ctx, "phin-tech", "github")
	if err != nil || installed.ID != "github" || installed.Registry != "phin-tech" || installed.Trusted {
		t.Fatalf("install = %#v, err = %v", installed, err)
	}
	trusted, err := daemon.TrustPlugin(ctx, "github")
	if err != nil || !trusted.Trusted {
		t.Fatalf("trust = %#v, err = %v", trusted, err)
	}
	project, err := daemon.CreateProject(ctx, protocol.CreateProjectRequest{Name: "App", RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	attached, err := daemon.RunPluginProjectAttachmentTemplate(ctx, "github", "github.issue", protocol.RunPluginProjectAttachmentTemplateRequest{
		ProjectID: project.ID,
		Values:    map[string]string{"repo": "owner/repo", "issue": "1"},
	})
	if err != nil || len(attached.Attachments) != 1 || attached.Attachments[0].Provider != "github" {
		t.Fatalf("attached = %#v, err = %v", attached.Attachments, err)
	}
}

func TestHTTPClientReturnsNormalizedAgentHookEventMetadata(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{EventSink: newFakeEventBus()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event, err := daemon.RecordAgentHookEvent(ctx, protocol.AgentBridgeHookRequest{
		Provider:  "codex",
		EventName: "UserPromptSubmit",
		Message:   "Implement the feature.",
		RawPayload: map[string]any{
			"session_id": "codex_session_01",
			"whisk": map[string]any{
				"sessionId": "whisk_session_01",
				"ptyId":     "pty_01",
				"cwd":       "/repo",
				"agent":     "codex",
			},
		},
	})
	if err != nil {
		t.Fatalf("record agent hook event: %v", err)
	}

	if event.Kind != "prompt" ||
		event.Title != "Codex prompt" ||
		event.ProviderSessionID != "codex_session_01" ||
		event.SessionID != "whisk_session_01" ||
		event.PTYID != "pty_01" ||
		event.CWD != "/repo" ||
		event.Agent != "codex" ||
		event.Answerable {
		t.Fatalf("normalized agent hook event = %#v", event)
	}
}

func TestHTTPClientAgentHookIntegrations(t *testing.T) {
	paths := clientTestAgentHookPaths(t)
	runtime := app.NewRuntime(app.RuntimeConfig{AgentHookPaths: &paths})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)
	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx := context.Background()

	listed, err := daemon.ListAgentHookIntegrations(ctx)
	if err != nil {
		t.Fatalf("list integrations: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("listed = %#v", listed)
	}

	installed, err := daemon.InstallAgentHookIntegration(ctx, protocol.AgentHookIntegrationRequest{Provider: "claude"})
	if err != nil {
		t.Fatalf("install integration: %v", err)
	}
	if installed.Provider != "claude" || installed.Status != "current" {
		t.Fatalf("installed = %#v", installed)
	}
	checked, err := daemon.CheckAgentHookIntegration(ctx, protocol.AgentHookIntegrationRequest{Provider: "claude"})
	if err != nil || checked.Provider != "claude" || checked.Status != "current" {
		t.Fatalf("checked = %#v err=%v", checked, err)
	}
	removed, err := daemon.RemoveAgentHookIntegration(ctx, protocol.AgentHookIntegrationRequest{Provider: "claude"})
	if err != nil || removed.Provider != "claude" || removed.Status != "missing" {
		t.Fatalf("removed = %#v err=%v", removed, err)
	}
}

func TestHTTPClientAgentBridgeAndHookLog(t *testing.T) {
	tmp := t.TempDir()
	logPaths := agenthooklog.Paths{
		ConfigRoot: tmp,
		LogPath:    filepath.Join(tmp, "agent-hooks", "hooks.jsonl"),
	}
	runtime := app.NewRuntime(app.RuntimeConfig{AgentHookLogPaths: &logPaths})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)
	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx := context.Background()

	if daemon.BaseURL() != httpServer.URL {
		t.Fatalf("base url = %q, want %q", daemon.BaseURL(), httpServer.URL)
	}

	// Enable hook logging, then record an event so it is appended to the log on disk.
	enabled := true
	status, err := daemon.SetAgentHookLogSettings(ctx, protocol.SetAgentHookLogSettingsRequest{Enabled: &enabled})
	if err != nil || !status.Enabled {
		t.Fatalf("set hook log = %#v, err = %v", status, err)
	}

	event, err := daemon.RecordAgentHookEvent(ctx, protocol.AgentBridgeHookRequest{
		Provider:  "claude",
		EventName: "Notification",
		Message:   "build finished",
	})
	if err != nil || event.ID == "" {
		t.Fatalf("record event = %#v, err = %v", event, err)
	}

	events, err := daemon.ListAgentBridgeEvents(ctx, protocol.ListAgentBridgeEventsRequest{})
	if err != nil || len(events) != 1 || events[0].ID != event.ID {
		t.Fatalf("events = %#v, err = %v", events, err)
	}

	approvals, err := daemon.ListAgentBridgeApprovals(ctx, protocol.ListAgentBridgeApprovalsRequest{Status: "pending"})
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if len(approvals) != 0 {
		t.Fatalf("approvals = %#v, want none", approvals)
	}
	prompts, err := daemon.ListAgentPrompts(ctx, protocol.ListAgentPromptsRequest{Status: "pending"})
	if err != nil {
		t.Fatalf("list prompts: %v", err)
	}
	if len(prompts) != 0 {
		t.Fatalf("prompts = %#v, want none", prompts)
	}

	// Resolving an unknown approval and hooking an unknown bridge both surface daemon errors,
	// exercising the client error paths.
	if _, err := daemon.ResolveAgentBridgeApproval(ctx, "missing", protocol.ResolveAgentBridgeApprovalRequest{Action: "allow"}); err == nil {
		t.Fatalf("expected error resolving unknown approval")
	}
	if _, err := daemon.ResolveAgentPrompt(ctx, "missing", protocol.ResolveAgentPromptRequest{Answer: "ok"}); err == nil {
		t.Fatalf("expected error resolving unknown prompt")
	}
	if _, err := daemon.AgentBridgeHook(ctx, "bridge_missing", protocol.AgentBridgeHookRequest{Token: "nope"}); err == nil {
		t.Fatalf("expected unauthorized bridge hook error")
	}

	logStatus, err := daemon.AgentHookLogStatus(ctx)
	if err != nil || !logStatus.Enabled || logStatus.SizeBytes == 0 {
		t.Fatalf("log status = %#v, err = %v", logStatus, err)
	}

	cleared, err := daemon.ClearAgentHookLog(ctx)
	if err != nil || cleared.SizeBytes != 0 {
		t.Fatalf("cleared = %#v, err = %v", cleared, err)
	}
}

func TestHTTPClientExitPlanModeHookSubmitsDraftPlanForReview(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	nextID := 0
	ptyBackend := &clientMemoryPTYBackend{records: map[string]app.PTYRecord{}}
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		PTYBackend: ptyBackend,
		DaemonURL:  "http://127.0.0.1:8787",
		CLIPath:    "/usr/local/bin/whisk",
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)
	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx := context.Background()

	project, err := daemon.CreateProject(ctx, protocol.CreateProjectRequest{Name: "App", RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := daemon.CreateWorkItem(ctx, protocol.CreateWorkItemRequest{ProjectID: project.ID, Title: "Plan first"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	run, err := daemon.StartPlanning(ctx, protocol.StartPlanningRequest{
		WorkItemID:     item.ID,
		Launch:         true,
		AgentProfileID: "claude-plan",
		Actor:          "agent",
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	if run.PromptTemplateID != workitem.PromptTemplatePlan {
		t.Fatalf("run = %#v", run)
	}
	if len(ptyBackend.spawns) != 1 {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
	env := ptyBackend.spawns[0].Env
	bridgeID, token := env["WHISK_AGENT_BRIDGE_ID"], env["WHISK_AGENT_BRIDGE_TOKEN"]
	if bridgeID == "" || token == "" {
		t.Fatalf("missing bridge credentials: env = %#v", env)
	}

	planBody := "## Plan\nDo it."
	resp, err := daemon.AgentBridgeHook(ctx, bridgeID, protocol.AgentBridgeHookRequest{
		Token:     token,
		Provider:  "claude",
		EventName: "PreToolUse",
		ToolName:  "ExitPlanMode",
		ToolInput: map[string]any{"plan": planBody},
	})
	if err != nil {
		t.Fatalf("agent bridge hook: %v", err)
	}
	hookSpecific, ok := resp.Output["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("response = %#v", resp.Output)
	}
	reason, _ := hookSpecific["permissionDecisionReason"].(string)
	if hookSpecific["permissionDecision"] != "deny" ||
		!strings.Contains(reason, "Whisk planning run") ||
		!strings.Contains(reason, "submitted") ||
		!strings.Contains(reason, "review") {
		t.Fatalf("hookSpecificOutput = %#v", hookSpecific)
	}
	artifacts, err := daemon.ListArtifacts(ctx, item.ID)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) != 1 ||
		artifacts[0].Kind != workitem.ArtifactKindPlan ||
		artifacts[0].Status != workitem.ArtifactStatusDraft ||
		artifacts[0].Body != planBody ||
		artifacts[0].RunID != run.ID {
		t.Fatalf("artifacts = %#v", artifacts)
	}
}

func TestHTTPClientOpenAgentHookLog(t *testing.T) {
	// Use a stub server so the client method is exercised without the daemon actually shelling out
	// to open the log file.
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enabled":true,"path":"/tmp/hooks.jsonl","sizeBytes":0}`))
	}))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	status, err := daemon.OpenAgentHookLog(context.Background())
	if err != nil || !status.Enabled || status.Path != "/tmp/hooks.jsonl" {
		t.Fatalf("open hook log = %#v, err = %v", status, err)
	}
}

func TestHTTPClientCoversLightweightRouteWrappers(t *testing.T) {
	seen := map[string]bool{}
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.Method+" "+r.URL.RequestURI()] = true
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /v1/onboarding":
			_, _ = w.Write([]byte(`{"shouldShow":true,"localDaemon":true,"statePath":"/tmp/onboarding.json"}`))
		case "POST /v1/onboarding/apply":
			_, _ = w.Write([]byte(`{"shouldShow":false,"localDaemon":true,"statePath":"/tmp/onboarding.json"}`))
		case "POST /v1/plugins/rescan":
			_, _ = w.Write([]byte(`[{"id":"github","name":"GitHub","valid":true}]`))
		case "POST /v1/plugins/github/untrust":
			_, _ = w.Write([]byte(`{"id":"github","name":"GitHub","valid":true}`))
		case "DELETE /v1/ptys/pty_01":
			w.WriteHeader(http.StatusNoContent)
		case "POST /v1/projects/proj_01/delete":
			_, _ = w.Write([]byte(`{"id":"proj_01","name":"App","rootDir":"/repo"}`))
		case "POST /v1/projects/proj_01/attachments":
			_, _ = w.Write([]byte(`{"id":"proj_01","name":"App","rootDir":"/repo","attachments":[{"id":"att_01","kind":"note"}]}`))
		case "POST /v1/project-attachments/att_01/update":
			_, _ = w.Write([]byte(`{"id":"proj_01","name":"App","rootDir":"/repo","attachments":[{"id":"att_01","title":"Updated"}]}`))
		case "POST /v1/project-attachments/att_01/delete":
			_, _ = w.Write([]byte(`{"id":"proj_01","name":"App","rootDir":"/repo"}`))
		case "GET /v1/projects/proj_01/context":
			_, _ = w.Write([]byte(`{"projectId":"proj_01","items":[{"kind":"note","delivery":"inline","content":"Context"}]}`))
		case "POST /v1/workflow-definitions/validate-file":
			_, _ = w.Write([]byte(`{"valid":true,"identity":"workflow@1"}`))
		case "POST /v1/workflow-definitions/import-file":
			_, _ = w.Write([]byte(`{"id":"workflow","version":1,"sourcePath":"/tmp/workflow.json"}`))
		case "POST /v1/workflow-definitions/export-file":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx := context.Background()

	if status, err := daemon.OnboardingStatus(ctx); err != nil || !status.ShouldShow {
		t.Fatalf("onboarding status = %#v, err = %v", status, err)
	}
	if status, err := daemon.ApplyOnboarding(ctx, protocol.OnboardingApplyRequest{ItemIDs: []string{"daemon:version"}}); err != nil || status.ShouldShow {
		t.Fatalf("apply onboarding = %#v, err = %v", status, err)
	}
	if plugins, err := daemon.RescanPlugins(ctx); err != nil || len(plugins) != 1 {
		t.Fatalf("rescan plugins = %#v, err = %v", plugins, err)
	}
	if plugin, err := daemon.UntrustPlugin(ctx, "github"); err != nil || plugin.ID != "github" || plugin.Trusted {
		t.Fatalf("untrust plugin = %#v, err = %v", plugin, err)
	}
	if err := daemon.DeletePTY(ctx, protocol.DeletePTYRequest{PTYID: "pty_01"}); err != nil {
		t.Fatalf("delete pty: %v", err)
	}
	if project, err := daemon.DeleteProject(ctx, "proj_01", protocol.DeleteProjectRequest{Actor: "human"}); err != nil || project.ID != "proj_01" {
		t.Fatalf("delete project = %#v, err = %v", project, err)
	}
	if project, err := daemon.AddProjectAttachment(ctx, protocol.AddProjectAttachmentRequest{ProjectID: "proj_01", Kind: "note", Note: "Context"}); err != nil || len(project.Attachments) != 1 {
		t.Fatalf("add project attachment = %#v, err = %v", project, err)
	}
	title := "Updated"
	if project, err := daemon.UpdateProjectAttachment(ctx, "att_01", protocol.UpdateProjectAttachmentRequest{Title: &title}); err != nil || project.Attachments[0].Title != title {
		t.Fatalf("update project attachment = %#v, err = %v", project, err)
	}
	if project, err := daemon.DeleteProjectAttachment(ctx, "att_01", protocol.DeleteProjectAttachmentRequest{ProjectID: "proj_01"}); err != nil || len(project.Attachments) != 0 {
		t.Fatalf("delete project attachment = %#v, err = %v", project, err)
	}
	if contextBundle, err := daemon.GetProjectContext(ctx, "proj_01"); err != nil || contextBundle.ProjectID != "proj_01" {
		t.Fatalf("project context = %#v, err = %v", contextBundle, err)
	}
	if report, err := daemon.ValidateWorkflowDefinitionFile(ctx, protocol.ValidateWorkflowDefinitionFileRequest{Path: "/tmp/workflow.json"}); err != nil || !report.Valid {
		t.Fatalf("validate workflow file = %#v, err = %v", report, err)
	}
	if record, err := daemon.ImportWorkflowDefinitionFile(ctx, protocol.ImportWorkflowDefinitionFileRequest{Path: "/tmp/workflow.json"}); err != nil || record.SourcePath != "/tmp/workflow.json" {
		t.Fatalf("import workflow file = %#v, err = %v", record, err)
	}
	if err := daemon.ExportWorkflowDefinitionFile(ctx, protocol.ExportWorkflowDefinitionFileRequest{ID: "workflow", Version: 1, Path: "/tmp/out.json"}); err != nil {
		t.Fatalf("export workflow file: %v", err)
	}

	for _, key := range []string{
		"GET /v1/onboarding",
		"POST /v1/onboarding/apply",
		"POST /v1/plugins/rescan",
		"POST /v1/plugins/github/untrust",
		"DELETE /v1/ptys/pty_01",
		"POST /v1/projects/proj_01/delete",
		"POST /v1/projects/proj_01/attachments",
		"POST /v1/project-attachments/att_01/update",
		"POST /v1/project-attachments/att_01/delete",
		"GET /v1/projects/proj_01/context",
		"POST /v1/workflow-definitions/validate-file",
		"POST /v1/workflow-definitions/import-file",
		"POST /v1/workflow-definitions/export-file",
	} {
		if !seen[key] {
			t.Fatalf("missing request %s; seen = %#v", key, seen)
		}
	}
}

func TestHTTPClientNextEventTimeoutReturnsNoop(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend(), EventSink: newFakeEventBus()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	event, err := daemon.NextEvent(ctx, protocol.NextEventRequest{TimeoutMs: 1})
	if err != nil {
		t.Fatalf("next event: %v", err)
	}
	if event.Event.Type != protocol.RuntimeEventNone || event.Missed {
		t.Fatalf("event = %#v", event)
	}
}

func TestHTTPClientNextEventSendsCursorAndParsesMissed(t *testing.T) {
	eventBus := newFakeEventBus()
	runtime := app.NewRuntime(app.RuntimeConfig{EventSink: eventBus})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	eventBus.missed = true
	eventBus.ch <- app.RuntimeEvent{Seq: 8, Type: app.EventPTYOutput, PtyID: "pty_01", Offset: 42}
	event, err := daemon.NextEvent(ctx, protocol.NextEventRequest{TimeoutMs: 25, AfterSeq: 4})
	if err != nil {
		t.Fatalf("next event: %v", err)
	}
	if eventBus.afterSeq != 4 {
		t.Fatalf("after seq = %d, want 4", eventBus.afterSeq)
	}
	if !event.Missed || event.Event.Seq != 8 || event.Event.Type != "pty.output" || event.Event.PtyID != "pty_01" || event.Event.Offset != 42 {
		t.Fatalf("event = %#v", event)
	}
}

type fakeEventBus struct {
	ch       chan app.RuntimeEvent
	afterSeq uint64
	missed   bool
}

func newFakeEventBus() *fakeEventBus {
	return &fakeEventBus{ch: make(chan app.RuntimeEvent, 64)}
}

func (b *fakeEventBus) Publish(_ context.Context, event app.RuntimeEvent) error {
	b.ch <- event
	return nil
}

func (b *fakeEventBus) Next(ctx context.Context, afterSeq uint64) (app.NextRuntimeEventResult, error) {
	b.afterSeq = afterSeq
	select {
	case event := <-b.ch:
		return app.NextRuntimeEventResult{Event: event, Missed: b.missed}, nil
	case <-ctx.Done():
		return app.NextRuntimeEventResult{}, ctx.Err()
	}
}

type pluginRegistryFake struct {
	statuses []app.PluginStatus
}

func (f *pluginRegistryFake) ListPlugins(context.Context) ([]app.PluginStatus, error) {
	return f.statuses, nil
}

func (f *pluginRegistryFake) RescanPlugins(context.Context) ([]app.PluginStatus, error) {
	return f.statuses, nil
}

func (f *pluginRegistryFake) TrustPlugin(_ context.Context, id string) (app.PluginStatus, error) {
	status := app.PluginStatus{ID: id, Name: "GitHub", Valid: true, Trusted: true}
	f.statuses = []app.PluginStatus{status}
	return status, nil
}

func (f *pluginRegistryFake) UntrustPlugin(_ context.Context, id string) (app.PluginStatus, error) {
	status := app.PluginStatus{ID: id, Name: "GitHub", Valid: true}
	f.statuses = []app.PluginStatus{status}
	return status, nil
}

func (f *pluginRegistryFake) ListRegistryPlugins(context.Context) ([]app.RegistryPlugin, error) {
	return []app.RegistryPlugin{{Registry: "phin-tech", ID: "github", Name: "GitHub Issues", SourceType: "path", Installed: false}}, nil
}

func (f *pluginRegistryFake) InstallPlugin(_ context.Context, registry, id string) (app.PluginStatus, error) {
	status := app.PluginStatus{ID: id, Registry: registry, Name: "GitHub Issues", Valid: true}
	f.statuses = []app.PluginStatus{status}
	return status, nil
}

func (f *pluginRegistryFake) RunProjectAttachmentTemplate(_ context.Context, req app.RunPluginProjectAttachmentTemplateRequest) (app.AddProjectAttachmentRequest, error) {
	return app.AddProjectAttachmentRequest{
		ProjectID:        req.ProjectID,
		Kind:             workitem.AttachmentKindExternal,
		Provider:         "github",
		Target:           "owner/repo#1",
		URL:              "https://github.com/owner/repo/issues/1",
		Title:            "Issue",
		IncludeInContext: true,
	}, nil
}

func (f *pluginRegistryFake) ResolveProjectAttachmentProvider(string) app.ProjectContextResolver {
	return nil
}

func TestHTTPClientEnsureCompatibleAcceptsLegacyAPIVersion(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"apiVersion":%d,"gitSha":"abc"}`, protocol.ProtocolVersion)
	}))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	compatibility, err := daemon.EnsureCompatible(context.Background())
	if err != nil {
		t.Fatalf("ensure compatible: %v", err)
	}
	if compatibility.DaemonProtocolVersion() != protocol.ProtocolVersion {
		t.Fatalf("compatibility = %#v", compatibility)
	}
}

func TestHTTPClientEnsureCompatibleReturnsTypedError(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(
			w,
			`{"apiVersion":%d,"protocolVersion":%d,"gitSha":"abc"}`,
			protocol.ProtocolVersion+1,
			protocol.ProtocolVersion+1,
		)
	}))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	_, err := daemon.EnsureCompatible(context.Background())
	if err == nil {
		t.Fatalf("expected compatibility error")
	}
	var compatibilityErr *protocol.CompatibilityError
	if !errors.As(err, &compatibilityErr) {
		t.Fatalf("expected CompatibilityError, got %T", err)
	}
}

func TestHTTPClientReportsDaemonErrors(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":false}`))
		case "/v1/sessions":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad session"}`))
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`plain failure`))
		}
	}))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx := context.Background()

	if err := daemon.Health(ctx); err == nil || !strings.Contains(err.Error(), "health") {
		t.Fatalf("health error = %v", err)
	}
	if _, err := daemon.CreateSession(ctx, protocol.CreateSessionRequest{}); err == nil || !strings.Contains(err.Error(), "bad session") {
		t.Fatalf("create error = %v", err)
	}
	if _, err := daemon.Output(ctx, protocol.OutputRequest{PtyID: "missing"}); err == nil || !strings.Contains(err.Error(), "plain failure") {
		t.Fatalf("output error = %v", err)
	}
}

func clientTestAgentHookPaths(t *testing.T) agenthooks.Paths {
	t.Helper()
	root := t.TempDir()
	helperSource := filepath.Join(root, "whisk")
	if err := os.WriteFile(helperSource, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("write helper source: %v", err)
	}
	return agenthooks.Paths{
		ConfigRoot:         filepath.Join(root, ".config", "whisk"),
		HelperSourcePath:   helperSource,
		ClaudeSettingsPath: filepath.Join(root, ".claude", "settings.json"),
		CodexHooksPath:     filepath.Join(root, ".codex", "hooks.json"),
	}
}

type clientMemoryPTYBackend struct {
	spawns  []app.SpawnPTYRequest
	records map[string]app.PTYRecord
}

func (b *clientMemoryPTYBackend) Spawn(_ context.Context, req app.SpawnPTYRequest) (app.PTYRecord, error) {
	b.spawns = append(b.spawns, req)
	record := app.PTYRecord{
		ID:         req.ID,
		WorkingDir: req.WorkingDir,
		Cols:       req.Cols,
		Rows:       req.Rows,
		Running:    true,
	}
	b.records[record.ID] = record
	return record, nil
}

func (b *clientMemoryPTYBackend) Write(context.Context, string, []byte) error {
	return nil
}

func (b *clientMemoryPTYBackend) Resize(context.Context, string, app.PTYSize) error {
	return nil
}

func (b *clientMemoryPTYBackend) Kill(_ context.Context, ptyID string) (app.PTYRecord, error) {
	record, ok := b.records[ptyID]
	if !ok {
		return app.PTYRecord{}, fmt.Errorf("pty %s not found", ptyID)
	}
	record.Running = false
	b.records[ptyID] = record
	return record, nil
}

func (b *clientMemoryPTYBackend) Delete(_ context.Context, ptyID string) error {
	delete(b.records, ptyID)
	return nil
}

func (b *clientMemoryPTYBackend) Attach(context.Context, app.AttachPTYRequest) (*app.PTYAttach, error) {
	return nil, fmt.Errorf("attach unsupported")
}

func (b *clientMemoryPTYBackend) Output(_ context.Context, ptyID string, _ uint64) (app.PTYOutputSnapshot, error) {
	record, ok := b.records[ptyID]
	if !ok {
		return app.PTYOutputSnapshot{}, fmt.Errorf("pty %s not found", ptyID)
	}
	return app.PTYOutputSnapshot{Record: record}, nil
}

func (b *clientMemoryPTYBackend) List(context.Context) ([]app.PTYRecord, error) {
	out := make([]app.PTYRecord, 0, len(b.records))
	for _, record := range b.records {
		out = append(out, record)
	}
	return out, nil
}

func (b *clientMemoryPTYBackend) Shutdown(context.Context) error {
	return nil
}
