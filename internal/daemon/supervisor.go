package daemon

import (
	"context"
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

// Ensure makes sure a compatible daemon is reachable at baseURL, starting one if needed.
// It reports whether it started a new daemon (started == true) versus adopting one that was
// already running (started == false). Callers use this to decide ownership: a daemon the app
// started itself should be stopped when the app exits, while one started elsewhere (e.g. a
// developer's `whisk daemon run`) must be left alone.
func Ensure(ctx context.Context, baseURL string) (started bool, err error) {
	daemonClient := client.NewHTTP(baseURL, nil)
	compatibilityErr := compatibilityCheck(ctx, daemonClient)
	if compatibilityErr == nil {
		return false, nil
	}
	if healthCheck(ctx, daemonClient) == nil {
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
		if err := compatibilityCheck(ctx, daemonClient); err == nil {
			log.Printf("whiskd ready at %s", baseURL)
			return true, nil
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return false, fmt.Errorf("wait for whiskd: %w", ctx.Err())
		}
	}
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
	checkCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	return daemonClient.Health(checkCtx)
}

func compatibilityCheck(ctx context.Context, daemonClient *client.HTTPClient) error {
	if err := healthCheck(ctx, daemonClient); err != nil {
		return err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	compatibility, err := daemonClient.Compatibility(checkCtx)
	if err != nil {
		return fmt.Errorf("daemon is missing required compatibility API: %w", err)
	}
	if compatibility.APIVersion != protocol.DaemonAPIVersion {
		return fmt.Errorf("daemon api version %d does not match required %d", compatibility.APIVersion, protocol.DaemonAPIVersion)
	}
	return nil
}

func shutdownExisting(ctx context.Context, baseURL string) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
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
