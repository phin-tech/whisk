package daemon_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/daemon"
	"github.com/phin-tech/whisk/internal/protocol"
)

func TestEnsureStartsDaemonWhenDown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	t.Setenv("WHISKD_PATH", writeWhiskdHelper(t))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	started, err := daemon.Ensure(ctx, "http://"+addr)
	if err != nil {
		t.Fatalf("ensure daemon: %v", err)
	}
	if !started {
		t.Fatalf("expected Ensure to report it started the daemon")
	}
}

func TestEnsureRestartsIncompatibleDaemon(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()

	shutdownCalled := make(chan struct{})
	oldServer := &http.Server{ReadHeaderTimeout: time.Second}
	mux := http.NewServeMux()
	oldServer.Handler = mux
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/v1/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		close(shutdownCalled)
		go func() {
			_ = oldServer.Shutdown(context.Background())
		}()
	})
	go func() {
		_ = oldServer.Serve(listener)
	}()
	t.Cleanup(func() { _ = oldServer.Shutdown(context.Background()) })

	t.Setenv("WHISKD_PATH", writeWhiskdHelper(t))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	started, err := daemon.Ensure(ctx, "http://"+addr)
	if err != nil {
		t.Fatalf("ensure daemon: %v", err)
	}
	if !started {
		t.Fatalf("expected Ensure to report it started the replacement daemon")
	}
	select {
	case <-shutdownCalled:
	default:
		t.Fatalf("incompatible daemon was not asked to shut down")
	}
}

func TestWhiskdHelperProcess(t *testing.T) {
	if os.Getenv("WHISKD_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}

	var addr string
	if len(args) < 2 || args[0] != "daemon" || args[1] != "run" {
		os.Exit(4)
	}
	args = args[2:]
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-addr" {
			addr = args[i+1]
		}
	}
	if addr == "" {
		os.Exit(2)
	}

	done := make(chan struct{})
	var doneOnce sync.Once
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/v1/compat", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"apiVersion":%d}`, protocol.DaemonAPIVersion)
		doneOnce.Do(func() { close(done) })
	})
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: time.Second}
	go func() {
		_ = server.ListenAndServe()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		os.Exit(3)
	}
	_ = server.Shutdown(context.Background())
	os.Exit(0)
}

func writeWhiskdHelper(t *testing.T) string {
	t.Helper()
	helper := filepath.Join(t.TempDir(), "whiskd-helper")
	script := fmt.Sprintf("#!/bin/sh\nWHISKD_HELPER_PROCESS=1 exec %q -test.run TestWhiskdHelperProcess -- \"$@\"\n", os.Args[0])
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	return helper
}

func TestStopPIDSignalsRecordedProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}

	script := filepath.Join(t.TempDir(), "wait.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntrap 'exit 0' INT TERM\nwhile true; do sleep 1; done\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	cmd := exec.Command(script)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start script: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	baseURL := "http://127.0.0.1:19991"
	pidPath, err := daemon.PIDPath(baseURL)
	if err != nil {
		t.Fatalf("pid path: %v", err)
	}
	if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o600); err != nil {
		t.Fatalf("write pid: %v", err)
	}

	if err := daemon.StopPID(baseURL); err != nil {
		t.Fatalf("stop pid: %v", err)
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatalf("pid file still exists or unexpected stat error: %v", err)
	}
}

func TestSupervisorRejectsInvalidDaemonURLAndPIDFiles(t *testing.T) {
	if _, err := daemon.PIDPath("://bad-url"); err == nil {
		t.Fatalf("expected invalid pid path URL error")
	}
	if _, err := daemon.PIDPath("http:///missing-host"); err == nil {
		t.Fatalf("expected missing host error")
	}
	if err := daemon.StopPID("http://127.0.0.1:1"); err == nil {
		t.Fatalf("expected missing pid file error")
	}
	pidPath, err := daemon.PIDPath("http://127.0.0.1:19992")
	if err != nil {
		t.Fatalf("pid path: %v", err)
	}
	if err := os.WriteFile(pidPath, []byte("not-a-pid\n"), 0o600); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(pidPath) })
	if err := daemon.StopPID("http://127.0.0.1:19992"); err == nil {
		t.Fatalf("expected invalid pid file error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if _, err := daemon.Ensure(ctx, "http:///missing-host"); err == nil {
		t.Fatalf("expected ensure URL error")
	}
}

func TestDaemonPathCanFindBundledWhiskHelper(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("WHISKD_PATH", "")

	bundleDir := t.TempDir()
	// The GUI binary is "whisk-app" and the bundled CLI/daemon helper is "whisk". These names
	// must differ so they do not collide on case-insensitive filesystems.
	appExecutable := filepath.Join(bundleDir, "whisk-app")
	helper := filepath.Join(bundleDir, "whisk")
	if err := os.WriteFile(appExecutable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write app executable: %v", err)
	}
	if err := os.WriteFile(helper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	path, err := daemon.DaemonPathForTest(appExecutable)
	if err != nil {
		t.Fatalf("daemon path: %v", err)
	}
	if path != helper {
		t.Fatalf("daemon path = %q, want %q", path, helper)
	}
}

// TestDaemonPathRejectsRunningExecutable guards against the fork loop: if daemon discovery
// ever resolves to the running executable itself (e.g. a name collision aliases the GUI
// binary into a daemon-candidate path), it must be skipped rather than relaunched.
func TestDaemonPathRejectsRunningExecutable(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("WHISKD_PATH", "")

	bundleDir := t.TempDir()
	appExecutable := filepath.Join(bundleDir, "whisk-app")
	if err := os.WriteFile(appExecutable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write app executable: %v", err)
	}
	// Hard-link a "whisk" candidate to the same inode as the running executable. os.SameFile
	// must detect the alias and refuse it, leaving no valid daemon path.
	alias := filepath.Join(bundleDir, "whisk")
	if err := os.Link(appExecutable, alias); err != nil {
		t.Fatalf("link alias: %v", err)
	}

	if path, err := daemon.DaemonPathForTest(appExecutable); err == nil {
		t.Fatalf("expected no daemon path, got %q", path)
	}
}
