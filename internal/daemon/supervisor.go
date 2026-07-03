package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/protocol"
)

var (
	errDaemonStillHealthy = errors.New("daemon still answers health checks")
	errProcessStillAlive  = errors.New("process still running")
)

const daemonStateVersion = 1

type daemonStateFile struct {
	Version          int    `json:"version"`
	PID              int    `json:"pid"`
	ProcessStartTime string `json:"processStartTime"`
	ListenAddress    string `json:"listenAddress"`
	APIVersion       int    `json:"apiVersion"`
	BinaryPath       string `json:"binaryPath"`
}

// Policy holds every timing knob used by daemon supervision. The zero value uses documented
// defaults chosen for normal desktop/CLI control paths.
type Policy struct {
	// ControlTimeout defaults to the sum of the stop ladder plus one second.
	ControlTimeout time.Duration
	// HealthCheckTimeout defaults to 250ms.
	HealthCheckTimeout time.Duration
	// CompatibilityProbeTimeout defaults to 250ms.
	CompatibilityProbeTimeout time.Duration
	// CompatibilityRetryWindow defaults to 2s.
	CompatibilityRetryWindow time.Duration
	// CompatibilityInitialBackoff defaults to 50ms.
	CompatibilityInitialBackoff time.Duration
	// CompatibilityMaxBackoff defaults to 250ms.
	CompatibilityMaxBackoff time.Duration
	// ShutdownRequestTimeout defaults to 500ms.
	ShutdownRequestTimeout time.Duration
	// HealthDownGrace defaults to 2s.
	HealthDownGrace time.Duration
	// ProcessExitGrace defaults to 10s.
	ProcessExitGrace time.Duration
	// SignalGrace defaults to 2s.
	SignalGrace time.Duration
	// KillGrace defaults to 1s.
	KillGrace time.Duration
	// SpawnCleanupTimeout defaults to 2s.
	SpawnCleanupTimeout time.Duration
	// SpawnInterruptGrace defaults to 500ms.
	SpawnInterruptGrace time.Duration
	// SpawnKillGrace defaults to 500ms.
	SpawnKillGrace time.Duration
	// PollInterval defaults to 50ms.
	PollInterval time.Duration
}

func selectedPolicy(policies []Policy) Policy {
	if len(policies) == 0 {
		return Policy{}.normalized()
	}
	return policies[0].normalized()
}

func (policy Policy) normalized() Policy {
	if policy.HealthCheckTimeout <= 0 {
		policy.HealthCheckTimeout = 250 * time.Millisecond
	}
	if policy.CompatibilityProbeTimeout <= 0 {
		policy.CompatibilityProbeTimeout = 250 * time.Millisecond
	}
	if policy.CompatibilityRetryWindow <= 0 {
		policy.CompatibilityRetryWindow = 2 * time.Second
	}
	if policy.CompatibilityInitialBackoff <= 0 {
		policy.CompatibilityInitialBackoff = 50 * time.Millisecond
	}
	if policy.CompatibilityMaxBackoff <= 0 {
		policy.CompatibilityMaxBackoff = 250 * time.Millisecond
	}
	if policy.ShutdownRequestTimeout <= 0 {
		policy.ShutdownRequestTimeout = 500 * time.Millisecond
	}
	if policy.HealthDownGrace <= 0 {
		policy.HealthDownGrace = 2 * time.Second
	}
	if policy.ProcessExitGrace <= 0 {
		policy.ProcessExitGrace = 10 * time.Second
	}
	if policy.SignalGrace <= 0 {
		policy.SignalGrace = 2 * time.Second
	}
	if policy.KillGrace <= 0 {
		policy.KillGrace = time.Second
	}
	if policy.SpawnCleanupTimeout <= 0 {
		policy.SpawnCleanupTimeout = 2 * time.Second
	}
	if policy.SpawnInterruptGrace <= 0 {
		policy.SpawnInterruptGrace = 500 * time.Millisecond
	}
	if policy.SpawnKillGrace <= 0 {
		policy.SpawnKillGrace = 500 * time.Millisecond
	}
	if policy.PollInterval <= 0 {
		policy.PollInterval = 50 * time.Millisecond
	}
	if policy.ControlTimeout <= 0 {
		policy.ControlTimeout = policy.ShutdownRequestTimeout +
			policy.HealthDownGrace +
			policy.ProcessExitGrace +
			policy.SignalGrace +
			policy.KillGrace +
			time.Second
	}
	return policy
}

func (policy Policy) withControlTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok || policy.ControlTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, policy.ControlTimeout)
}

type compatibilityDecision int

const (
	compatibilityUnknown compatibilityDecision = iota
	compatibilityCompatible
	compatibilityIncompatible
)

type daemonStartOptions struct {
	writeState bool
	binaryPath string
	env        map[string]string
	label      string
}

type spawnedDaemon struct {
	cmd    *exec.Cmd
	waitCh chan error
	stderr *limitedCapture
}

// Ensure makes sure a compatible daemon is reachable at baseURL, starting one if needed.
// It reports whether it started a new daemon (started == true) versus adopting one that was
// already running (started == false). Callers use this to decide ownership: a daemon the app
// started itself should be stopped when the app exits, while one started elsewhere (e.g. a
// developer's `whisk daemon run`) must be left alone.
func Ensure(ctx context.Context, baseURL string, policies ...Policy) (started bool, err error) {
	policy := selectedPolicy(policies)
	ctx, cancel := policy.withControlTimeout(ctx)
	defer cancel()
	lock, err := lockDaemonState(ctx, baseURL)
	if err != nil {
		return false, err
	}
	defer lock.Close()
	return ensureLocked(ctx, baseURL, policy)
}

func ensureLocked(ctx context.Context, baseURL string, policy Policy) (started bool, err error) {
	daemonClient := client.NewHTTP(baseURL, nil)
	if healthCheck(ctx, daemonClient, policy) == nil {
		decision, compatibilityErr := compatibilityCheckWithRetry(ctx, daemonClient, policy)
		if decision == compatibilityCompatible {
			return false, nil
		}
		if decision == compatibilityUnknown {
			return false, fmt.Errorf("check whiskd compatibility: %w", compatibilityErr)
		}
		path, err := daemonPath()
		if err != nil {
			return false, fmt.Errorf("find replacement whiskd: %w", err)
		}
		if err := verifyReplacementDaemon(ctx, path, policy); err != nil {
			return false, fmt.Errorf("replacement whiskd at %s failed compatibility verification; leaving existing daemon running: %w", path, err)
		}
		log.Printf("whiskd at %s is incompatible (%v); shutting it down", baseURL, compatibilityErr)
		if err := stopLocked(ctx, baseURL, policy); err != nil {
			return false, fmt.Errorf("stop incompatible whiskd: %w", err)
		}
	}

	addr, err := addrFromURL(baseURL)
	if err != nil {
		return false, err
	}
	path, err := daemonPath()
	if err != nil {
		return false, err
	}
	binaryPath := path
	if abs, err := filepath.Abs(path); err == nil {
		binaryPath = abs
	}

	if _, err := startDaemonAndWait(ctx, baseURL, addr, binaryPath, policy, daemonStartOptions{writeState: true, binaryPath: binaryPath}); err != nil {
		return false, err
	}
	return true, nil
}

func verifyReplacementDaemon(ctx context.Context, path string, policy Policy) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("reserve probe address: %w", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return fmt.Errorf("release probe address: %w", err)
	}
	configDir, err := os.MkdirTemp("", "whiskd-verify-*")
	if err != nil {
		return fmt.Errorf("create probe config dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(configDir) }()

	probeCtx, cancel := context.WithTimeout(ctx, policy.CompatibilityRetryWindow)
	defer cancel()
	proc, err := startDaemonAndWait(probeCtx, "http://"+addr, addr, path, policy, daemonStartOptions{
		env:   map[string]string{"XDG_CONFIG_HOME": configDir},
		label: "replacement probe whisk daemon",
	})
	if err != nil {
		return err
	}
	stopCtx, stopCancel := context.WithTimeout(context.Background(), policy.SpawnCleanupTimeout)
	defer stopCancel()
	if err := stopSpawnedDaemon(stopCtx, "http://"+addr, proc, false, policy); err != nil {
		return fmt.Errorf("stop replacement probe whiskd: %w", err)
	}
	return nil
}

func startDaemonAndWait(ctx context.Context, baseURL, addr, path string, policy Policy, opts daemonStartOptions) (*spawnedDaemon, error) {
	label := opts.label
	if label == "" {
		label = "whisk daemon"
	}
	log.Printf("starting %s at %s from %s", label, baseURL, path)
	proc, err := startDaemonProcess(baseURL, addr, path, opts)
	if err != nil {
		return nil, err
	}
	daemonClient := client.NewHTTP(baseURL, nil)

	ticker := time.NewTicker(policy.PollInterval)
	defer ticker.Stop()
	for {
		decision, err := compatibilityProbe(ctx, daemonClient, policy, policy.CompatibilityProbeTimeout)
		if decision == compatibilityCompatible {
			if proc.stderr != nil {
				proc.stderr.StopRecording()
			}
			log.Printf("whiskd ready at %s", baseURL)
			return proc, nil
		}
		if decision == compatibilityIncompatible {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), policy.SpawnCleanupTimeout)
			_ = stopSpawnedDaemon(cleanupCtx, baseURL, proc, opts.writeState, policy)
			cancel()
			return nil, fmt.Errorf("started whiskd is incompatible: %w%s", err, proc.diagnostics(baseURL))
		}
		select {
		case waitErr := <-proc.waitCh:
			if opts.writeState {
				removeStateFile(baseURL)
			}
			return nil, proc.exitBeforeReadyError(baseURL, waitErr)
		case <-ticker.C:
		case <-ctx.Done():
			cleanupCtx, cancel := context.WithTimeout(context.Background(), policy.SpawnCleanupTimeout)
			_ = stopSpawnedDaemon(cleanupCtx, baseURL, proc, opts.writeState, policy)
			cancel()
			return nil, fmt.Errorf("wait for whiskd: %w%s", ctx.Err(), proc.diagnostics(baseURL))
		}
	}
}

func startDaemonProcess(baseURL, addr, path string, opts daemonStartOptions) (*spawnedDaemon, error) {
	cmd := exec.CommandContext(context.Background(), path, "daemon", "run", "-addr", addr)
	if len(opts.env) > 0 {
		cmd.Env = environWithOverrides(opts.env)
	}
	stderrCapture := newLimitedCapture(supervisorStderrCaptureBytes)
	cmd.Stderr = stderrCapture
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start whiskd: %w%s", err, daemonStartDiagnostics(baseURL, stderrCapture))
	}
	if opts.writeState {
		binaryPath := opts.binaryPath
		if binaryPath == "" {
			binaryPath = path
		}
		startTime, err := processStartTime(cmd.Process.Pid)
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return nil, fmt.Errorf("read whiskd process start time: %w%s", err, daemonStartDiagnostics(baseURL, stderrCapture))
		}
		if err := writeStateFile(baseURL, daemonStateFile{
			Version:          daemonStateVersion,
			PID:              cmd.Process.Pid,
			ProcessStartTime: startTime,
			ListenAddress:    addr,
			APIVersion:       protocol.DaemonAPIVersion,
			BinaryPath:       binaryPath,
		}); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return nil, fmt.Errorf("write whiskd state file: %w%s", err, daemonStartDiagnostics(baseURL, stderrCapture))
		}
	}
	proc := &spawnedDaemon{
		cmd:    cmd,
		waitCh: make(chan error, 1),
		stderr: stderrCapture,
	}
	go func() {
		err := cmd.Wait()
		proc.waitCh <- err
		if err != nil {
			log.Printf("whiskd exited: %v", err)
		}
	}()
	return proc, nil
}

func stopSpawnedDaemon(ctx context.Context, baseURL string, proc *spawnedDaemon, removeState bool, policy Policy) error {
	if proc == nil {
		return nil
	}
	defer func() {
		if removeState {
			removeStateFile(baseURL)
		}
	}()
	select {
	case err := <-proc.waitCh:
		return err
	default:
	}

	if proc.cmd.Process != nil {
		if err := proc.cmd.Process.Signal(os.Interrupt); err != nil {
			_ = proc.cmd.Process.Kill()
		}
	}
	grace := time.NewTimer(policy.SpawnInterruptGrace)
	defer grace.Stop()
	select {
	case err := <-proc.waitCh:
		return err
	case <-grace.C:
	case <-ctx.Done():
	}
	if proc.cmd.Process != nil {
		_ = proc.cmd.Process.Kill()
	}
	select {
	case <-proc.waitCh:
		return nil
	case <-ctx.Done():
		if proc.cmd.Process != nil {
			_ = proc.cmd.Process.Kill()
		}
		return ctx.Err()
	case <-time.After(policy.SpawnKillGrace):
		return ctx.Err()
	}
}

func environWithOverrides(overrides map[string]string) []string {
	env := os.Environ()
	out := env[:0]
	for _, entry := range env {
		key := entry
		if idx := strings.IndexByte(entry, '='); idx >= 0 {
			key = entry[:idx]
		}
		if _, ok := overrides[key]; ok {
			continue
		}
		out = append(out, entry)
	}
	for key, value := range overrides {
		out = append(out, key+"="+value)
	}
	return out
}

func (p *spawnedDaemon) exitBeforeReadyError(baseURL string, waitErr error) error {
	return fmt.Errorf("started whiskd exited before readiness: %s%s", exitStatusText(waitErr), p.diagnostics(baseURL))
}

func exitStatusText(err error) string {
	if err == nil {
		return "exit status 0"
	}
	return err.Error()
}

func (p *spawnedDaemon) diagnostics(baseURL string) string {
	if p == nil {
		return ""
	}
	return daemonStartDiagnostics(baseURL, p.stderr)
}

// Stop shuts down the daemon at baseURL whether or not this process started it.
func Stop(ctx context.Context, baseURL string, policies ...Policy) error {
	policy := selectedPolicy(policies)
	ctx, cancel := policy.withControlTimeout(ctx)
	defer cancel()
	lock, err := lockDaemonState(ctx, baseURL)
	if err != nil {
		return err
	}
	defer lock.Close()
	return stopLocked(ctx, baseURL, policy)
}

type StatusReport struct {
	Running    bool   `json:"running"`
	Address    string `json:"address"`
	Managed    bool   `json:"managed"`
	APIVersion int    `json:"apiVersion"`
	GitSHA     string `json:"gitSha"`
	Version    string `json:"version"`
	Dirty      bool   `json:"dirty"`
	Error      string `json:"error"`
}

// Status reports the daemon's current health, ownership, compatibility, and build metadata.
func Status(ctx context.Context, baseURL string, policies ...Policy) StatusReport {
	policy := selectedPolicy(policies)
	status := StatusReport{Address: strings.TrimRight(baseURL, "/")}
	if _, managed, err := verifiedStateFile(baseURL, false); err == nil && managed {
		status.Managed = true
	}

	daemonClient := client.NewHTTP(baseURL, nil)
	if err := healthCheck(ctx, daemonClient, policy); err != nil {
		status.Error = err.Error()
		return status
	}
	status.Running = true

	checkCtx, cancel := context.WithTimeout(ctx, policy.CompatibilityProbeTimeout)
	defer cancel()
	compatibility, err := daemonClient.Compatibility(checkCtx)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.APIVersion = compatibility.APIVersion
	status.GitSHA = compatibility.GitSHA
	status.Version = compatibility.Version
	status.Dirty = compatibility.Dirty
	if compatibility.APIVersion != protocol.DaemonAPIVersion {
		status.Error = fmt.Sprintf("daemon api version %d does not match required %d", compatibility.APIVersion, protocol.DaemonAPIVersion)
	}
	return status
}

func stopLocked(ctx context.Context, baseURL string, policy Policy) error {
	daemonClient := client.NewHTTP(baseURL, nil)

	state, stateVerified, stateErr := verifiedStateFile(baseURL, true)
	pid := state.PID
	pidKnown := stateVerified && pid > 0
	if stateErr != nil && !errors.Is(stateErr, os.ErrNotExist) {
		if _, pathErr := statePath(baseURL); pathErr != nil {
			return pathErr
		}
		log.Printf("read whiskd state file for %s: %v", baseURL, stateErr)
	}

	healthy := healthCheck(ctx, daemonClient, policy) == nil
	waitForGracefulProcessExit := !healthy
	var shutdownErr error
	if healthy {
		if err := shutdownExistingWithPolicy(ctx, baseURL, policy); err != nil {
			shutdownErr = err
			log.Printf("shutdown whiskd at %s: %v", baseURL, err)
			if !pidKnown {
				return fmt.Errorf("stop whiskd: %w", err)
			}
		}
		if err := waitUntilDownWithPolicy(ctx, daemonClient, policy); err != nil {
			if !errors.Is(err, errDaemonStillHealthy) {
				return fmt.Errorf("wait for whiskd health down: %w", err)
			}
			log.Printf("whiskd at %s still answers health after shutdown request", baseURL)
			if !pidKnown {
				if shutdownErr != nil {
					return fmt.Errorf("stop whiskd: %w", shutdownErr)
				}
				return fmt.Errorf("stop whiskd: %w", err)
			}
		} else {
			waitForGracefulProcessExit = true
		}
	}

	if !pidKnown {
		return nil
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find whiskd process %d: %w", pid, err)
	}
	if waitForGracefulProcessExit {
		if err := waitForProcessExitWithPolicy(ctx, pid, policy.ProcessExitGrace, policy); err == nil {
			removeStateFile(baseURL)
			return nil
		} else if !errors.Is(err, errProcessStillAlive) {
			return fmt.Errorf("wait for whiskd process exit: %w", err)
		}
	} else if !processAlive(pid) {
		removeStateFile(baseURL)
		return nil
	}

	if waitForGracefulProcessExit {
		log.Printf("whiskd process %d still running after %s; sending SIGTERM", pid, policy.ProcessExitGrace)
	} else {
		log.Printf("whiskd process %d still running while health is up; sending SIGTERM", pid)
	}
	if err := signalProcessTerm(process); err != nil && processAlive(pid) {
		log.Printf("SIGTERM whiskd process %d: %v", pid, err)
	}
	if err := waitForProcessExitWithPolicy(ctx, pid, policy.SignalGrace, policy); err == nil {
		removeStateFile(baseURL)
		return nil
	} else if !errors.Is(err, errProcessStillAlive) {
		return fmt.Errorf("wait for whiskd process after SIGTERM: %w", err)
	}

	log.Printf("whiskd process %d still running after SIGTERM; sending SIGKILL", pid)
	if err := process.Kill(); err != nil && processAlive(pid) {
		return fmt.Errorf("SIGKILL whiskd process %d: %w", pid, err)
	}
	if err := waitForProcessExitWithPolicy(ctx, pid, policy.KillGrace, policy); err != nil && !errors.Is(err, errProcessStillAlive) {
		return fmt.Errorf("wait for whiskd process after SIGKILL: %w", err)
	}
	removeStateFile(baseURL)
	return nil
}

func waitForProcessExitWithPolicy(ctx context.Context, pid int, grace time.Duration, policy Policy) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !processAlive(pid) {
		return nil
	}
	timer := time.NewTimer(grace)
	defer timer.Stop()
	ticker := time.NewTicker(policy.PollInterval)
	defer ticker.Stop()
	for {
		if !processAlive(pid) {
			return nil
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			if !processAlive(pid) {
				return nil
			}
			return errProcessStillAlive
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func statePath(baseURL string) (string, error) {
	addr, err := addrFromURL(baseURL)
	if err != nil {
		return "", err
	}
	root, err := daemonStateRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "daemon-"+sanitizeDaemonAddr(addr)+".json"), nil
}

func lockDaemonState(ctx context.Context, baseURL string) (*stateFileLock, error) {
	statePath, err := statePath(baseURL)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return nil, err
	}
	return lockFile(ctx, statePath+".lock")
}

func daemonStateRoot() (string, error) {
	if dir := os.Getenv("WHISK_STATE_DIR"); dir != "" {
		return filepath.Clean(dir), nil
	}
	if runtime.GOOS == "windows" {
		if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
			return filepath.Join(dir, "Whisk"), nil
		}
		if dir := os.Getenv("APPDATA"); dir != "" {
			return filepath.Join(dir, "Whisk"), nil
		}
	}
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "whisk"), nil
	}
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "whisk"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".local", "state", "whisk"), nil
}

func sanitizeDaemonAddr(addr string) string {
	replacer := strings.NewReplacer(":", "_", ".", "_", "[", "", "]", "")
	return replacer.Replace(addr)
}

func readStateFile(baseURL string) (daemonStateFile, error) {
	statePath, err := statePath(baseURL)
	if err != nil {
		return daemonStateFile{}, err
	}
	data, err := os.ReadFile(statePath)
	if err != nil {
		return daemonStateFile{}, err
	}
	var state daemonStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return daemonStateFile{}, err
	}
	if state.Version != daemonStateVersion {
		return daemonStateFile{}, fmt.Errorf("unsupported daemon state version %d", state.Version)
	}
	addr, err := addrFromURL(baseURL)
	if err != nil {
		return daemonStateFile{}, err
	}
	if state.ListenAddress != addr {
		return daemonStateFile{}, fmt.Errorf("daemon state listen address %q does not match %q", state.ListenAddress, addr)
	}
	if state.PID <= 0 {
		return daemonStateFile{}, fmt.Errorf("daemon state pid required")
	}
	if state.ProcessStartTime == "" {
		return daemonStateFile{}, fmt.Errorf("daemon state process start time required")
	}
	return state, nil
}

func verifiedStateFile(baseURL string, cleanupMismatch bool) (daemonStateFile, bool, error) {
	state, err := readStateFile(baseURL)
	if err != nil {
		return daemonStateFile{}, false, err
	}
	liveStartTime, err := processStartTime(state.PID)
	if err != nil || liveStartTime != state.ProcessStartTime {
		if cleanupMismatch {
			removeStateFile(baseURL)
		}
		return state, false, nil
	}
	return state, true, nil
}

func writeStateFile(baseURL string, state daemonStateFile) error {
	statePath, err := statePath(baseURL)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(filepath.Dir(statePath), ".daemon-state-*.json")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempPath, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tempPath, statePath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func removeStateFile(baseURL string) {
	statePath, err := statePath(baseURL)
	if err == nil {
		_ = os.Remove(statePath)
	}
}

func healthCheck(ctx context.Context, daemonClient *client.HTTPClient, policy Policy) error {
	checkCtx, cancel := context.WithTimeout(ctx, policy.HealthCheckTimeout)
	defer cancel()
	return daemonClient.Health(checkCtx)
}

func compatibilityProbe(ctx context.Context, daemonClient *client.HTTPClient, policy Policy, timeout time.Duration) (compatibilityDecision, error) {
	if err := healthCheck(ctx, daemonClient, policy); err != nil {
		return compatibilityUnknown, fmt.Errorf("daemon health check failed before compatibility probe: %w", err)
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	compatibility, err := daemonClient.Compatibility(checkCtx)
	if err != nil {
		return compatibilityUnknown, fmt.Errorf("daemon compatibility probe is unknown: %w", err)
	}
	if compatibility.APIVersion != protocol.DaemonAPIVersion {
		return compatibilityIncompatible, fmt.Errorf("daemon api version %d does not match required %d", compatibility.APIVersion, protocol.DaemonAPIVersion)
	}
	return compatibilityCompatible, nil
}

func compatibilityCheckWithRetry(ctx context.Context, daemonClient *client.HTTPClient, policy Policy) (compatibilityDecision, error) {
	retryCtx, cancel := context.WithTimeout(ctx, policy.CompatibilityRetryWindow)
	defer cancel()

	probeTimeout := policy.CompatibilityProbeTimeout
	backoff := policy.CompatibilityInitialBackoff
	var lastErr error
	for {
		decision, err := compatibilityProbe(retryCtx, daemonClient, policy, probeTimeout)
		if decision == compatibilityCompatible || decision == compatibilityIncompatible {
			return decision, err
		}
		lastErr = err

		timer := time.NewTimer(backoff)
		select {
		case <-timer.C:
		case <-retryCtx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			if lastErr == nil {
				lastErr = retryCtx.Err()
			}
			return compatibilityUnknown, fmt.Errorf("compatibility probe did not complete before deadline: %w", lastErr)
		}

		probeTimeout = policy.CompatibilityRetryWindow
		backoff *= 2
		if backoff > policy.CompatibilityMaxBackoff {
			backoff = policy.CompatibilityMaxBackoff
		}
	}
}

func shutdownExistingWithPolicy(ctx context.Context, baseURL string, policy Policy) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, policy.ShutdownRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(shutdownCtx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/shutdown", nil)
	if err != nil {
		return err
	}
	if err := client.NewHTTP(baseURL, nil).AuthorizeRequest(req); err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("shutdown status: %s", resp.Status)
	}
	return nil
}

func waitUntilDownWithPolicy(ctx context.Context, daemonClient *client.HTTPClient, policy Policy) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(policy.HealthDownGrace)
	defer timer.Stop()
	ticker := time.NewTicker(policy.PollInterval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if healthCheck(ctx, daemonClient, policy) != nil {
			return nil
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			if healthCheck(ctx, daemonClient, policy) != nil {
				return nil
			}
			return errDaemonStillHealthy
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func addrFromURL(baseURL string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("daemon URL requires host: %s", baseURL)
	}
	return parsed.Host, nil
}

func daemonPath() (string, error) {
	return daemonPathForExecutable("")
}

func daemonPathForExecutable(executable string) (string, error) {
	candidates := []string{}
	if path := os.Getenv("WHISKD_PATH"); path != "" {
		candidates = append(candidates, path)
	}
	if executable == "" {
		if path, err := os.Executable(); err == nil {
			executable = path
		}
	}
	if executable != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "whisk"))
	}
	if path, err := exec.LookPath("whisk"); err == nil {
		candidates = append(candidates, path)
	}

	var selfInfo os.FileInfo
	if executable != "" {
		if info, err := os.Stat(executable); err == nil {
			selfInfo = info
		}
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		// Never select the currently running executable as the daemon. macOS filesystems
		// are case-insensitive by default, so a GUI binary ("whisk-app") and a sibling
		// candidate path can resolve to the same file; relaunching ourselves with daemon
		// args would fork-loop instead of starting the real daemon.
		if selfInfo != nil && os.SameFile(selfInfo, info) {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("whisk daemon executable not found; run `task build:daemon` or set WHISKD_PATH to the whisk CLI")
}
