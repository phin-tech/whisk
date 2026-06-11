package daemon

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start whiskd: %w", err)
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
