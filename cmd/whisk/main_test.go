package main

import (
	"net/http"
	"net/http/httptest"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/shutdown" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := run([]string{"daemon", "stop", "-url", server.URL}); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if !called {
		t.Fatalf("shutdown endpoint was not called")
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
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
