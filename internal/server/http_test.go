package server_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/phin-tech/whisk/internal/adapters/agenthooklog"
	"github.com/phin-tech/whisk/internal/adapters/agenthooks"
	"github.com/phin-tech/whisk/internal/adapters/agents"
	"github.com/phin-tech/whisk/internal/adapters/transcriptstore"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPServerSessionAndPTYFlow(t *testing.T) {
	backend := newFakePTYBackend()
	eventBus := newFakeEventBus()
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
	if err := transcripts.AppendPTYOutput(context.Background(), app.PTYTranscriptOutput{PTYID: "pty_history", Bytes: []byte("saved output")}); err != nil {
		t.Fatalf("append transcript: %v", err)
	}
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: eventBus, TranscriptStore: transcripts})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	handler := server.NewHTTP(runtime)

	health := getJSON[map[string]bool](t, handler, "/v1/health", http.StatusOK)
	if !health["ok"] {
		t.Fatalf("health = %#v", health)
	}
	compatibility := getJSON[protocol.CompatibilityResponse](t, handler, "/v1/compat", http.StatusOK)
	if compatibility.APIVersion != protocol.DaemonAPIVersion {
		t.Fatalf("compatibility = %#v", compatibility)
	}
	if compatibility.ProtocolVersion != protocol.ProtocolVersion {
		t.Fatalf("compatibility missing protocol version: %#v", compatibility)
	}
	if compatibility.GitSHA == "" {
		t.Fatalf("compatibility missing git sha: %#v", compatibility)
	}
	if compatibility.Version == "" {
		t.Fatalf("compatibility missing version: %#v", compatibility)
	}
	agentEvent := postJSON[protocol.AgentBridgeEvent](t, handler, "/v1/agent-hook-events", protocol.AgentBridgeHookRequest{
		Provider:  "claude",
		EventName: "Notification",
		Message:   "Need input.",
	}, http.StatusCreated)
	readAgentEvent := postJSON[protocol.AgentBridgeEvent](t, handler, "/v1/agent-bridge-events/"+agentEvent.ID+"/read", protocol.MarkAgentBridgeEventReadRequest{}, http.StatusOK)
	if readAgentEvent.Status != "read" {
		t.Fatalf("read agent event = %#v", readAgentEvent)
	}
	pendingAgentEvents := getJSON[[]protocol.AgentBridgeEvent](t, handler, "/v1/agent-bridge-events?status=pending", http.StatusOK)
	if len(pendingAgentEvents) != 0 {
		t.Fatalf("pending agent events = %#v", pendingAgentEvents)
	}
	for range 2 {
		event := getJSON[protocol.NextEventResponse](t, handler, "/v1/events/next?timeoutMs=10", http.StatusOK)
		if event.Event.Type != "agent_hook_events.changed" || event.Event.Seq == 0 {
			t.Fatalf("agent hook event = %#v", event)
		}
	}

	created := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusCreated)
	if created.Session.ID == "" || created.WindowID == "" || created.PaneID == "" || created.PTYID == nil || created.MainPtyID == "" {
		t.Fatalf("created session missing ids: %#v", created)
	}

	sessions := getJSON[[]map[string]any](t, handler, "/v1/sessions", http.StatusOK)
	if len(sessions) != 1 {
		t.Fatalf("sessions = %#v", sessions)
	}

	postNoContent(t, handler, "/v1/ptys/"+created.MainPtyID+"/resize", protocol.ResizePTYRequest{Cols: 120, Rows: 40})
	if backend.records[created.MainPtyID].Cols != 120 || backend.records[created.MainPtyID].Rows != 40 {
		t.Fatalf("backend size = %#v", backend.records[created.MainPtyID])
	}

	postNoContent(t, handler, "/v1/ptys/"+created.MainPtyID+"/write", protocol.WritePTYRequest{Data: "hello"})
	snapshot := getJSON[protocol.OutputSnapshot](t, handler, "/v1/ptys/"+created.MainPtyID+"/output?from=0", http.StatusOK)
	if snapshot.Output != "hello" || snapshot.OutputBase64 != "aGVsbG8=" || snapshot.Offset != 5 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	assertStatus(t, handler, http.MethodPost, "/v1/ptys/missing/write", `{"data":"x"}`, http.StatusBadRequest)
	if runtime.PTYInputRecent("missing", time.Hour) {
		t.Fatalf("failed HTTP pty write should not be recent")
	}
	assertStatus(t, handler, http.MethodPost, "/v1/ptys/"+created.MainPtyID+"/bookmarks", `{"offset":3}`, http.StatusNotFound)
	assertStatus(t, handler, http.MethodGet, "/v1/ptys/"+created.MainPtyID+"/bookmarks", "", http.StatusNotFound)
	assertStatus(t, handler, http.MethodDelete, "/v1/pty-bookmarks/bm_01", "", http.StatusNotFound)

	split := postJSON[protocol.SplitPaneResult](t, handler, "/v1/sessions/"+created.Session.ID+"/split", protocol.SplitPaneRequest{
		WindowID:     created.WindowID,
		TargetPaneID: created.PaneID,
		Direction:    "vertical",
		InitialPTY:   &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusOK)
	if split.PaneID == "" || split.PTYID == nil || split.PtyID == "" {
		t.Fatalf("split = %#v", split)
	}

	ptys := getJSON[[]protocol.PTYInfo](t, handler, "/v1/ptys", http.StatusOK)
	if len(ptys) != 2 {
		t.Fatalf("ptys = %#v", ptys)
	}
	byID := map[string]protocol.PTYInfo{}
	for _, pty := range ptys {
		byID[pty.ID] = pty
	}
	if byID[created.MainPtyID].SessionID != created.Session.ID || byID[created.MainPtyID].WindowID != created.WindowID || byID[created.MainPtyID].PaneID != created.PaneID {
		t.Fatalf("main pty = %#v", byID[created.MainPtyID])
	}
	history := getJSON[[]protocol.PTYHistorySummary](t, handler, "/v1/pty-history", http.StatusOK)
	if len(history) != 3 || history[0].WorkingDir == "" {
		t.Fatalf("pty history = %#v", history)
	}
	selectedHistory := getJSON[protocol.PTYHistory](t, handler, "/v1/pty-history/pty_history", http.StatusOK)
	if selectedHistory.Output != "saved output" || selectedHistory.SessionID != "sess_history" {
		t.Fatalf("selected pty history = %#v", selectedHistory)
	}

	event := getJSON[protocol.NextEventResponse](t, handler, "/v1/events/next?timeoutMs=10", http.StatusOK)
	if event.Event.Type != "session.changed" || event.Event.Seq == 0 {
		t.Fatalf("event = %#v", event)
	}

	cleared := postJSON[protocol.ClearDaemonResponse](t, handler, "/v1/daemon/clear", protocol.ClearDaemonRequest{}, http.StatusOK)
	if cleared.SessionsCleared != 1 || cleared.PTYsCleared != 2 {
		t.Fatalf("cleared = %#v", cleared)
	}
	sessions = getJSON[[]map[string]any](t, handler, "/v1/sessions", http.StatusOK)
	if len(sessions) != 0 {
		t.Fatalf("sessions after clear = %#v", sessions)
	}
	ptys = getJSON[[]protocol.PTYInfo](t, handler, "/v1/ptys", http.StatusOK)
	if len(ptys) != 0 {
		t.Fatalf("ptys after clear = %#v", ptys)
	}

}

func TestHTTPServerListsDetectedAgents(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "claude"))
	t.Setenv("PATH", binDir)

	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: newFakePTYBackend()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	handler := server.NewHTTP(runtime)

	detected := getJSON[[]protocol.DetectedAgent](t, handler, "/v1/agents/detected", http.StatusOK)
	byID := map[string]protocol.DetectedAgent{}
	for _, agent := range detected {
		byID[agent.ProfileID] = agent
	}
	claude, ok := byID["claude"]
	if !ok ||
		claude.Provider != "claude" ||
		claude.Label != "Claude Code" ||
		claude.DetectCommand != "claude" ||
		claude.Path != filepath.Join(binDir, "claude") {
		t.Fatalf("claude detection = %#v (ok=%v), all = %#v", claude, ok, detected)
	}
	if _, ok := byID["claude-plan"]; !ok {
		t.Fatalf("expected claude-plan detection, got %#v", detected)
	}
	if _, ok := byID["codex"]; ok {
		t.Fatalf("unexpected codex detection = %#v", detected)
	}
}

func TestHTTPServerPluginRoutes(t *testing.T) {
	used := 1.0
	limit := 10.0
	plugins := &pluginRegistryFake{statuses: []app.PluginStatus{{
		ID:      "github",
		Name:    "GitHub",
		Trusted: true,
		Valid:   true,
		UsageResolvers: []app.PluginUsageResolver{{
			ID:             "github.usage",
			Provider:       "github",
			Label:          "GitHub",
			Profiles:       []string{"codex"},
			TimeoutMs:      10000,
			OutputCapBytes: 262144,
			MinRefreshMs:   300000,
			StaleAfterMs:   1800000,
		}},
		UIPanels: []app.PluginUIPanel{{
			ID:    "github.issue.panel",
			Title: "GitHub Issue",
			Scope: app.PluginUIScope("workItem"),
			Kind:  "view",
			Read:  &app.PluginUICommandRef{TimeoutMs: 10000, OutputCapBytes: 262144},
			Actions: []app.PluginUICommandRef{{
				ID:             "sync",
				Label:          "Sync",
				TimeoutMs:      10000,
				OutputCapBytes: 262144,
			}},
		}},
		UICommands: []app.PluginUICommand{{
			ID:             "github.open",
			Label:          "GitHub: Open",
			Scope:          app.PluginUIScope("global"),
			TimeoutMs:      10000,
			OutputCapBytes: 262144,
		}},
		ReviewActions: []app.PluginReviewAction{{
			ID:          "github.review",
			Label:       "GitHub Review",
			Scope:       app.PluginUIScope("workItem"),
			URLTemplate: "https://github.com/{{project.id.url}}",
			HasSubmit:   true,
			Blocking:    true,
		}},
		Permissions: &app.PluginPermissions{Network: []string{"api.github.com"}},
	}}, usageResult: app.UsageResolverResult{
		Summary: "ok",
		Metrics: []app.UsageResolverMetric{{
			ID:    "requests",
			Kind:  app.UsageMetricKindUsage,
			Used:  &used,
			Limit: &limit,
		}},
	}}
	runtime := app.NewRuntime(app.RuntimeConfig{Plugins: plugins})
	handler := server.NewHTTP(runtime)

	listed := getJSON[[]protocol.PluginStatus](t, handler, "/v1/plugins", http.StatusOK)
	if len(listed) != 1 || listed[0].ID != "github" {
		t.Fatalf("plugins = %#v", listed)
	}
	if len(listed[0].UsageResolvers) != 1 ||
		listed[0].UsageResolvers[0].ID != "github.usage" ||
		listed[0].UsageResolvers[0].Provider != "github" ||
		listed[0].UsageResolvers[0].Label != "GitHub" ||
		len(listed[0].UsageResolvers[0].Profiles) != 1 ||
		listed[0].UsageResolvers[0].Profiles[0] != "codex" ||
		listed[0].UsageResolvers[0].TimeoutMs != 10000 ||
		listed[0].UsageResolvers[0].OutputCapBytes != 262144 ||
		listed[0].UsageResolvers[0].MinRefreshMs != 300000 ||
		listed[0].UsageResolvers[0].StaleAfterMs != 1800000 {
		t.Fatalf("usage resolvers = %#v", listed[0].UsageResolvers)
	}
	if len(listed[0].UIPanels) != 1 ||
		listed[0].UIPanels[0].Read == nil ||
		len(listed[0].UIPanels[0].Actions) != 1 ||
		listed[0].UIPanels[0].Actions[0].ID != "sync" ||
		len(listed[0].UICommands) != 1 ||
		listed[0].UICommands[0].ID != "github.open" ||
		len(listed[0].ReviewActions) != 1 ||
		!listed[0].ReviewActions[0].HasSubmit ||
		!listed[0].ReviewActions[0].Blocking ||
		listed[0].Permissions == nil ||
		len(listed[0].Permissions.Network) != 1 ||
		listed[0].Permissions.Network[0] != "api.github.com" {
		t.Fatalf("plugin ui catalog = %#v", listed[0])
	}
	usageResolvers := getJSON[[]protocol.UsageResolverReadModel](t, handler, "/v1/usage-resolvers", http.StatusOK)
	if len(usageResolvers) != 1 ||
		usageResolvers[0].PluginID != "github" ||
		usageResolvers[0].ResolverID != "github.usage" ||
		usageResolvers[0].Profile != "codex" ||
		usageResolvers[0].Status != app.UsageResolverStatusPending {
		t.Fatalf("usage resolvers = %#v", usageResolvers)
	}
	refreshed := postJSON[protocol.UsageResolverReadModel](t, handler, "/v1/plugins/github/usage-resolvers/github.usage/refresh", protocol.RefreshUsageResolverRequest{Profile: "codex"}, http.StatusOK)
	if refreshed.Status != app.UsageResolverStatusOK ||
		refreshed.Result == nil ||
		refreshed.Result.Summary != "ok" ||
		refreshed.Result.Metrics[0].ID != "requests" ||
		refreshed.RefreshedAt == nil {
		t.Fatalf("refreshed usage resolver = %#v", refreshed)
	}
	if plugins.usageReq.PluginID != "github" || plugins.usageReq.ResolverID != "github.usage" || plugins.usageReq.Profile != "codex" {
		t.Fatalf("usage request = %#v", plugins.usageReq)
	}
	uiContribs := getJSON[protocol.UIContributionsResponse](t, handler, "/v1/ui-contributions?workItemId=wi_01", http.StatusOK)
	if len(uiContribs.Plugins) != 1 || uiContribs.Plugins[0].PluginID != "github" {
		t.Fatalf("ui contributions = %#v", uiContribs)
	}
	if uiContribs.Scope.WorkItemID != "wi_01" {
		t.Fatalf("ui contributions scope workItemId = %q, want wi_01", uiContribs.Scope.WorkItemID)
	}
	if len(uiContribs.Plugins[0].Panels) != 1 ||
		uiContribs.Plugins[0].Panels[0].ID != "github.issue.panel" ||
		!uiContribs.Plugins[0].Enabled ||
		uiContribs.Plugins[0].DisabledReason != "" ||
		len(uiContribs.Plugins[0].Commands) != 1 ||
		uiContribs.Plugins[0].Commands[0].ID != "github.open" ||
		len(uiContribs.Plugins[0].ReviewActions) != 1 ||
		uiContribs.Plugins[0].ReviewActions[0].ID != "github.review" {
		t.Fatalf("ui contributions plugin = %#v", uiContribs.Plugins[0])
	}
	if uiContribs.Plugins[0].Permissions == nil || len(uiContribs.Plugins[0].Permissions.Network) != 1 {
		t.Fatalf("ui contributions permissions = %#v", uiContribs.Plugins[0].Permissions)
	}
	globalOnly := getJSON[protocol.UIContributionsResponse](t, handler, "/v1/ui-contributions", http.StatusOK)
	if len(globalOnly.Plugins) != 1 ||
		len(globalOnly.Plugins[0].Commands) != 1 ||
		globalOnly.Plugins[0].Commands[0].ID != "github.open" ||
		len(globalOnly.Plugins[0].Panels) != 0 ||
		len(globalOnly.Plugins[0].ReviewActions) != 0 {
		t.Fatalf("global ui contributions = %#v", globalOnly)
	}
	phaseOnly := getJSON[protocol.UIContributionsResponse](t, handler, "/v1/ui-contributions?workItemId=&phase=review", http.StatusOK)
	if phaseOnly.Scope.WorkItemID != "" || phaseOnly.Scope.Phase != "review" {
		t.Fatalf("phase-only scope = %#v", phaseOnly.Scope)
	}
	if len(phaseOnly.Plugins) != 1 ||
		len(phaseOnly.Plugins[0].Commands) != 1 ||
		phaseOnly.Plugins[0].Commands[0].ID != "github.open" ||
		len(phaseOnly.Plugins[0].Panels) != 0 ||
		len(phaseOnly.Plugins[0].ReviewActions) != 0 {
		t.Fatalf("phase-only ui contributions = %#v", phaseOnly)
	}
	registry := getJSON[[]protocol.RegistryPlugin](t, handler, "/v1/plugin-registry", http.StatusOK)
	if len(registry) != 1 || registry[0].ID != "github" || registry[0].Registry != "phin-tech" || registry[0].SourceType != "path" {
		t.Fatalf("registry = %#v", registry)
	}
	installed := postJSON[protocol.PluginStatus](t, handler, "/v1/plugin-registry/install", protocol.InstallRegistryPluginRequest{Registry: "phin-tech", ID: "github"}, http.StatusCreated)
	if installed.ID != "github" || installed.Registry != "phin-tech" || installed.Trusted {
		t.Fatalf("installed = %#v", installed)
	}
	trusted := postJSON[protocol.PluginStatus](t, handler, "/v1/plugins/github/trust", struct{}{}, http.StatusOK)
	if !trusted.Trusted {
		t.Fatalf("trusted = %#v", trusted)
	}
	rescanned := postJSON[[]protocol.PluginStatus](t, handler, "/v1/plugins/rescan", struct{}{}, http.StatusOK)
	if len(rescanned) != 1 || rescanned[0].ID != "github" {
		t.Fatalf("rescanned = %#v", rescanned)
	}
	untrusted := postJSON[protocol.PluginStatus](t, handler, "/v1/plugins/github/untrust", struct{}{}, http.StatusOK)
	if untrusted.Trusted {
		t.Fatalf("untrusted = %#v", untrusted)
	}
	project := postJSON[protocol.Project](t, handler, "/v1/projects", protocol.CreateProjectRequest{
		Name:    "App",
		RootDir: t.TempDir(),
	}, http.StatusCreated)
	attached := postJSON[protocol.Project](t, handler, "/v1/plugins/github/project-attachment-templates/github.issue", protocol.RunPluginProjectAttachmentTemplateRequest{
		ProjectID: project.ID,
		Values:    map[string]string{"repo": "owner/repo", "issue": "1"},
	}, http.StatusCreated)
	if len(attached.Attachments) != 1 || attached.Attachments[0].Provider != "github" || attached.Attachments[0].URL == "" {
		t.Fatalf("attached = %#v", attached.Attachments)
	}
}

func TestHTTPServerAgentHookIntegrationRoutes(t *testing.T) {
	paths := testAgentHookPaths(t)
	runtime := app.NewRuntime(app.RuntimeConfig{AgentHookPaths: &paths})
	handler := server.NewHTTP(runtime)

	listed := getJSON[[]protocol.AgentHookIntegration](t, handler, "/v1/agent-hook-integrations", http.StatusOK)
	if len(listed) != 2 {
		t.Fatalf("listed = %#v", listed)
	}

	installed := postJSON[protocol.AgentHookIntegration](t, handler, "/v1/agent-hook-integrations/install", protocol.AgentHookIntegrationRequest{
		Provider: "claude",
	}, http.StatusOK)
	if installed.Provider != "claude" || installed.Status != "current" || installed.HelperPath == "" || installed.ManifestPath == "" {
		t.Fatalf("installed = %#v", installed)
	}

	checked := postJSON[protocol.AgentHookIntegration](t, handler, "/v1/agent-hook-integrations/check", protocol.AgentHookIntegrationRequest{
		Provider: "claude",
	}, http.StatusOK)
	if checked.Provider != "claude" || checked.Status != "current" {
		t.Fatalf("checked = %#v", checked)
	}

	removed := postJSON[protocol.AgentHookIntegration](t, handler, "/v1/agent-hook-integrations/remove", protocol.AgentHookIntegrationRequest{
		Provider: "claude",
	}, http.StatusOK)
	if removed.Provider != "claude" || removed.Status != "missing" {
		t.Fatalf("removed = %#v", removed)
	}
}

func TestHTTPServerSessionLifecycleActions(t *testing.T) {
	backend := newFakePTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend})
	handler := server.NewHTTP(runtime)

	rootDir := t.TempDir()
	nextRoot := t.TempDir()
	paneDir := t.TempDir()

	created := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:    "Empty",
		RootDir: rootDir,
	}, http.StatusCreated)
	if created.PTYID != nil {
		t.Fatalf("created should be empty: %#v", created)
	}
	project := postJSON[protocol.Project](t, handler, "/v1/projects", protocol.CreateProjectRequest{
		Name:    "Session Project",
		RootDir: rootDir,
	}, http.StatusCreated)
	updated := postJSON[map[string]any](t, handler, "/v1/sessions/"+created.Session.ID+"/set-project", protocol.SetSessionProjectRequest{
		ProjectID: project.ID,
	}, http.StatusOK)
	if updated["projectId"] != project.ID {
		t.Fatalf("updated project = %#v", updated)
	}

	updated = postJSON[map[string]any](t, handler, "/v1/sessions/"+created.Session.ID+"/set-root-dir", protocol.SetSessionRootDirRequest{
		RootDir: nextRoot,
	}, http.StatusOK)
	if updated["rootDir"] != nextRoot {
		t.Fatalf("updated root = %#v", updated)
	}

	updated = postJSON[map[string]any](t, handler, "/v1/sessions/"+created.Session.ID+"/panes/"+created.PaneID+"/set-working-dir", protocol.SetPaneWorkingDirRequest{
		WorkingDir: paneDir,
	}, http.StatusOK)
	panes := updated["panes"].(map[string]any)
	pane := panes[created.PaneID].(map[string]any)
	if pane["workingDir"] != paneDir {
		t.Fatalf("pane = %#v", pane)
	}

	started := postJSON[protocol.StartedPanePTY](t, handler, "/v1/sessions/"+created.Session.ID+"/panes/"+created.PaneID+"/start-pty", protocol.StartPanePTYRequest{
		Options: protocol.StartPTYOptions{Cols: 90, Rows: 30, Command: "echo server-command"},
	}, http.StatusCreated)
	if started.PTYID == "" || backend.records[started.PTYID].WorkingDir != paneDir {
		t.Fatalf("started = %#v record = %#v", started, backend.records[started.PTYID])
	}
	if string(backend.outputs[started.PTYID]) != "echo server-command\r" {
		t.Fatalf("initial command output = %q", string(backend.outputs[started.PTYID]))
	}
	execCreated := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:    "Exec",
		RootDir: rootDir,
		InitialPTY: &protocol.StartPTYOptions{
			Cols:    81,
			Rows:    25,
			Command: "codex",
			Args:    []string{"--ask-for-approval=never", "Plan"},
			Env:     map[string]string{"WHISK_TEST": "1"},
			Exec:    true,
		},
	}, http.StatusCreated)
	if execCreated.PTYID == nil {
		t.Fatalf("exec created missing pty: %#v", execCreated)
	}
	execSpawn := backend.spawns[*execCreated.PTYID]
	if execSpawn.Command != "codex" || len(execSpawn.Args) != 2 || execSpawn.Args[1] != "Plan" || execSpawn.Env["WHISK_TEST"] != "1" {
		t.Fatalf("exec spawn = %#v", execSpawn)
	}
	if string(backend.outputs[*execCreated.PTYID]) != "" {
		t.Fatalf("exec pty should not receive command via stdin: %q", string(backend.outputs[*execCreated.PTYID]))
	}
	detached := postJSON[protocol.DetachedPanePTY](t, handler, "/v1/sessions/"+created.Session.ID+"/panes/"+created.PaneID+"/detach-pty", protocol.DetachPanePTYRequest{}, http.StatusOK)
	if detached.PTYID != started.PTYID {
		t.Fatalf("detached = %#v", detached)
	}
	ptys := getJSON[[]protocol.PTYInfo](t, handler, "/v1/ptys", http.StatusOK)
	byPTY := map[string]protocol.PTYInfo{}
	for _, pty := range ptys {
		byPTY[pty.ID] = pty
	}
	if byPTY[started.PTYID].SessionID != created.Session.ID || byPTY[started.PTYID].PaneID != "" || byPTY[started.PTYID].OriginPaneID != created.PaneID || byPTY[started.PTYID].Status != "running" {
		t.Fatalf("detached pty = %#v", byPTY[started.PTYID])
	}

	restartSession := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Restart",
		RootDir:    rootDir,
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusCreated)
	killed := postJSON[protocol.PTYInfo](t, handler, "/v1/ptys/"+restartSession.MainPtyID+"/kill", protocol.KillPTYRequest{}, http.StatusOK)
	if killed.Status != "killed" || killed.Running {
		t.Fatalf("killed = %#v", killed)
	}
	restarted := postJSON[protocol.RestartedPanePTY](t, handler, "/v1/sessions/"+restartSession.Session.ID+"/panes/"+restartSession.PaneID+"/restart-pty", protocol.RestartPanePTYRequest{
		Options: protocol.StartPTYOptions{Cols: 100, Rows: 40},
	}, http.StatusCreated)
	if restarted.PTYID == "" || restarted.OldPTYID != restartSession.MainPtyID || backend.records[restarted.PTYID].Cols != 100 {
		t.Fatalf("restarted = %#v record = %#v", restarted, backend.records[restarted.PTYID])
	}
	deleteNoContent(t, handler, "/v1/ptys/"+restarted.OldPTYID)
	if _, ok := backend.records[restarted.OldPTYID]; ok {
		t.Fatalf("deleted pty still in backend: %#v", backend.records[restarted.OldPTYID])
	}

	closeSession := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:    "Close",
		RootDir: rootDir,
	}, http.StatusCreated)
	split := postJSON[protocol.SplitPaneResult](t, handler, "/v1/sessions/"+closeSession.Session.ID+"/split", protocol.SplitPaneRequest{
		WindowID:     closeSession.WindowID,
		TargetPaneID: closeSession.PaneID,
		Direction:    "horizontal",
	}, http.StatusOK)
	closed := postJSON[map[string]any](t, handler, "/v1/sessions/"+closeSession.Session.ID+"/windows/"+closeSession.WindowID+"/panes/"+split.PaneID+"/close", protocol.ClosePaneRequest{}, http.StatusOK)
	closedPanes := closed["panes"].(map[string]any)
	if _, ok := closedPanes[split.PaneID]; ok {
		t.Fatalf("closed pane still present: %#v", closedPanes)
	}

	sessionToClose := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Close session",
		RootDir:    rootDir,
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusCreated)
	remaining := deleteJSON[[]map[string]any](t, handler, "/v1/sessions/"+sessionToClose.Session.ID, http.StatusOK)
	for _, candidate := range remaining {
		if candidate["id"] == sessionToClose.Session.ID {
			t.Fatalf("closed session still present: %#v", remaining)
		}
	}
	ptys = getJSON[[]protocol.PTYInfo](t, handler, "/v1/ptys", http.StatusOK)
	for _, pty := range ptys {
		if pty.SessionID == sessionToClose.Session.ID && pty.Status != "killed" {
			t.Fatalf("closed session pty not killed: %#v", pty)
		}
	}
}

func TestHTTPServerAttachesPTYWebSocketOutputStream(t *testing.T) {
	backend := newFakePTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: newFakeEventBus()})
	handler := server.NewHTTP(runtime)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	created := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusCreated)
	postNoContent(t, handler, "/v1/ptys/"+created.MainPtyID+"/write", protocol.WritePTYRequest{Data: "hello"})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, strings.Replace(httpServer.URL, "http", "ws", 1)+"/v1/ptys/"+created.MainPtyID+"/attach?from=1", nil)
	if err != nil {
		t.Fatalf("dial attach: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	replay := readPTYStreamFrame(t, ctx, conn)
	if replay.Type != "output" || replay.PtyID != created.MainPtyID || replay.Offset != 1 || replay.OutputBase64 != "ZWxsbw==" {
		t.Fatalf("replay frame = %#v", replay)
	}

	postNoContent(t, handler, "/v1/ptys/"+created.MainPtyID+"/write", protocol.WritePTYRequest{Data: "!"})
	live := readPTYStreamFrame(t, ctx, conn)
	if live.Type != "output" || live.PtyID != created.MainPtyID || live.Offset != 5 || live.OutputBase64 != "IQ==" {
		t.Fatalf("live frame = %#v", live)
	}

	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"input","ptyId":"`+created.MainPtyID+`","data":"?"}`)); err != nil {
		t.Fatalf("write websocket input: %v", err)
	}
	inputEcho := readPTYStreamFrame(t, ctx, conn)
	if inputEcho.Type != "output" || inputEcho.PtyID != created.MainPtyID || inputEcho.Offset != 6 || inputEcho.OutputBase64 != "Pw==" {
		t.Fatalf("input echo frame = %#v", inputEcho)
	}
}

func TestHTTPServerAttachesPTYWebSocketBinaryOutputStream(t *testing.T) {
	backend := newFakePTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: newFakeEventBus()})
	handler := server.NewHTTP(runtime)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	created := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusCreated)
	postNoContent(t, handler, "/v1/ptys/"+created.MainPtyID+"/write", protocol.WritePTYRequest{Data: "hello"})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, strings.Replace(httpServer.URL, "http", "ws", 1)+"/v1/ptys/"+created.MainPtyID+"/attach?from=1&binary=1", nil)
	if err != nil {
		t.Fatalf("dial attach: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	replayOffset, replay := readPTYBinaryOutputFrame(t, ctx, conn)
	if replayOffset != 1 || string(replay) != "ello" {
		t.Fatalf("replay frame offset=%d output=%q", replayOffset, replay)
	}

	postNoContent(t, handler, "/v1/ptys/"+created.MainPtyID+"/write", protocol.WritePTYRequest{Data: "!"})
	liveOffset, live := readPTYBinaryOutputFrame(t, ctx, conn)
	if liveOffset != 5 || string(live) != "!" {
		t.Fatalf("live frame offset=%d output=%q", liveOffset, live)
	}

	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"input","ptyId":"`+created.MainPtyID+`","data":"?"}`)); err != nil {
		t.Fatalf("write websocket input: %v", err)
	}
	inputOffset, inputEcho := readPTYBinaryOutputFrame(t, ctx, conn)
	if inputOffset != 6 || string(inputEcho) != "?" {
		t.Fatalf("input echo frame offset=%d output=%q", inputOffset, inputEcho)
	}
}

func TestHTTPServerAttachPTYWebSocketResynchronizesOutputGap(t *testing.T) {
	backend := newFakePTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: newFakeEventBus()})
	handler := server.NewHTTP(runtime)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	created := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusCreated)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, strings.Replace(httpServer.URL, "http", "ws", 1)+"/v1/ptys/"+created.MainPtyID+"/attach?from=0", nil)
	if err != nil {
		t.Fatalf("dial attach: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	payload := []byte("0123456789ABCDE")
	backend.mu.Lock()
	backend.outputs[created.MainPtyID] = append([]byte(nil), payload...)
	events := backend.events
	backend.mu.Unlock()
	if events == nil {
		t.Fatalf("attach did not register event stream")
	}
	events <- app.PTYEvent{Kind: app.PTYOutput, Offset: 10, Bytes: payload[10:]}

	recovered := readPTYStreamFrame(t, ctx, conn)
	if recovered.Type != "output" ||
		recovered.PtyID != created.MainPtyID ||
		recovered.Offset != 0 ||
		recovered.OutputBase64 != base64.StdEncoding.EncodeToString(payload) {
		t.Fatalf("recovered frame = %#v", recovered)
	}
}

func TestHTTPServerAttachPTYWebSocketBinaryResynchronizesOutputGap(t *testing.T) {
	backend := newFakePTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: newFakeEventBus()})
	handler := server.NewHTTP(runtime)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	created := postJSON[protocol.CreatedSession](t, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    t.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	}, http.StatusCreated)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, strings.Replace(httpServer.URL, "http", "ws", 1)+"/v1/ptys/"+created.MainPtyID+"/attach?from=0&binary=1", nil)
	if err != nil {
		t.Fatalf("dial attach: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	payload := []byte("0123456789ABCDE")
	backend.mu.Lock()
	backend.outputs[created.MainPtyID] = append([]byte(nil), payload...)
	events := backend.events
	backend.mu.Unlock()
	if events == nil {
		t.Fatalf("attach did not register event stream")
	}
	events <- app.PTYEvent{Kind: app.PTYOutput, Offset: 10, Bytes: payload[10:]}

	recoveredOffset, recovered := readPTYBinaryOutputFrame(t, ctx, conn)
	if recoveredOffset != 0 || !bytes.Equal(recovered, payload) {
		t.Fatalf("recovered frame offset=%d output=%q", recoveredOffset, recovered)
	}
}

func TestHTTPServerAttachPTYWebSocketBatchesBackgroundOutput(t *testing.T) {
	backend := newFakePTYBackend()
	_, err := backend.Spawn(context.Background(), app.SpawnPTYRequest{
		ID:         "pty_background",
		WorkingDir: t.TempDir(),
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("spawn fake pty: %v", err)
	}
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: newFakeEventBus()})
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, strings.Replace(httpServer.URL, "http", "ws", 1)+"/v1/ptys/pty_background/attach?from=0", nil)
	if err != nil {
		t.Fatalf("dial attach: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	backend.waitForSubscriber(t)

	backend.emitOutput(t, "pty_background", []byte("a"))
	backend.emitOutput(t, "pty_background", []byte("b"))

	batched := readPTYStreamFrame(t, ctx, conn)
	if batched.Type != "output" ||
		batched.PtyID != "pty_background" ||
		batched.Offset != 0 ||
		batched.OutputBase64 != base64.StdEncoding.EncodeToString([]byte("ab")) {
		t.Fatalf("batched frame = %#v", batched)
	}
}

func TestHTTPServerAttachPTYWebSocketBinaryBatchesBackgroundOutput(t *testing.T) {
	backend := newFakePTYBackend()
	_, err := backend.Spawn(context.Background(), app.SpawnPTYRequest{
		ID:         "pty_binary_background",
		WorkingDir: t.TempDir(),
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("spawn fake pty: %v", err)
	}
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: newFakeEventBus()})
	httpServer := httptest.NewServer(server.NewHTTP(runtime, server.WithPTYOutputBatchInterval(200*time.Millisecond)))
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, strings.Replace(httpServer.URL, "http", "ws", 1)+"/v1/ptys/pty_binary_background/attach?from=0&binary=1", nil)
	if err != nil {
		t.Fatalf("dial attach: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	backend.waitForSubscriber(t)

	backend.emitOutput(t, "pty_binary_background", []byte("a"))
	backend.emitOutput(t, "pty_binary_background", []byte("b"))

	batchedOffset, batched := readPTYBinaryOutputFrame(t, ctx, conn)
	if batchedOffset != 0 || string(batched) != "ab" {
		t.Fatalf("batched frame offset=%d output=%q", batchedOffset, batched)
	}
}

func TestHTTPServerAttachPTYWebSocketFlushesPendingOutputBeforeExit(t *testing.T) {
	backend := newFakePTYBackend()
	_, err := backend.Spawn(context.Background(), app.SpawnPTYRequest{
		ID:         "pty_exit",
		WorkingDir: t.TempDir(),
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("spawn fake pty: %v", err)
	}
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: newFakeEventBus()})
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, strings.Replace(httpServer.URL, "http", "ws", 1)+"/v1/ptys/pty_exit/attach?from=0", nil)
	if err != nil {
		t.Fatalf("dial attach: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	backend.waitForSubscriber(t)

	backend.emitOutput(t, "pty_exit", []byte("done"))
	backend.emitExit(t, "pty_exit", 7)

	output := readPTYStreamFrame(t, ctx, conn)
	if output.Type != "output" ||
		output.PtyID != "pty_exit" ||
		output.Offset != 0 ||
		output.OutputBase64 != base64.StdEncoding.EncodeToString([]byte("done")) {
		t.Fatalf("output frame = %#v", output)
	}
	exit := readPTYStreamFrame(t, ctx, conn)
	if exit.Type != "exit" || exit.PtyID != "pty_exit" || exit.Code == nil || *exit.Code != 7 {
		t.Fatalf("exit frame = %#v", exit)
	}
}

func TestHTTPServerAttachPTYWebSocketBinaryFlushesPendingOutputBeforeExit(t *testing.T) {
	backend := newFakePTYBackend()
	_, err := backend.Spawn(context.Background(), app.SpawnPTYRequest{
		ID:         "pty_binary_exit",
		WorkingDir: t.TempDir(),
		Cols:       80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("spawn fake pty: %v", err)
	}
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: newFakeEventBus()})
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, strings.Replace(httpServer.URL, "http", "ws", 1)+"/v1/ptys/pty_binary_exit/attach?from=0&binary=1", nil)
	if err != nil {
		t.Fatalf("dial attach: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	backend.waitForSubscriber(t)

	backend.emitOutput(t, "pty_binary_exit", []byte("done"))
	backend.emitExit(t, "pty_binary_exit", 7)

	outputOffset, output := readPTYBinaryOutputFrame(t, ctx, conn)
	if outputOffset != 0 || string(output) != "done" {
		t.Fatalf("output frame offset=%d output=%q", outputOffset, output)
	}
	exit := readPTYStreamFrame(t, ctx, conn)
	if exit.Type != "exit" || exit.PtyID != "pty_binary_exit" || exit.Code == nil || *exit.Code != 7 {
		t.Fatalf("exit frame = %#v", exit)
	}
}

func readPTYStreamFrame(t *testing.T, ctx context.Context, conn *websocket.Conn) protocol.PTYStreamFrame {
	t.Helper()
	typ, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read websocket frame: %v", err)
	}
	if typ != websocket.MessageText {
		t.Fatalf("websocket message type = %v", typ)
	}
	var frame protocol.PTYStreamFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		t.Fatalf("decode websocket frame: %v", err)
	}
	return frame
}

func readPTYBinaryOutputFrame(t *testing.T, ctx context.Context, conn *websocket.Conn) (uint64, []byte) {
	t.Helper()
	typ, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read websocket frame: %v", err)
	}
	if typ != websocket.MessageBinary {
		t.Fatalf("websocket message type = %v", typ)
	}
	offset, output, err := protocol.DecodePTYBinaryOutputFrame(data)
	if err != nil {
		t.Fatalf("decode binary output frame: %v", err)
	}
	return offset, output
}

func TestHTTPServerNextEventTimeoutReturnsNoop(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: newFakePTYBackend(),
		EventSink:  newFakeEventBus(),
	})
	handler := server.NewHTTP(runtime)

	event := getJSON[protocol.NextEventResponse](t, handler, "/v1/events/next?timeoutMs=1", http.StatusOK)
	if event.Event.Type != protocol.RuntimeEventNone || event.Missed {
		t.Fatalf("event = %#v", event)
	}
}

func TestHTTPServerOnboardingRoutes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".state"))

	runtime := app.NewRuntime(app.RuntimeConfig{
		DaemonURL:           "http://127.0.0.1:8787",
		OnboardingStatePath: filepath.Join(t.TempDir(), "onboarding.json"),
		AgentHookPaths: &agenthooks.Paths{
			ConfigRoot:         filepath.Join(t.TempDir(), "whisk"),
			HelperSourcePath:   os.Args[0],
			ClaudeSettingsPath: filepath.Join(t.TempDir(), "claude.json"),
			CodexHooksPath:     filepath.Join(t.TempDir(), "codex.json"),
		},
	})
	handler := server.NewHTTP(runtime)

	status := getJSON[protocol.OnboardingStatus](t, handler, "/v1/onboarding", http.StatusOK)
	if !status.LocalDaemon || status.StatePath == "" || len(status.Items) == 0 {
		t.Fatalf("onboarding status = %#v", status)
	}
	applied := postJSON[protocol.OnboardingStatus](t, handler, "/v1/onboarding/apply", protocol.OnboardingApplyRequest{
		ItemIDs: []string{"daemon:version"},
	}, http.StatusOK)
	if !applied.LocalDaemon || applied.StatePath != status.StatePath {
		t.Fatalf("applied onboarding = %#v, status = %#v", applied, status)
	}
}

func TestHTTPServerAgentBridgeHooksValidateTokenAndReturnProviderOutput(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	backend := newFakePTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: backend,
		IDGenerator: func() string {
			nextID++
			return "id_" + strconv.Itoa(nextID)
		},
		DaemonURL: "http://127.0.0.1:8787",
	})
	handler := server.NewHTTP(runtime)

	project := postJSON[protocol.Project](t, handler, "/v1/projects", protocol.CreateProjectRequest{
		Name:    "App",
		RootDir: t.TempDir(),
	}, http.StatusCreated)
	item := postJSON[protocol.WorkItem](t, handler, "/v1/work-items", protocol.CreateWorkItemRequest{
		ProjectID: project.ID,
		Title:     "Bridge endpoint",
		Actor:     "human",
	}, http.StatusCreated)
	run := postJSON[protocol.WorkItemRun](t, handler, "/v1/work-item-runs", protocol.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Launch:           true,
		AgentProfileID:   "claude",
		Actor:            "agent",
	}, http.StatusCreated)
	spawn := backend.spawns[run.PTYID]
	bridgeID := spawn.Env["WHISK_AGENT_BRIDGE_ID"]
	token := spawn.Env["WHISK_AGENT_BRIDGE_TOKEN"]
	if bridgeID == "" || token == "" {
		t.Fatalf("spawn bridge env = %#v", spawn.Env)
	}

	assertStatus(t, handler, http.MethodPost, "/v1/agent-bridges/"+bridgeID+"/hooks", `{"token":"wrong"}`, http.StatusUnauthorized)

	allow := postJSON[protocol.AgentBridgeHookResponse](t, handler, "/v1/agent-bridges/"+bridgeID+"/hooks", protocol.AgentBridgeHookRequest{
		Token:     token,
		Provider:  "claude",
		EventName: "PermissionRequest",
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "pwd"},
		Decision:  protocol.AgentBridgeHookDecision{Action: "allow"},
	}, http.StatusOK)
	if allow.Output != nil {
		t.Fatalf("allow output = %#v", allow.Output)
	}

	deny := postJSON[protocol.AgentBridgeHookResponse](t, handler, "/v1/agent-bridges/"+bridgeID+"/hooks", protocol.AgentBridgeHookRequest{
		Token:     token,
		Provider:  "claude",
		EventName: "PermissionRequest",
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "rm -rf /tmp/x"},
		Decision:  protocol.AgentBridgeHookDecision{Action: "deny", Reason: "blocked by policy"},
	}, http.StatusOK)
	hookSpecific, ok := deny.Output["hookSpecificOutput"].(map[string]any)
	if !ok || hookSpecific["permissionDecision"] != "deny" || hookSpecific["permissionDecisionReason"] != "blocked by policy" {
		t.Fatalf("deny output = %#v", deny.Output)
	}
}

func TestHTTPServerAgentHookLogAndBridgeRoutes(t *testing.T) {
	tmp := t.TempDir()
	logPaths := agenthooklog.Paths{
		ConfigRoot: tmp,
		LogPath:    filepath.Join(tmp, "agent-hooks", "hooks.jsonl"),
	}
	runtime := app.NewRuntime(app.RuntimeConfig{AgentHookLogPaths: &logPaths})
	handler := server.NewHTTP(runtime)

	enabled := true
	status := postJSON[protocol.AgentHookLogStatus](t, handler, "/v1/agent-hook-log/settings",
		protocol.SetAgentHookLogSettingsRequest{Enabled: &enabled}, http.StatusOK)
	if !status.Enabled {
		t.Fatalf("settings status = %#v", status)
	}

	event := postJSON[protocol.AgentBridgeEvent](t, handler, "/v1/agent-hook-events",
		protocol.AgentBridgeHookRequest{Provider: "claude", EventName: "Notification", Message: "ping"},
		http.StatusCreated)
	if event.ID == "" {
		t.Fatalf("event = %#v", event)
	}

	events := getJSON[[]protocol.AgentBridgeEvent](t, handler, "/v1/agent-bridge-events", http.StatusOK)
	if len(events) != 1 || events[0].ID != event.ID {
		t.Fatalf("events = %#v", events)
	}

	approvals := getJSON[[]protocol.AgentBridgeApproval](t, handler, "/v1/agent-bridge-approvals?status=pending", http.StatusOK)
	if len(approvals) != 0 {
		t.Fatalf("approvals = %#v", approvals)
	}

	// Resolving an unknown approval is a bad request.
	assertStatus(t, handler, http.MethodPost, "/v1/agent-bridge-approvals/missing/resolve", `{"action":"allow"}`, http.StatusBadRequest)

	logStatus := getJSON[protocol.AgentHookLogStatus](t, handler, "/v1/agent-hook-log", http.StatusOK)
	if !logStatus.Enabled || logStatus.SizeBytes == 0 {
		t.Fatalf("log status = %#v", logStatus)
	}

	cleared := postJSON[protocol.AgentHookLogStatus](t, handler, "/v1/agent-hook-log/clear", struct{}{}, http.StatusOK)
	if cleared.SizeBytes != 0 {
		t.Fatalf("cleared = %#v", cleared)
	}
}

func TestHTTPServerAgentHookReceiverRejectsOversizedPayload(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{})
	handler := server.NewHTTP(runtime, server.WithAgentHookReceiverLimits(128, time.Second))

	body := `{"provider":"claude","eventName":"Notification","rawPayload":{"body":"` + strings.Repeat("x", 512) + `"}}`
	assertStatus(t, handler, http.MethodPost, "/v1/agent-hook-events", body, http.StatusRequestEntityTooLarge)
}

func TestHTTPServerAgentHookReceiverRejectsTrailingJSON(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{})
	handler := server.NewHTTP(runtime, server.WithAgentHookReceiverLimits(1024, time.Second))

	for name, body := range map[string]string{
		"garbage":      `{"provider":"claude","eventName":"Notification"} trailing`,
		"second_value": `{"provider":"claude","eventName":"Notification"} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			assertStatus(t, handler, http.MethodPost, "/v1/agent-hook-events", body, http.StatusBadRequest)
		})
	}
}

func TestHTTPServerAgentHookReceiverRejectsOversizedTrailingBody(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{})
	handler := server.NewHTTP(runtime, server.WithAgentHookReceiverLimits(64, time.Second))

	body := `{"provider":"claude","eventName":"Notification"}` + strings.Repeat(" ", 128)
	assertStatus(t, handler, http.MethodPost, "/v1/agent-hook-events", body, http.StatusRequestEntityTooLarge)
}

func TestHTTPServerAgentHookReceiverTimesOutSlowBody(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{})
	handler := server.NewHTTP(runtime, server.WithAgentHookReceiverLimits(1024, 10*time.Millisecond))
	body := newBlockingReadCloser()
	req := httptest.NewRequest(http.MethodPost, "/v1/agent-hook-events", nil)
	req.Body = body
	recorder := httptest.NewRecorder()

	started := time.Now()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusRequestTimeout {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("timeout took too long: %v", elapsed)
	}
}

func TestHTTPServerAgentBridgeApprovalLifecycle(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	backend := newFakePTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend:                 backend,
		IDGenerator:                func() string { nextID++; return "id_" + strconv.Itoa(nextID) },
		DaemonURL:                  "http://127.0.0.1:8787",
		AgentBridgeApprovalTimeout: 5 * time.Second,
	})
	handler := server.NewHTTP(runtime)

	project := postJSON[protocol.Project](t, handler, "/v1/projects", protocol.CreateProjectRequest{
		Name:    "App",
		RootDir: t.TempDir(),
	}, http.StatusCreated)
	item := postJSON[protocol.WorkItem](t, handler, "/v1/work-items", protocol.CreateWorkItemRequest{
		ProjectID: project.ID,
		Title:     "Bridge approval",
		Actor:     "human",
	}, http.StatusCreated)
	run := postJSON[protocol.WorkItemRun](t, handler, "/v1/work-item-runs", protocol.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Launch:           true,
		AgentProfileID:   "claude",
		Actor:            "agent",
	}, http.StatusCreated)
	spawn := backend.spawns[run.PTYID]
	bridgeID := spawn.Env["WHISK_AGENT_BRIDGE_ID"]
	token := spawn.Env["WHISK_AGENT_BRIDGE_TOKEN"]
	if bridgeID == "" || token == "" {
		t.Fatalf("spawn bridge env = %#v", spawn.Env)
	}

	// A permission hook with no pre-supplied decision blocks until the approval is resolved.
	hookBody, err := json.Marshal(protocol.AgentBridgeHookRequest{
		Token:     token,
		Provider:  "claude",
		EventName: "PermissionRequest",
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "ls"},
	})
	if err != nil {
		t.Fatalf("marshal hook: %v", err)
	}
	hookStatus := make(chan int, 1)
	go func() {
		recorder := httptest.NewRecorder()
		recorder.Body = &bytes.Buffer{}
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/agent-bridges/"+bridgeID+"/hooks", bytes.NewReader(hookBody)))
		hookStatus <- recorder.Code
	}()

	var approvalID string
	for i := 0; i < 200; i++ {
		approvals := getJSON[[]protocol.AgentBridgeApproval](t, handler, "/v1/agent-bridge-approvals?status=pending", http.StatusOK)
		if len(approvals) == 1 {
			approvalID = approvals[0].ID
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if approvalID == "" {
		t.Fatalf("no pending approval appeared")
	}
	pendingPrompts := getJSON[[]protocol.AgentPrompt](t, handler, "/v1/agent-prompts?status=pending", http.StatusOK)
	if len(pendingPrompts) != 1 || pendingPrompts[0].ID != approvalID || pendingPrompts[0].Kind != "approval" || len(pendingPrompts[0].Options) != 2 {
		t.Fatalf("pending prompts = %#v, approval = %q", pendingPrompts, approvalID)
	}

	resolved := postJSON[protocol.AgentBridgeApproval](t, handler, "/v1/agent-bridge-approvals/"+approvalID+"/resolve",
		protocol.ResolveAgentBridgeApprovalRequest{Action: "allow"}, http.StatusOK)
	if resolved.ID != approvalID || resolved.Decision.Action != "allow" {
		t.Fatalf("resolved approval = %#v", resolved)
	}

	select {
	case code := <-hookStatus:
		if code != http.StatusOK {
			t.Fatalf("hook status = %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("hook did not return after approval")
	}

	secondHookStatus := make(chan int, 1)
	go func() {
		recorder := httptest.NewRecorder()
		recorder.Body = &bytes.Buffer{}
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/agent-bridges/"+bridgeID+"/hooks", bytes.NewReader(hookBody)))
		secondHookStatus <- recorder.Code
	}()

	var promptID string
	for i := 0; i < 200; i++ {
		prompts := getJSON[[]protocol.AgentPrompt](t, handler, "/v1/agent-prompts?status=pending", http.StatusOK)
		for _, prompt := range prompts {
			if prompt.ID != approvalID {
				promptID = prompt.ID
				break
			}
		}
		if promptID != "" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if promptID == "" {
		t.Fatalf("no second pending prompt appeared")
	}
	resolvedPrompt := postJSON[protocol.AgentPrompt](t, handler, "/v1/agent-prompts/"+promptID+"/resolve",
		protocol.ResolveAgentPromptRequest{Answer: "deny"}, http.StatusOK)
	if resolvedPrompt.ID != promptID || resolvedPrompt.Status != "resolved" || resolvedPrompt.Answer != "deny" {
		t.Fatalf("resolved prompt = %#v", resolvedPrompt)
	}
	resolvedPrompts := getJSON[[]protocol.AgentPrompt](t, handler, "/v1/agent-prompts?status=resolved", http.StatusOK)
	if len(resolvedPrompts) == 0 {
		t.Fatalf("resolved prompts = %#v", resolvedPrompts)
	}
	select {
	case code := <-secondHookStatus:
		if code != http.StatusOK {
			t.Fatalf("second hook status = %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("second hook did not return after prompt resolution")
	}
}

func TestHTTPServerWorkItemWorkflowRoutes(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: newFakePTYBackend(),
		DaemonURL:  "http://127.0.0.1:8787",
		CLIPath:    "/usr/local/bin/whisk",
	})
	handler := server.NewHTTP(runtime)

	project := postJSON[protocol.Project](t, handler, "/v1/projects", protocol.CreateProjectRequest{
		Name:    "App",
		RootDir: t.TempDir(),
		Preferences: protocol.ProjectPreferences{
			AutoRun: workitem.AutoRunNever,
		},
	}, http.StatusCreated)
	projects := getJSON[[]protocol.Project](t, handler, "/v1/projects", http.StatusOK)
	if len(projects) != 1 || projects[0].ID != project.ID {
		t.Fatalf("projects = %#v", projects)
	}
	updatedName := "Renamed App"
	updatedSlug := "renamed-app"
	updatedProject := postJSON[protocol.Project](t, handler, "/v1/projects/"+project.ID+"/update", protocol.UpdateProjectRequest{
		Name: &updatedName,
		Slug: &updatedSlug,
	}, http.StatusOK)
	if updatedProject.Name != updatedName || updatedProject.Slug != updatedSlug {
		t.Fatalf("updated project = %#v", updatedProject)
	}
	noteProject := postJSON[protocol.Project](t, handler, "/v1/projects/"+project.ID+"/attachments", protocol.AddProjectAttachmentRequest{
		Kind:             workitem.AttachmentKindNote,
		Title:            "Decision",
		Note:             "Keep it small.",
		IncludeInContext: true,
	}, http.StatusCreated)
	if len(noteProject.Attachments) != 1 || noteProject.Attachments[0].Title != "Decision" {
		t.Fatalf("note project = %#v", noteProject)
	}
	noteAttachmentID := noteProject.Attachments[0].ID
	updatedTitle := "Tiny decision"
	updatedNote := "Still keep it small."
	updatedAttachmentProject := postJSON[protocol.Project](t, handler, "/v1/project-attachments/"+noteAttachmentID+"/update", protocol.UpdateProjectAttachmentRequest{
		ProjectID: project.ID,
		Title:     &updatedTitle,
		Note:      &updatedNote,
	}, http.StatusOK)
	if updatedAttachmentProject.Attachments[0].Title != updatedTitle || updatedAttachmentProject.Attachments[0].Note != updatedNote {
		t.Fatalf("updated attachment project = %#v", updatedAttachmentProject)
	}
	projectContext := getJSON[protocol.ProjectContext](t, handler, "/v1/projects/"+project.ID+"/context", http.StatusOK)
	if len(projectContext.Items) != 1 || projectContext.Items[0].Content != updatedNote {
		t.Fatalf("project context = %#v", projectContext)
	}
	detail := getJSON[protocol.ProjectDetail](t, handler, "/v1/projects/"+project.ID+"/detail", http.StatusOK)
	if detail.Project.ID != project.ID || len(detail.WorkItems) != 0 {
		t.Fatalf("detail = %#v", detail)
	}
	deletedAttachmentProject := postJSON[protocol.Project](t, handler, "/v1/project-attachments/"+noteAttachmentID+"/delete", protocol.DeleteProjectAttachmentRequest{ProjectID: project.ID}, http.StatusOK)
	if len(deletedAttachmentProject.Attachments) != 0 {
		t.Fatalf("deleted attachment project = %#v", deletedAttachmentProject)
	}
	if templates := getJSON[[]protocol.WorkflowTemplate](t, handler, "/v1/workflow-templates", http.StatusOK); len(templates) == 0 {
		t.Fatalf("workflow templates = %#v", templates)
	}
	if prompts := getJSON[[]protocol.PromptTemplate](t, handler, "/v1/prompt-templates", http.StatusOK); len(prompts) == 0 {
		t.Fatalf("prompt templates = %#v", prompts)
	}
	if profiles := getJSON[[]protocol.AgentProfile](t, handler, "/v1/agent-profiles", http.StatusOK); len(profiles) == 0 || profiles[0].ID == "" || profiles[0].PromptInjectionMode == "" || profiles[0].Source != "builtin" || !profiles[0].Launchable {
		t.Fatalf("agent profiles = %#v", profiles)
	}

	routeWorkflow := workitem.DefaultWorkflowDefinition()
	routeWorkflow.ID = "route-workflow"
	routeWorkflow.Version = 7
	report := postJSON[protocol.WorkflowValidationReport](t, handler, "/v1/workflow-definitions/validate", protocol.ValidateWorkflowDefinitionRequest{
		Definition: routeWorkflow,
	}, http.StatusOK)
	if !report.Valid || report.Identity != "route-workflow@7" {
		t.Fatalf("workflow validation = %#v", report)
	}
	imported := postJSON[protocol.WorkflowDefinitionRecord](t, handler, "/v1/workflow-definitions/import", protocol.ImportWorkflowDefinitionRequest{
		Definition: routeWorkflow,
		Source:     "test",
	}, http.StatusCreated)
	if imported.ID != routeWorkflow.ID || imported.Version != routeWorkflow.Version {
		t.Fatalf("imported workflow = %#v", imported)
	}
	definitions := getJSON[[]protocol.WorkflowDefinitionRecord](t, handler, "/v1/workflow-definitions", http.StatusOK)
	if len(definitions) < 2 {
		t.Fatalf("workflow definitions = %#v", definitions)
	}

	fileWorkflow := routeWorkflow
	fileWorkflow.ID = "route-workflow-file"
	fileWorkflow.Version = 1
	workflowPath := filepath.Join(t.TempDir(), "workflow.json")
	workflowPayload, err := json.Marshal(fileWorkflow)
	if err != nil {
		t.Fatalf("marshal workflow: %v", err)
	}
	if err := os.WriteFile(workflowPath, workflowPayload, 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	fileReport := postJSON[protocol.WorkflowValidationReport](t, handler, "/v1/workflow-definitions/validate-file", protocol.ValidateWorkflowDefinitionFileRequest{
		Path: workflowPath,
	}, http.StatusOK)
	if !fileReport.Valid || fileReport.Identity != "route-workflow-file@1" {
		t.Fatalf("file workflow validation = %#v", fileReport)
	}
	fileImported := postJSON[protocol.WorkflowDefinitionRecord](t, handler, "/v1/workflow-definitions/import-file", protocol.ImportWorkflowDefinitionFileRequest{
		Path: workflowPath,
	}, http.StatusCreated)
	if fileImported.ID != fileWorkflow.ID || fileImported.Source != "file" {
		t.Fatalf("file imported workflow = %#v", fileImported)
	}
	exportPath := filepath.Join(t.TempDir(), "exported-workflow.json")
	postNoContent(t, handler, "/v1/workflow-definitions/export-file", protocol.ExportWorkflowDefinitionFileRequest{
		ID:      fileImported.ID,
		Version: fileImported.Version,
		Path:    exportPath,
	})
	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("exported workflow stat: %v", err)
	}
	migration := postJSON[protocol.WorkflowMigrationPlan](t, handler, "/v1/projects/"+project.ID+"/workflow-migration-plan", protocol.PlanProjectWorkflowMigrationRequest{
		ID:      imported.ID,
		Version: imported.Version,
	}, http.StatusOK)
	if migration.ProjectID != project.ID || migration.TargetID != imported.ID {
		t.Fatalf("migration = %#v", migration)
	}
	project = postJSON[protocol.Project](t, handler, "/v1/projects/"+project.ID+"/workflow-definition", protocol.SetProjectWorkflowDefinitionRequest{
		ID:      imported.ID,
		Version: imported.Version,
	}, http.StatusOK)
	if project.Workflow.DefinitionID != imported.ID || project.Workflow.DefinitionVersion != imported.Version {
		t.Fatalf("workflow project = %#v", project.Workflow)
	}
	deletedWorkflow := postJSON[protocol.WorkflowDefinitionRecord](t, handler, "/v1/workflow-definitions/"+fileImported.ID+"/"+strconv.Itoa(fileImported.Version)+"/delete", struct{}{}, http.StatusOK)
	if deletedWorkflow.ID != fileImported.ID {
		t.Fatalf("deleted workflow = %#v", deletedWorkflow)
	}

	item := postJSON[protocol.WorkItem](t, handler, "/v1/work-items", protocol.CreateWorkItemRequest{
		ProjectID:    project.ID,
		Title:        "Ship route workflow",
		BodyMarkdown: "Use the daemon route surface.",
		Actor:        "human",
	}, http.StatusCreated)
	if item.ID == "" || item.Number != 1 {
		t.Fatalf("item = %#v", item)
	}
	editedTitle := "Ship edited route workflow"
	editedBody := ""
	editedItem := postJSON[protocol.WorkItem](t, handler, "/v1/work-items/"+item.ID+"/update", protocol.UpdateWorkItemRequest{
		Title:        &editedTitle,
		BodyMarkdown: &editedBody,
		Actor:        "human",
	}, http.StatusOK)
	if editedItem.Title != editedTitle || editedItem.BodyMarkdown != "" {
		t.Fatalf("updated item = %#v", editedItem)
	}
	item = editedItem
	items := getJSON[[]protocol.WorkItem](t, handler, "/v1/work-items?projectId="+project.ID, http.StatusOK)
	if len(items) != 1 || items[0].ID != item.ID {
		t.Fatalf("items = %#v", items)
	}
	actions := getJSON[[]protocol.WorkflowActionAvailability](t, handler, "/v1/work-items/"+item.ID+"/workflow-actions", http.StatusOK)
	if len(actions) == 0 || actions[0].Action.ID == "" {
		t.Fatalf("workflow actions = %#v", actions)
	}
	moved := postJSON[protocol.WorkItem](t, handler, "/v1/work-items/"+item.ID+"/move", protocol.MoveWorkItemRequest{
		StageID: workitem.StagePlanning,
		Actor:   "human",
	}, http.StatusOK)
	if moved.StageID != workitem.StagePlanning {
		t.Fatalf("moved = %#v", moved)
	}
	bound := postJSON[protocol.WorkItem](t, handler, "/v1/work-items/"+item.ID+"/bind-worktree", protocol.BindWorkItemWorktreeRequest{
		Branch:       "whisk/app-1-route-workflow",
		WorktreePath: t.TempDir(),
		Actor:        "human",
	}, http.StatusOK)
	if bound.Worktree == nil || bound.Worktree.Branch != "whisk/app-1-route-workflow" {
		t.Fatalf("bound = %#v", bound)
	}
	attached := postJSON[protocol.WorkItem](t, handler, "/v1/work-items/"+item.ID+"/attachments", protocol.AddWorkItemAttachmentRequest{
		Kind:  "file",
		Path:  "docs/spec.md",
		Actor: "human",
	}, http.StatusCreated)
	if len(attached.Attachments) != 1 || attached.Attachments[0].Path != "docs/spec.md" {
		t.Fatalf("attached = %#v", attached)
	}

	planning := postJSON[protocol.WorkItemRun](t, handler, "/v1/work-items/"+item.ID+"/start-planning", protocol.StartPlanningRequest{
		Actor: "agent",
	}, http.StatusCreated)
	if planning.PromptTemplateID != workitem.PromptTemplatePlan {
		t.Fatalf("planning = %#v", planning)
	}
	draft := postJSON[protocol.Artifact](t, handler, "/v1/work-items/"+item.ID+"/plan-drafts", protocol.SubmitDraftPlanRequest{
		RunID: planning.ID,
		Title: "Plan",
		Body:  "Implement it.",
		Actor: "agent",
	}, http.StatusCreated)
	if draft.Kind != workitem.ArtifactKindPlan || draft.Status != workitem.ArtifactStatusDraft {
		t.Fatalf("draft = %#v", draft)
	}
	ready := postJSON[protocol.WorkItem](t, handler, "/v1/work-items/"+item.ID+"/approve-plan", protocol.ApprovePlanRequest{
		ArtifactID: draft.ID,
		Actor:      "human",
	}, http.StatusOK)
	if ready.StageID != workitem.StageReady {
		t.Fatalf("ready = %#v", ready)
	}
	queued := postJSON[protocol.WorkItemRun](t, handler, "/v1/work-items/"+item.ID+"/queue-execution", protocol.QueueExecutionRequest{
		Actor: "human",
	}, http.StatusCreated)
	if queued.Status != workitem.RunStateQueued || queued.PromptTemplateID != workitem.PromptTemplateImplement {
		t.Fatalf("queued = %#v", queued)
	}
	launched := postJSON[protocol.WorkItemRun](t, handler, "/v1/work-item-runs/"+queued.ID+"/launch", protocol.LaunchWorkItemRunRequest{
		Actor: "agent",
	}, http.StatusOK)
	if launched.Status != workitem.RunStateRunning || launched.SessionID == "" || launched.PTYID == "" {
		t.Fatalf("launched = %#v", launched)
	}
	runs := getJSON[[]protocol.WorkItemRun](t, handler, "/v1/work-item-runs?workItemId="+item.ID, http.StatusOK)
	if len(runs) != 2 || runs[1].ID != queued.ID {
		t.Fatalf("runs = %#v", runs)
	}

	status := postJSON[protocol.ReportStatusResponse](t, handler, "/v1/status", protocol.ReportStatusRequest{
		Kind:       workitem.StatusKindQuestion,
		Message:    "Running tests.",
		Actor:      "agent",
		ProjectID:  project.ID,
		WorkItemID: item.ID,
		RunID:      launched.ID,
		SessionID:  launched.SessionID,
		PTYID:      launched.PTYID,
	}, http.StatusCreated)
	if status.Event.ID == "" || status.Run == nil || status.WorkItem == nil {
		t.Fatalf("status = %#v", status)
	}
	events := getJSON[[]protocol.StatusEvent](t, handler, "/v1/status-events?workItemId="+item.ID+"&unreadOnly=true", http.StatusOK)
	if len(events) != 1 || events[0].ID != status.Event.ID {
		t.Fatalf("status events = %#v", events)
	}
	read := postJSON[protocol.StatusEvent](t, handler, "/v1/status-events/"+status.Event.ID+"/read", protocol.MarkStatusEventReadRequest{}, http.StatusOK)
	if read.ReadAt == nil {
		t.Fatalf("read = %#v", read)
	}

	question := postJSON[protocol.Question](t, handler, "/v1/questions", protocol.AskQuestionRequest{
		WorkItemID: item.ID,
		RunID:      launched.ID,
		Prompt:     "Which key?",
		Actor:      "agent",
	}, http.StatusCreated)
	if question.Status != workitem.QuestionStatusOpen {
		t.Fatalf("question = %#v", question)
	}
	answered := postJSON[protocol.Question](t, handler, "/v1/questions/"+question.ID+"/answer", protocol.AnswerQuestionRequest{
		Answer: "Staging.",
		Actor:  "human",
	}, http.StatusOK)
	if answered.Status != workitem.QuestionStatusAnswered {
		t.Fatalf("answered = %#v", answered)
	}
	questions := getJSON[[]protocol.Question](t, handler, "/v1/questions?workItemId="+item.ID, http.StatusOK)
	if len(questions) != 1 || questions[0].ID != question.ID {
		t.Fatalf("questions = %#v", questions)
	}

	review := postJSON[protocol.WorkItem](t, handler, "/v1/work-item-runs/"+launched.ID+"/complete-execution", protocol.CompleteExecutionRequest{
		Message: "Done.",
		Actor:   "agent",
	}, http.StatusOK)
	if review.StageID != workitem.StageReview {
		t.Fatalf("review = %#v", review)
	}
	gates := getJSON[[]protocol.GateReport](t, handler, "/v1/gate-reports?workItemId="+item.ID, http.StatusOK)
	if len(gates) != 1 || gates[0].Status != workitem.GateStatusPending {
		t.Fatalf("gates = %#v", gates)
	}
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/"+item.ID+"/approve-done", `{"actor":"human"}`, http.StatusBadRequest)
	feedback := postJSON[protocol.Artifact](t, handler, "/v1/work-items/"+item.ID+"/review-feedback", protocol.SubmitReviewFeedbackRequest{
		RunID: launched.ID,
		Body:  "Tighten validation.",
		Actor: "human",
	}, http.StatusCreated)
	if feedback.Kind != workitem.ArtifactKindFeedback {
		t.Fatalf("feedback = %#v", feedback)
	}
	artifacts := getJSON[[]protocol.Artifact](t, handler, "/v1/artifacts?workItemId="+item.ID, http.StatusOK)
	if len(artifacts) != 2 {
		t.Fatalf("artifacts = %#v", artifacts)
	}
	passed := postJSON[protocol.GateReport](t, handler, "/v1/gate-reports/"+gates[0].ID+"/complete", protocol.CompleteGateRequest{
		Status: workitem.GateStatusPassed,
		Actor:  "agent",
	}, http.StatusOK)
	if passed.Status != workitem.GateStatusPassed {
		t.Fatalf("passed = %#v", passed)
	}
	done := postJSON[protocol.WorkItem](t, handler, "/v1/work-items/"+item.ID+"/approve-done", protocol.ApproveDoneRequest{
		Reason: "review passed",
		Actor:  "human",
	}, http.StatusOK)
	if done.StageID != workitem.StageDone {
		t.Fatalf("done = %#v", done)
	}
	workflowEvents := getJSON[[]protocol.WorkflowEvent](t, handler, "/v1/workflow-events?workItemId="+item.ID, http.StatusOK)
	if len(workflowEvents) == 0 || workflowEvents[len(workflowEvents)-1].Type != workitem.WorkflowEventDoneApproved {
		t.Fatalf("workflow events = %#v", workflowEvents)
	}

	runOnlyItem := postJSON[protocol.WorkItem](t, handler, "/v1/work-items", protocol.CreateWorkItemRequest{
		ProjectID: project.ID,
		Title:     "Standalone run",
		Actor:     "human",
	}, http.StatusCreated)
	runOnly := postJSON[protocol.WorkItemRun](t, handler, "/v1/work-item-runs", protocol.StartWorkItemRunRequest{
		WorkItemID:       runOnlyItem.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Actor:            "agent",
	}, http.StatusCreated)
	cancelled := postJSON[protocol.WorkItemRun](t, handler, "/v1/work-item-runs/"+runOnly.ID+"/cancel", protocol.CancelWorkItemRunRequest{
		Actor: "human",
	}, http.StatusOK)
	if cancelled.Status != workitem.RunStateCancelled {
		t.Fatalf("cancelled = %#v", cancelled)
	}

	execItem := createReadyWorkItemViaHTTP(t, handler, project.ID, "Start execution")
	execution := postJSON[protocol.WorkItemRun](t, handler, "/v1/work-items/"+execItem.ID+"/start-execution", protocol.StartExecutionRequest{
		Actor: "agent",
	}, http.StatusCreated)
	if execution.Status != workitem.RunStateQueued {
		t.Fatalf("execution = %#v", execution)
	}
	launchItem := createReadyWorkItemViaHTTP(t, handler, project.ID, "Launch execution")
	launchExecution := postJSON[protocol.WorkItemRun](t, handler, "/v1/work-items/"+launchItem.ID+"/launch-execution", protocol.LaunchExecutionRequest{
		Actor: "agent",
	}, http.StatusCreated)
	if launchExecution.Status != workitem.RunStateRunning || launchExecution.PTYID == "" {
		t.Fatalf("launch execution = %#v", launchExecution)
	}
	blockedItem := postJSON[protocol.WorkItem](t, handler, "/v1/work-items", protocol.CreateWorkItemRequest{
		ProjectID: project.ID,
		Title:     "Blocked by dependency",
		Actor:     "human",
	}, http.StatusCreated)
	blockedItem = postJSON[protocol.WorkItem](t, handler, "/v1/work-items/"+blockedItem.ID+"/move", protocol.MoveWorkItemRequest{
		StageID: workitem.StageReady,
		Actor:   "human",
	}, http.StatusOK)
	blockerItem := postJSON[protocol.WorkItem](t, handler, "/v1/work-items", protocol.CreateWorkItemRequest{
		ProjectID: project.ID,
		Title:     "Dependency",
		Actor:     "human",
	}, http.StatusCreated)
	blockerItem = postJSON[protocol.WorkItem](t, handler, "/v1/work-items/"+blockerItem.ID+"/move", protocol.MoveWorkItemRequest{
		StageID: workitem.StageReady,
		Actor:   "human",
	}, http.StatusOK)
	link := postJSON[protocol.WorkItemLink](t, handler, "/v1/work-item-links", protocol.AddWorkItemLinkRequest{
		SourceWorkItemID: blockedItem.ID,
		TargetWorkItemID: blockerItem.ID,
		Type:             workitem.WorkItemLinkBlocks,
		Actor:            "human",
	}, http.StatusCreated)
	if link.SourceWorkItemID != blockedItem.ID || link.TargetWorkItemID != blockerItem.ID {
		t.Fatalf("link = %#v", link)
	}
	links := getJSON[[]protocol.WorkItemLink](t, handler, "/v1/work-item-links?workItemId="+blockedItem.ID, http.StatusOK)
	if len(links) != 1 || links[0].ID != link.ID {
		t.Fatalf("links = %#v", links)
	}
	readyWork := getJSON[protocol.ReadyWorkExplanation](t, handler, "/v1/ready-work?projectId="+project.ID, http.StatusOK)
	foundBlocked := false
	for _, blocked := range readyWork.Blocked {
		if blocked.WorkItem.ID == blockedItem.ID {
			foundBlocked = true
			break
		}
	}
	if !foundBlocked || readyWork.Summary.TotalBlocked == 0 {
		t.Fatalf("ready work = %#v", readyWork)
	}

	deleteItem := postJSON[protocol.WorkItem](t, handler, "/v1/work-items", protocol.CreateWorkItemRequest{
		ProjectID: project.ID,
		Title:     "Delete me",
		Actor:     "human",
	}, http.StatusCreated)
	deleted := postJSON[protocol.WorkItem](t, handler, "/v1/work-items/"+deleteItem.ID+"/delete", protocol.DeleteWorkItemRequest{
		Actor: "human",
	}, http.StatusOK)
	if deleted.ID != deleteItem.ID || len(deleted.History) == 0 || deleted.History[len(deleted.History)-1].Type != workitem.HistoryDeleted {
		t.Fatalf("deleted = %#v", deleted)
	}
	deletedProject := postJSON[protocol.Project](t, handler, "/v1/projects/"+project.ID+"/delete", protocol.DeleteProjectRequest{Actor: "human"}, http.StatusOK)
	if deletedProject.ID != project.ID {
		t.Fatalf("deleted project = %#v", deletedProject)
	}
	if projects := getJSON[[]protocol.Project](t, handler, "/v1/projects", http.StatusOK); len(projects) != 0 {
		t.Fatalf("projects after delete = %#v", projects)
	}
}

func TestHTTPServerWorktreeFlow(t *testing.T) {
	backend := &fakeWorktreeBackend{
		status: app.WorktrunkStatus{
			Available:   true,
			ConfigFound: true,
			Binary:      app.WorktrunkBinary{Path: "/bin/wt", Version: "0.44.0"},
		},
		worktrees: []app.Worktree{{Branch: "feature", Path: "/repo/.worktrees/feature", Kind: "worktree", Dirty: true, Locked: true}},
		created:   app.CreatedWorktree{Path: "/repo/.worktrees/created"},
	}
	runtime := app.NewRuntime(app.RuntimeConfig{Worktrees: backend})
	handler := server.NewHTTP(runtime)

	status := postJSON[protocol.WorktrunkStatus](t, handler, "/v1/worktrunk/detect", protocol.DetectWorktrunkRequest{
		RepoPath:     "/repo",
		OverridePath: "/custom/wt",
	}, http.StatusOK)
	if !status.Available || !status.ConfigFound || status.Binary.Path != "/bin/wt" {
		t.Fatalf("status = %#v", status)
	}
	if backend.detectReq.OverridePath != "/custom/wt" {
		t.Fatalf("detect req = %#v", backend.detectReq)
	}

	worktrees := postJSON[[]protocol.Worktree](t, handler, "/v1/worktrees/list", protocol.ListWorktreesRequest{RepoPath: "/repo"}, http.StatusOK)
	if len(worktrees) != 1 || worktrees[0].Branch != "feature" || !worktrees[0].Dirty || !worktrees[0].Locked {
		t.Fatalf("worktrees = %#v", worktrees)
	}

	created := postJSON[protocol.CreatedWorktree](t, handler, "/v1/worktrees/create", protocol.CreateWorktreeRequest{
		RepoPath: "/repo",
		Branch:   "created",
		Base:     "main",
	}, http.StatusCreated)
	if created.Path != "/repo/.worktrees/created" || backend.createReq.Base != "main" {
		t.Fatalf("created = %#v req = %#v", created, backend.createReq)
	}

	postNoContent(t, handler, "/v1/worktrees/remove", protocol.RemoveWorktreeRequest{
		RepoPath:     "/repo",
		WorktreePath: "/repo/.worktrees/created",
	})
	if backend.removeReq.WorktreePath != "/repo/.worktrees/created" {
		t.Fatalf("remove req = %#v", backend.removeReq)
	}
}

func TestHTTPServerHTTPForwardFlow(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/base/nested" || r.URL.RawQuery != "q=1" {
			t.Fatalf("target path = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Forward-Test", "hit")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("target:" + string(body)))
	}))
	t.Cleanup(target.Close)

	runtime := app.NewRuntime(app.RuntimeConfig{})
	handler := server.NewHTTP(runtime)

	created := postJSON[protocol.HTTPForward](t, handler, "/v1/http-forwards", protocol.CreateHTTPForwardRequest{
		Name:      "difit",
		TargetURL: target.URL + "/base",
		SessionID: "session_01",
	}, http.StatusCreated)
	if created.ID == "" || created.Name != "difit" || created.SessionID != "session_01" {
		t.Fatalf("created = %#v", created)
	}

	forwards := getJSON[[]protocol.HTTPForward](t, handler, "/v1/http-forwards", http.StatusOK)
	if len(forwards) != 1 || forwards[0].ID != created.ID {
		t.Fatalf("forwards = %#v", forwards)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/http-forwards/"+created.ID+"/proxy/nested?q=1", strings.NewReader("body"))
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusAccepted || recorder.Header().Get("X-Forward-Test") != "hit" || recorder.Body.String() != "target:body" {
		t.Fatalf("proxy status=%d header=%q body=%q", recorder.Code, recorder.Header().Get("X-Forward-Test"), recorder.Body.String())
	}

	deleteNoContent(t, handler, "/v1/http-forwards/"+created.ID)
	assertStatus(t, handler, http.MethodGet, "/v1/http-forwards/"+created.ID+"/proxy", "", http.StatusNotFound)
}

func TestHTTPServerReportsBadRequests(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: newFakePTYBackend()})
	handler := server.NewHTTP(runtime)

	assertStatus(t, handler, http.MethodPost, "/v1/sessions", "{", http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/sessions/missing/split", `{"targetPaneId":"pane","direction":"diagonal"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodGet, "/v1/ptys/missing/output?from=nope", "", http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/ptys/missing/write", `{"data":"x"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/ptys/missing/resize", `{"cols":0,"rows":24}`, http.StatusBadRequest)

	assertStatus(t, handler, http.MethodPost, "/v1/worktrunk/detect", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/worktrunk/detect", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/worktrees/list", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/worktrees/create", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/worktrees/remove", `{}`, http.StatusBadRequest)

	assertStatus(t, handler, http.MethodPost, "/v1/http-forwards", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/http-forwards", `{"targetUrl":"http://example.com"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodDelete, "/v1/http-forwards/missing", "", http.StatusNotFound)
	assertStatus(t, handler, http.MethodGet, "/v1/http-forwards/missing/proxy", "", http.StatusNotFound)

	assertStatus(t, handler, http.MethodPost, "/v1/projects", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/projects", `{"name":"App"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items", `{"title":"Task"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/update", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/update", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/update", `{"title":"Task"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/move", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/move", `{"stageId":"execution"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/start-planning", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/start-planning", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/plan-drafts", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/plan-drafts", `{"body":"Plan"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/approve-plan", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/approve-plan", `{"artifactId":"missing"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/start-execution", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/start-execution", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/queue-execution", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/queue-execution", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/launch-execution", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/launch-execution", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/questions", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/questions", `{"prompt":"Which key?"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/questions/missing/answer", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/questions/missing/answer", `{"answer":"Staging"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-item-runs/missing/complete-execution", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-item-runs/missing/complete-execution", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/review-feedback", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/review-feedback", `{"body":"Fix it."}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/approve-done", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/approve-done", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/gate-reports/missing/complete", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/gate-reports/missing/complete", `{"status":"passed"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/bind-worktree", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/bind-worktree", `{"branch":"feature","worktreePath":"/tmp/work"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/attachments", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/attachments", `{"kind":"note","note":"context"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/delete", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-items/missing/delete", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-item-runs", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-item-runs", `{"workItemId":"missing"}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-item-runs/missing/launch", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-item-runs/missing/launch", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-item-runs/missing/cancel", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/work-item-runs/missing/cancel", `{}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/status", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/status", `{"kind":"question","message":"Need input."}`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/status-events/missing/read", `{`, http.StatusBadRequest)
	assertStatus(t, handler, http.MethodPost, "/v1/status-events/missing/read", `{}`, http.StatusBadRequest)
}

type fakePTYBackend struct {
	mu      sync.Mutex
	records map[string]app.PTYRecord
	outputs map[string][]byte
	spawns  map[string]app.SpawnPTYRequest
	events  chan app.PTYEvent
}

func newFakePTYBackend() *fakePTYBackend {
	return &fakePTYBackend{
		records: map[string]app.PTYRecord{},
		outputs: map[string][]byte{},
		spawns:  map[string]app.SpawnPTYRequest{},
	}
}

type blockingReadCloser struct {
	once sync.Once
	done chan struct{}
}

func newBlockingReadCloser() *blockingReadCloser {
	return &blockingReadCloser{done: make(chan struct{})}
}

func (b *blockingReadCloser) Read([]byte) (int, error) {
	<-b.done
	return 0, io.EOF
}

func (b *blockingReadCloser) Close() error {
	b.once.Do(func() { close(b.done) })
	return nil
}

func (b *fakePTYBackend) Spawn(_ context.Context, req app.SpawnPTYRequest) (app.PTYRecord, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	record := app.PTYRecord{
		ID:         req.ID,
		WorkingDir: req.WorkingDir,
		Cols:       req.Cols,
		Rows:       req.Rows,
		Running:    true,
	}
	b.records[req.ID] = record
	b.spawns[req.ID] = req
	return record, nil
}

func (b *fakePTYBackend) Write(_ context.Context, ptyID string, data []byte) error {
	b.mu.Lock()
	if _, ok := b.records[ptyID]; !ok {
		b.mu.Unlock()
		return errNotFound(ptyID)
	}
	offset := uint64(len(b.outputs[ptyID]))
	b.outputs[ptyID] = append(b.outputs[ptyID], data...)
	event := app.PTYEvent{Kind: app.PTYOutput, Offset: offset, Bytes: append([]byte(nil), data...)}
	if b.events != nil {
		b.events <- event
	}
	b.mu.Unlock()
	return nil
}

func (b *fakePTYBackend) waitForSubscriber(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for {
		b.mu.Lock()
		ready := b.events != nil
		b.mu.Unlock()
		if ready {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for fake PTY subscriber")
		}
		time.Sleep(time.Millisecond)
	}
}

func (b *fakePTYBackend) emitOutput(t *testing.T, ptyID string, data []byte) {
	t.Helper()
	b.mu.Lock()
	if _, ok := b.records[ptyID]; !ok {
		b.mu.Unlock()
		t.Fatalf("pty %s not found", ptyID)
	}
	offset := uint64(len(b.outputs[ptyID]))
	b.outputs[ptyID] = append(b.outputs[ptyID], data...)
	event := app.PTYEvent{Kind: app.PTYOutput, Offset: offset, Bytes: append([]byte(nil), data...)}
	events := b.events
	b.mu.Unlock()
	if events == nil {
		t.Fatalf("pty %s has no subscriber", ptyID)
	}
	events <- event
}

func (b *fakePTYBackend) emitExit(t *testing.T, ptyID string, code int) {
	t.Helper()
	b.mu.Lock()
	record, ok := b.records[ptyID]
	if !ok {
		b.mu.Unlock()
		t.Fatalf("pty %s not found", ptyID)
	}
	record.Running = false
	b.records[ptyID] = record
	events := b.events
	b.mu.Unlock()
	if events == nil {
		t.Fatalf("pty %s has no subscriber", ptyID)
	}
	events <- app.PTYEvent{Kind: app.PTYExit, Code: &code}
}

func (b *fakePTYBackend) Resize(_ context.Context, ptyID string, size app.PTYSize) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	record, ok := b.records[ptyID]
	if !ok {
		return errNotFound(ptyID)
	}
	record.Cols = size.Cols
	record.Rows = size.Rows
	b.records[ptyID] = record
	return nil
}

func (b *fakePTYBackend) Kill(_ context.Context, ptyID string) (app.PTYRecord, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	record, ok := b.records[ptyID]
	if !ok {
		return app.PTYRecord{}, errNotFound(ptyID)
	}
	record.Running = false
	b.records[ptyID] = record
	return record, nil
}

func (b *fakePTYBackend) Delete(_ context.Context, ptyID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	record, ok := b.records[ptyID]
	if !ok {
		return errNotFound(ptyID)
	}
	if record.Running {
		return fmt.Errorf("cannot delete running pty %s", ptyID)
	}
	delete(b.records, ptyID)
	return nil
}

func (b *fakePTYBackend) Attach(ctx context.Context, req app.AttachPTYRequest) (*app.PTYAttach, error) {
	b.mu.Lock()
	record, ok := b.records[req.PtyID]
	if !ok {
		b.mu.Unlock()
		return nil, errNotFound(req.PtyID)
	}
	output := b.outputs[req.PtyID]
	replayOffset := req.ReplayFromOffset
	if replayOffset > uint64(len(output)) {
		replayOffset = uint64(len(output))
	}
	replay := append([]byte(nil), output[replayOffset:]...)
	ch := make(chan app.PTYEvent, 16)
	b.events = ch
	b.mu.Unlock()

	var once sync.Once
	closeAttach := func() {
		once.Do(func() {
			b.mu.Lock()
			if b.events == ch {
				b.events = nil
			}
			b.mu.Unlock()
			close(ch)
		})
	}
	return &app.PTYAttach{
		Record:       record,
		ReplayBytes:  replay,
		ReplayOffset: replayOffset,
		Events:       ch,
		CloseFunc:    closeAttach,
	}, nil
}

func (b *fakePTYBackend) Output(_ context.Context, ptyID string, fromOffset uint64) (app.PTYOutputSnapshot, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	record, ok := b.records[ptyID]
	if !ok {
		return app.PTYOutputSnapshot{}, errNotFound(ptyID)
	}
	output := b.outputs[ptyID]
	if fromOffset > uint64(len(output)) {
		fromOffset = uint64(len(output))
	}
	return app.PTYOutputSnapshot{
		Record:      record,
		Offset:      fromOffset,
		OutputBytes: append([]byte(nil), output[fromOffset:]...),
	}, nil
}

func (b *fakePTYBackend) List(context.Context) ([]app.PTYRecord, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]app.PTYRecord, 0, len(b.records))
	for _, record := range b.records {
		out = append(out, record)
	}
	return out, nil
}

func (b *fakePTYBackend) Shutdown(context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.records = map[string]app.PTYRecord{}
	b.outputs = map[string][]byte{}
	b.events = nil
	return nil
}

type fakeEventBus struct {
	ch       chan app.RuntimeEvent
	afterSeq uint64
	missed   bool
}

type pluginRegistryFake struct {
	statuses    []app.PluginStatus
	usageResult app.UsageResolverResult
	usageReq    app.RunUsageResolverRequest
	usageErr    error
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
	return []app.RegistryPlugin{{Registry: "phin-tech", ID: "github", Name: "GitHub Issues", SourceType: "path"}}, nil
}

func (f *pluginRegistryFake) InstallPlugin(_ context.Context, registry, id string) (app.PluginStatus, error) {
	status := app.PluginStatus{ID: id, Registry: registry, Name: "GitHub Issues", Valid: true}
	f.statuses = []app.PluginStatus{status}
	return status, nil
}

func (f *pluginRegistryFake) ListAgentProfiles(context.Context) ([]agents.ProfileInfo, error) {
	return nil, nil
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

func (f *pluginRegistryFake) RunUsageResolver(_ context.Context, req app.RunUsageResolverRequest) (app.UsageResolverResult, error) {
	f.usageReq = req
	if f.usageErr != nil {
		return app.UsageResolverResult{}, f.usageErr
	}
	return f.usageResult, nil
}

func (f *pluginRegistryFake) ResolveProjectAttachmentProvider(string) app.ProjectContextResolver {
	return nil
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

type errNotFound string

func (e errNotFound) Error() string {
	return "pty " + string(e) + " not found"
}

type fakeWorktreeBackend struct {
	status    app.WorktrunkStatus
	worktrees []app.Worktree
	created   app.CreatedWorktree

	detectReq app.DetectWorktrunkRequest
	listReq   app.ListWorktreesRequest
	createReq app.CreateWorktreeRequest
	removeReq app.RemoveWorktreeRequest
}

func (b *fakeWorktreeBackend) DetectWorktrunk(_ context.Context, req app.DetectWorktrunkRequest) (app.WorktrunkStatus, error) {
	b.detectReq = req
	return b.status, nil
}

func (b *fakeWorktreeBackend) ListWorktrees(_ context.Context, req app.ListWorktreesRequest) ([]app.Worktree, error) {
	b.listReq = req
	return b.worktrees, nil
}

func (b *fakeWorktreeBackend) CreateWorktree(_ context.Context, req app.CreateWorktreeRequest) (app.CreatedWorktree, error) {
	b.createReq = req
	return b.created, nil
}

func (b *fakeWorktreeBackend) RemoveWorktree(_ context.Context, req app.RemoveWorktreeRequest) error {
	b.removeReq = req
	return nil
}

func createReadyWorkItemViaHTTP(t *testing.T, handler http.Handler, projectID string, title string) protocol.WorkItem {
	t.Helper()
	item := postJSON[protocol.WorkItem](t, handler, "/v1/work-items", protocol.CreateWorkItemRequest{
		ProjectID: projectID,
		Title:     title,
		Actor:     "human",
	}, http.StatusCreated)
	planning := postJSON[protocol.WorkItemRun](t, handler, "/v1/work-items/"+item.ID+"/start-planning", protocol.StartPlanningRequest{
		Actor: "agent",
	}, http.StatusCreated)
	draft := postJSON[protocol.Artifact](t, handler, "/v1/work-items/"+item.ID+"/plan-drafts", protocol.SubmitDraftPlanRequest{
		RunID: planning.ID,
		Body:  "Implement it.",
		Actor: "agent",
	}, http.StatusCreated)
	return postJSON[protocol.WorkItem](t, handler, "/v1/work-items/"+item.ID+"/approve-plan", protocol.ApprovePlanRequest{
		ArtifactID: draft.ID,
		Actor:      "human",
	}, http.StatusOK)
}

func postJSON[T any](t *testing.T, handler http.Handler, path string, body any, wantStatus int) T {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded)))
	if recorder.Code != wantStatus {
		t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
	}
	var out T
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func postNoContent(t *testing.T, handler http.Handler, path string, body any) {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded)))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
	}
}

func deleteNoContent(t *testing.T, handler http.Handler, path string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, path, nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
	}
}

func deleteJSON[T any](t *testing.T, handler http.Handler, path string, wantStatus int) T {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, path, nil))
	if recorder.Code != wantStatus {
		t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
	}
	var out T
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func getJSON[T any](t *testing.T, handler http.Handler, path string, wantStatus int) T {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != wantStatus {
		t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
	}
	var out T
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func assertStatus(t *testing.T, handler http.Handler, method string, path string, body string, wantStatus int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s status = %d body = %s", method, path, recorder.Code, recorder.Body.String())
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func testAgentHookPaths(t *testing.T) agenthooks.Paths {
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
