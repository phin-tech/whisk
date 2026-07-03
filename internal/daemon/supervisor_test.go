package daemon_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
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

	t.Setenv("WHISKD_PATH", writeWhiskHelper(t))
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
	mux.HandleFunc("/v1/compat", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"apiVersion":%d}`, protocol.DaemonAPIVersion+1)
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

	t.Setenv("WHISKD_PATH", writeWhiskHelper(t))
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

func TestEnsureAdoptsSlowCompatibleDaemonWithoutShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()

	shutdownCalled := make(chan struct{})
	server := &http.Server{ReadHeaderTimeout: time.Second}
	mux := http.NewServeMux()
	server.Handler = mux
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/v1/compat", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(350 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"apiVersion":%d}`, protocol.DaemonAPIVersion)
	})
	mux.HandleFunc("/v1/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		close(shutdownCalled)
	})
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })

	t.Setenv("PATH", "")
	t.Setenv("WHISKD_PATH", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	started, err := daemon.Ensure(ctx, "http://"+addr)
	if err != nil {
		t.Fatalf("ensure daemon: %v", err)
	}
	if started {
		t.Fatalf("expected Ensure to adopt the existing daemon")
	}
	select {
	case <-shutdownCalled:
		t.Fatalf("slow compatible daemon was asked to shut down")
	default:
	}
}

func TestEnsureLeavesDaemonRunningWhenCompatibilityUnknown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()

	shutdownCalled := make(chan struct{})
	server := &http.Server{ReadHeaderTimeout: time.Second}
	mux := http.NewServeMux()
	server.Handler = mux
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/v1/compat", func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	mux.HandleFunc("/v1/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		close(shutdownCalled)
	})
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })

	t.Setenv("PATH", "")
	t.Setenv("WHISKD_PATH", "")
	ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel()

	started, err := daemon.Ensure(ctx, "http://"+addr)
	if err == nil {
		t.Fatalf("expected compatibility error")
	}
	if started {
		t.Fatalf("expected Ensure not to start a replacement daemon")
	}
	if !strings.Contains(err.Error(), "compatibility") {
		t.Fatalf("expected clear compatibility error, got %v", err)
	}
	select {
	case <-shutdownCalled:
		t.Fatalf("unknown compatibility daemon was asked to shut down")
	default:
	}
}

func TestWhiskHelperProcess(t *testing.T) {
	if os.Getenv("WHISK_HELPER_PROCESS") != "1" {
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

func writeWhiskHelper(t *testing.T) string {
	t.Helper()
	helper := filepath.Join(t.TempDir(), "whisk-helper")
	script := fmt.Sprintf("#!/bin/sh\nWHISK_HELPER_PROCESS=1 exec %q -test.run TestWhiskHelperProcess -- \"$@\"\n", os.Args[0])
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	return helper
}

func TestStopAllowsSlowDrainWithoutKillingProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}

	addr := freeSupervisorAddr(t)
	baseURL := "http://" + addr
	signalPath := filepath.Join(t.TempDir(), "signals.log")
	drain := 5 * time.Second
	cmd, wait := startSlowDrainHelper(t, addr, drain, signalPath)
	waitForSupervisorHealth(t, baseURL)
	writeSupervisorPID(t, baseURL, cmd.Process.Pid)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	started := time.Now()
	if err := daemon.Stop(ctx, baseURL); err != nil {
		t.Fatalf("stop slow-draining daemon: %v", err)
	}
	if elapsed := time.Since(started); elapsed < drain-250*time.Millisecond {
		t.Fatalf("stop returned before helper drained: elapsed=%s drain=%s", elapsed, drain)
	}
	if err := wait(); err != nil {
		t.Fatalf("helper did not exit gracefully: %v", err)
	}
	if signals := readSupervisorSignals(t, signalPath); strings.TrimSpace(signals) != "" {
		t.Fatalf("graceful HTTP shutdown should not signal helper, got %q", signals)
	}
	pidPath, err := daemon.PIDPath(baseURL)
	if err != nil {
		t.Fatalf("pid path: %v", err)
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatalf("pid file still exists or unexpected stat error: %v", err)
	}
}

func TestStopWithPolicySendsSIGTERMAfterGrace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}

	signalPath := filepath.Join(t.TempDir(), "signals.log")
	cmd := exec.Command(os.Args[0], "-test.run", "TestSupervisorSignalHelperProcess")
	cmd.Env = append(os.Environ(), "WHISK_SUPERVISOR_SIGNAL_HELPER_PROCESS=1", "WHISK_SUPERVISOR_HELPER_SIGNAL_PATH="+signalPath)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start signal helper: %v", err)
	}
	wait := waitForSupervisorProcess(t, cmd)

	baseURL := "http://127.0.0.1:19991"
	writeSupervisorPID(t, baseURL, cmd.Process.Pid)

	policy := daemon.DefaultStopPolicy()
	policy.ProcessExitGrace = 50 * time.Millisecond
	policy.SignalGrace = time.Second
	policy.HealthDownGrace = 50 * time.Millisecond
	policy.PollInterval = 10 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := daemon.StopWithPolicy(ctx, baseURL, policy); err != nil {
		t.Fatalf("stop pid-backed daemon: %v", err)
	}
	if err := wait(); err != nil {
		t.Fatalf("helper did not exit after SIGTERM: %v", err)
	}
	if signals := readSupervisorSignals(t, signalPath); signals != "TERM" {
		t.Fatalf("signals = %q, want TERM", signals)
	}
	pidPath, err := daemon.PIDPath(baseURL)
	if err != nil {
		t.Fatalf("pid path: %v", err)
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatalf("pid file still exists or unexpected stat error: %v", err)
	}
}

func TestSupervisorSlowDrainHelperProcess(t *testing.T) {
	if os.Getenv("WHISK_SUPERVISOR_HELPER_PROCESS") != "1" {
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

	drain, err := time.ParseDuration(os.Getenv("WHISK_SUPERVISOR_HELPER_DRAIN"))
	if err != nil {
		os.Exit(5)
	}
	signalPath := os.Getenv("WHISK_SUPERVISOR_HELPER_SIGNAL_PATH")
	signals := make(chan os.Signal, 8)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		for sig := range signals {
			_ = appendSupervisorSignal(signalPath, sig.String())
		}
	}()

	shutdown := make(chan struct{})
	var shutdownOnce sync.Once
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/v1/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		shutdownOnce.Do(func() { close(shutdown) })
	})
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: time.Second}
	serveErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case <-shutdown:
	case err := <-serveErr:
		if err != nil {
			os.Exit(6)
		}
		os.Exit(7)
	case <-time.After(10 * time.Second):
		os.Exit(8)
	}
	_ = server.Shutdown(context.Background())
	time.Sleep(drain)
	os.Exit(0)
}

func TestSupervisorSignalHelperProcess(t *testing.T) {
	if os.Getenv("WHISK_SUPERVISOR_SIGNAL_HELPER_PROCESS") != "1" {
		return
	}
	signalPath := os.Getenv("WHISK_SUPERVISOR_HELPER_SIGNAL_PATH")
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case sig := <-signals:
		switch sig {
		case os.Interrupt:
			_ = appendSupervisorSignal(signalPath, "INT")
		case syscall.SIGTERM:
			_ = appendSupervisorSignal(signalPath, "TERM")
		default:
			_ = appendSupervisorSignal(signalPath, sig.String())
		}
		os.Exit(0)
	case <-time.After(10 * time.Second):
		os.Exit(9)
	}
}

func freeSupervisorAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return addr
}

func startSlowDrainHelper(t *testing.T, addr string, drain time.Duration, signalPath string) (*exec.Cmd, func() error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run", "TestSupervisorSlowDrainHelperProcess", "--", "daemon", "run", "-addr", addr)
	cmd.Env = append(
		os.Environ(),
		"WHISK_SUPERVISOR_HELPER_PROCESS=1",
		"WHISK_SUPERVISOR_HELPER_DRAIN="+drain.String(),
		"WHISK_SUPERVISOR_HELPER_SIGNAL_PATH="+signalPath,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start slow drain helper: %v", err)
	}
	return cmd, waitForSupervisorProcess(t, cmd)
}

func waitForSupervisorProcess(t *testing.T, cmd *exec.Cmd) func() error {
	t.Helper()
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()
	var waitOnce sync.Once
	var waitErr error
	wait := func() error {
		waitOnce.Do(func() {
			waitErr = <-waitDone
		})
		return waitErr
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = wait()
	})
	return wait
}

func waitForSupervisorHealth(t *testing.T, baseURL string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/v1/health", nil)
		if err != nil {
			t.Fatalf("health request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatalf("helper never became healthy at %s", baseURL)
		}
	}
}

func writeSupervisorPID(t *testing.T, baseURL string, pid int) {
	t.Helper()
	pidPath, err := daemon.PIDPath(baseURL)
	if err != nil {
		t.Fatalf("pid path: %v", err)
	}
	if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", pid)), 0o600); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(pidPath) })
}

func readSupervisorSignals(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatalf("read signals: %v", err)
	}
	return string(data)
}

func appendSupervisorSignal(path string, name string) error {
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(name)
	return err
}

func TestIsManagedReflectsLivePIDFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal-based liveness check is unix-only")
	}

	baseURL := "http://127.0.0.1:19993"
	if daemon.IsManaged(baseURL) {
		t.Fatalf("expected not managed with no pid file")
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

	pidPath, err := daemon.PIDPath(baseURL)
	if err != nil {
		t.Fatalf("pid path: %v", err)
	}
	if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o600); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(pidPath) })

	if !daemon.IsManaged(baseURL) {
		t.Fatalf("expected managed while pid file names a live process")
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill process: %v", err)
	}
	_, _ = cmd.Process.Wait()
	if daemon.IsManaged(baseURL) {
		t.Fatalf("expected not managed after process exits")
	}
}

func TestStopReturnsNilWhenDaemonAlreadyDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := daemon.Stop(ctx, "http://127.0.0.1:19994"); err != nil {
		t.Fatalf("stop already-down daemon: %v", err)
	}
}

func TestSupervisorRejectsInvalidDaemonURLAndPIDFiles(t *testing.T) {
	if _, err := daemon.PIDPath("://bad-url"); err == nil {
		t.Fatalf("expected invalid pid path URL error")
	}
	if _, err := daemon.PIDPath("http:///missing-host"); err == nil {
		t.Fatalf("expected missing host error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := daemon.Stop(ctx, "http:///missing-host"); err == nil {
		t.Fatalf("expected stop URL error")
	}
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

func TestDaemonPathIgnoresWorkingDirectoryBinCandidate(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("WHISKD_PATH", "")

	workDir := t.TempDir()
	binDir := filepath.Join(workDir, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatalf("make bin dir: %v", err)
	}
	cwdHelper := filepath.Join(binDir, "whisk")
	if err := os.WriteFile(cwdHelper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write cwd helper: %v", err)
	}
	t.Chdir(workDir)

	appDir := t.TempDir()
	appExecutable := filepath.Join(appDir, "whisk-app")
	if err := os.WriteFile(appExecutable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write app executable: %v", err)
	}

	if path, err := daemon.DaemonPathForTest(appExecutable); err == nil {
		t.Fatalf("expected cwd-relative bin/whisk to be ignored, got %q", path)
	}
}

func TestDaemonPathCanFindWhiskOnPATH(t *testing.T) {
	t.Setenv("WHISKD_PATH", "")

	pathDir := t.TempDir()
	helperName := "whisk"
	if runtime.GOOS == "windows" {
		helperName = "whisk.exe"
	}
	helper := filepath.Join(pathDir, helperName)
	if err := os.WriteFile(helper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write path helper: %v", err)
	}
	t.Setenv("PATH", pathDir)

	appDir := t.TempDir()
	appExecutable := filepath.Join(appDir, "whisk-app")
	if err := os.WriteFile(appExecutable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write app executable: %v", err)
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
