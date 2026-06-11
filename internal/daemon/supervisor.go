package daemon

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/phin-tech/whisk/internal/client"
)

func Ensure(ctx context.Context, baseURL string) error {
	daemonClient := client.NewHTTP(baseURL, nil)
	if healthCheck(ctx, daemonClient) == nil {
		return nil
	}

	addr, err := addrFromURL(baseURL)
	if err != nil {
		return err
	}
	path, err := daemonPath()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(context.Background(), path, "-addr", addr)
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "whiskd.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open whiskd log: %w", err)
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start whiskd: %w", err)
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
		if err := healthCheck(ctx, daemonClient); err == nil {
			return nil
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return fmt.Errorf("wait for whiskd: %w", ctx.Err())
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
		if killErr := process.Kill(); killErr != nil {
			return err
		}
	}
	_ = os.Remove(pidPath)
	return nil
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
	candidates := []string{}
	if path := os.Getenv("WHISKD_PATH"); path != "" {
		candidates = append(candidates, path)
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "whiskd"))
	}
	candidates = append(candidates, filepath.Join("bin", "whiskd"))
	if path, err := exec.LookPath("whiskd"); err == nil {
		candidates = append(candidates, path)
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("whiskd not found; run `task build:daemon` or set WHISKD_PATH")
}
