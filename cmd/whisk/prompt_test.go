package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phin-tech/whisk/internal/protocol"
)

func TestRunPromptListAndResolveUseDaemonAPI(t *testing.T) {
	var resolved protocol.ResolveAgentPromptRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/agent-prompts":
			if r.URL.Query().Get("status") != "pending" {
				t.Fatalf("status query = %q", r.URL.Query().Get("status"))
			}
			_ = json.NewEncoder(w).Encode([]protocol.AgentPrompt{{
				ID:       "prompt_01",
				Kind:     "question",
				Provider: "claude",
				Message:  "Pick one",
				Status:   "pending",
			}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/agent-prompts/prompt_01/resolve":
			if err := json.NewDecoder(r.Body).Decode(&resolved); err != nil {
				t.Fatalf("decode resolve: %v", err)
			}
			_ = json.NewEncoder(w).Encode(protocol.AgentPrompt{ID: "prompt_01", Status: "resolved", Answer: resolved.Answer})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	if err := run([]string{"prompt", "list", "-url", server.URL, "-json"}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if err := run([]string{"prompt", "resolve", "prompt_01", "-answer", "ship", "-url", server.URL, "-json"}); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Answer != "ship" {
		t.Fatalf("resolve request = %#v", resolved)
	}
}
