package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunDaemonStatusUsesHealthEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/health" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	if err := run([]string{"daemon", "status", "-url", server.URL}); err != nil {
		t.Fatalf("status: %v", err)
	}
}

func TestRunDaemonStopUsesShutdownEndpoint(t *testing.T) {
	called := false
	healthy := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/health":
			if !healthy {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/shutdown":
			called = true
			healthy = false
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := run([]string{"daemon", "stop", "-url", server.URL}); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if !called {
		t.Fatalf("shutdown endpoint was not called")
	}
}

func TestRunDaemonClearRequiresConfirmation(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := run([]string{"daemon", "clear", "-url", server.URL})
	if err == nil || !strings.Contains(err.Error(), "requires -yes") {
		t.Fatalf("clear error = %v", err)
	}
	if called {
		t.Fatalf("clear endpoint was called without confirmation")
	}
}

func TestRunDaemonClearUsesClearEndpoint(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/daemon/clear" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sessionsCleared":1,"ptysCleared":2,"projectsCleared":4,"workItemsCleared":5,"forwardsCleared":6}`))
	}))
	defer server.Close()

	output, err := captureStdout(func() error {
		return run([]string{"daemon", "clear", "-url", server.URL, "-yes"})
	})
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if strings.Contains(output, "bookmarks") {
		t.Fatalf("clear output still includes bookmarks: %q", output)
	}
	if !strings.Contains(output, "sessions=1 ptys=2 projects=4 workItems=5 forwards=6") {
		t.Fatalf("clear output = %q", output)
	}
	if !called {
		t.Fatalf("clear endpoint was not called")
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	if err := run([]string{"daemon", "bogus"}); err == nil {
		t.Fatalf("expected unknown command error")
	}
	if err := run([]string{}); err == nil {
		t.Fatalf("expected usage error")
	}
}

func TestRunVersionReportsWhiskVersion(t *testing.T) {
	if err := run([]string{"version"}); err != nil {
		t.Fatalf("version: %v", err)
	}
}

func TestRunDaemonStatusReportsUnavailableDaemon(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`nope`))
	}))
	defer server.Close()

	if err := run([]string{"daemon", "status", "-url", server.URL}); err == nil {
		t.Fatalf("expected unavailable daemon error")
	}
}

func TestRunDaemonStopReportsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/shutdown":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := run([]string{"daemon", "stop", "-url", server.URL}); err == nil {
		t.Fatalf("expected stop error")
	}
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("WHISK_TEST_ENV", "set")
	if got := envOrDefault("WHISK_TEST_ENV", "fallback"); got != "set" {
		t.Fatalf("env value = %q", got)
	}
	if got := envOrDefault("WHISK_TEST_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("fallback = %q", got)
	}
}
