package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/agentbridge"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/server"
)

// buildTestWhiskBin builds cmd/whisk once per test process and returns its path.
// The binary is placed in a temp dir that is intentionally not cleaned up —
// it persists for the lifetime of the test process and is removed by the OS.
var (
	buildWhiskOnce   sync.Once
	testWhiskBinPath string
	testWhiskBinErr  error
)

func testWhiskBin(t *testing.T) string {
	t.Helper()
	buildWhiskOnce.Do(func() {
		dir, err := os.MkdirTemp("", "whisk-testagent-bin-*")
		if err != nil {
			testWhiskBinErr = fmt.Errorf("mktemp: %w", err)
			return
		}
		bin := filepath.Join(dir, "whisk")
		out, err := exec.Command("go", "build", "-o", bin, "github.com/phin-tech/whisk/cmd/whisk").CombinedOutput()
		if err != nil {
			os.RemoveAll(dir)
			testWhiskBinErr = fmt.Errorf("build cmd/whisk: %w\n%s", err, out)
			return
		}
		testWhiskBinPath = bin
	})
	if testWhiskBinErr != nil {
		t.Fatalf("whisk binary unavailable: %v", testWhiskBinErr)
	}
	return testWhiskBinPath
}

// agentBridgeHTTPFixture holds the runtime, HTTP server, and bridge credentials
// needed by the testagent integration tests.
type agentBridgeHTTPFixture struct {
	Runtime    *app.Runtime
	Server     *httptest.Server
	BridgeID   string
	HookScript string
	// AgentEnv is the full environment to pass to the testagent subprocess.
	// It includes all system env vars plus the bridge credentials needed by hook.sh.
	AgentEnv []string
}

// launchAgentBridgeWithHTTP starts a real HTTP server wrapping a live runtime,
// then launches a work-item run so that bridgeinstaller writes hook.sh to disk.
// The returned fixture has everything testagent needs to fire hooks against the daemon.
func launchAgentBridgeWithHTTP(t *testing.T, approvalTimeout time.Duration) agentBridgeHTTPFixture {
	t.Helper()
	whiskBin := testWhiskBin(t)

	// Pre-bind a listener so the URL is known before the runtime is constructed.
	// The runtime embeds the URL in hook.sh via bridgeinstaller.Install.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	daemonURL := "http://" + l.Addr().String()

	ctx := context.Background()
	nextID := 0
	ptyBackend := newMemoryPTYBackend()
	rt := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore:              &memoryWorkItemStore{},
		PTYBackend:                 ptyBackend,
		DaemonURL:                  daemonURL,
		CLIPath:                    whiskBin,
		AgentBridgeApprovalTimeout: approvalTimeout,
	})
	t.Cleanup(func() { _ = rt.Shutdown(ctx) })

	srv := httptest.NewUnstartedServer(server.NewHTTP(rt))
	srv.Listener = l
	srv.Start()
	t.Cleanup(srv.Close)

	project, err := rt.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := rt.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Hook test"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	item, err = rt.BindWorkItemWorktree(ctx, app.BindWorkItemWorktreeRequest{
		ID:           item.ID,
		Branch:       "whisk/app-1-hook-test",
		WorktreePath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("bind worktree: %v", err)
	}
	if _, err := rt.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
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
		t.Fatalf("expected 1 spawn, got %d", len(ptyBackend.spawns))
	}

	spawnEnv := ptyBackend.spawns[0].Env
	bridgeID := spawnEnv["WHISK_AGENT_BRIDGE_ID"]
	hookScript := spawnEnv["WHISK_AGENT_BRIDGE_HOOK_SCRIPT"]
	if bridgeID == "" || hookScript == "" {
		t.Fatalf("missing bridge env in spawn: %#v", spawnEnv)
	}

	// Start with the full system environment (including PATH with go, etc.),
	// then overlay the bridge-specific credentials that hook.sh needs.
	agentEnv := os.Environ()
	for _, key := range []string{
		"WHISK_CLI", "WHISKD_URL",
		"WHISK_AGENT_BRIDGE_ID", "WHISK_AGENT_BRIDGE_TOKEN", "WHISK_AGENT_BRIDGE_PROVIDER",
		"WHISK_SESSION_ID", "WHISK_PTY_ID",
		"WHISK_PROJECT_ID", "WHISK_WORK_ITEM_ID", "WHISK_RUN_ID", "WHISK_ACTOR",
		"PATH",
	} {
		if val := spawnEnv[key]; val != "" {
			agentEnv = append(agentEnv, key+"="+val)
		}
	}

	return agentBridgeHTTPFixture{
		Runtime:    rt,
		Server:     srv,
		BridgeID:   bridgeID,
		HookScript: hookScript,
		AgentEnv:   agentEnv,
	}
}

func launchCustomAgentBridgeWithHTTP(t *testing.T, approvalTimeout time.Duration, launch func(context.Context, *app.Runtime, workitem.Project) error) agentBridgeHTTPFixture {
	t.Helper()
	whiskBin := testWhiskBin(t)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	daemonURL := "http://" + l.Addr().String()

	ctx := context.Background()
	nextID := 0
	ptyBackend := newMemoryPTYBackend()
	rt := app.NewRuntime(app.RuntimeConfig{
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("id_%02d", nextID)
		},
		WorkItemStore:              &memoryWorkItemStore{},
		PTYBackend:                 ptyBackend,
		DaemonURL:                  daemonURL,
		CLIPath:                    whiskBin,
		AgentBridgeApprovalTimeout: approvalTimeout,
	})
	t.Cleanup(func() { _ = rt.Shutdown(ctx) })

	srv := httptest.NewUnstartedServer(server.NewHTTP(rt))
	srv.Listener = l
	srv.Start()
	t.Cleanup(srv.Close)

	project, err := rt.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	definition := customAgentBridgeWorkflowDefinition()
	imported, err := rt.ImportWorkflowDefinition(ctx, app.ImportWorkflowDefinitionRequest{
		Definition: definition,
		Source:     "test",
	})
	if err != nil {
		t.Fatalf("import workflow: %v", err)
	}
	project, err = rt.SetProjectWorkflowDefinition(ctx, app.SetProjectWorkflowDefinitionRequest{
		ProjectID: project.ID,
		ID:        imported.ID,
		Version:   imported.Version,
	})
	if err != nil {
		t.Fatalf("set workflow: %v", err)
	}
	if err := launch(ctx, rt, project); err != nil {
		t.Fatalf("launch custom bridge: %v", err)
	}
	if len(ptyBackend.spawns) != 1 {
		t.Fatalf("expected 1 spawn, got %d", len(ptyBackend.spawns))
	}

	spawnEnv := ptyBackend.spawns[0].Env
	bridgeID := spawnEnv["WHISK_AGENT_BRIDGE_ID"]
	hookScript := spawnEnv["WHISK_AGENT_BRIDGE_HOOK_SCRIPT"]
	if bridgeID == "" || hookScript == "" {
		t.Fatalf("missing bridge env in spawn: %#v", spawnEnv)
	}

	agentEnv := os.Environ()
	for _, key := range []string{
		"WHISK_CLI", "WHISKD_URL",
		"WHISK_AGENT_BRIDGE_ID", "WHISK_AGENT_BRIDGE_TOKEN", "WHISK_AGENT_BRIDGE_PROVIDER",
		"WHISK_SESSION_ID", "WHISK_PTY_ID",
		"WHISK_PROJECT_ID", "WHISK_WORK_ITEM_ID", "WHISK_RUN_ID", "WHISK_ACTOR",
		"PATH",
	} {
		if val := spawnEnv[key]; val != "" {
			agentEnv = append(agentEnv, key+"="+val)
		}
	}

	return agentBridgeHTTPFixture{
		Runtime:    rt,
		Server:     srv,
		BridgeID:   bridgeID,
		HookScript: hookScript,
		AgentEnv:   agentEnv,
	}
}

func launchCustomPlanningAgentBridgeWithHTTP(t *testing.T) agentBridgeHTTPFixture {
	t.Helper()
	return launchCustomAgentBridgeWithHTTP(t, 5*time.Second, func(ctx context.Context, rt *app.Runtime, project workitem.Project) error {
		item, err := rt.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Plan first"})
		if err != nil {
			return fmt.Errorf("create work item: %w", err)
		}
		if _, err := rt.StartPlanning(ctx, app.StartPlanningRequest{
			WorkItemID:     item.ID,
			Launch:         true,
			AgentProfileID: "claude-plan",
			Actor:          "agent",
		}); err != nil {
			return fmt.Errorf("start planning: %w", err)
		}
		return nil
	})
}

func launchCustomExecutionAgentBridgeWithHTTP(t *testing.T) agentBridgeHTTPFixture {
	t.Helper()
	return launchCustomAgentBridgeWithHTTP(t, 5*time.Second, func(ctx context.Context, rt *app.Runtime, project workitem.Project) error {
		item, err := rt.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Auto review"})
		if err != nil {
			return fmt.Errorf("create work item: %w", err)
		}
		planningRun, err := rt.StartPlanning(ctx, app.StartPlanningRequest{WorkItemID: item.ID, Actor: "agent"})
		if err != nil {
			return fmt.Errorf("start planning: %w", err)
		}
		draft, err := rt.SubmitDraftPlan(ctx, app.SubmitDraftPlanRequest{
			WorkItemID: item.ID,
			RunID:      planningRun.ID,
			Body:       "Implement it.",
			Actor:      "agent",
		})
		if err != nil {
			return fmt.Errorf("submit draft plan: %w", err)
		}
		if _, err := rt.ApprovePlan(ctx, app.ApprovePlanRequest{WorkItemID: item.ID, ArtifactID: draft.ID, Actor: "human"}); err != nil {
			return fmt.Errorf("approve plan: %w", err)
		}
		if _, err := rt.StartExecution(ctx, app.StartExecutionRequest{
			WorkItemID:     item.ID,
			Launch:         true,
			AgentProfileID: "claude",
			Actor:          "agent",
		}); err != nil {
			return fmt.Errorf("start execution: %w", err)
		}
		return nil
	})
}

// writeTestagentSettings writes a Claude-format settings.json that registers hookScript
// as a Type="command" handler for all relevant hook events. Returns the file path.
func writeTestagentSettings(t *testing.T, hookScript string) string {
	t.Helper()
	hook := []any{map[string]any{
		"hooks": []any{map[string]any{"type": "command", "command": hookScript}},
	}}
	settings := map[string]any{
		"hooks": map[string]any{
			"SessionStart":      hook,
			"SessionEnd":        hook,
			"Stop":              hook,
			"PreToolUse":        hook,
			"PermissionRequest": hook,
			"PostToolUse":       hook,
		},
	}
	raw, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write testagent settings: %v", err)
	}
	return path
}

// execTestagent runs "go tool testagent <provider> --settings <path>" with script on stdin.
// Safe to call from a goroutine — it does not call t.Fatal.
func execTestagent(provider, settingsPath string, env []string, script string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "tool", "testagent", provider, "--settings", settingsPath)
	cmd.Stdin = strings.NewReader(script)
	cmd.Env = env
	return cmd.CombinedOutput()
}

// waitForApproval polls until a pending approval appears or the deadline passes.
func waitForApproval(t *testing.T, rt *app.Runtime, timeout time.Duration) agentbridge.Approval {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		approvals, err := rt.ListAgentBridgeApprovals(context.Background(), app.ListAgentBridgeApprovalsRequest{Status: "pending"})
		if err != nil {
			t.Fatalf("list approvals: %v", err)
		}
		if len(approvals) > 0 {
			return approvals[0]
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out after %v waiting for pending approval", timeout)
	return agentbridge.Approval{}
}

func TestTestagentExitPlanModeSubmitsDraftPlanForCustomWorkflow(t *testing.T) {
	fix := launchCustomPlanningAgentBridgeWithHTTP(t)
	settings := writeTestagentSettings(t, fix.HookScript)

	out, err := execTestagent(
		"claude",
		settings,
		fix.AgentEnv,
		"/fake-tool ExitPlanMode {\"plan\":\"## Plan\\nDo it.\"}\n/exit\n",
		60*time.Second,
	)
	if err != nil {
		t.Fatalf("testagent: %v\noutput:\n%s", err, out)
	}

	items, err := fix.Runtime.ListWorkItems(context.Background(), "")
	if err != nil {
		t.Fatalf("list work items: %v", err)
	}
	if len(items) != 1 || items[0].StageID != "design" {
		t.Fatalf("items = %#v", items)
	}
	artifacts, err := fix.Runtime.ListArtifacts(context.Background(), items[0].ID)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) != 1 ||
		artifacts[0].Kind != workitem.ArtifactKindPlan ||
		artifacts[0].Status != workitem.ArtifactStatusDraft ||
		artifacts[0].Body != "## Plan\nDo it." {
		t.Fatalf("artifacts = %#v", artifacts)
	}
}

func TestTestagentStopCompletesExecutionForCustomWorkflow(t *testing.T) {
	fix := launchCustomExecutionAgentBridgeWithHTTP(t)
	settings := writeTestagentSettings(t, fix.HookScript)

	out, err := execTestagent(
		"claude",
		settings,
		fix.AgentEnv,
		"Implemented and tested.\n/exit\n",
		60*time.Second,
	)
	if err != nil {
		t.Fatalf("testagent: %v\noutput:\n%s", err, out)
	}

	items, err := fix.Runtime.ListWorkItems(context.Background(), "")
	if err != nil {
		t.Fatalf("list work items: %v", err)
	}
	if len(items) != 1 || items[0].StageID != "reviewing" {
		t.Fatalf("items = %#v", items)
	}
	runs, err := fix.Runtime.ListWorkItemRuns(context.Background(), items[0].ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	run, ok := executionRun(runs)
	if !ok || run.Status != workitem.RunStateCompleted {
		t.Fatalf("runs = %#v", runs)
	}
}

// TestTestagentPassiveHooksAreLogged verifies that passive hook events fired by
// a real testagent process arrive at the daemon through the HTTP layer and are
// recorded by the runtime. This exercises the full path: testagent → hook.sh →
// whisk agent-bridge hook → POST /v1/agent-bridges/{id}/hooks → runtime.
func TestTestagentPassiveHooksAreLogged(t *testing.T) {
	fix := launchAgentBridgeWithHTTP(t, 5*time.Second)
	settings := writeTestagentSettings(t, fix.HookScript)

	// /exit fires SessionEnd; SessionStart fires at boot. Both are passive — no
	// blocking approval needed. The daemon records them via recordAgentBridgeEvent.
	out, err := execTestagent("claude", settings, fix.AgentEnv, "/exit\n", 60*time.Second)
	if err != nil {
		t.Fatalf("testagent: %v\noutput:\n%s", err, out)
	}

	events, err := fix.Runtime.ListAgentBridgeEvents(context.Background(), app.ListAgentBridgeEventsRequest{})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("expected hook events logged by daemon, got none")
	}
	var names []string
	for _, e := range events {
		names = append(names, e.EventName)
	}
	t.Logf("logged events: %v", names)
}

// TestTestagentApprovalFlowThroughHTTP verifies the blocking approval path:
// testagent fires a PermissionRequest hook, the daemon suspends waiting for
// human approval, the test resolves it via the runtime API, and testagent
// receives the allow decision and continues. This exercises the round-trip
// that would otherwise require a real claude/codex binary.
func TestTestagentApprovalFlowThroughHTTP(t *testing.T) {
	fix := launchAgentBridgeWithHTTP(t, 60*time.Second)
	settings := writeTestagentSettings(t, fix.HookScript)

	type result struct {
		out []byte
		err error
	}
	done := make(chan result, 1)
	go func() {
		// Fires PermissionRequest (blocking) then exits after approval resolves.
		out, err := execTestagent(
			"claude", settings, fix.AgentEnv,
			"/fake-permission-request Bash {\"command\":\"ls\"}\n/exit\n",
			90*time.Second,
		)
		done <- result{out, err}
	}()

	// Wait for the PermissionRequest hook to arrive and block in the daemon.
	approval := waitForApproval(t, fix.Runtime, 60*time.Second)

	// Resolve with allow — this unblocks the hook.sh subprocess.
	if _, err := fix.Runtime.ResolveAgentBridgeApproval(context.Background(), app.ResolveAgentBridgeApprovalRequest{
		ID:     approval.ID,
		Action: "allow",
	}); err != nil {
		t.Fatalf("resolve approval: %v", err)
	}

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("testagent: %v\noutput:\n%s", res.err, res.out)
		}
		t.Logf("testagent output:\n%s", res.out)
	case <-time.After(30 * time.Second):
		t.Fatalf("testagent did not finish after approval was resolved")
	}
}
