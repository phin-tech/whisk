package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

func TestRunSessionListUsesSessionsEndpoint(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/sessions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]session.Session{
			{ID: "ses_01", Name: "app", RootDir: "/repo", Panes: map[string]session.Pane{"pane_01": {ID: "pane_01"}}},
		})
	}))
	defer server.Close()

	if err := run([]string{"session", "list", "-url", server.URL}); err != nil {
		t.Fatalf("session list: %v", err)
	}
	if !called {
		t.Fatalf("sessions endpoint was not called")
	}
}

func TestRunSessionCreatePostsSessionRequest(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/sessions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req protocol.CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Name != "app" || req.RootDir != root {
			t.Fatalf("request = %#v", req)
		}
		if req.InitialPTY == nil || req.InitialPTY.Command != "codex" {
			t.Fatalf("initial pty = %#v", req.InitialPTY)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.CreatedSession{
			Session: session.Session{ID: "ses_01", Name: "app", RootDir: root, Panes: map[string]session.Pane{}},
			PaneID:  "pane_01",
		})
	}))
	defer server.Close()

	if err := run([]string{"session", "create", "-url", server.URL, "-root", root, "-name", "app", "-command", "codex"}); err != nil {
		t.Fatalf("session create: %v", err)
	}
}

func TestRunSessionCreateResolvesRelativeRoot(t *testing.T) {
	absRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req protocol.CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.RootDir != absRoot {
			t.Fatalf("root dir = %q, want %q", req.RootDir, absRoot)
		}
		if req.InitialPTY != nil {
			t.Fatalf("initial pty = %#v", req.InitialPTY)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.CreatedSession{
			Session: session.Session{ID: "ses_01", RootDir: absRoot, Panes: map[string]session.Pane{}},
			PaneID:  "pane_01",
		})
	}))
	defer server.Close()

	if err := run([]string{"session", "create", "-url", server.URL, "-root", ".", "-pty=false"}); err != nil {
		t.Fatalf("session create: %v", err)
	}
}

func TestRunSessionCreateRejectsCommandWithoutInitialPTY(t *testing.T) {
	if err := run([]string{"session", "create", "-root", ".", "-pty=false", "-command", "codex"}); err == nil {
		t.Fatalf("expected command without initial pty error")
	}
}

func TestRunSessionSetRootUsesSessionSetRootEndpoint(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/sessions/ses_01/set-root-dir" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req protocol.SetSessionRootDirRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.SessionID != "ses_01" || req.RootDir != root {
			t.Fatalf("request = %#v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(session.Session{ID: "ses_01", RootDir: root, Panes: map[string]session.Pane{}})
	}))
	defer server.Close()

	if err := run([]string{"session", "set-root", "-url", server.URL, "ses_01", root}); err != nil {
		t.Fatalf("session set-root: %v", err)
	}
}

func TestRunSessionCloseUsesSessionDeleteEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/sessions/ses_01" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]session.Session{})
	}))
	defer server.Close()

	if err := run([]string{"session", "close", "-url", server.URL, "ses_01"}); err != nil {
		t.Fatalf("session close: %v", err)
	}
}

func TestRunSessionPTYListUsesPTYEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/ptys" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]protocol.PTYInfo{
			{ID: "pty_01", Status: "running", SessionID: "ses_01", PaneID: "pane_01", WorkingDir: "/repo"},
			{ID: "pty_02", Status: "running", SessionID: "ses_02", PaneID: "pane_02", WorkingDir: "/repo"},
		})
	}))
	defer server.Close()

	if err := run([]string{"session", "pty", "list", "-url", server.URL, "ses_01"}); err != nil {
		t.Fatalf("session pty list: %v", err)
	}
}

func TestRunSessionPTYKillUsesPTYKillEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/ptys/pty_01/kill" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req protocol.KillPTYRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.PTYID != "pty_01" {
			t.Fatalf("request = %#v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.PTYInfo{ID: "pty_01", Status: "killed"})
	}))
	defer server.Close()

	if err := run([]string{"session", "pty", "kill", "-url", server.URL, "pty_01"}); err != nil {
		t.Fatalf("session pty kill: %v", err)
	}
}

func TestRunSessionRejectsInvalidUsage(t *testing.T) {
	if err := run([]string{"session", "create"}); err == nil {
		t.Fatalf("expected create usage error")
	}
	if err := run([]string{"session", "set-root"}); err == nil {
		t.Fatalf("expected set-root usage error")
	}
	if err := run([]string{"session", "close"}); err == nil {
		t.Fatalf("expected close usage error")
	}
	if err := run([]string{"session", "pty", "kill"}); err == nil {
		t.Fatalf("expected pty kill usage error")
	}
}
