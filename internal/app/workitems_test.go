package app_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
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

func TestRuntimeStartWorkItemRunLaunchesAgentPTY(t *testing.T) {
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
		env["PATH"] != "/usr/local/bin:/usr/bin:/bin" ||
		env["WHISK_PROJECT_ID"] != project.ID ||
		env["WHISK_WORK_ITEM_ID"] != item.ID ||
		env["WHISK_RUN_ID"] != run.ID ||
		env["WHISK_SESSION_ID"] != run.SessionID ||
		env["WHISK_PTY_ID"] != run.PTYID ||
		env["WHISK_ACTOR"] != "agent" {
		t.Fatalf("spawn env = %#v", env)
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
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Wire status"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID: item.ID,
		SessionID:  "sess_01",
		PTYID:      "pty_01",
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

type memoryWorkItemStore struct {
	saved workitem.Snapshot
}

func (s *memoryWorkItemStore) LoadWorkItems(context.Context) (workitem.Snapshot, error) {
	return s.saved, nil
}

func (s *memoryWorkItemStore) SaveWorkItems(_ context.Context, snapshot workitem.Snapshot) error {
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
	records map[string]app.PTYRecord
	spawns  []app.SpawnPTYRequest
	writes  map[string][][]byte
}

func newMemoryPTYBackend() *memoryPTYBackend {
	return &memoryPTYBackend{
		records: map[string]app.PTYRecord{},
		writes:  map[string][][]byte{},
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
	if _, ok := b.records[ptyID]; !ok {
		return fmt.Errorf("pty %s not found", ptyID)
	}
	b.writes[ptyID] = append(b.writes[ptyID], append([]byte(nil), data...))
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

func (b *memoryPTYBackend) Attach(context.Context, app.AttachPTYRequest) (*app.PTYAttach, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *memoryPTYBackend) Output(_ context.Context, ptyID string, _ uint64) (app.PTYOutputSnapshot, error) {
	record, ok := b.records[ptyID]
	if !ok {
		return app.PTYOutputSnapshot{}, fmt.Errorf("pty %s not found", ptyID)
	}
	return app.PTYOutputSnapshot{Record: record}, nil
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
