package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/agents"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/agentbridge"
	"github.com/phin-tech/whisk/internal/domain/workitem"
)

func TestRuntimeWorkItemRunLifecyclePersistsAndPublishes(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	sink := &memoryEventSink{}
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		EventSink:     sink,
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Wire runs"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		SessionID:        "sess_01",
		PTYID:            "pty_01",
		Actor:            "agent",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.Status != workitem.RunStateQueued || run.PromptSnapshot == "" {
		t.Fatalf("run = %#v", run)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil || len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("runs = %#v, err = %v", runs, err)
	}
	cancelled, err := runtime.CancelWorkItemRun(ctx, app.CancelWorkItemRunRequest{ID: run.ID, Actor: "agent"})
	if err != nil {
		t.Fatalf("cancel run: %v", err)
	}
	if cancelled.Status != workitem.RunStateCancelled {
		t.Fatalf("cancelled = %#v", cancelled)
	}
	if store.saved.Runs[len(store.saved.Runs)-1].Status != workitem.RunStateCancelled {
		t.Fatalf("saved runs = %#v", store.saved.Runs)
	}
	if len(sink.events) != 4 {
		t.Fatalf("events = %#v", sink.events)
	}
}

func TestRuntimeUpdateWorkItemPersistsAndPublishes(t *testing.T) {
	ctx := context.Background()
	store := &memoryWorkItemStore{}
	sink := &memoryEventSink{}
	runtime := app.NewRuntime(app.RuntimeConfig{
		WorkItemStore: store,
		EventSink:     sink,
	})
	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{
		ProjectID:    project.ID,
		Title:        "Old",
		BodyMarkdown: "Old body",
	})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	sink.events = nil
	title := "New"
	body := "New body"

	updated, err := runtime.UpdateWorkItem(ctx, app.UpdateWorkItemRequest{
		ID:           item.ID,
		Title:        &title,
		BodyMarkdown: &body,
		Actor:        "human",
	})
	if err != nil {
		t.Fatalf("update work item: %v", err)
	}
	if updated.Title != title || updated.BodyMarkdown != body {
		t.Fatalf("updated item = %#v", updated)
	}
	if len(store.saved.Items) != 1 || store.saved.Items[0].Title != title {
		t.Fatalf("saved snapshot = %#v", store.saved.Items)
	}
	if !hasRuntimeEvent(sink.events, app.EventWorkItemsChanged) {
		t.Fatalf("events = %#v", sink.events)
	}
}

func TestRuntimeStartWorkItemRunLaunchesAgentPTY(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	worktreeDir := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Wire agent"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	item, err = runtime.BindWorkItemWorktree(ctx, app.BindWorkItemWorktreeRequest{
		ID:           item.ID,
		Branch:       "whisk/app-1-wire-agent",
		WorktreePath: worktreeDir,
	})
	if err != nil {
		t.Fatalf("bind worktree: %v", err)
	}

	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Launch:           true,
		AgentProfileID:   "codex",
		SystemPrompt:     "Be direct.",
		Actor:            "agent",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.Status != workitem.RunStateRunning || run.SessionID == "" || run.PTYID == "" {
		t.Fatalf("run = %#v", run)
	}
	if len(ptyBackend.spawns) != 1 || ptyBackend.spawns[0].WorkingDir != worktreeDir {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
	if ptyBackend.spawns[0].Command != "codex" ||
		!containsArg(ptyBackend.spawns[0].Args, "instructions=Be direct.") ||
		!containsArgWith(ptyBackend.spawns[0].Args, "Implement the work item.") ||
		!containsArgWith(ptyBackend.spawns[0].Args, "Wire agent") {
		t.Fatalf("spawn command/args = %q %#v", ptyBackend.spawns[0].Command, ptyBackend.spawns[0].Args)
	}
	env := ptyBackend.spawns[0].Env
	if env["WHISKD_URL"] != "http://127.0.0.1:8787" ||
		env["WHISK_CLI"] != "/usr/local/bin/whisk" ||
		env["WHISK_PROJECT_ID"] != project.ID ||
		env["WHISK_WORK_ITEM_ID"] != item.ID ||
		env["WHISK_RUN_ID"] != run.ID ||
		env["WHISK_SESSION_ID"] != run.SessionID ||
		env["WHISK_PTY_ID"] != run.PTYID ||
		env["WHISK_ACTOR"] != "agent" {
		t.Fatalf("spawn env = %#v", env)
	}
	wantPath := []string{"/usr/local/bin", "/opt/homebrew/bin", filepath.Join(home, ".local", "bin"), filepath.Join(home, "bin"), "/usr/bin", "/bin"}
	if gotPath := filepath.SplitList(env["PATH"]); !reflect.DeepEqual(gotPath, wantPath) {
		t.Fatalf("spawn PATH = %#v, want %#v", gotPath, wantPath)
	}
	writes := ptyBackend.writes[run.PTYID]
	if len(writes) != 0 {
		t.Fatalf("writes = %#v", writes)
	}
	sessions, err := runtime.ListSessions(ctx)
	if err != nil || len(sessions) != 1 || sessions[0].RootDir != worktreeDir {
		t.Fatalf("sessions = %#v, err = %v", sessions, err)
	}
	if sessions[0].Name != "#1 Execution - Wire agent" {
		t.Fatalf("session name = %q", sessions[0].Name)
	}
}

func TestRuntimeStartPlanningUsesCodexPlanProfileForCodexReaderDefault(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: &memoryWorkItemStore{},
		PTYBackend:    ptyBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{
		Name:    "App",
		RootDir: root,
		Preferences: workitem.ProjectPreferences{
			DefaultPhaseAgents: map[string]string{
				workitem.RunPresetReader: "codex",
			},
		},
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Plan with Codex"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}

	run, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{
		WorkItemID: item.ID,
		Launch:     true,
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	if run.Status != workitem.RunStateRunning {
		t.Fatalf("run = %#v", run)
	}
	if len(ptyBackend.spawns) != 1 {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
	spawn := ptyBackend.spawns[0]
	if spawn.Command != "codex" ||
		!containsArg(spawn.Args, "--sandbox") ||
		!containsArg(spawn.Args, "read-only") ||
		!containsArgWith(spawn.Args, "Plan the work item.") ||
		!containsArgWith(spawn.Args, "Plan with Codex") {
		t.Fatalf("spawn command/args = %q %#v", spawn.Command, spawn.Args)
	}
}

func TestRuntimeStartWorkItemRunUsesInteractiveAgentShellWhenEnabled(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{
		Name:    "App",
		RootDir: root,
		Preferences: workitem.ProjectPreferences{
			UseInteractiveAgentShell: true,
		},
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Wire agent"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Launch:           true,
		AgentProfileID:   "claude",
		SystemPrompt:     "Don't overbuild.",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	if len(ptyBackend.spawns) != 1 {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
	spawn := ptyBackend.spawns[0]
	if spawn.Command != "/bin/zsh" || len(spawn.Args) != 2 || spawn.Args[0] != "-lc" {
		t.Fatalf("spawn command/args = %q %#v", spawn.Command, spawn.Args)
	}
	// The prompt rides in argv (claude auto-runs the first turn), so it appears in
	// the launched command line — not typed into the PTY.
	if !strings.Contains(spawn.Args[1], "claude --append-system-prompt") || !strings.Contains(spawn.Args[1], "overbuild.") {
		t.Fatalf("spawn command line = %q", spawn.Args[1])
	}
	if !strings.Contains(spawn.Args[1], "Implement the work item.") {
		t.Fatalf("prompt should be passed as an argument, command line = %q", spawn.Args[1])
	}
	if writes := ptyBackend.writes[run.PTYID]; len(writes) != 0 {
		t.Fatalf("interactive agent prompt must not be typed into the PTY, writes = %#v", writes)
	}
}

func TestRuntimeStartWorkItemRunDoesNotBlockWaitingForInteractiveAgentReady(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	setupCtx := context.Background()
	project, err := runtime.CreateProject(setupCtx, app.CreateProjectRequest{
		Name:    "App",
		RootDir: root,
		Preferences: workitem.ProjectPreferences{
			UseInteractiveAgentShell: true,
		},
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(setupCtx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Wire agent"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Launch:           true,
		AgentProfileID:   "claude",
	})
	if err != nil {
		t.Fatalf("start run should not wait for Claude readiness: %v", err)
	}
	if run.PTYID == "" {
		t.Fatalf("run missing pty: %#v", run)
	}
}

func TestSubmitReviewFeedbackSubmitsIntoRunningAgentTUI(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Review me"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Launch:           true,
		AgentProfileID:   "claude",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.PTYID == "" {
		t.Fatalf("run missing pty: %#v", run)
	}

	if _, err := runtime.SubmitReviewFeedback(ctx, app.SubmitReviewFeedbackRequest{
		WorkItemID: item.ID,
		RunID:      run.ID,
		Body:       "Tighten the tests.\nAdd an edge case.",
		Actor:      "reviewer",
	}); err != nil {
		t.Fatalf("submit review feedback: %v", err)
	}

	// Delivery is async (clear → paste → settle → separate Enter).
	var writes [][]byte
	deadline := time.Now().Add(3 * time.Second)
	for {
		ptyBackend.mu.Lock()
		writes = append([][]byte(nil), ptyBackend.writes[run.PTYID]...)
		ptyBackend.mu.Unlock()
		if len(writes) >= 3 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if len(writes) != 3 {
		t.Fatalf("expected clear+paste+enter writes, got %d: %q", len(writes), writes)
	}
	if string(writes[0]) != "\x01\x0b" {
		t.Fatalf("first write must clear the draft (Ctrl-A Ctrl-K), got %q", writes[0])
	}
	paste := string(writes[1])
	if !strings.HasPrefix(paste, "\x1b[200~") || !strings.HasSuffix(paste, "\x1b[201~") {
		t.Fatalf("second write must be a bracketed paste, got %q", paste)
	}
	if !strings.Contains(paste, "Tighten the tests.") || strings.Contains(paste, "\n") {
		t.Fatalf("paste body must be CR-joined with no line feed, got %q", paste)
	}
	if string(writes[2]) != "\r" {
		t.Fatalf("final write must be a separate submit Enter, got %q", writes[2])
	}
}

func TestRuntimeStartWorkItemRunPassesAgentProfileEnvToPTY(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "or-runtime-key")
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "Remote", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Remote smoke"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}

	_, err = runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Launch:           true,
		AgentProfileID:   "claude-openrouter",
		Actor:            "agent",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if len(ptyBackend.spawns) != 1 {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
	env := ptyBackend.spawns[0].Env
	if env["ANTHROPIC_BASE_URL"] != "https://openrouter.ai/api" ||
		env["ANTHROPIC_AUTH_TOKEN"] != "or-runtime-key" ||
		env["ANTHROPIC_API_KEY"] != "" ||
		env["WHISK_RUN_ID"] == "" {
		t.Fatalf("env = %#v", env)
	}
}

func TestRuntimeStartWorkItemRunInjectsAgentBridgeEnvForProviderHooks(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Bridge hooks"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}

	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Launch:           true,
		AgentProfileID:   "claude",
		Actor:            "agent",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.Status != workitem.RunStateRunning || run.SessionID == "" || run.PTYID == "" {
		t.Fatalf("run = %#v", run)
	}
	if len(ptyBackend.spawns) != 1 {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
	env := ptyBackend.spawns[0].Env
	if env["WHISK_AGENT_BRIDGE_ID"] == "" ||
		env["WHISK_AGENT_BRIDGE_TOKEN"] == "" ||
		env["WHISK_AGENT_BRIDGE_PROVIDER"] != "claude" ||
		env["WHISK_AGENT_BRIDGE_HOOK_URL"] != "http://127.0.0.1:8787/v1/agent-bridges/"+env["WHISK_AGENT_BRIDGE_ID"]+"/hooks" ||
		env["WHISK_AGENT_BRIDGE_CONFIG_DIR"] == "" {
		t.Fatalf("bridge env = %#v", env)
	}
	if strings.Contains(env["WHISK_AGENT_BRIDGE_CONFIG_DIR"], env["WHISK_AGENT_BRIDGE_TOKEN"]) {
		t.Fatalf("bridge config dir must not contain hook token: %#v", env)
	}
	if _, err := os.Stat(filepath.Join(env["WHISK_AGENT_BRIDGE_CONFIG_DIR"], "bridge.json")); err != nil {
		t.Fatalf("bridge config not installed: %v", err)
	}
	if info, err := os.Stat(env["WHISK_AGENT_BRIDGE_HOOK_SCRIPT"]); err != nil || info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("bridge hook script not executable: info=%v err=%v", info, err)
	}
}

func TestRuntimeCreateSessionCanEnableAgentBridgeHooksForRegularTerminal(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		PTYBackend: ptyBackend,
		DaemonURL:  "http://127.0.0.1:8787",
		CLIPath:    "/usr/local/bin/whisk",
	})

	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:    "Agent terminal",
		RootDir: root,
		InitialPTY: &app.StartPTYOptions{
			Command: "claude",
			Exec:    true,
			AgentBridge: &app.StartPTYAgentBridgeOptions{
				Enabled:  true,
				Provider: "claude",
			},
		},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.MainPtyID == "" || len(ptyBackend.spawns) != 1 {
		t.Fatalf("created = %#v spawns = %#v", created, ptyBackend.spawns)
	}
	env := ptyBackend.spawns[0].Env
	if env["WHISK_AGENT_BRIDGE_ID"] == "" ||
		env["WHISK_AGENT_BRIDGE_TOKEN"] == "" ||
		env["WHISK_AGENT_BRIDGE_PROVIDER"] != "claude" ||
		env["WHISK_AGENT_BRIDGE_HOOK_SCRIPT"] == "" {
		t.Fatalf("agent bridge env = %#v", env)
	}
	if _, err := os.Stat(filepath.Join(env["WHISK_AGENT_BRIDGE_CONFIG_DIR"], "bridge.json")); err != nil {
		t.Fatalf("bridge config not written: %v", err)
	}
}

func TestRuntimeAgentBridgeHookWaitsForApprovalResolution(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	ptyBackend := newMemoryPTYBackend()
	sink := &memoryEventSink{}
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		PTYBackend:                 ptyBackend,
		EventSink:                  sink,
		DaemonURL:                  "http://127.0.0.1:8787",
		CLIPath:                    "/usr/local/bin/whisk",
		AgentBridgeApprovalTimeout: time.Second,
	})

	if _, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:    "Agent terminal",
		RootDir: root,
		InitialPTY: &app.StartPTYOptions{
			Command: "claude",
			Exec:    true,
			AgentBridge: &app.StartPTYAgentBridgeOptions{
				Enabled:  true,
				Provider: "claude",
			},
		},
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	env := ptyBackend.spawns[0].Env
	resultCh := make(chan app.AgentBridgeHookResponse, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := runtime.HandleAgentBridgeHook(ctx, app.AgentBridgeHookRequest{
			BridgeID:  env["WHISK_AGENT_BRIDGE_ID"],
			Token:     env["WHISK_AGENT_BRIDGE_TOKEN"],
			Provider:  "claude",
			EventName: "PermissionRequest",
			ToolName:  "Bash",
			ToolInput: map[string]any{"command": "pwd"},
		})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- resp
	}()

	approval := waitForPendingAgentBridgeApproval(t, runtime)
	if approval.ToolName != "Bash" || approval.ToolInput["command"] != "pwd" {
		t.Fatalf("approval = %#v", approval)
	}
	if _, err := runtime.ResolveAgentBridgeApproval(ctx, app.ResolveAgentBridgeApprovalRequest{
		ID:     approval.ID,
		Action: "deny",
		Reason: "blocked by human",
	}); err != nil {
		t.Fatalf("resolve approval: %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("hook error: %v", err)
	case resp := <-resultCh:
		hookSpecific, ok := resp.Output["hookSpecificOutput"].(map[string]any)
		if !ok || hookSpecific["permissionDecision"] != "deny" || hookSpecific["permissionDecisionReason"] != "blocked by human" {
			t.Fatalf("hook response = %#v", resp)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for hook response")
	}
	if !hasRuntimeEvent(sink.events, app.EventAgentBridgeApprovalsChanged) || !hasRuntimeEvent(sink.events, app.EventAgentHookEventsChanged) {
		t.Fatalf("events = %#v", sink.events)
	}
}

func TestRuntimeAgentBridgeHookWaitsForPromptResolution(t *testing.T) {
	runtime, bridgeID, token := launchAgentBridge(t, time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resultCh := make(chan app.AgentBridgeHookResponse, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := runtime.HandleAgentBridgeHook(ctx, app.AgentBridgeHookRequest{
			BridgeID:      bridgeID,
			Token:         token,
			Provider:      "claude",
			EventName:     "Elicitation",
			ElicitationID: "ask_01",
			RawPayload: map[string]any{
				"tool_input": map[string]any{
					"questions": []any{
						map[string]any{
							"question": "What now?",
							"options": []any{
								map[string]any{"label": "Ship it", "value": "ship"},
							},
						},
					},
				},
			},
		})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- resp
	}()

	prompt := waitForPendingAgentPrompt(t, runtime)
	if prompt.Kind != agentbridge.PromptKindQuestion || prompt.Message != "What now?" || len(prompt.Options) != 1 {
		t.Fatalf("prompt = %#v", prompt)
	}
	if _, err := runtime.ResolveAgentPrompt(ctx, app.ResolveAgentPromptRequest{
		ID:     prompt.ID,
		Answer: "ship",
	}); err != nil {
		t.Fatalf("resolve prompt: %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("hook error: %v", err)
	case resp := <-resultCh:
		hookSpecific, ok := resp.Output["hookSpecificOutput"].(map[string]any)
		if !ok || hookSpecific["decision"] != "ship" || hookSpecific["elicitationId"] != "ask_01" {
			t.Fatalf("response = %#v", resp.Output)
		}
		events, err := runtime.ListAgentBridgeEvents(ctx, app.ListAgentBridgeEventsRequest{Status: "pending"})
		if err != nil || len(events) != 0 {
			t.Fatalf("pending hook events = %#v, err = %v", events, err)
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for hook response")
	}
}

func TestRuntimeAgentBridgeHookAnswersClaudeAskUserQuestion(t *testing.T) {
	runtime, bridgeID, token := launchAgentBridge(t, time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	toolInput := map[string]any{
		"questions": []any{
			map[string]any{
				"question": "Which planet rotates on its side?",
				"options": []any{
					map[string]any{"label": "Saturn"},
					map[string]any{"label": "Uranus"},
				},
			},
		},
	}
	resultCh := make(chan app.AgentBridgeHookResponse, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := runtime.HandleAgentBridgeHook(ctx, app.AgentBridgeHookRequest{
			BridgeID:   bridgeID,
			Token:      token,
			Provider:   "claude",
			EventName:  "PermissionRequest",
			ToolName:   "AskUserQuestion",
			ToolInput:  toolInput,
			RawPayload: map[string]any{"tool_input": toolInput},
		})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- resp
	}()

	prompt := waitForPendingAgentPrompt(t, runtime)
	if prompt.Message != "Which planet rotates on its side?" || len(prompt.Options) != 2 {
		t.Fatalf("prompt = %#v", prompt)
	}
	if _, err := runtime.ResolveAgentPrompt(ctx, app.ResolveAgentPromptRequest{
		ID:     prompt.ID,
		Answer: "Uranus",
	}); err != nil {
		t.Fatalf("resolve prompt: %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("hook error: %v", err)
	case resp := <-resultCh:
		hookSpecific, ok := resp.Output["hookSpecificOutput"].(map[string]any)
		if !ok || hookSpecific["permissionDecision"] != "allow" {
			t.Fatalf("response = %#v", resp.Output)
		}
		updatedInput, ok := hookSpecific["updatedInput"].(map[string]any)
		if !ok {
			t.Fatalf("updatedInput = %#v", hookSpecific["updatedInput"])
		}
		answers, ok := updatedInput["answers"].(map[string]any)
		if !ok || answers["Which planet rotates on its side?"] != "Uranus" {
			t.Fatalf("answers = %#v", updatedInput["answers"])
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for hook response")
	}
}

func hasRuntimeEvent(events []app.RuntimeEvent, eventType app.RuntimeEventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func waitForPendingAgentBridgeApproval(t *testing.T, runtime *app.Runtime) agentbridge.Approval {
	t.Helper()
	deadline := time.After(time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for pending approval")
		case <-tick.C:
			approvals, err := runtime.ListAgentBridgeApprovals(context.Background(), app.ListAgentBridgeApprovalsRequest{Status: "pending"})
			if err != nil {
				t.Fatalf("list approvals: %v", err)
			}
			if len(approvals) > 0 {
				return approvals[0]
			}
		}
	}
}

func waitForPendingAgentPrompt(t *testing.T, runtime *app.Runtime) agentbridge.Prompt {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		prompts, err := runtime.ListAgentPrompts(context.Background(), app.ListAgentPromptsRequest{Status: "pending"})
		if err != nil {
			t.Fatalf("list prompts: %v", err)
		}
		if len(prompts) > 0 {
			return prompts[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for pending prompt")
	return agentbridge.Prompt{}
}

func TestRuntimeLaunchQueuedWorkItemRunLaunchesExistingRun(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Launch queued"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Actor:            "agent",
	})
	if err != nil {
		t.Fatalf("queue run: %v", err)
	}
	if run.Status != workitem.RunStateQueued {
		t.Fatalf("queued run = %#v", run)
	}

	launched, err := runtime.LaunchWorkItemRun(ctx, app.LaunchWorkItemRunRequest{
		ID:             run.ID,
		AgentProfileID: "codex",
		SystemPrompt:   "Be direct.",
		Actor:          "agent",
	})
	if err != nil {
		t.Fatalf("launch queued run: %v", err)
	}
	if launched.ID != run.ID || launched.Status != workitem.RunStateRunning || launched.SessionID == "" || launched.PTYID == "" {
		t.Fatalf("launched run = %#v", launched)
	}
	sessions, err := runtime.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ProjectID != project.ID {
		t.Fatalf("sessions = %#v", sessions)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("runs = %#v", runs)
	}
	if len(ptyBackend.spawns) != 1 || !containsArgWith(ptyBackend.spawns[0].Args, "Launch queued") {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
}

func TestRuntimeLaunchFailuresMarkRunsFailed(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	sink := &memoryEventSink{}
	runtime := app.NewRuntime(app.RuntimeConfig{
		WorkItemStore: store,
		EventSink:     sink,
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Launch failure"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	launched, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetReviewer,
		PromptTemplateID: workitem.PromptTemplateReview,
		Launch:           true,
		Actor:            "agent",
	})
	if err == nil || !strings.Contains(err.Error(), "pty backend required") {
		t.Fatalf("expected pty backend error, got run=%#v err=%v", launched, err)
	}
	if launched.Status != workitem.RunStateFailed || launched.CompletedAt == nil {
		t.Fatalf("failed launch run = %#v", launched)
	}
	queued, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetManager,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Actor:            "agent",
	})
	if err != nil {
		t.Fatalf("queue run: %v", err)
	}
	failed, err := runtime.LaunchWorkItemRun(ctx, app.LaunchWorkItemRunRequest{ID: queued.ID, Actor: "agent"})
	if err == nil || !strings.Contains(err.Error(), "pty backend required") {
		t.Fatalf("expected launch error, got run=%#v err=%v", failed, err)
	}
	if failed.ID != queued.ID || failed.Status != workitem.RunStateFailed || failed.CompletedAt == nil {
		t.Fatalf("failed queued launch = %#v", failed)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 2 || runs[0].Status != workitem.RunStateFailed || runs[1].Status != workitem.RunStateFailed {
		t.Fatalf("runs = %#v", runs)
	}
	if len(store.saved.Runs) != 2 || len(sink.events) < 4 {
		t.Fatalf("saved=%#v events=%#v", store.saved.Runs, sink.events)
	}
}

func TestRuntimeQueueExecutionMovesReadyItemAndCreatesQueuedRun(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	sink := &memoryEventSink{}
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		EventSink:     sink,
	})

	item := createApprovedWorkItem(t, ctx, runtime, root, "Queue execution")
	run, err := runtime.QueueExecution(ctx, app.QueueExecutionRequest{
		WorkItemID: item.ID,
		Actor:      "human",
	})
	if err != nil {
		t.Fatalf("queue execution: %v", err)
	}
	if run.Status != workitem.RunStateQueued || run.PromptTemplateID != workitem.PromptTemplateImplement {
		t.Fatalf("run = %#v", run)
	}
	updated := findWorkItem(t, ctx, runtime, item.ID)
	if updated.StageID != workitem.StageExecution {
		t.Fatalf("updated item = %#v", updated)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 2 || runs[1].ID != run.ID {
		t.Fatalf("runs = %#v", runs)
	}
	if store.saved.Runs[len(store.saved.Runs)-1].ID != run.ID {
		t.Fatalf("saved runs = %#v", store.saved.Runs)
	}
	if len(sink.events) != 6 {
		t.Fatalf("events = %#v", sink.events)
	}
}

func TestRuntimeLaunchExecutionMovesReadyItemAndLaunchesRun(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	worktreeDir := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})

	item := createApprovedWorkItem(t, ctx, runtime, root, "Launch execution")
	if _, err := runtime.BindWorkItemWorktree(ctx, app.BindWorkItemWorktreeRequest{
		ID:           item.ID,
		Branch:       "whisk/app-1-launch-execution",
		WorktreePath: worktreeDir,
	}); err != nil {
		t.Fatalf("bind worktree: %v", err)
	}

	run, err := runtime.LaunchExecution(ctx, app.LaunchExecutionRequest{
		WorkItemID:     item.ID,
		AgentProfileID: "codex",
		SystemPrompt:   "Be direct.",
		Actor:          "agent",
	})
	if err != nil {
		t.Fatalf("launch execution: %v", err)
	}
	if run.Status != workitem.RunStateRunning || run.SessionID == "" || run.PTYID == "" {
		t.Fatalf("run = %#v", run)
	}
	updated := findWorkItem(t, ctx, runtime, item.ID)
	if updated.StageID != workitem.StageExecution {
		t.Fatalf("updated item = %#v", updated)
	}
	if len(ptyBackend.spawns) != 1 || ptyBackend.spawns[0].WorkingDir != worktreeDir {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
	if ptyBackend.spawns[0].Command != "codex" ||
		!containsArg(ptyBackend.spawns[0].Args, "instructions=Be direct.") ||
		!containsArgWith(ptyBackend.spawns[0].Args, "Implement the work item.") ||
		!containsArgWith(ptyBackend.spawns[0].Args, "Launch execution") {
		t.Fatalf("spawn command/args = %q %#v", ptyBackend.spawns[0].Command, ptyBackend.spawns[0].Args)
	}
}

func TestRuntimeLaunchExecutionCreatesAndBindsWorktreeWhenMissing(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	worktreeBackend := &worktreeBackendFake{
		created: app.CreatedWorktree{Path: filepath.Join(root, ".worktrees", "app-1-launch-execution")},
	}
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
		Worktrees:     worktreeBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})

	item := createApprovedWorkItem(t, ctx, runtime, root, "Launch execution")
	run, err := runtime.LaunchExecution(ctx, app.LaunchExecutionRequest{
		WorkItemID:           item.ID,
		AgentProfileID:       "codex",
		WorktreeOverridePath: "/custom/wt",
		Actor:                "agent",
	})
	if err != nil {
		t.Fatalf("launch execution: %v", err)
	}
	if run.Status != workitem.RunStateRunning {
		t.Fatalf("run = %#v", run)
	}
	if worktreeBackend.createReq.RepoPath != root ||
		worktreeBackend.createReq.Branch != "whisk/app-1-launch-execution" ||
		worktreeBackend.createReq.OverridePath != "/custom/wt" {
		t.Fatalf("create worktree req = %#v", worktreeBackend.createReq)
	}
	updated := findWorkItem(t, ctx, runtime, item.ID)
	if updated.Worktree == nil ||
		updated.Worktree.Branch != "whisk/app-1-launch-execution" ||
		updated.Worktree.WorktreePath != filepath.Join(root, ".worktrees", "app-1-launch-execution") {
		t.Fatalf("updated work item = %#v", updated)
	}
	if len(ptyBackend.spawns) != 1 || ptyBackend.spawns[0].WorkingDir != updated.Worktree.WorktreePath {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
	if ptyBackend.spawns[0].Command != "codex" {
		t.Fatalf("spawn command/args = %q %#v", ptyBackend.spawns[0].Command, ptyBackend.spawns[0].Args)
	}
}

func TestRuntimeCloseLinkedRunSessionCancelsActiveRun(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})

	item := createApprovedWorkItem(t, ctx, runtime, root, "Close linked terminal")
	run, err := runtime.LaunchExecution(ctx, app.LaunchExecutionRequest{
		WorkItemID:     item.ID,
		AgentProfileID: "codex",
		Actor:          "agent",
	})
	if err != nil {
		t.Fatalf("launch execution: %v", err)
	}
	if run.Status != workitem.RunStateRunning || run.SessionID == "" || run.PTYID == "" {
		t.Fatalf("run = %#v", run)
	}

	if _, err := runtime.CloseSession(ctx, app.CloseSessionRequest{SessionID: run.SessionID}); err != nil {
		t.Fatalf("close session: %v", err)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	found := false
	for _, got := range runs {
		if got.ID != run.ID {
			continue
		}
		found = true
		if got.Status != workitem.RunStateCancelled || got.CompletedAt == nil {
			t.Fatalf("closed session run = %#v", got)
		}
	}
	if !found {
		t.Fatalf("run missing from %#v", runs)
	}
	updated := findWorkItem(t, ctx, runtime, item.ID)
	if updated.RunState != workitem.RunStateCancelled {
		t.Fatalf("work item = %#v", updated)
	}
}

func TestRuntimeKillLinkedRunPTYCancelsActiveRun(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	ptyBackend := newMemoryPTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: ptyBackend,
		DaemonURL:  "http://127.0.0.1:8787",
		CLIPath:    "/usr/local/bin/whisk",
	})

	item := createApprovedWorkItem(t, ctx, runtime, root, "Kill linked terminal")
	run, err := runtime.LaunchExecution(ctx, app.LaunchExecutionRequest{
		WorkItemID:     item.ID,
		AgentProfileID: "codex",
		Actor:          "agent",
	})
	if err != nil {
		t.Fatalf("launch execution: %v", err)
	}

	if _, err := runtime.KillPTY(ctx, app.KillPTYRequest{PTYID: run.PTYID}); err != nil {
		t.Fatalf("kill pty: %v", err)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	for _, got := range runs {
		if got.ID == run.ID && got.Status != workitem.RunStateCancelled {
			t.Fatalf("killed pty run = %#v", got)
		}
	}
}

func TestRuntimePTYExitEventCancelsActiveLinkedRun(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	ptyBackend := newAttachableMemoryPTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{
		PTYBackend: ptyBackend,
		DaemonURL:  "http://127.0.0.1:8787",
		CLIPath:    "/usr/local/bin/whisk",
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	item := createApprovedWorkItem(t, ctx, runtime, root, "Exited linked terminal")
	run, err := runtime.LaunchExecution(ctx, app.LaunchExecutionRequest{
		WorkItemID:     item.ID,
		AgentProfileID: "codex",
		Actor:          "agent",
	})
	if err != nil {
		t.Fatalf("launch execution: %v", err)
	}

	ptyBackend.exit(run.PTYID, 0)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
		if err != nil {
			t.Fatalf("list runs: %v", err)
		}
		for _, got := range runs {
			if got.ID == run.ID && got.Status == workitem.RunStateCancelled {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	t.Fatalf("run was not cancelled after pty exit: %#v", runs)
}

func TestRuntimeLoadCancelsActiveRunsWithMissingTerminal(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	state := workitem.NewState()
	project, err := state.CreateProject(workitem.CreateProject{
		ID:      "proj_01",
		Name:    "App",
		RootDir: t.TempDir(),
		Now:     now,
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := state.CreateWorkItem(workitem.CreateWorkItem{
		ID:        "wi_01",
		HistoryID: "hist_01",
		ProjectID: project.ID,
		Title:     "Reload stale run",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	queued, err := state.StartRun(workitem.StartRun{
		ID:               "run_01",
		HistoryID:        "hist_02",
		RunHistoryID:     "run_hist_01",
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		SessionID:        "missing_session",
		PTYID:            "missing_pty",
		Actor:            "agent",
		Now:              now,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if _, err := state.MarkRunRunning(workitem.MarkRunRunning{
		ID:           queued.ID,
		RunHistoryID: "run_hist_02",
		SessionID:    "missing_session",
		PTYID:        "missing_pty",
		DaemonOwned:  true,
		Actor:        "agent",
		Now:          now.Add(time.Second),
	}); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	store := &memoryWorkItemStore{saved: state.Snapshot()}
	runtime, err := app.NewRuntimeWithError(app.RuntimeConfig{WorkItemStore: store})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != workitem.RunStateCancelled || runs[0].CompletedAt == nil {
		t.Fatalf("loaded runs = %#v", runs)
	}
	if len(store.saved.Runs) != 1 || store.saved.Runs[0].Status != workitem.RunStateCancelled {
		t.Fatalf("saved snapshot = %#v", store.saved.Runs)
	}
}

func TestRuntimeDeleteWorkItemClosesLinkedRunSessions(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Delete cleanup"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	run, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{
		WorkItemID:     item.ID,
		Launch:         true,
		AgentProfileID: "codex",
		Actor:          "agent",
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	if run.SessionID == "" || run.PTYID == "" {
		t.Fatalf("run = %#v", run)
	}

	if _, err := runtime.DeleteWorkItem(ctx, app.DeleteWorkItemRequest{ID: item.ID, Actor: "user"}); err != nil {
		t.Fatalf("delete work item: %v", err)
	}
	sessions, err := runtime.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions remain = %#v", sessions)
	}
	record := ptyBackend.records[run.PTYID]
	if record.Running {
		t.Fatalf("pty still running = %#v", record)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("runs remain = %#v", runs)
	}
	if len(store.saved.Runs) != 0 || len(store.saved.Items) != 0 {
		t.Fatalf("saved work item state = %#v", store.saved)
	}
}

func TestRuntimeApprovePlanClosesDaemonLaunchedPlanningSession(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})
	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Draft plan"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	run, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{
		WorkItemID:     item.ID,
		Launch:         true,
		AgentProfileID: "codex",
		Actor:          "agent",
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	draft, err := runtime.SubmitDraftPlan(ctx, app.SubmitDraftPlanRequest{
		WorkItemID: item.ID,
		RunID:      run.ID,
		Body:       "Ship the smallest thing.",
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("submit draft plan: %v", err)
	}

	approved, err := runtime.ApprovePlan(ctx, app.ApprovePlanRequest{ArtifactID: draft.ID, WorkItemID: item.ID, Actor: "human"})
	if err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	if approved.StageID != workitem.StageReady {
		t.Fatalf("approved item = %#v", approved)
	}
	sessions, err := runtime.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions remain = %#v", sessions)
	}
	record := ptyBackend.records[run.PTYID]
	if record.Running {
		t.Fatalf("pty still running = %#v", record)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != workitem.RunStateCompleted {
		t.Fatalf("runs = %#v", runs)
	}
}

func TestRuntimeSubmitDraftPlanCompletesDaemonLaunchedPlanningRun(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: &memoryWorkItemStore{},
		PTYBackend:    ptyBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Draft plan"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	run, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{
		WorkItemID:     item.ID,
		Launch:         true,
		AgentProfileID: "codex",
		Actor:          "agent",
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}

	if _, err := runtime.SubmitDraftPlan(ctx, app.SubmitDraftPlanRequest{
		WorkItemID: item.ID,
		RunID:      run.ID,
		Body:       "Ship the smallest thing.",
		Actor:      "agent",
	}); err != nil {
		t.Fatalf("submit draft plan: %v", err)
	}

	items, err := runtime.ListWorkItems(ctx, project.ID)
	if err != nil {
		t.Fatalf("list work items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %#v", items)
	}
	if items[0].RunState != workitem.RunStateCompleted {
		t.Fatalf("item run state = %q, want completed", items[0].RunState)
	}
	if items[0].StageID != workitem.StagePlanning {
		t.Fatalf("item stage = %q, want planning", items[0].StageID)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != workitem.RunStateCompleted {
		t.Fatalf("runs = %#v", runs)
	}
	sessions, err := runtime.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions remain = %#v", sessions)
	}
	record := ptyBackend.records[run.PTYID]
	if record.Running {
		t.Fatalf("pty still running = %#v", record)
	}
}

func TestRuntimeApprovePlanDoesNotCloseManuallyBoundPlanningSession(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
	})
	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Manual plan"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:    "manual",
		RootDir: root,
		InitialPTY: &app.StartPTYOptions{
			Command: "sh",
			Exec:    true,
		},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	run, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{
		WorkItemID: item.ID,
		SessionID:  created.Session.ID,
		PTYID:      created.MainPtyID,
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	draft, err := runtime.SubmitDraftPlan(ctx, app.SubmitDraftPlanRequest{
		WorkItemID: item.ID,
		RunID:      run.ID,
		Body:       "Manual session stays open.",
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("submit draft plan: %v", err)
	}
	if _, err := runtime.ApprovePlan(ctx, app.ApprovePlanRequest{ArtifactID: draft.ID, WorkItemID: item.ID, Actor: "human"}); err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	sessions, err := runtime.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != created.Session.ID {
		t.Fatalf("sessions = %#v", sessions)
	}
	record := ptyBackend.records[created.MainPtyID]
	if !record.Running {
		t.Fatalf("manual pty was closed = %#v", record)
	}
}

func TestRuntimeCompleteExecutionClosesDaemonLaunchedExecutionSession(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		PTYBackend:    ptyBackend,
		DaemonURL:     "http://127.0.0.1:8787",
		CLIPath:       "/usr/local/bin/whisk",
	})
	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Execute"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	planningRun, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{WorkItemID: item.ID, Actor: "agent"})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	draft, err := runtime.SubmitDraftPlan(ctx, app.SubmitDraftPlanRequest{
		WorkItemID: item.ID,
		RunID:      planningRun.ID,
		Body:       "Implement it.",
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("submit draft plan: %v", err)
	}
	if _, err := runtime.ApprovePlan(ctx, app.ApprovePlanRequest{ArtifactID: draft.ID, WorkItemID: item.ID, Actor: "human"}); err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	run, err := runtime.StartExecution(ctx, app.StartExecutionRequest{
		WorkItemID:     item.ID,
		Launch:         true,
		AgentProfileID: "codex",
		Actor:          "agent",
	})
	if err != nil {
		t.Fatalf("start execution: %v", err)
	}

	review, err := runtime.CompleteExecution(ctx, app.CompleteExecutionRequest{RunID: run.ID, Message: "ready", Actor: "agent"})
	if err != nil {
		t.Fatalf("complete execution: %v", err)
	}
	if review.StageID != workitem.StageReview {
		t.Fatalf("review item = %#v", review)
	}
	sessions, err := runtime.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions remain = %#v", sessions)
	}
	record := ptyBackend.records[run.PTYID]
	if record.Running {
		t.Fatalf("pty still running = %#v", record)
	}
	runs, err := runtime.ListWorkItemRuns(ctx, item.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	found := false
	for _, got := range runs {
		if got.ID == run.ID {
			found = true
			if got.Status != workitem.RunStateCompleted {
				t.Fatalf("execution run = %#v", got)
			}
		}
	}
	if !found {
		t.Fatalf("execution run missing from %#v", runs)
	}
	feedback, err := runtime.SubmitReviewFeedback(ctx, app.SubmitReviewFeedbackRequest{
		WorkItemID: item.ID,
		RunID:      run.ID,
		Body:       "Tighten the tests.",
		Actor:      "reviewer",
	})
	if err != nil {
		t.Fatalf("submit review feedback: %v", err)
	}
	if feedback.Kind != workitem.ArtifactKindFeedback {
		t.Fatalf("feedback = %#v", feedback)
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func createApprovedWorkItem(t *testing.T, ctx context.Context, runtime *app.Runtime, root string, title string) workitem.WorkItem {
	t.Helper()
	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: title})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	planningRun, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{WorkItemID: item.ID, Actor: "agent"})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	draft, err := runtime.SubmitDraftPlan(ctx, app.SubmitDraftPlanRequest{
		WorkItemID: item.ID,
		RunID:      planningRun.ID,
		Body:       "Implement it.",
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("submit draft plan: %v", err)
	}
	approved, err := runtime.ApprovePlan(ctx, app.ApprovePlanRequest{
		ArtifactID: draft.ID,
		WorkItemID: item.ID,
		Actor:      "human",
	})
	if err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	if approved.StageID != workitem.StageReady {
		t.Fatalf("approved item = %#v", approved)
	}
	return approved
}

func findWorkItem(t *testing.T, ctx context.Context, runtime *app.Runtime, id string) workitem.WorkItem {
	t.Helper()
	items, err := runtime.ListWorkItems(ctx, "")
	if err != nil {
		t.Fatalf("list work items: %v", err)
	}
	for _, item := range items {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("work item %s missing from %#v", id, items)
	return workitem.WorkItem{}
}

func containsArgWith(args []string, want string) bool {
	for _, arg := range args {
		if strings.Contains(arg, want) {
			return true
		}
	}
	return false
}

func TestRuntimeReportStatusPersistsAndPublishes(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	sink := &memoryEventSink{}
	ptyBackend := newMemoryPTYBackend()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		EventSink:     sink,
		PTYBackend:    ptyBackend,
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Wire status"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	created, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:       "Run session",
		RootDir:    root,
		ProjectID:  project.ID,
		InitialPTY: &app.StartPTYOptions{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID: item.ID,
		SessionID:  created.Session.ID,
		PTYID:      created.MainPtyID,
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	report, err := runtime.ReportStatus(ctx, app.ReportStatusRequest{
		Kind:    workitem.StatusKindQuestion,
		Message: "Need staging API key.",
		Actor:   "agent",
		RunID:   run.ID,
	})
	if err != nil {
		t.Fatalf("report status: %v", err)
	}
	if report.Event.Kind != workitem.StatusKindQuestion || !report.Event.RequiresAttention {
		t.Fatalf("report = %#v", report)
	}
	if report.Event.SessionID != created.Session.ID || report.Event.PaneID != created.PaneID || report.Event.PTYID != created.MainPtyID {
		t.Fatalf("report target = %#v, created = %#v", report.Event, created)
	}
	if report.Event.NotificationKey != fmt.Sprintf("status|session:%s|pane:%s|actor:agent|kind:question", created.Session.ID, created.PaneID) {
		t.Fatalf("notification key = %q", report.Event.NotificationKey)
	}
	if report.Run == nil || report.Run.Status != workitem.RunStateAwaitingInput {
		t.Fatalf("report run = %#v", report.Run)
	}
	if report.WorkItem == nil || report.WorkItem.RunState != workitem.RunStateAwaitingInput {
		t.Fatalf("report work item = %#v", report.WorkItem)
	}
	if len(store.saved.StatusEvents) != 1 || store.saved.StatusEvents[0].Message != "Need staging API key." {
		t.Fatalf("saved status events = %#v", store.saved.StatusEvents)
	}
	if len(sink.events) == 0 || sink.events[len(sink.events)-1].Type != app.EventStatusChanged {
		t.Fatalf("events = %#v", sink.events)
	}
}

func TestRuntimeWorkflowListsAndAuxiliaryActions(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := &memoryWorkItemStore{}
	sink := &memoryEventSink{}
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore: store,
		EventSink:     sink,
	})

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if templates, err := runtime.ListWorkflowTemplates(ctx); err != nil || len(templates) == 0 {
		t.Fatalf("workflow templates = %#v, err = %v", templates, err)
	}
	if templates, err := runtime.ListPromptTemplates(ctx); err != nil || len(templates) == 0 {
		t.Fatalf("prompt templates = %#v, err = %v", templates, err)
	}
	if profiles, err := runtime.ListAgentProfiles(ctx); err != nil || len(profiles) == 0 || profiles[0].ID == "" {
		t.Fatalf("agent profiles = %#v, err = %v", profiles, err)
	}
	routeWorkflow := workitem.DefaultWorkflowDefinition()
	routeWorkflow.ID = "runtime-workflow"
	routeWorkflow.Version = 3
	if report, err := runtime.ValidateWorkflowDefinition(ctx, app.ValidateWorkflowDefinitionRequest{Definition: routeWorkflow}); err != nil || !report.Valid {
		t.Fatalf("validate workflow = %#v, err = %v", report, err)
	}
	imported, err := runtime.ImportWorkflowDefinition(ctx, app.ImportWorkflowDefinitionRequest{
		Definition: routeWorkflow,
		Source:     "test",
	})
	if err != nil {
		t.Fatalf("import workflow: %v", err)
	}
	if definitions, err := runtime.ListWorkflowDefinitions(ctx); err != nil || len(definitions) < 2 {
		t.Fatalf("workflow definitions = %#v, err = %v", definitions, err)
	}
	fileWorkflow := routeWorkflow
	fileWorkflow.ID = "runtime-workflow-file"
	fileWorkflow.Version = 1
	workflowPath := filepath.Join(t.TempDir(), "workflow.json")
	workflowPayload, err := json.Marshal(fileWorkflow)
	if err != nil {
		t.Fatalf("marshal workflow: %v", err)
	}
	if err := os.WriteFile(workflowPath, workflowPayload, 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if report, err := runtime.ValidateWorkflowDefinitionFile(ctx, app.ValidateWorkflowDefinitionFileRequest{Path: workflowPath}); err != nil || !report.Valid {
		t.Fatalf("validate workflow file = %#v, err = %v", report, err)
	}
	fileImported, err := runtime.ImportWorkflowDefinitionFile(ctx, app.ImportWorkflowDefinitionFileRequest{Path: workflowPath})
	if err != nil {
		t.Fatalf("import workflow file: %v", err)
	}
	exportPath := filepath.Join(t.TempDir(), "workflow-export.json")
	if err := runtime.ExportWorkflowDefinitionFile(ctx, app.ExportWorkflowDefinitionFileRequest{ID: fileImported.ID, Version: fileImported.Version, Path: exportPath}); err != nil {
		t.Fatalf("export workflow: %v", err)
	}
	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("exported workflow stat: %v", err)
	}
	migration, err := runtime.PlanProjectWorkflowMigration(ctx, app.PlanProjectWorkflowMigrationRequest{
		ProjectID: project.ID,
		ID:        imported.ID,
		Version:   imported.Version,
	})
	if err != nil {
		t.Fatalf("plan workflow migration: %v", err)
	}
	if migration.TargetID != imported.ID || migration.ProjectID != project.ID {
		t.Fatalf("migration = %#v", migration)
	}
	project, err = runtime.SetProjectWorkflowDefinition(ctx, app.SetProjectWorkflowDefinitionRequest{
		ProjectID: project.ID,
		ID:        imported.ID,
		Version:   imported.Version,
	})
	if err != nil {
		t.Fatalf("set workflow definition: %v", err)
	}
	if project.Workflow.DefinitionID != imported.ID || project.Workflow.DefinitionVersion != imported.Version {
		t.Fatalf("project workflow = %#v", project.Workflow)
	}
	if deletedWorkflow, err := runtime.DeleteWorkflowDefinition(ctx, app.DeleteWorkflowDefinitionRequest{ID: fileImported.ID, Version: fileImported.Version}); err != nil || deletedWorkflow.ID != fileImported.ID {
		t.Fatalf("delete workflow = %#v, err = %v", deletedWorkflow, err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{
		ProjectID:    project.ID,
		Title:        "Auxiliary flow",
		BodyMarkdown: "Exercise app adapter methods.",
		Actor:        "human",
	})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	actions, err := runtime.ListWorkItemWorkflowActions(ctx, item.ID)
	if err != nil {
		t.Fatalf("list workflow actions: %v", err)
	}
	if len(actions) == 0 || actions[0].Action.ID == "" {
		t.Fatalf("workflow actions = %#v", actions)
	}
	moved, err := runtime.MoveWorkItem(ctx, app.MoveWorkItemRequest{ID: item.ID, StageID: workitem.StagePlanning, Actor: "human"})
	if err != nil {
		t.Fatalf("move work item: %v", err)
	}
	if moved.StageID != workitem.StagePlanning {
		t.Fatalf("moved = %#v", moved)
	}
	attached, err := runtime.AddWorkItemAttachment(ctx, app.AddWorkItemAttachmentRequest{
		WorkItemID: item.ID,
		Kind:       workitem.AttachmentKindURL,
		URL:        "https://example.test/spec",
		Actor:      "human",
	})
	if err != nil {
		t.Fatalf("add attachment: %v", err)
	}
	if len(attached.Attachments) != 1 || attached.Attachments[0].URL == "" {
		t.Fatalf("attached = %#v", attached)
	}
	planning, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{WorkItemID: item.ID, Actor: "agent"})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	question, err := runtime.AskQuestion(ctx, app.AskQuestionRequest{
		WorkItemID: item.ID,
		RunID:      planning.ID,
		SessionID:  "sess_01",
		PTYID:      "pty_01",
		Prompt:     "Which key?",
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("ask question: %v", err)
	}
	answered, err := runtime.AnswerQuestion(ctx, app.AnswerQuestionRequest{ID: question.ID, Answer: "Staging.", Actor: "human"})
	if err != nil {
		t.Fatalf("answer question: %v", err)
	}
	if answered.Status != workitem.QuestionStatusAnswered {
		t.Fatalf("answered = %#v", answered)
	}
	if questions, err := runtime.ListQuestions(ctx, item.ID); err != nil || len(questions) != 1 {
		t.Fatalf("questions = %#v, err = %v", questions, err)
	}
	draft, err := runtime.SubmitDraftPlan(ctx, app.SubmitDraftPlanRequest{
		WorkItemID: item.ID,
		RunID:      planning.ID,
		Title:      "Plan",
		Body:       "Implement it.",
		Actor:      "agent",
	})
	if err != nil {
		t.Fatalf("submit plan: %v", err)
	}
	if artifacts, err := runtime.ListArtifacts(ctx, item.ID); err != nil || len(artifacts) != 1 {
		t.Fatalf("artifacts = %#v, err = %v", artifacts, err)
	}
	if _, err := runtime.ApprovePlan(ctx, app.ApprovePlanRequest{ArtifactID: draft.ID, WorkItemID: item.ID, Actor: "human"}); err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	execution, err := runtime.StartExecution(ctx, app.StartExecutionRequest{WorkItemID: item.ID, Actor: "agent"})
	if err != nil {
		t.Fatalf("start execution: %v", err)
	}
	review, err := runtime.CompleteExecution(ctx, app.CompleteExecutionRequest{RunID: execution.ID, Message: "ready", Actor: "agent"})
	if err != nil {
		t.Fatalf("complete execution: %v", err)
	}
	if review.StageID != workitem.StageReview {
		t.Fatalf("review = %#v", review)
	}
	gates, err := runtime.ListGateReports(ctx, item.ID)
	if err != nil {
		t.Fatalf("list gates: %v", err)
	}
	if len(gates) != 1 || gates[0].Status != workitem.GateStatusPending {
		t.Fatalf("gates = %#v", gates)
	}
	gate, err := runtime.CompleteGate(ctx, app.CompleteGateRequest{
		ID:             gates[0].ID,
		Status:         workitem.GateStatusOverridden,
		OverrideReason: "Manual review passed.",
		Actor:          "human",
	})
	if err != nil {
		t.Fatalf("complete gate: %v", err)
	}
	if gate.Status != workitem.GateStatusOverridden {
		t.Fatalf("gate = %#v", gate)
	}
	done, err := runtime.ApproveDone(ctx, app.ApproveDoneRequest{WorkItemID: item.ID, Reason: "review passed", Actor: "human"})
	if err != nil {
		t.Fatalf("approve done: %v", err)
	}
	if done.StageID != workitem.StageDone {
		t.Fatalf("done = %#v", done)
	}
	blockedItem, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Blocked by dependency", Actor: "human"})
	if err != nil {
		t.Fatalf("create blocked item: %v", err)
	}
	blockedItem, err = runtime.MoveWorkItem(ctx, app.MoveWorkItemRequest{ID: blockedItem.ID, StageID: workitem.StageReady, Actor: "human"})
	if err != nil {
		t.Fatalf("ready blocked item: %v", err)
	}
	blockerItem, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Dependency", Actor: "human"})
	if err != nil {
		t.Fatalf("create blocker item: %v", err)
	}
	blockerItem, err = runtime.MoveWorkItem(ctx, app.MoveWorkItemRequest{ID: blockerItem.ID, StageID: workitem.StageReady, Actor: "human"})
	if err != nil {
		t.Fatalf("ready blocker item: %v", err)
	}
	link, err := runtime.AddWorkItemLink(ctx, app.AddWorkItemLinkRequest{
		SourceWorkItemID: blockedItem.ID,
		TargetWorkItemID: blockerItem.ID,
		Type:             workitem.WorkItemLinkBlocks,
		Actor:            "human",
	})
	if err != nil {
		t.Fatalf("add link: %v", err)
	}
	if links, err := runtime.ListWorkItemLinks(ctx, blockedItem.ID); err != nil || len(links) != 1 || links[0].ID != link.ID {
		t.Fatalf("links = %#v, err = %v", links, err)
	}
	readyWork, err := runtime.ReadyWork(ctx, app.ReadyWorkRequest{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("ready work: %v", err)
	}
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
	if events, err := runtime.ListWorkflowEvents(ctx, item.ID); err != nil || len(events) == 0 {
		t.Fatalf("workflow events = %#v, err = %v", events, err)
	}
	status, err := runtime.ReportStatus(ctx, app.ReportStatusRequest{
		Kind:      workitem.StatusKindBlocked,
		Message:   "Need branch name.",
		Actor:     "agent",
		SessionID: "sess_01",
		PTYID:     "pty_01",
	})
	if err != nil {
		t.Fatalf("report session status: %v", err)
	}
	statusEvents, err := runtime.ListStatusEvents(ctx, app.ListStatusEventsRequest{SessionID: "sess_01", UnreadOnly: true})
	if err != nil || len(statusEvents) != 1 || statusEvents[0].ID != status.Event.ID {
		t.Fatalf("status events = %#v, err = %v", statusEvents, err)
	}
	read, err := runtime.MarkStatusEventRead(ctx, app.MarkStatusEventReadRequest{ID: status.Event.ID})
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if read.ReadAt == nil {
		t.Fatalf("read = %#v", read)
	}
	if len(store.saved.StatusEvents) != 1 || store.saved.StatusEvents[0].ReadAt == nil {
		t.Fatalf("saved status events = %#v", store.saved.StatusEvents)
	}
	deletedProject, err := runtime.DeleteProject(ctx, app.DeleteProjectRequest{ID: project.ID, Actor: "human"})
	if err != nil {
		t.Fatalf("delete project: %v", err)
	}
	if deletedProject.ID != project.ID {
		t.Fatalf("deleted project = %#v", deletedProject)
	}
	if projects, err := runtime.ListProjects(ctx); err != nil || len(projects) != 0 {
		t.Fatalf("projects after delete = %#v, err = %v", projects, err)
	}
	if len(sink.events) == 0 {
		t.Fatalf("events were not published")
	}
}

func TestRuntimeWorkItemActionsRejectInvalidRequests(t *testing.T) {
	ctx := context.Background()
	runtime := app.NewRuntime(app.RuntimeConfig{})
	if _, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "Missing root"}); err == nil {
		t.Fatalf("expected create project error")
	}
	if _, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: "missing", Title: "Task"}); err == nil {
		t.Fatalf("expected create work item error")
	}
	if _, err := runtime.MoveWorkItem(ctx, app.MoveWorkItemRequest{ID: "missing", StageID: workitem.StagePlanning}); err == nil {
		t.Fatalf("expected move work item error")
	}
	if _, err := runtime.BindWorkItemWorktree(ctx, app.BindWorkItemWorktreeRequest{ID: "missing", Branch: "feature", WorktreePath: "/tmp/work"}); err == nil {
		t.Fatalf("expected bind worktree error")
	}
	if _, err := runtime.AddWorkItemAttachment(ctx, app.AddWorkItemAttachmentRequest{WorkItemID: "missing", Kind: workitem.AttachmentKindNote, Note: "context"}); err == nil {
		t.Fatalf("expected add attachment error")
	}
	if _, err := runtime.DeleteWorkItem(ctx, app.DeleteWorkItemRequest{ID: "missing"}); err == nil {
		t.Fatalf("expected delete work item error")
	}
	if _, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{WorkItemID: "missing"}); err == nil {
		t.Fatalf("expected start run error")
	}
	if _, err := runtime.LaunchWorkItemRun(ctx, app.LaunchWorkItemRunRequest{ID: "missing"}); err == nil {
		t.Fatalf("expected launch run error")
	}
	if _, err := runtime.CancelWorkItemRun(ctx, app.CancelWorkItemRunRequest{ID: "missing"}); err == nil {
		t.Fatalf("expected cancel run error")
	}
	if _, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{WorkItemID: "missing"}); err == nil {
		t.Fatalf("expected start planning error")
	}
	if _, err := runtime.SubmitDraftPlan(ctx, app.SubmitDraftPlanRequest{WorkItemID: "missing", Body: "Plan"}); err == nil {
		t.Fatalf("expected submit draft error")
	}
	if _, err := runtime.ApprovePlan(ctx, app.ApprovePlanRequest{WorkItemID: "missing", ArtifactID: "missing"}); err == nil {
		t.Fatalf("expected approve plan error")
	}
	if _, err := runtime.StartExecution(ctx, app.StartExecutionRequest{WorkItemID: "missing"}); err == nil {
		t.Fatalf("expected start execution error")
	}
	if _, err := runtime.QueueExecution(ctx, app.QueueExecutionRequest{WorkItemID: "missing"}); err == nil {
		t.Fatalf("expected queue execution error")
	}
	if _, err := runtime.LaunchExecution(ctx, app.LaunchExecutionRequest{WorkItemID: "missing"}); err == nil {
		t.Fatalf("expected launch execution error")
	}
	if _, err := runtime.AskQuestion(ctx, app.AskQuestionRequest{WorkItemID: "missing", Prompt: "Question?"}); err == nil {
		t.Fatalf("expected ask question error")
	}
	if _, err := runtime.AnswerQuestion(ctx, app.AnswerQuestionRequest{ID: "missing", Answer: "Answer"}); err == nil {
		t.Fatalf("expected answer question error")
	}
	if _, err := runtime.CompleteExecution(ctx, app.CompleteExecutionRequest{RunID: "missing"}); err == nil {
		t.Fatalf("expected complete execution error")
	}
	if _, err := runtime.SubmitReviewFeedback(ctx, app.SubmitReviewFeedbackRequest{WorkItemID: "missing", Body: "Fix it."}); err == nil {
		t.Fatalf("expected submit feedback error")
	}
	if _, err := runtime.CompleteGate(ctx, app.CompleteGateRequest{ID: "missing", Status: workitem.GateStatusPassed}); err == nil {
		t.Fatalf("expected complete gate error")
	}
	if _, err := runtime.ApproveDone(ctx, app.ApproveDoneRequest{WorkItemID: "missing"}); err == nil {
		t.Fatalf("expected approve done error")
	}
	if _, err := runtime.ReportStatus(ctx, app.ReportStatusRequest{Kind: "bad", Message: "nope", SessionID: "sess"}); err == nil {
		t.Fatalf("expected report status error")
	}
	if _, err := runtime.MarkStatusEventRead(ctx, app.MarkStatusEventReadRequest{ID: "missing"}); err == nil {
		t.Fatalf("expected mark status read error")
	}
}

func TestRuntimeWorkItemPersistenceErrorsBubble(t *testing.T) {
	ctx := context.Background()
	saveErr := fmt.Errorf("save failed")
	withRuntime := func(t *testing.T) (*app.Runtime, *memoryWorkItemStore, workitem.Project, workitem.WorkItem) {
		t.Helper()
		store := &memoryWorkItemStore{}
		runtime := app.NewRuntime(app.RuntimeConfig{WorkItemStore: store})
		project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: t.TempDir()})
		if err != nil {
			t.Fatalf("create project: %v", err)
		}
		item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Task", Actor: "human"})
		if err != nil {
			t.Fatalf("create work item: %v", err)
		}
		store.saveErr = saveErr
		return runtime, store, project, item
	}
	expectSaveErr := func(t *testing.T, err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), saveErr.Error()) {
			t.Fatalf("err = %v, want %v", err, saveErr)
		}
	}

	t.Run("create project", func(t *testing.T) {
		runtime := app.NewRuntime(app.RuntimeConfig{WorkItemStore: &memoryWorkItemStore{saveErr: saveErr}})
		_, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: t.TempDir()})
		expectSaveErr(t, err)
	})
	t.Run("update project", func(t *testing.T) {
		runtime, _, project, _ := withRuntime(t)
		name := "New name"
		_, err := runtime.UpdateProject(ctx, app.UpdateProjectRequest{ID: project.ID, Name: &name})
		expectSaveErr(t, err)
	})
	t.Run("project attachments", func(t *testing.T) {
		runtime, store, project, _ := withRuntime(t)
		_, err := runtime.AddProjectAttachment(ctx, app.AddProjectAttachmentRequest{ProjectID: project.ID, Kind: workitem.AttachmentKindNote, Note: "note"})
		expectSaveErr(t, err)
		store.saveErr = nil
		project, err = runtime.AddProjectAttachment(ctx, app.AddProjectAttachmentRequest{ProjectID: project.ID, Kind: workitem.AttachmentKindNote, Note: "note"})
		if err != nil {
			t.Fatalf("add project attachment: %v", err)
		}
		store.saveErr = saveErr
		title := "Updated"
		_, err = runtime.UpdateProjectAttachment(ctx, app.UpdateProjectAttachmentRequest{ID: project.Attachments[0].ID, ProjectID: project.ID, Title: &title})
		expectSaveErr(t, err)
		_, err = runtime.DeleteProjectAttachment(ctx, app.DeleteProjectAttachmentRequest{ID: project.Attachments[0].ID, ProjectID: project.ID})
		expectSaveErr(t, err)
	})
	t.Run("work item mutations", func(t *testing.T) {
		runtime, _, _, item := withRuntime(t)
		_, err := runtime.MoveWorkItem(ctx, app.MoveWorkItemRequest{ID: item.ID, StageID: workitem.StagePlanning, Actor: "human"})
		expectSaveErr(t, err)
		_, err = runtime.BindWorkItemWorktree(ctx, app.BindWorkItemWorktreeRequest{ID: item.ID, Branch: "feature", WorktreePath: t.TempDir(), Actor: "human"})
		expectSaveErr(t, err)
		_, err = runtime.AddWorkItemAttachment(ctx, app.AddWorkItemAttachmentRequest{WorkItemID: item.ID, Kind: workitem.AttachmentKindFile, Path: "docs/spec.md", Actor: "human"})
		expectSaveErr(t, err)
	})
	t.Run("run lifecycle", func(t *testing.T) {
		runtime, store, _, item := withRuntime(t)
		_, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{WorkItemID: item.ID, Preset: workitem.RunPresetWriter, PromptTemplateID: workitem.PromptTemplateImplement, Actor: "agent"})
		expectSaveErr(t, err)
		store.saveErr = nil
		run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{WorkItemID: item.ID, Preset: workitem.RunPresetWriter, PromptTemplateID: workitem.PromptTemplateImplement, Actor: "agent"})
		if err != nil {
			t.Fatalf("start run: %v", err)
		}
		store.saveErr = saveErr
		_, err = runtime.AskQuestion(ctx, app.AskQuestionRequest{WorkItemID: item.ID, RunID: run.ID, Prompt: "Which?", Actor: "agent"})
		expectSaveErr(t, err)
		_, err = runtime.ReportStatus(ctx, app.ReportStatusRequest{Kind: workitem.StatusKindBlocked, Message: "blocked", WorkItemID: item.ID, RunID: run.ID, Actor: "agent"})
		expectSaveErr(t, err)
		_, err = runtime.CancelWorkItemRun(ctx, app.CancelWorkItemRunRequest{ID: run.ID, Actor: "human"})
		expectSaveErr(t, err)
	})
	t.Run("planning", func(t *testing.T) {
		runtime, store, _, item := withRuntime(t)
		_, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{WorkItemID: item.ID, Actor: "agent"})
		expectSaveErr(t, err)
		store.saveErr = nil
		planning, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{WorkItemID: item.ID, Actor: "agent"})
		if err != nil {
			t.Fatalf("start planning: %v", err)
		}
		store.saveErr = saveErr
		_, err = runtime.SubmitDraftPlan(ctx, app.SubmitDraftPlanRequest{WorkItemID: item.ID, RunID: planning.ID, Body: "plan", Actor: "agent"})
		expectSaveErr(t, err)
	})
}

type memoryWorkItemStore struct {
	saved   workitem.Snapshot
	saveErr error
}

func (s *memoryWorkItemStore) LoadWorkItems(context.Context) (workitem.Snapshot, error) {
	return s.saved, nil
}

func (s *memoryWorkItemStore) SaveWorkItems(_ context.Context, snapshot workitem.Snapshot) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = snapshot
	return nil
}

type memoryEventSink struct {
	events []app.RuntimeEvent
}

func (s *memoryEventSink) Publish(_ context.Context, event app.RuntimeEvent) error {
	s.events = append(s.events, event)
	return nil
}

type memoryPTYBackend struct {
	mu          sync.Mutex
	records     map[string]app.PTYRecord
	spawns      []app.SpawnPTYRequest
	writes      map[string][][]byte
	outputBytes []byte
	outputCalls map[string]int
}

func newMemoryPTYBackend() *memoryPTYBackend {
	return &memoryPTYBackend{
		records:     map[string]app.PTYRecord{},
		writes:      map[string][][]byte{},
		outputCalls: map[string]int{},
	}
}

func (b *memoryPTYBackend) Spawn(_ context.Context, req app.SpawnPTYRequest) (app.PTYRecord, error) {
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

func (b *memoryPTYBackend) Write(_ context.Context, ptyID string, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.records[ptyID]; !ok {
		return fmt.Errorf("pty %s not found", ptyID)
	}
	b.writes[ptyID] = append(b.writes[ptyID], append([]byte(nil), data...))
	b.outputBytes = append(b.outputBytes, data...)
	return nil
}

func (b *memoryPTYBackend) Resize(context.Context, string, app.PTYSize) error {
	return nil
}

func (b *memoryPTYBackend) Kill(_ context.Context, ptyID string) (app.PTYRecord, error) {
	record, ok := b.records[ptyID]
	if !ok {
		return app.PTYRecord{}, fmt.Errorf("pty %s not found", ptyID)
	}
	record.Running = false
	b.records[ptyID] = record
	return record, nil
}

func (b *memoryPTYBackend) Delete(_ context.Context, ptyID string) error {
	record, ok := b.records[ptyID]
	if !ok {
		return fmt.Errorf("pty %s not found", ptyID)
	}
	if record.Running {
		return fmt.Errorf("cannot delete running pty %s", ptyID)
	}
	delete(b.records, ptyID)
	return nil
}

func (b *memoryPTYBackend) Attach(context.Context, app.AttachPTYRequest) (*app.PTYAttach, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *memoryPTYBackend) Output(_ context.Context, ptyID string, _ uint64) (app.PTYOutputSnapshot, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	record, ok := b.records[ptyID]
	if !ok {
		return app.PTYOutputSnapshot{}, fmt.Errorf("pty %s not found", ptyID)
	}
	b.outputCalls[ptyID]++
	return app.PTYOutputSnapshot{Record: record, OutputBytes: append([]byte(nil), b.outputBytes...)}, nil
}

func (b *memoryPTYBackend) List(context.Context) ([]app.PTYRecord, error) {
	out := make([]app.PTYRecord, 0, len(b.records))
	for _, record := range b.records {
		out = append(out, record)
	}
	return out, nil
}

func (b *memoryPTYBackend) Shutdown(context.Context) error {
	b.records = map[string]app.PTYRecord{}
	return nil
}

type attachableMemoryPTYBackend struct {
	*memoryPTYBackend
	events map[string]chan app.PTYEvent
}

func newAttachableMemoryPTYBackend() *attachableMemoryPTYBackend {
	return &attachableMemoryPTYBackend{
		memoryPTYBackend: newMemoryPTYBackend(),
		events:           map[string]chan app.PTYEvent{},
	}
}

func (b *attachableMemoryPTYBackend) Spawn(ctx context.Context, req app.SpawnPTYRequest) (app.PTYRecord, error) {
	record, err := b.memoryPTYBackend.Spawn(ctx, req)
	if err != nil {
		return app.PTYRecord{}, err
	}
	b.events[record.ID] = make(chan app.PTYEvent, 4)
	return record, nil
}

func (b *attachableMemoryPTYBackend) Attach(_ context.Context, req app.AttachPTYRequest) (*app.PTYAttach, error) {
	events, ok := b.events[req.PtyID]
	if !ok {
		return nil, fmt.Errorf("pty %s not found", req.PtyID)
	}
	return &app.PTYAttach{
		Record: b.records[req.PtyID],
		Events: events,
	}, nil
}

func (b *attachableMemoryPTYBackend) exit(ptyID string, code int) {
	record := b.records[ptyID]
	record.Running = false
	b.records[ptyID] = record
	b.events[ptyID] <- app.PTYEvent{Kind: app.PTYExit, Code: &code}
}

func TestRuntimeProjectContextResolvesIncludedAttachments(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	nextID := 0
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		ContextResolvers: map[string]app.ProjectContextResolver{
			"github": staticProjectContextResolver{},
		},
	})
	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	renamed := "Renamed"
	slug := "renamed"
	project, err = runtime.UpdateProject(ctx, app.UpdateProjectRequest{ID: project.ID, Name: &renamed, Slug: &slug})
	if err != nil {
		t.Fatalf("update project: %v", err)
	}
	if project.Name != renamed || project.Slug != slug {
		t.Fatalf("updated project = %#v", project)
	}
	noteProject, err := runtime.AddProjectAttachment(ctx, app.AddProjectAttachmentRequest{ProjectID: project.ID, Kind: workitem.AttachmentKindNote, Title: "Note", Note: "Remember this.", IncludeInContext: true})
	if err != nil {
		t.Fatalf("add note: %v", err)
	}
	noteAttachmentID := noteProject.Attachments[0].ID
	updatedTitle := "Updated note"
	updatedNote := "Remember less."
	if _, err := runtime.UpdateProjectAttachment(ctx, app.UpdateProjectAttachmentRequest{ID: noteAttachmentID, ProjectID: project.ID, Title: &updatedTitle, Note: &updatedNote}); err != nil {
		t.Fatalf("update note: %v", err)
	}
	if _, err := runtime.AddProjectAttachment(ctx, app.AddProjectAttachmentRequest{ProjectID: project.ID, Kind: workitem.AttachmentKindExternal, Provider: "github", Target: "owner/repo#1", IncludeInContext: true}); err != nil {
		t.Fatalf("add external: %v", err)
	}
	urlProject, err := runtime.AddProjectAttachment(ctx, app.AddProjectAttachmentRequest{ProjectID: project.ID, Kind: workitem.AttachmentKindURL, URL: "https://example.test"})
	if err != nil {
		t.Fatalf("add skipped url: %v", err)
	}
	urlAttachmentID := urlProject.Attachments[len(urlProject.Attachments)-1].ID
	if _, err := runtime.DeleteProjectAttachment(ctx, app.DeleteProjectAttachmentRequest{ID: urlAttachmentID, ProjectID: project.ID}); err != nil {
		t.Fatalf("delete url: %v", err)
	}
	detail, err := runtime.GetProjectDetail(ctx, project.ID)
	if err != nil {
		t.Fatalf("project detail: %v", err)
	}
	if detail.Project.ID != project.ID || detail.Project.Name != renamed {
		t.Fatalf("detail = %#v", detail)
	}

	contextBundle, err := runtime.ProjectContext(ctx, project.ID)
	if err != nil {
		t.Fatalf("project context: %v", err)
	}
	if len(contextBundle.Items) != 2 {
		t.Fatalf("context = %#v", contextBundle)
	}
	if contextBundle.Items[0].Delivery != "inline" || contextBundle.Items[0].Content != updatedNote {
		t.Fatalf("note context = %#v", contextBundle.Items[0])
	}
	if contextBundle.Items[1].Delivery != "inline" || contextBundle.Items[1].Content != "resolved issue" {
		t.Fatalf("external context = %#v", contextBundle.Items[1])
	}
}

type staticProjectContextResolver struct{}

func (staticProjectContextResolver) ResolveProjectAttachment(context.Context, app.ResolveProjectAttachmentRequest) (app.ResolvedProjectAttachment, error) {
	return app.ResolvedProjectAttachment{Title: "GitHub issue", Delivery: "inline", ContentType: "text/markdown", Content: "resolved issue", SourceURL: "https://github.com/owner/repo/issues/1"}, nil
}

func TestRuntimePluginRegistryMethods(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	plugins := &memoryPluginRegistry{
		status: app.PluginStatus{
			ID:      "github",
			Name:    "GitHub",
			Version: "1.0.0",
			Valid:   true,
			Resolvers: []app.PluginResolver{{
				Provider: "github",
				Kinds:    []string{workitem.AttachmentKindExternal},
			}},
			ProjectAttachmentTemplates: []app.ProjectAttachmentTemplate{{
				ID:       "github.issue",
				Label:    "GitHub issue",
				Provider: "github",
				Kind:     workitem.AttachmentKindExternal,
			}},
		},
		registry: app.RegistryPlugin{Registry: "phin-tech", ID: "github", Name: "GitHub", SourceType: "path"},
		resolver: staticProjectContextResolver{},
	}
	runtime := app.NewRuntime(app.RuntimeConfig{Plugins: plugins})

	listed, err := runtime.ListPlugins(ctx)
	if err != nil || len(listed) != 1 || listed[0].ID != "github" {
		t.Fatalf("list plugins = %#v, err = %v", listed, err)
	}
	rescanned, err := runtime.RescanPlugins(ctx)
	if err != nil || len(rescanned) != 1 || !plugins.rescanned {
		t.Fatalf("rescan plugins = %#v, rescanned=%v, err = %v", rescanned, plugins.rescanned, err)
	}
	trusted, err := runtime.TrustPlugin(ctx, "github")
	if err != nil || !trusted.Trusted || plugins.trustedID != "github" {
		t.Fatalf("trust plugin = %#v, trustedID=%q, err = %v", trusted, plugins.trustedID, err)
	}
	untrusted, err := runtime.UntrustPlugin(ctx, "github")
	if err != nil || untrusted.Trusted || plugins.untrustedID != "github" {
		t.Fatalf("untrust plugin = %#v, untrustedID=%q, err = %v", untrusted, plugins.untrustedID, err)
	}
	available, err := runtime.ListRegistryPlugins(ctx)
	if err != nil || len(available) != 1 || available[0].Registry != "phin-tech" {
		t.Fatalf("registry plugins = %#v, err = %v", available, err)
	}
	installed, err := runtime.InstallPlugin(ctx, "phin-tech", "github")
	if err != nil || installed.Registry != "phin-tech" || plugins.installedID != "github" {
		t.Fatalf("install plugin = %#v, installedID=%q, err = %v", installed, plugins.installedID, err)
	}

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: root})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	withAttachment, err := runtime.RunPluginProjectAttachmentTemplate(ctx, app.RunPluginProjectAttachmentTemplateRequest{
		PluginID:   "github",
		TemplateID: "github.issue",
		ProjectID:  project.ID,
		Values:     map[string]string{"repo": "owner/repo", "issue": "1"},
	})
	if err != nil {
		t.Fatalf("run plugin template: %v", err)
	}
	if len(withAttachment.Attachments) != 1 || withAttachment.Attachments[0].Provider != "github" {
		t.Fatalf("with attachment = %#v", withAttachment)
	}
	contextBundle, err := runtime.ProjectContext(ctx, project.ID)
	if err != nil {
		t.Fatalf("project context: %v", err)
	}
	if len(contextBundle.Items) != 1 || contextBundle.Items[0].Content != "resolved issue" {
		t.Fatalf("context = %#v", contextBundle)
	}
}

func TestRuntimeListAgentProfilesMergesPluginProfiles(t *testing.T) {
	ctx := context.Background()
	plugins := &memoryPluginRegistry{
		agentProfiles: []agents.ProfileInfo{{
			ID:                  "plugin:phin-tech/gemini/gemini-cli",
			Provider:            "gemini",
			Label:               "Gemini CLI",
			Source:              agents.ProfileSourcePlugin,
			PluginID:            "phin-tech/gemini",
			Launchable:          false,
			LaunchBlockedReason: "plugin phin-tech/gemini is not trusted",
			PromptInjectionMode: agents.PromptInjectionArgv,
		}},
	}
	runtime := app.NewRuntime(app.RuntimeConfig{Plugins: plugins})

	profiles, err := runtime.ListAgentProfiles(ctx)
	if err != nil {
		t.Fatalf("list agent profiles: %v", err)
	}
	byID := map[string]agents.ProfileInfo{}
	for _, profile := range profiles {
		byID[profile.ID] = profile
	}
	if builtin := byID["codex"]; builtin.Source != agents.ProfileSourceBuiltin || !builtin.Launchable {
		t.Fatalf("builtin profile = %#v", builtin)
	}
	plugin := byID["plugin:phin-tech/gemini/gemini-cli"]
	if plugin.Source != agents.ProfileSourcePlugin ||
		plugin.PluginID != "phin-tech/gemini" ||
		plugin.Launchable ||
		plugin.LaunchBlockedReason == "" {
		t.Fatalf("plugin profile = %#v", plugin)
	}
}

type memoryPluginRegistry struct {
	status        app.PluginStatus
	registry      app.RegistryPlugin
	agentProfiles []agents.ProfileInfo
	resolver      app.ProjectContextResolver
	rescanned     bool
	trustedID     string
	untrustedID   string
	installedID   string
}

func (r *memoryPluginRegistry) ListPlugins(context.Context) ([]app.PluginStatus, error) {
	return []app.PluginStatus{r.status}, nil
}

func (r *memoryPluginRegistry) RescanPlugins(context.Context) ([]app.PluginStatus, error) {
	r.rescanned = true
	return []app.PluginStatus{r.status}, nil
}

func (r *memoryPluginRegistry) TrustPlugin(_ context.Context, id string) (app.PluginStatus, error) {
	r.trustedID = id
	status := r.status
	status.ID = id
	status.Trusted = true
	return status, nil
}

func (r *memoryPluginRegistry) UntrustPlugin(_ context.Context, id string) (app.PluginStatus, error) {
	r.untrustedID = id
	status := r.status
	status.ID = id
	status.Trusted = false
	return status, nil
}

func (r *memoryPluginRegistry) ListRegistryPlugins(context.Context) ([]app.RegistryPlugin, error) {
	return []app.RegistryPlugin{r.registry}, nil
}

func (r *memoryPluginRegistry) InstallPlugin(_ context.Context, registry, id string) (app.PluginStatus, error) {
	r.installedID = id
	status := r.status
	status.ID = id
	status.Registry = registry
	return status, nil
}

func (r *memoryPluginRegistry) ListAgentProfiles(context.Context) ([]agents.ProfileInfo, error) {
	return append([]agents.ProfileInfo(nil), r.agentProfiles...), nil
}

func (r *memoryPluginRegistry) RunProjectAttachmentTemplate(_ context.Context, req app.RunPluginProjectAttachmentTemplateRequest) (app.AddProjectAttachmentRequest, error) {
	return app.AddProjectAttachmentRequest{
		ProjectID:        req.ProjectID,
		Kind:             workitem.AttachmentKindExternal,
		Provider:         "github",
		Target:           req.Values["repo"] + "#" + req.Values["issue"],
		Title:            "Issue",
		IncludeInContext: true,
	}, nil
}

func (r *memoryPluginRegistry) ResolveProjectAttachmentProvider(provider string) app.ProjectContextResolver {
	if provider == "github" {
		return r.resolver
	}
	return nil
}
