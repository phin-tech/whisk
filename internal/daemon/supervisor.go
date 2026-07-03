package daemon

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
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

const (
	compatibilityProbeTimeout   = 250 * time.Millisecond
	compatibilityRetryWindow    = 2 * time.Second
	compatibilityInitialBackoff = 50 * time.Millisecond
	compatibilityMaxBackoff     = 250 * time.Millisecond
	daemonLogTailBytes          = 4096
)

type compatibilityDecision int

const (
	compatibilityUnknown compatibilityDecision = iota
	compatibilityCompatible
	compatibilityIncompatible
)

type daemonStartOptions struct {
	writePID bool
	env      map[string]string
	label    string
}

type spawnedDaemon struct {
	cmd       *exec.Cmd
	waitCh    chan error
	logPath   string
	logOffset int64
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
		path, err := daemonPath()
		if err != nil {
			return false, fmt.Errorf("find replacement whiskd: %w", err)
		}
		if err := verifyReplacementDaemon(ctx, path); err != nil {
			return false, fmt.Errorf("replacement whiskd at %s failed compatibility verification; leaving existing daemon running: %w", path, err)
		}
		log.Printf("whiskd at %s is incompatible (%v); shutting it down", baseURL, compatibilityErr)
		// Capture the PID before StopPID removes the file so we can wait for the actual
		// process to exit, not just for the HTTP server to stop answering.
		existingPID, _ := readPIDFile(baseURL)
		if err := shutdownExisting(ctx, baseURL); err != nil {
			log.Printf("shutdown incompatible whiskd: %v", err)
		}
		_ = StopPID(baseURL)
		if err := waitUntilDown(ctx, daemonClient); err != nil {
			return false, fmt.Errorf("stop incompatible whiskd: %w", err)
		}
		// httpServer.Shutdown stops answering health immediately, but the process keeps
		// draining (PTYs/NATS/sqlite) for some time afterwards. Wait for the process itself
		// to be gone before spawning a replacement, otherwise the old and new daemons
		// coexist on the same address.
		if existingPID > 0 {
			if err := waitForProcessExit(ctx, existingPID); err != nil {
				return false, fmt.Errorf("stop incompatible whiskd: %w", err)
			}
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

	if _, err := startDaemonAndWait(ctx, baseURL, addr, path, daemonStartOptions{writePID: true}); err != nil {
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
			log.Printf("whiskd ready at %s", baseURL)
			return proc, nil
		}
		if decision == compatibilityIncompatible {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = stopSpawnedDaemon(cleanupCtx, baseURL, proc, opts.writePID)
			cancel()
			return nil, fmt.Errorf("started whiskd is incompatible: %w%s", err, proc.logTailMessage())
		}
		select {
		case waitErr := <-proc.waitCh:
			if opts.writePID {
				removePIDFile(baseURL)
			}
			return nil, proc.exitBeforeReadyError(waitErr)
		case <-ticker.C:
		case <-ctx.Done():
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = stopSpawnedDaemon(cleanupCtx, baseURL, proc, opts.writePID)
			cancel()
			return nil, fmt.Errorf("wait for whiskd: %w%s", ctx.Err(), proc.logTailMessage())
		}
	}
}

func startDaemonProcess(baseURL, addr, path string, opts daemonStartOptions) (*spawnedDaemon, error) {
	logPath := filepath.Join(os.TempDir(), "whiskd.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open whiskd log: %w", err)
	}
	info, statErr := logFile.Stat()
	if statErr != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("stat whiskd log: %w", statErr)
	}
	logOffset := info.Size()
	cmd := exec.CommandContext(context.Background(), path, "daemon", "run", "-addr", addr)
	if len(opts.env) > 0 {
		cmd.Env = environWithOverrides(opts.env)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	detach(cmd)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("start whiskd: %w", err)
	}
	if err := logFile.Close(); err != nil {
		log.Printf("close whiskd log handle: %v", err)
	}
	if opts.writePID {
		if err := writePIDFile(baseURL, cmd.Process.Pid); err != nil {
			log.Printf("write whiskd pid file: %v", err)
		}
	}
	proc := &spawnedDaemon{
		cmd:       cmd,
		waitCh:    make(chan error, 1),
		logPath:   logPath,
		logOffset: logOffset,
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

func stopSpawnedDaemon(ctx context.Context, baseURL string, proc *spawnedDaemon, removePID bool) error {
	if proc == nil {
		return nil
	}
	defer func() {
		if removePID {
			removePIDFile(baseURL)
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

func removePIDFile(baseURL string) {
	pidPath, err := PIDPath(baseURL)
	if err != nil {
		log.Printf("remove whiskd pid file: %v", err)
		return
	}
	_ = os.Remove(pidPath)
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

func (p *spawnedDaemon) exitBeforeReadyError(waitErr error) error {
	return fmt.Errorf("started whiskd exited before readiness: %s%s", exitStatusText(waitErr), p.logTailMessage())
}

func exitStatusText(err error) string {
	if err == nil {
		return "exit status 0"
	}
	return err.Error()
}

func (p *spawnedDaemon) logTailMessage() string {
	tail := readLogTail(p.logPath, p.logOffset, daemonLogTailBytes)
	if tail == "" {
		return ""
	}
	return "; whiskd log tail:\n" + tail
}

func readLogTail(path string, offset int64, maxBytes int64) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() <= offset {
		return ""
	}
	start := info.Size() - maxBytes
	if start < offset {
		start = offset
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return ""
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func StopPID(baseURL string) error {
	pidPath, err := PIDPath(baseURL)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return err
	}
	pid := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid); err != nil {
		return err
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := process.Signal(os.Interrupt); err != nil {
		// Interrupt is unsupported (e.g. Windows) or the process is already gone; force-kill.
		if killErr := process.Kill(); killErr != nil {
			return err
		}
	} else {
		// Give the daemon a moment to shut down gracefully, then force-kill if it lingers so
		// we never leave an orphaned process holding the port.
		deadline := 2 * time.Second
		interval := 50 * time.Millisecond
		for waited := time.Duration(0); waited < deadline && processAlive(pid); waited += interval {
			time.Sleep(interval)
		}
		if processAlive(pid) {
			_ = process.Kill()
		}
	}
	_ = os.Remove(pidPath)
	return nil
}

// Stop shuts down the daemon at baseURL whether or not this process started it. It first asks the
// daemon to shut itself down over HTTP, then signals the recorded PID as a fallback, and waits for
// it to stop answering health checks. A daemon that is already down is treated as success.
func Stop(ctx context.Context, baseURL string) error {
	daemonClient := client.NewHTTP(baseURL, nil)
	if healthCheck(ctx, daemonClient) != nil {
		_ = StopPID(baseURL) // clean up a stale PID file if one is lying around
		return nil
	}
	if err := shutdownExisting(ctx, baseURL); err != nil {
		log.Printf("shutdown whiskd at %s: %v", baseURL, err)
	}
	_ = StopPID(baseURL)
	return waitUntilDown(ctx, daemonClient)
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

func waitForProcessExit(ctx context.Context, pid int) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if !processAlive(pid) {
			return nil
		}
		select {
		case <-ticker.C:
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

func healthCheck(ctx context.Context, daemonClient *client.HTTPClient) error {
	checkCtx, cancel := context.WithTimeout(ctx, compatibilityProbeTimeout)
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

func shutdownExisting(ctx context.Context, baseURL string) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
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

func waitUntilDown(ctx context.Context, daemonClient *client.HTTPClient) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if healthCheck(ctx, daemonClient) != nil {
			return nil
		}
		select {
		case <-ticker.C:
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
