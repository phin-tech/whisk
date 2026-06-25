package app_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/agenthooklog"
	"github.com/phin-tech/whisk/internal/adapters/agenthooks"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/workitem"
)

func appTestAgentHookPaths(t *testing.T) agenthooks.Paths {
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

func TestRuntimeAgentHookLogSettings(t *testing.T) {
	tmp := t.TempDir()
	logPaths := agenthooklog.Paths{
		ConfigRoot: tmp,
		LogPath:    filepath.Join(tmp, "agent-hooks", "hooks.jsonl"),
	}
	runtime := app.NewRuntime(app.RuntimeConfig{AgentHookLogPaths: &logPaths})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	ctx := context.Background()

	enabled, clear := true, true
	status, err := runtime.SetAgentHookLogSettings(ctx, app.SetAgentHookLogSettingsRequest{
		Enabled:           &enabled,
		ClearAfterSession: &clear,
	})
	if err != nil || !status.Enabled || !status.ClearAfterSession {
		t.Fatalf("set hook log = %#v, err = %v", status, err)
	}

	// Record an event so the on-disk log grows, then confirm the status reflects it.
	if _, err := runtime.RecordAgentHookEvent(ctx, app.AgentBridgeHookRequest{
		Provider:  "claude",
		EventName: "Notification",
		Message:   "coverage event",
	}); err != nil {
		t.Fatalf("record event: %v", err)
	}

	status, err = runtime.AgentHookLogStatus(ctx)
	if err != nil || status.SizeBytes == 0 {
		t.Fatalf("hook log status = %#v, err = %v", status, err)
	}

	cleared, err := runtime.ClearAgentHookLog(ctx)
	if err != nil || cleared.SizeBytes != 0 {
		t.Fatalf("clear hook log = %#v, err = %v", cleared, err)
	}
}

func TestRuntimeAgentHookLogWritesNormalizedQuestions(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "agent-hooks", "hooks.jsonl")
	logPaths := agenthooklog.Paths{ConfigRoot: tmp, LogPath: logPath}
	runtime := app.NewRuntime(app.RuntimeConfig{AgentHookLogPaths: &logPaths})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	if _, err := runtime.RecordAgentHookEvent(context.Background(), app.AgentBridgeHookRequest{
		Provider:  "claude",
		EventName: "PermissionRequest",
		ToolName:  "AskUserQuestion",
		RawPayload: map[string]any{
			"tool_input": map[string]any{
				"questions": []any{
					map[string]any{
						"question": "What would you like to work on today?",
						"options": []any{
							map[string]any{"label": "Fix a bug"},
							map[string]any{"label": "Build a feature"},
						},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("record event: %v", err)
	}

	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("missing log line")
	}
	var line map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
		t.Fatalf("parse log line: %v", err)
	}
	options, _ := line["options"].([]any)
	if line["kind"] != "question" ||
		line["title"] != "Claude question" ||
		line["message"] != "What would you like to work on today?" ||
		len(options) != 2 {
		t.Fatalf("log line = %#v", line)
	}
}

func TestRuntimeAgentBridgeListsAndResolveError(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	ctx := context.Background()

	if _, err := runtime.RecordAgentHookEvent(ctx, app.AgentBridgeHookRequest{
		Provider:  "claude",
		EventName: "Notification",
	}); err != nil {
		t.Fatalf("record event: %v", err)
	}

	events, err := runtime.ListAgentBridgeEvents(ctx, app.ListAgentBridgeEventsRequest{})
	if err != nil || len(events) != 1 {
		t.Fatalf("events = %#v, err = %v", events, err)
	}

	read, err := runtime.MarkAgentBridgeEventRead(ctx, app.MarkAgentBridgeEventReadRequest{ID: events[0].ID})
	if err != nil || read.ID != events[0].ID || read.Status != "read" {
		t.Fatalf("mark event read = %#v, err = %v", read, err)
	}
	if _, err := runtime.MarkAgentBridgeEventRead(ctx, app.MarkAgentBridgeEventReadRequest{ID: "missing"}); err == nil {
		t.Fatalf("expected error marking unknown event read")
	}

	approvals, err := runtime.ListAgentBridgeApprovals(ctx, app.ListAgentBridgeApprovalsRequest{Status: "pending"})
	if err != nil || len(approvals) != 0 {
		t.Fatalf("approvals = %#v, err = %v", approvals, err)
	}

	if _, err := runtime.ResolveAgentBridgeApproval(ctx, app.ResolveAgentBridgeApprovalRequest{
		ID:     "missing",
		Action: "allow",
	}); err == nil {
		t.Fatalf("expected error resolving unknown approval")
	}
}

// launchAgentBridge starts a launched work-item run and returns its runtime plus the agent bridge
// credentials handed to the spawned agent, so tests can drive HandleAgentBridgeHook directly.
func launchAgentBridge(t *testing.T, approvalTimeout time.Duration) (*app.Runtime, string, string) {
	t.Helper()
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	nextID := 0
	ptyBackend := newMemoryPTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore:              &memoryWorkItemStore{},
		PTYBackend:                 ptyBackend,
		DaemonURL:                  "http://127.0.0.1:8787",
		CLIPath:                    "/usr/local/bin/whisk",
		AgentBridgeApprovalTimeout: approvalTimeout,
	})
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Bridge approval"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	item, err = runtime.BindWorkItemWorktree(ctx, app.BindWorkItemWorktreeRequest{
		ID:           item.ID,
		Branch:       "whisk/app-1-bridge-approval",
		WorktreePath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("bind worktree: %v", err)
	}
	if _, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		Launch:           true,
		AgentProfileID:   "claude",
		Actor:            "agent",
	}); err != nil {
		t.Fatalf("start run: %v", err)
	}
	if len(ptyBackend.spawns) != 1 {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
	env := ptyBackend.spawns[0].Env
	bridgeID, token := env["WHISK_AGENT_BRIDGE_ID"], env["WHISK_AGENT_BRIDGE_TOKEN"]
	if bridgeID == "" || token == "" {
		t.Fatalf("missing bridge credentials: env = %#v", env)
	}
	return runtime, bridgeID, token
}

func toolCallHook(bridgeID, token string) app.AgentBridgeHookRequest {
	return app.AgentBridgeHookRequest{
		BridgeID:  bridgeID,
		Token:     token,
		Provider:  "claude",
		EventName: "PermissionRequest",
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "ls"},
	}
}

func launchPlanningAgentBridge(t *testing.T) (*app.Runtime, string, string) {
	t.Helper()
	t.Setenv("PATH", "/usr/bin:/bin")
	ctx := context.Background()
	nextID := 0
	ptyBackend := newMemoryPTYBackend()
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

	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Plan first"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	if _, err := runtime.StartPlanning(ctx, app.StartPlanningRequest{
		WorkItemID:     item.ID,
		Launch:         true,
		AgentProfileID: "claude-plan",
		Actor:          "agent",
	}); err != nil {
		t.Fatalf("start planning: %v", err)
	}
	if len(ptyBackend.spawns) != 1 {
		t.Fatalf("spawns = %#v", ptyBackend.spawns)
	}
	env := ptyBackend.spawns[0].Env
	bridgeID, token := env["WHISK_AGENT_BRIDGE_ID"], env["WHISK_AGENT_BRIDGE_TOKEN"]
	if bridgeID == "" || token == "" {
		t.Fatalf("missing bridge credentials: env = %#v", env)
	}
	return runtime, bridgeID, token
}

func TestRuntimeExitPlanModeHookStopsPlanningRunAtWhiskSubmission(t *testing.T) {
	runtime, bridgeID, token := launchPlanningAgentBridge(t)
	ctx := context.Background()

	resp, err := runtime.HandleAgentBridgeHook(ctx, app.AgentBridgeHookRequest{
		BridgeID:  bridgeID,
		Token:     token,
		Provider:  "claude",
		EventName: "PreToolUse",
		ToolName:  "ExitPlanMode",
		ToolInput: map[string]any{"plan": "## Plan\nDo it."},
	})
	if err != nil {
		t.Fatalf("hook: %v", err)
	}
	hookSpecific, ok := resp.Output["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("response = %#v", resp.Output)
	}
	reason, _ := hookSpecific["permissionDecisionReason"].(string)
	if hookSpecific["hookEventName"] != "PreToolUse" ||
		hookSpecific["permissionDecision"] != "deny" ||
		!strings.Contains(reason, "Whisk planning run") ||
		!strings.Contains(reason, "workflow submit-plan") ||
		!strings.Contains(reason, "Do not") {
		t.Fatalf("hookSpecificOutput = %#v", hookSpecific)
	}

	executionRuntime, executionBridgeID, executionToken := launchAgentBridge(t, time.Second)
	executionResp, err := executionRuntime.HandleAgentBridgeHook(ctx, app.AgentBridgeHookRequest{
		BridgeID:  executionBridgeID,
		Token:     executionToken,
		Provider:  "claude",
		EventName: "PreToolUse",
		ToolName:  "ExitPlanMode",
		ToolInput: map[string]any{"plan": "## Plan\nDo it."},
	})
	if err != nil {
		t.Fatalf("execution hook: %v", err)
	}
	if executionResp.Output != nil {
		t.Fatalf("execution response = %#v", executionResp.Output)
	}
}

func TestRuntimeAgentBridgeLogsClaudePreToolUseWithoutApproval(t *testing.T) {
	runtime, bridgeID, token := launchAgentBridge(t, time.Second)
	ctx := context.Background()

	resp, err := runtime.HandleAgentBridgeHook(ctx, app.AgentBridgeHookRequest{
		BridgeID:  bridgeID,
		Token:     token,
		Provider:  "claude",
		EventName: "PreToolUse",
		ToolName:  "Write",
		ToolInput: map[string]any{"file_path": "/tmp/plan.md", "content": "draft"},
	})
	if err != nil {
		t.Fatalf("hook: %v", err)
	}
	if resp.Output != nil {
		t.Fatalf("response = %#v", resp.Output)
	}
	approvals, err := runtime.ListAgentBridgeApprovals(ctx, app.ListAgentBridgeApprovalsRequest{Status: "pending"})
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if len(approvals) != 0 {
		t.Fatalf("approvals = %#v", approvals)
	}
}

func TestRuntimeAgentBridgeApprovalLifecycle(t *testing.T) {
	runtime, bridgeID, token := launchAgentBridge(t, 5*time.Second)
	ctx := context.Background()

	// A permission hook with no pre-supplied decision blocks until the approval is resolved.
	hookErr := make(chan error, 1)
	go func() {
		_, err := runtime.HandleAgentBridgeHook(ctx, toolCallHook(bridgeID, token))
		hookErr <- err
	}()

	var approvalID string
	for i := 0; i < 200; i++ {
		approvals, listErr := runtime.ListAgentBridgeApprovals(ctx, app.ListAgentBridgeApprovalsRequest{Status: "pending"})
		if listErr != nil {
			t.Fatalf("list approvals: %v", listErr)
		}
		if len(approvals) == 1 {
			approvalID = approvals[0].ID
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if approvalID == "" {
		t.Fatalf("no pending approval appeared")
	}

	approval, err := runtime.ResolveAgentBridgeApproval(ctx, app.ResolveAgentBridgeApprovalRequest{
		ID:     approvalID,
		Action: "allow",
	})
	if err != nil {
		t.Fatalf("resolve approval: %v", err)
	}
	if approval.ID != approvalID {
		t.Fatalf("resolved approval = %#v", approval)
	}

	select {
	case err := <-hookErr:
		if err != nil {
			t.Fatalf("hook returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("hook did not return after approval")
	}
}

func TestRuntimeAgentBridgeApprovalTimesOut(t *testing.T) {
	runtime, bridgeID, token := launchAgentBridge(t, time.Millisecond)

	if _, err := runtime.HandleAgentBridgeHook(context.Background(), toolCallHook(bridgeID, token)); err != nil {
		t.Fatalf("hook: %v", err)
	}

	// After timing out, the approval is no longer pending.
	approvals, err := runtime.ListAgentBridgeApprovals(context.Background(), app.ListAgentBridgeApprovalsRequest{Status: "pending"})
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if len(approvals) != 0 {
		t.Fatalf("expected no pending approvals after timeout, got %#v", approvals)
	}
}

func TestRuntimeAgentBridgeApprovalCancelled(t *testing.T) {
	runtime, bridgeID, token := launchAgentBridge(t, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runtime.HandleAgentBridgeHook(ctx, toolCallHook(bridgeID, token)); err != nil {
		t.Fatalf("hook with cancelled context: %v", err)
	}
}

func TestRuntimeAgentHookIntegrations(t *testing.T) {
	paths := appTestAgentHookPaths(t)
	runtime := app.NewRuntime(app.RuntimeConfig{AgentHookPaths: &paths})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	ctx := context.Background()

	listed, err := runtime.ListAgentHookIntegrations(ctx)
	if err != nil || len(listed) == 0 {
		t.Fatalf("list integrations = %#v, err = %v", listed, err)
	}

	installed, err := runtime.InstallAgentHookIntegration(ctx, app.AgentHookIntegrationRequest{Provider: "claude"})
	if err != nil || installed.Provider != "claude" {
		t.Fatalf("install = %#v, err = %v", installed, err)
	}

	checked, err := runtime.CheckAgentHookIntegration(ctx, app.AgentHookIntegrationRequest{Provider: "claude"})
	if err != nil || checked.Provider != "claude" {
		t.Fatalf("check = %#v, err = %v", checked, err)
	}

	removed, err := runtime.RemoveAgentHookIntegration(ctx, app.AgentHookIntegrationRequest{Provider: "claude"})
	if err != nil || removed.Provider != "claude" {
		t.Fatalf("remove = %#v, err = %v", removed, err)
	}
}
