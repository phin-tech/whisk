package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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
	compatibilityProbeTimeout   = 250 * time.Millisecond
	compatibilityRetryWindow    = 2 * time.Second
	compatibilityInitialBackoff = 50 * time.Millisecond
	compatibilityMaxBackoff     = 250 * time.Millisecond
)

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

// Ensure makes sure a compatible daemon is reachable at baseURL, starting one if needed.
// It reports whether it started a new daemon (started == true) versus adopting one that was
// already running (started == false). Callers use this to decide ownership: a daemon the app
// started itself should be stopped when the app exits, while one started elsewhere (e.g. a
// developer's `whisk daemon run`) must be left alone.
func Ensure(ctx context.Context, baseURL string) (started bool, err error) {
	daemonClient := client.NewHTTP(baseURL, nil)
	if healthCheck(ctx, daemonClient) == nil {
		decision, compatibilityErr := compatibilityCheckWithRetry(ctx, daemonClient)
		if decision == compatibilityCompatible {
			return false, nil
		}
		if decision == compatibilityUnknown {
			return false, fmt.Errorf("check whiskd compatibility: %w", compatibilityErr)
		}
		log.Printf("whiskd at %s is incompatible (%v); shutting it down", baseURL, compatibilityErr)
		if err := Stop(ctx, baseURL); err != nil {
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

	log.Printf("starting whisk daemon at %s from %s", baseURL, path)
	cmd := exec.CommandContext(context.Background(), path, "daemon", "run", "-addr", addr)
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "whiskd.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return false, fmt.Errorf("open whiskd log: %w", err)
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return false, fmt.Errorf("start whiskd: %w", err)
	}
	if err := writePIDFile(baseURL, cmd.Process.Pid); err != nil {
		log.Printf("write whiskd pid file: %v", err)
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("whiskd exited: %v", err)
		}
	}()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		decision, err := compatibilityProbe(ctx, daemonClient, compatibilityProbeTimeout)
		if decision == compatibilityCompatible {
			log.Printf("whiskd ready at %s", baseURL)
			return true, nil
		}
		if decision == compatibilityIncompatible {
			return false, fmt.Errorf("started whiskd is incompatible: %w", err)
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return false, fmt.Errorf("wait for whiskd: %w", ctx.Err())
		}
	}
}

// Stop shuts down the daemon at baseURL whether or not this process started it.
func Stop(ctx context.Context, baseURL string) error {
	return StopWithPolicy(ctx, baseURL, DefaultStopPolicy())
}

// StopWithPolicy applies the single daemon stop ladder: request HTTP shutdown, wait for health to
// go down, wait for the recorded process to exit, escalate to SIGTERM, and use SIGKILL only last.
func StopWithPolicy(ctx context.Context, baseURL string, policy StopPolicy) error {
	policy = policy.normalized()
	daemonClient := client.NewHTTP(baseURL, nil)

	pid, pidErr := readPIDFile(baseURL)
	pidKnown := pidErr == nil && pid > 0
	if pidErr != nil && !errors.Is(pidErr, os.ErrNotExist) {
		if _, pathErr := PIDPath(baseURL); pathErr != nil {
			return pathErr
		}
		log.Printf("read whiskd pid file for %s: %v", baseURL, pidErr)
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
			removePIDFile(baseURL)
			return nil
		} else if !errors.Is(err, errProcessStillAlive) {
			return fmt.Errorf("wait for whiskd process exit: %w", err)
		}
	} else if !processAlive(pid) {
		removePIDFile(baseURL)
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
		removePIDFile(baseURL)
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
	removePIDFile(baseURL)
	return nil
}

// IsManaged reports whether the daemon at baseURL was started by this machine's whisk app, i.e. a
// PID file exists and names a live process. Used to distinguish a daemon the app owns from one a
// developer started independently.
func IsManaged(baseURL string) bool {
	pid, err := readPIDFile(baseURL)
	if err != nil {
		return false
	}
	return processAlive(pid)
}

func readPIDFile(baseURL string) (int, error) {
	pidPath, err := PIDPath(baseURL)
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, err
	}
	pid := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid); err != nil {
		return 0, err
	}
	return pid, nil
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

func PIDPath(baseURL string) (string, error) {
	addr, err := addrFromURL(baseURL)
	if err != nil {
		return "", err
	}
	replacer := strings.NewReplacer(":", "_", ".", "_", "[", "", "]", "")
	return filepath.Join(os.TempDir(), "whiskd-"+replacer.Replace(addr)+".pid"), nil
}

func writePIDFile(baseURL string, pid int) error {
	pidPath, err := PIDPath(baseURL)
	if err != nil {
		return err
	}
	return os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", pid)), 0o600)
}

func removePIDFile(baseURL string) {
	pidPath, err := PIDPath(baseURL)
	if err == nil {
		_ = os.Remove(pidPath)
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
	candidates = append(candidates, filepath.Join("bin", "whisk"))
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
