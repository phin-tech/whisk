package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phin-tech/whisk/internal/protocol"
)

func TestRunOnboardingStatusUsesStatusEndpoint(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/onboarding" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.OnboardingStatus{ShouldShow: true, LocalDaemon: true})
	}))
	defer server.Close()

	if err := run([]string{"onboarding", "status", "-json", "-url", server.URL}); err != nil {
		t.Fatalf("status: %v", err)
	}
	if !called {
		t.Fatalf("status endpoint was not called")
	}
}

func TestRunOnboardingApplyUsesDefaultSelectedItems(t *testing.T) {
	var got protocol.OnboardingApplyRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/onboarding":
			_ = json.NewEncoder(w).Encode(protocol.OnboardingStatus{
				Items: []protocol.OnboardingItem{
					{ID: "skill:codex", SelectedByDefault: true},
					{ID: "plugin:github", SelectedByDefault: false},
				},
				LocalDaemon: true,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/onboarding/apply":
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatalf("decode apply: %v", err)
			}
			_ = json.NewEncoder(w).Encode(protocol.OnboardingStatus{LocalDaemon: true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := run([]string{"onboarding", "apply", "-json", "-url", server.URL}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(got.ItemIDs) != 1 || got.ItemIDs[0] != "skill:codex" {
		t.Fatalf("apply req = %#v", got)
	}
}
