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
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/daemon"
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

	helper := filepath.Join(t.TempDir(), "whiskd-helper")
	script := fmt.Sprintf("#!/bin/sh\nWHISKD_HELPER_PROCESS=1 exec %q -test.run TestWhiskdHelperProcess -- \"$@\"\n", os.Args[0])
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	t.Setenv("WHISKD_PATH", helper)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := daemon.Ensure(ctx, "http://"+addr); err != nil {
		t.Fatalf("ensure daemon: %v", err)
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
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-addr" {
			addr = args[i+1]
		}
	}
	if addr == "" {
		os.Exit(2)
	}

	done := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
		close(done)
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
