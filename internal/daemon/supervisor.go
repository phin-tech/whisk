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

const (
	daemonStateVersion          = 1
	compatibilityProbeTimeout   = 250 * time.Millisecond
	compatibilityRetryWindow    = 2 * time.Second
	compatibilityInitialBackoff = 50 * time.Millisecond
	compatibilityMaxBackoff     = 250 * time.Millisecond
)

type daemonStateFile struct {
	Version          int    `json:"version"`
	PID              int    `json:"pid"`
	ProcessStartTime string `json:"processStartTime"`
	ListenAddress    string `json:"listenAddress"`
	APIVersion       int    `json:"apiVersion"`
	BinaryPath       string `json:"binaryPath"`
}

type compatibilityDecision int

const (
	compatibilityUnknown compatibilityDecision = iota
	compatibilityCompatible
	compatibilityIncompatible
)

// StopPolicy holds every timing used by the daemon stop escalation ladder.
type StopPolicy struct {
	ShutdownRequestTimeout time.Duration
	HealthCheckTimeout     time.Duration
	HealthDownGrace        time.Duration
	ProcessExitGrace       time.Duration
	SignalGrace            time.Duration
	KillGrace              time.Duration
	PollInterval           time.Duration
}

// DefaultStopPolicy is intentionally generous enough for normal daemon drains: HTTP shutdown
// should get the first chance to flush PTYs, transcripts, and sqlite before signals are used.
func DefaultStopPolicy() StopPolicy {
	return StopPolicy{
		ShutdownRequestTimeout: 500 * time.Millisecond,
		HealthCheckTimeout:     250 * time.Millisecond,
		HealthDownGrace:        2 * time.Second,
		ProcessExitGrace:       10 * time.Second,
		SignalGrace:            2 * time.Second,
		KillGrace:              time.Second,
		PollInterval:           50 * time.Millisecond,
	}
}

// DefaultControlTimeout gives UI/CLI callers a single timeout large enough to observe the default
// stop policy before their parent context cancels the ladder.
func DefaultControlTimeout() time.Duration {
	policy := DefaultStopPolicy().normalized()
	return policy.ShutdownRequestTimeout +
		policy.HealthDownGrace +
		policy.ProcessExitGrace +
		policy.SignalGrace +
		policy.KillGrace +
		time.Second
}

func (policy StopPolicy) normalized() StopPolicy {
	defaults := DefaultStopPolicy()
	if policy.ShutdownRequestTimeout <= 0 {
		policy.ShutdownRequestTimeout = defaults.ShutdownRequestTimeout
	}
	if policy.HealthCheckTimeout <= 0 {
		policy.HealthCheckTimeout = defaults.HealthCheckTimeout
	}
	if policy.HealthDownGrace <= 0 {
		policy.HealthDownGrace = defaults.HealthDownGrace
	}
	if policy.ProcessExitGrace <= 0 {
		policy.ProcessExitGrace = defaults.ProcessExitGrace
	}
	if policy.SignalGrace <= 0 {
		policy.SignalGrace = defaults.SignalGrace
	}
	if policy.KillGrace <= 0 {
		policy.KillGrace = defaults.KillGrace
	}
	if policy.PollInterval <= 0 {
		policy.PollInterval = defaults.PollInterval
	}
	return policy
}

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
func Ensure(ctx context.Context, baseURL string) (started bool, err error) {
	lock, err := lockDaemonState(ctx, baseURL)
	if err != nil {
		return false, err
	}
	defer lock.Close()
	return ensureLocked(ctx, baseURL)
}

func ensureLocked(ctx context.Context, baseURL string) (started bool, err error) {
	daemonClient := client.NewHTTP(baseURL, nil)
	if healthCheck(ctx, daemonClient) == nil {
		decision, compatibilityErr := compatibilityCheckWithRetry(ctx, daemonClient)
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
		if err := verifyReplacementDaemon(ctx, path); err != nil {
			return false, fmt.Errorf("replacement whiskd at %s failed compatibility verification; leaving existing daemon running: %w", path, err)
		}
		log.Printf("whiskd at %s is incompatible (%v); shutting it down", baseURL, compatibilityErr)
		if err := stopWithPolicyLocked(ctx, baseURL, DefaultStopPolicy()); err != nil {
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

	if _, err := startDaemonAndWait(ctx, baseURL, addr, binaryPath, daemonStartOptions{writeState: true, binaryPath: binaryPath}); err != nil {
		return false, err
	}
	return true, nil
}

func verifyReplacementDaemon(ctx context.Context, path string) error {
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

	probeCtx, cancel := context.WithTimeout(ctx, compatibilityRetryWindow)
	defer cancel()
	proc, err := startDaemonAndWait(probeCtx, "http://"+addr, addr, path, daemonStartOptions{
		env:   map[string]string{"XDG_CONFIG_HOME": configDir},
		label: "replacement probe whisk daemon",
	})
	if err != nil {
		return err
	}
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	if err := stopSpawnedDaemon(stopCtx, "http://"+addr, proc, false); err != nil {
		return fmt.Errorf("stop replacement probe whiskd: %w", err)
	}
	return nil
}

func startDaemonAndWait(ctx context.Context, baseURL, addr, path string, opts daemonStartOptions) (*spawnedDaemon, error) {
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

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		decision, err := compatibilityProbe(ctx, daemonClient, compatibilityProbeTimeout)
		if decision == compatibilityCompatible {
			if proc.stderr != nil {
				proc.stderr.StopRecording()
			}
			log.Printf("whiskd ready at %s", baseURL)
			return proc, nil
		}
		if decision == compatibilityIncompatible {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = stopSpawnedDaemon(cleanupCtx, baseURL, proc, opts.writeState)
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
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = stopSpawnedDaemon(cleanupCtx, baseURL, proc, opts.writeState)
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

func stopSpawnedDaemon(ctx context.Context, baseURL string, proc *spawnedDaemon, removeState bool) error {
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
	grace := time.NewTimer(500 * time.Millisecond)
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
	case <-time.After(500 * time.Millisecond):
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
func Stop(ctx context.Context, baseURL string) error {
	return StopWithPolicy(ctx, baseURL, DefaultStopPolicy())
}

// StopWithPolicy applies the single daemon stop ladder: request HTTP shutdown, wait for health to
// go down, wait for the recorded process to exit, escalate to SIGTERM, and use SIGKILL only last.
func StopWithPolicy(ctx context.Context, baseURL string, policy StopPolicy) error {
	lock, err := lockDaemonState(ctx, baseURL)
	if err != nil {
		return err
	}
	defer lock.Close()
	return stopWithPolicyLocked(ctx, baseURL, policy)
}

func stopWithPolicyLocked(ctx context.Context, baseURL string, policy StopPolicy) error {
	policy = policy.normalized()
	daemonClient := client.NewHTTP(baseURL, nil)

	state, stateVerified, stateErr := verifiedStateFile(baseURL, true)
	pid := state.PID
	pidKnown := stateVerified && pid > 0
	if stateErr != nil && !errors.Is(stateErr, os.ErrNotExist) {
		if _, pathErr := StatePath(baseURL); pathErr != nil {
			return pathErr
		}
		log.Printf("read whiskd state file for %s: %v", baseURL, stateErr)
	}

	healthy := healthCheckWithPolicy(ctx, daemonClient, policy) == nil
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

// IsManaged reports whether the daemon at baseURL was started by this machine's whisk app, i.e. a
// state file exists and matches a live process. Used to distinguish a daemon the app owns from one
// a developer started independently.
func IsManaged(baseURL string) bool {
	_, ok, _ := verifiedStateFile(baseURL, false)
	return ok
}

func waitForProcessExitWithPolicy(ctx context.Context, pid int, grace time.Duration, policy StopPolicy) error {
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

func StatePath(baseURL string) (string, error) {
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
	statePath, err := StatePath(baseURL)
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
	statePath, err := StatePath(baseURL)
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
	statePath, err := StatePath(baseURL)
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
	statePath, err := StatePath(baseURL)
	if err == nil {
		_ = os.Remove(statePath)
	}
}

func healthCheck(ctx context.Context, daemonClient *client.HTTPClient) error {
	return healthCheckWithPolicy(ctx, daemonClient, DefaultStopPolicy())
}

func healthCheckWithPolicy(ctx context.Context, daemonClient *client.HTTPClient, policy StopPolicy) error {
	checkCtx, cancel := context.WithTimeout(ctx, policy.HealthCheckTimeout)
	defer cancel()
	return daemonClient.Health(checkCtx)
}

func compatibilityProbe(ctx context.Context, daemonClient *client.HTTPClient, timeout time.Duration) (compatibilityDecision, error) {
	if err := healthCheck(ctx, daemonClient); err != nil {
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

func compatibilityCheckWithRetry(ctx context.Context, daemonClient *client.HTTPClient) (compatibilityDecision, error) {
	retryCtx, cancel := context.WithTimeout(ctx, compatibilityRetryWindow)
	defer cancel()

	probeTimeout := compatibilityProbeTimeout
	backoff := compatibilityInitialBackoff
	var lastErr error
	for {
		decision, err := compatibilityProbe(retryCtx, daemonClient, probeTimeout)
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

		probeTimeout = compatibilityRetryWindow
		backoff *= 2
		if backoff > compatibilityMaxBackoff {
			backoff = compatibilityMaxBackoff
		}
	}
}

func shutdownExistingWithPolicy(ctx context.Context, baseURL string, policy StopPolicy) error {
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

func waitUntilDownWithPolicy(ctx context.Context, daemonClient *client.HTTPClient, policy StopPolicy) error {
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
		if healthCheckWithPolicy(ctx, daemonClient, policy) != nil {
			return nil
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			if healthCheckWithPolicy(ctx, daemonClient, policy) != nil {
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
