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
	useSupervisorStateDir(t)

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
	statePath, err := daemon.StatePath("http://" + addr)
	if err != nil {
		t.Fatalf("state path: %v", err)
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected daemon state file: %v", err)
	}
}

func TestEnsureRestartsIncompatibleDaemon(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	useSupervisorStateDir(t)

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

func TestWhiskHelperProcess(t *testing.T) {
	if os.Getenv("WHISK_HELPER_PROCESS") != "1" {
		return
	}
	if startPath := os.Getenv("WHISK_HELPER_START_PATH"); startPath != "" {
		_ = appendSupervisorSignal(startPath, "start\n")
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
	shutdown := make(chan struct{})
	var shutdownOnce sync.Once
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
	mux.HandleFunc("/v1/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		shutdownOnce.Do(func() { close(shutdown) })
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
	if os.Getenv("WHISK_HELPER_STAY_ALIVE") == "1" {
		select {
		case <-shutdown:
		case <-time.After(10 * time.Second):
			os.Exit(5)
		}
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
	useSupervisorStateDir(t)

	addr := freeSupervisorAddr(t)
	baseURL := "http://" + addr
	signalPath := filepath.Join(t.TempDir(), "signals.log")
	drain := 5 * time.Second
	cmd, wait := startSlowDrainHelper(t, addr, drain, signalPath)
	waitForSupervisorHealth(t, baseURL)
	writeSupervisorState(t, baseURL, cmd.Process.Pid)

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
	statePath, err := daemon.StatePath(baseURL)
	if err != nil {
		t.Fatalf("state path: %v", err)
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("state file still exists or unexpected stat error: %v", err)
	}
}

func TestStopWithPolicySendsSIGTERMAfterGrace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	useSupervisorStateDir(t)

	signalPath := filepath.Join(t.TempDir(), "signals.log")
	cmd := exec.Command(os.Args[0], "-test.run", "TestSupervisorSignalHelperProcess")
	cmd.Env = append(os.Environ(), "WHISK_SUPERVISOR_SIGNAL_HELPER_PROCESS=1", "WHISK_SUPERVISOR_HELPER_SIGNAL_PATH="+signalPath)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start signal helper: %v", err)
	}
	wait := waitForSupervisorProcess(t, cmd)

	baseURL := "http://127.0.0.1:19991"
	writeSupervisorState(t, baseURL, cmd.Process.Pid)

	policy := daemon.DefaultStopPolicy()
	policy.ProcessExitGrace = 50 * time.Millisecond
	policy.SignalGrace = time.Second
	policy.HealthDownGrace = 50 * time.Millisecond
	policy.PollInterval = 10 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := daemon.StopWithPolicy(ctx, baseURL, policy); err != nil {
		t.Fatalf("stop state-backed daemon: %v", err)
	}
	if err := wait(); err != nil {
		t.Fatalf("helper did not exit after SIGTERM: %v", err)
	}
	if signals := readSupervisorSignals(t, signalPath); signals != "TERM" {
		t.Fatalf("signals = %q, want TERM", signals)
	}
	statePath, err := daemon.StatePath(baseURL)
	if err != nil {
		t.Fatalf("state path: %v", err)
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("state file still exists or unexpected stat error: %v", err)
	}
}

func TestStopIgnoresStateWithMismatchedProcessStartTime(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal-based liveness check is unix-only")
	}
	useSupervisorStateDir(t)

	signalPath := filepath.Join(t.TempDir(), "signals.log")
	cmd := exec.Command(os.Args[0], "-test.run", "TestSupervisorSignalHelperProcess")
	cmd.Env = append(os.Environ(), "WHISK_SUPERVISOR_SIGNAL_HELPER_PROCESS=1", "WHISK_SUPERVISOR_HELPER_SIGNAL_PATH="+signalPath)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start signal helper: %v", err)
	}
	wait := waitForSupervisorProcess(t, cmd)

	baseURL := "http://127.0.0.1:19992"
	actualStart, err := daemon.ProcessStartTimeForTest(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("process start time: %v", err)
	}
	if err := daemon.WriteStateForTest(baseURL, cmd.Process.Pid, "stale-"+actualStart, os.Args[0]); err != nil {
		t.Fatalf("write stale state: %v", err)
	}

	policy := daemon.DefaultStopPolicy()
	policy.ProcessExitGrace = 50 * time.Millisecond
	policy.SignalGrace = 50 * time.Millisecond
	policy.HealthDownGrace = 50 * time.Millisecond
	policy.PollInterval = 10 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := daemon.StopWithPolicy(ctx, baseURL, policy); err != nil {
		t.Fatalf("stop with stale state: %v", err)
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("stale state should not signal helper, process is gone: %v", err)
	}
	if signals := readSupervisorSignals(t, signalPath); signals != "" {
		t.Fatalf("stale state should not signal helper, got %q", signals)
	}
	statePath, err := daemon.StatePath(baseURL)
	if err != nil {
		t.Fatalf("state path: %v", err)
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("stale state file still exists or unexpected stat error: %v", err)
	}
	_ = cmd.Process.Kill()
	if err := wait(); err != nil && !strings.Contains(err.Error(), "signal: killed") {
		t.Fatalf("cleanup wait: %v", err)
	}
}

func TestEnsureConcurrentCallsSingleFlightThroughStateLock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	useSupervisorStateDir(t)

	addr := freeSupervisorAddr(t)
	baseURL := "http://" + addr
	startPath := filepath.Join(t.TempDir(), "starts.log")
	t.Setenv("WHISKD_PATH", writeWhiskHelper(t))
	t.Setenv("WHISK_HELPER_STAY_ALIVE", "1")
	t.Setenv("WHISK_HELPER_START_PATH", startPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	startedResults := make(chan bool, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			started, err := daemon.Ensure(ctx, baseURL)
			startedResults <- started
			errs <- err
		}()
	}
	wg.Wait()
	close(startedResults)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("ensure: %v", err)
		}
	}
	startedCount := 0
	for started := range startedResults {
		if started {
			startedCount++
		}
	}
	if startedCount != 1 {
		t.Fatalf("started count = %d, want 1", startedCount)
	}
	if got := strings.Count(readSupervisorSignals(t, startPath), "start\n"); got != 1 {
		t.Fatalf("helper starts = %d, want 1", got)
	}
	if !daemon.IsManaged(baseURL) {
		t.Fatalf("expected state file to describe a live managed daemon")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), daemon.DefaultControlTimeout())
	defer stopCancel()
	if err := daemon.Stop(stopCtx, baseURL); err != nil {
		t.Fatalf("cleanup stop: %v", err)
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

func useSupervisorStateDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "state")
	t.Setenv("WHISK_STATE_DIR", dir)
	return dir
}

func writeSupervisorState(t *testing.T, baseURL string, pid int) {
	t.Helper()
	startTime, err := daemon.ProcessStartTimeForTest(pid)
	if err != nil {
		t.Fatalf("process start time: %v", err)
	}
	if err := daemon.WriteStateForTest(baseURL, pid, startTime, os.Args[0]); err != nil {
		t.Fatalf("write state: %v", err)
	}
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

func TestIsManagedReflectsLiveStateFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal-based liveness check is unix-only")
	}
	useSupervisorStateDir(t)

	baseURL := "http://127.0.0.1:19993"
	if daemon.IsManaged(baseURL) {
		t.Fatalf("expected not managed with no state file")
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

	writeSupervisorState(t, baseURL, cmd.Process.Pid)

	if !daemon.IsManaged(baseURL) {
		t.Fatalf("expected managed while state file matches a live process")
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
	useSupervisorStateDir(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := daemon.Stop(ctx, "http://127.0.0.1:19994"); err != nil {
		t.Fatalf("stop already-down daemon: %v", err)
	}
}

func TestStatePathUsesConfiguredStateDirNotSystemTemp(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	tempDir := filepath.Join(t.TempDir(), "tmp")
	t.Setenv("WHISK_STATE_DIR", stateDir)
	t.Setenv("TMPDIR", tempDir)

	path, err := daemon.StatePath("http://127.0.0.1:19995")
	if err != nil {
		t.Fatalf("state path: %v", err)
	}
	if !strings.HasPrefix(path, stateDir+string(filepath.Separator)) {
		t.Fatalf("state path = %q, want under %q", path, stateDir)
	}
	if strings.HasPrefix(path, tempDir+string(filepath.Separator)) {
		t.Fatalf("state path = %q, should not use system temp dir %q", path, tempDir)
	}
}

func TestSupervisorRejectsInvalidDaemonURLAndStateFiles(t *testing.T) {
	useSupervisorStateDir(t)

	if _, err := daemon.StatePath("://bad-url"); err == nil {
		t.Fatalf("expected invalid state path URL error")
	}
	if _, err := daemon.StatePath("http:///missing-host"); err == nil {
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
