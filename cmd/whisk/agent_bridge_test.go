package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/protocol"
)

func TestRunAgentBridgeHookPostsProviderPayloadAndPrintsOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/agent-bridges/bridge_01/hooks" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req protocol.AgentBridgeHookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Token != "secret" ||
			req.Provider != "claude" ||
			req.EventName != "PreToolUse" ||
			req.ToolName != "Bash" ||
			req.ToolInput["command"] != "pwd" {
			t.Fatalf("request = %#v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.AgentBridgeHookResponse{
			Output: map[string]any{
				"hookSpecificOutput": map[string]any{
					"hookEventName":      "PreToolUse",
					"permissionDecision": "deny",
				},
			},
		})
	}))
	defer server.Close()

	var stdout bytes.Buffer
	stdin := strings.NewReader(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"pwd"}}`)
	err := runAgentBridgeHook([]string{
		"-url", server.URL,
		"-bridge", "bridge_01",
		"-token", "secret",
		"-provider", "claude",
	}, stdin, &stdout)
	if err != nil {
		t.Fatalf("hook: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout %q: %v", stdout.String(), err)
	}
	hookSpecific := output["hookSpecificOutput"].(map[string]any)
	if hookSpecific["permissionDecision"] != "deny" {
		t.Fatalf("output = %#v", output)
	}
}

func TestRunAgentBridgeHookFailsOpenOnDaemonError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	var stdout bytes.Buffer
	stdin := strings.NewReader(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"pwd"}}`)
	err := runAgentBridgeHook([]string{
		"-url", server.URL,
		"-bridge", "bridge_01",
		"-token", "secret",
		"-provider", "claude",
	}, stdin, &stdout)
	if err != nil {
		t.Fatalf("hook should fail open, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunAgentBridgeHookRecordsPassiveEventWithoutBridgeCredentials(t *testing.T) {
	var got protocol.AgentBridgeHookRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/agent-hook-events" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.AgentBridgeEvent{ID: "event_01", Provider: got.Provider, EventName: got.EventName})
	}))
	defer server.Close()

	stdin := strings.NewReader(`{"hook_event_name":"Notification","notification_type":"elicitation_dialog","message":"Need input","token":"redact-me"}`)
	var stdout bytes.Buffer
	err := runAgentBridgeHook([]string{
		"-url", server.URL,
		"-provider", "claude",
	}, stdin, &stdout)
	if err != nil {
		t.Fatalf("hook: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if got.Provider != "claude" ||
		got.EventName != "Notification" ||
		got.NotificationType != "elicitation_dialog" ||
		got.Message != "Need input" ||
		got.RawPayload["token"] != "redact-me" {
		t.Fatalf("request = %#v", got)
	}
}

func TestRunAgentBridgeHookAddsWhiskMetadataToRawPayload(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	t.Setenv("WHISK_SESSION_ID", "sess_01")
	t.Setenv("WHISK_PTY_ID", "pty_01")
	t.Setenv("WHISK_PROJECT_ID", "proj_01")
	t.Setenv("WHISK_PROJECT_ROOT", "/repo")
	t.Setenv("WHISK_WORK_ITEM_ID", "wi_01")
	t.Setenv("WHISK_RUN_ID", "run_01")
	t.Setenv("WHISK_ACTOR", "agent")

	var got protocol.AgentBridgeHookRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/agent-hook-events" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.AgentBridgeEvent{ID: "event_01", Provider: got.Provider, EventName: got.EventName})
	}))
	defer server.Close()

	stdin := strings.NewReader(`{"hook_event_name":"PostToolUse","tool_name":"Bash"}`)
	var stdout bytes.Buffer
	if err := runAgentBridgeHook([]string{"-url", server.URL, "-provider", "codex"}, stdin, &stdout); err != nil {
		t.Fatalf("hook: %v", err)
	}

	meta, ok := got.RawPayload["whisk"].(map[string]any)
	if !ok {
		t.Fatalf("whisk metadata missing: %#v", got.RawPayload)
	}
	if meta["cwd"] != cwd ||
		meta["sessionId"] != "sess_01" ||
		meta["ptyId"] != "pty_01" ||
		meta["projectId"] != "proj_01" ||
		meta["projectRoot"] != "/repo" ||
		meta["workItemId"] != "wi_01" ||
		meta["runId"] != "run_01" ||
		meta["actor"] != "agent" {
		t.Fatalf("whisk metadata = %#v", meta)
	}
}
