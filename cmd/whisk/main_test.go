package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
		_, _ = w.Write([]byte(`{"sessionsCleared":1,"ptysCleared":2,"bookmarksCleared":3,"projectsCleared":4,"workItemsCleared":5,"forwardsCleared":6}`))
	}))
	defer server.Close()

	if err := run([]string{"daemon", "clear", "-url", server.URL, "-yes"}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if !called {
		t.Fatalf("clear endpoint was not called")
	}
}

func TestRunDaemonRunValidatesListenAddress(t *testing.T) {
	err := run([]string{"daemon", "run", "-addr", "0.0.0.0:8787"})
	if err == nil {
		t.Fatalf("expected non-loopback daemon run address to be rejected")
	}
	if !strings.Contains(err.Error(), "refusing non-loopback bind") {
		t.Fatalf("daemon run error = %q", err.Error())
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

func TestRunDaemonRestartStopsThenStarts(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("WHISKD_PATH", "")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()

	shutdownCalled := false
	server := &http.Server{ReadHeaderTimeout: time.Second}
	mux := http.NewServeMux()
	server.Handler = mux
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/v1/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		shutdownCalled = true
		w.WriteHeader(http.StatusNoContent)
		go func() { _ = server.Shutdown(context.Background()) }()
	})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })

	// With no daemon binary available, restart stops the running daemon and then fails to bring a
	// fresh one up. That is enough to prove the CLI wires stop -> start like the GUI does.
	err = run([]string{"daemon", "restart", "-url", "http://" + addr})
	if err == nil {
		t.Fatalf("expected restart to fail without a daemon binary")
	}
	if !strings.Contains(err.Error(), "restart whiskd") {
		t.Fatalf("restart error = %v", err)
	}
	if !shutdownCalled {
		t.Fatalf("restart did not stop the running daemon")
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
