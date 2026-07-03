package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/daemon"
	"github.com/phin-tech/whisk/internal/protocol"
)

func TestRunDaemonStatusUsesHealthEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/compat":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"apiVersion":%d}`, protocol.DaemonAPIVersion)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
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

func TestConfigureDaemonLoggingWritesStateDirLogAndMirrorsStderr(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	t.Setenv("WHISK_STATE_DIR", stateDir)

	var stderr bytes.Buffer
	logPath, cleanup, err := configureDaemonLogging("127.0.0.1:19996", &stderr, daemon.LogRotation{
		MaxBytes:   1024,
		MaxBackups: 1,
	})
	if err != nil {
		t.Fatalf("configure daemon logging: %v", err)
	}
	log.Print("daemon mirror test")
	if err := cleanup(); err != nil {
		t.Fatalf("cleanup daemon logging: %v", err)
	}

	if !strings.HasPrefix(logPath, stateDir+string(filepath.Separator)) {
		t.Fatalf("log path = %q, want under %q", logPath, stateDir)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read daemon log: %v", err)
	}
	if !strings.Contains(string(data), "daemon mirror test") {
		t.Fatalf("daemon log did not contain message: %q", string(data))
	}
	if !strings.Contains(stderr.String(), "daemon mirror test") {
		t.Fatalf("stderr mirror did not contain message: %q", stderr.String())
	}
}

func TestServeDaemonWritesOwnPerAddressLogOnDuplicateListenAddress(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	t.Setenv("WHISK_STATE_DIR", stateDir)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	addr := listener.Addr().String()

	if err := serveDaemon(addr); err != nil {
		t.Fatalf("serve daemon on duplicate address: %v", err)
	}

	logPath, err := daemon.LogPath("http://" + addr)
	if err != nil {
		t.Fatalf("log path: %v", err)
	}
	if !strings.HasPrefix(logPath, stateDir+string(filepath.Separator)) {
		t.Fatalf("log path = %q, want under %q", logPath, stateDir)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read daemon log: %v", err)
	}
	if !strings.Contains(string(data), "another instance is already listening on "+addr) {
		t.Fatalf("daemon log did not contain duplicate-address message: %q", string(data))
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
