package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/controlauth"
)

func TestServeDaemonControlAuthSmoke(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("SHELL", "/bin/sh")

	addr := freeDaemonAuthAddr(t)
	baseURL := "http://" + addr
	errCh := make(chan error, 1)
	go func() {
		errCh <- serveDaemon(addr)
	}()
	defer stopAuthSmokeDaemon(t, baseURL, errCh)

	waitForAuthSmokeHealth(t, baseURL)
	token := waitForAuthSmokeToken(t)
	assertAuthSmokeTokenMode(t)

	client := &http.Client{Timeout: time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/v1/compat", nil)
	if err != nil {
		t.Fatalf("new unauth request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unauth compat: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth compat status = %s, want 401", resp.Status)
	}

	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/v1/compat", nil)
	if err != nil {
		t.Fatalf("new auth request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("auth compat: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("auth compat status = %s, want 200", resp.Status)
	}
}

func freeDaemonAuthAddr(t *testing.T) string {
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

func waitForAuthSmokeHealth(t *testing.T, baseURL string) {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/v1/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("daemon did not become healthy")
}

func waitForAuthSmokeToken(t *testing.T) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		token, err := controlauth.ReadToken()
		if err == nil && token != "" {
			return token
		}
		time.Sleep(25 * time.Millisecond)
	}
	path, _ := controlauth.TokenPath()
	t.Fatalf("token was not generated at %s", path)
	return ""
}

func assertAuthSmokeTokenMode(t *testing.T) {
	t.Helper()
	path, err := controlauth.TokenPath()
	if err != nil {
		t.Fatalf("token path: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat token: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("token mode = %o, want 600", got)
	}
}

func stopAuthSmokeDaemon(t *testing.T, baseURL string, errCh <-chan error) {
	t.Helper()
	token, _ := controlauth.ReadToken()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+"/v1/shutdown", nil)
	if err == nil && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if err == nil {
		resp, reqErr := (&http.Client{Timeout: time.Second}).Do(req)
		if reqErr == nil {
			_ = resp.Body.Close()
		}
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve daemon: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("daemon did not stop")
	}
}

func TestValidateListenAddrAllowsAuthenticatedNonLoopback(t *testing.T) {
	if err := validateListenAddr("0.0.0.0:8787"); err != nil {
		t.Fatalf("authenticated daemon should allow non-loopback binds: %v", err)
	}
	if err := validateListenAddr("[::]:8787"); err != nil {
		t.Fatalf("authenticated daemon should allow IPv6 wildcard binds: %v", err)
	}
	if err := validateListenAddr("bad-address"); err == nil {
		t.Fatalf("expected malformed address error")
	}
}

func TestRunDaemonRunDoesNotRejectNonLoopbackAddress(t *testing.T) {
	addr := freeDaemonAuthAddr(t)
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split addr: %v", err)
	}
	_ = host
	if err := validateListenAddr(fmt.Sprintf("0.0.0.0:%s", port)); err != nil {
		t.Fatalf("non-loopback validation: %v", err)
	}
}
