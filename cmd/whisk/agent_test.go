package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phin-tech/whisk/internal/protocol"
)

func TestRunAgentProfilesUsesDaemonAPI(t *testing.T) {
	var requested string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/agent-profiles" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		requested = r.URL.Path
		_ = json.NewEncoder(w).Encode([]protocol.AgentProfile{
			{ID: "claude", Provider: "claude", Label: "Claude Code", Description: "Claude Code with default permissions.", Source: "builtin", Launchable: true},
			{ID: "codex", Provider: "codex", Label: "Codex", Source: "builtin", Launchable: true},
		})
	}))
	defer server.Close()

	if err := run([]string{"agent", "profiles", "-url", server.URL, "-json"}); err != nil {
		t.Fatalf("agent profiles json: %v", err)
	}
	if err := run([]string{"agent", "profiles", "-url", server.URL}); err != nil {
		t.Fatalf("agent profiles table: %v", err)
	}
	if requested != "/v1/agent-profiles" {
		t.Fatalf("daemon route = %q", requested)
	}
}
