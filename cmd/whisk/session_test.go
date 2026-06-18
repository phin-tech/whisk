package main

import (
	"encoding/base64"
	"encoding/json"
	"math"
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

func TestRunSessionListFiltersProjectJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/sessions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]session.Session{
			{ID: "ses_01", ProjectID: "proj_01", Name: "one", RootDir: "/repo", Panes: map[string]session.Pane{"pane_01": {ID: "pane_01"}}},
			{ID: "ses_02", ProjectID: "proj_02", Name: "two", RootDir: "/repo", Panes: map[string]session.Pane{"pane_02": {ID: "pane_02"}}},
		})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"session", "list", "-url", server.URL, "-project", "proj_01", "-json"})
	})
	if err != nil {
		t.Fatalf("session list: %v", err)
	}
	var sessions []session.Session
	if err := json.Unmarshal([]byte(output), &sessions); err != nil {
		t.Fatalf("json output %q: %v", output, err)
	}
	if len(sessions) != 1 || sessions[0].ID != "ses_01" {
		t.Fatalf("sessions = %#v", sessions)
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

func TestRunSessionCreatePostsProjectID(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/sessions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req protocol.CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.ProjectID != "proj_01" {
			t.Fatalf("project id = %q", req.ProjectID)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.CreatedSession{
			Session: session.Session{ID: "ses_01", ProjectID: req.ProjectID, RootDir: root, Panes: map[string]session.Pane{}},
			PaneID:  "pane_01",
		})
	}))
	defer server.Close()

	if err := run([]string{"session", "create", "-url", server.URL, "-root", root, "-project", "proj_01", "-pty=false"}); err != nil {
		t.Fatalf("session create: %v", err)
	}
}

func TestRunSessionCreatePostsWorkingDir(t *testing.T) {
	root := t.TempDir()
	workingDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/sessions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req protocol.CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.RootDir != root || req.WorkingDir != workingDir {
			t.Fatalf("request = %#v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.CreatedSession{
			Session: session.Session{ID: "ses_01", RootDir: root, Panes: map[string]session.Pane{}},
			PaneID:  "pane_01",
		})
	}))
	defer server.Close()

	if err := run([]string{"session", "create", "-url", server.URL, "-root", root, "-working-dir", workingDir, "-pty=false"}); err != nil {
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

func TestRunSessionUpdateSetsProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/sessions/ses_01/set-project" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req protocol.SetSessionProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.SessionID != "ses_01" || req.ProjectID != "proj_01" {
			t.Fatalf("request = %#v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(session.Session{ID: "ses_01", ProjectID: req.ProjectID, RootDir: "/repo", Panes: map[string]session.Pane{}})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"session", "update", "-url", server.URL, "-project", "proj_01", "-json", "ses_01"})
	})
	if err != nil {
		t.Fatalf("session update: %v", err)
	}
	var updated session.Session
	if err := json.Unmarshal([]byte(output), &updated); err != nil {
		t.Fatalf("json output %q: %v", output, err)
	}
	if updated.ProjectID != "proj_01" {
		t.Fatalf("updated = %#v", updated)
	}
}

func TestRunSessionUpdateClearsProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req protocol.SetSessionProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.ProjectID != "" {
			t.Fatalf("project id = %q", req.ProjectID)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(session.Session{ID: "ses_01", RootDir: "/repo", Panes: map[string]session.Pane{}})
	}))
	defer server.Close()

	if err := run([]string{"session", "update", "-url", server.URL, "-clear-project", "ses_01"}); err != nil {
		t.Fatalf("session update: %v", err)
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

func TestRunSessionPTYWriteUsesPTYWriteEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/ptys/pty_01/write" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req protocol.WritePTYRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.PtyID != "pty_01" || req.Data != "printf ok\n" {
			t.Fatalf("request = %#v", req)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := run([]string{"session", "pty", "write", "-url", server.URL, "-data", "printf ok\n", "pty_01"}); err != nil {
		t.Fatalf("session pty write: %v", err)
	}
}

func TestRunSessionPTYResizeUsesPTYResizeEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/ptys/pty_01/resize" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req protocol.ResizePTYRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.PtyID != "pty_01" || req.Cols != 120 || req.Rows != 40 {
			t.Fatalf("request = %#v", req)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := run([]string{"session", "pty", "resize", "-url", server.URL, "-cols", "120", "-rows", "40", "pty_01"}); err != nil {
		t.Fatalf("session pty resize: %v", err)
	}
}

func TestRunSessionPTYOutputUsesPTYOutputEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/ptys/pty_01/output" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("from") != "7" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.OutputSnapshot{
			PtyID:        "pty_01",
			Offset:       19,
			OutputBase64: base64.StdEncoding.EncodeToString([]byte("prompt text")),
		})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"session", "pty", "output", "-url", server.URL, "-from", "7", "pty_01"})
	})
	if err != nil {
		t.Fatalf("session pty output: %v", err)
	}
	if output != "prompt text" {
		t.Fatalf("output = %q", output)
	}
}

func TestRunSessionPTYOutputCanStripANSIEscapes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/ptys/pty_01/output" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.OutputSnapshot{
			PtyID:        "pty_01",
			Offset:       42,
			OutputBase64: base64.StdEncoding.EncodeToString([]byte("\x1b[31mPlan\x1b[0m\r\n\x1b]0;title\aReady")),
		})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"session", "pty", "output", "-url", server.URL, "-plain", "pty_01"})
	})
	if err != nil {
		t.Fatalf("session pty output: %v", err)
	}
	if output != "Plan\nReady" {
		t.Fatalf("output = %q", output)
	}
}

func TestRunSessionPTYTailPollsPTYOutputEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/ptys/pty_01/output" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("from") != "19" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.OutputSnapshot{
			PtyID:        "pty_01",
			Offset:       31,
			OutputBase64: base64.StdEncoding.EncodeToString([]byte("next output")),
		})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"session", "pty", "tail", "-url", server.URL, "-from", "19", "-once", "pty_01"})
	})
	if err != nil {
		t.Fatalf("session pty tail: %v", err)
	}
	if output != "next output" {
		t.Fatalf("output = %q", output)
	}
}

func TestRunSessionPTYTailDefaultsToEndOffset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/ptys/pty_01/output" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("from") != "18446744073709551615" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.OutputSnapshot{
			PtyID:  "pty_01",
			Offset: math.MaxUint64,
		})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"session", "pty", "tail", "-url", server.URL, "-once", "pty_01"})
	})
	if err != nil {
		t.Fatalf("session pty tail: %v", err)
	}
	if output != "" {
		t.Fatalf("output = %q", output)
	}
}

func TestRunSessionPTYTailCanStripANSIEscapes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/ptys/pty_01/output" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(protocol.OutputSnapshot{
			PtyID:        "pty_01",
			Offset:       31,
			OutputBase64: base64.StdEncoding.EncodeToString([]byte("\x1b[1mnext\x1b[22m output")),
		})
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"session", "pty", "tail", "-url", server.URL, "-plain", "-once", "pty_01"})
	})
	if err != nil {
		t.Fatalf("session pty tail: %v", err)
	}
	if output != "next output" {
		t.Fatalf("output = %q", output)
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
	if err := run([]string{"session", "pty", "output"}); err == nil {
		t.Fatalf("expected pty output usage error")
	}
	if err := run([]string{"session", "pty", "tail"}); err == nil {
		t.Fatalf("expected pty tail usage error")
	}
	if err := run([]string{"session", "pty", "tail", "-from", "nope", "pty_01"}); err == nil {
		t.Fatalf("expected pty tail from usage error")
	}
}
