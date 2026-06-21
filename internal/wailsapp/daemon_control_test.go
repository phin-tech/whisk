package wailsapp_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/wailsapp"
)

// daemonHealthCompatServer stands in for a running daemon, answering the health and (optionally)
// compatibility endpoints the status panel relies on.
func daemonHealthCompatServer(t *testing.T, apiVersion int, withCompat bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	if withCompat {
		mux.HandleFunc("/v1/compat", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"apiVersion":%d,"gitSha":"abc1234def","version":"dev","dirty":true}`, apiVersion)
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// freeLoopbackAddr returns a loopback address that nothing is listening on, so health checks fail.
func freeLoopbackAddr(t *testing.T) string {
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

func TestDaemonControlRequiresHTTPClient(t *testing.T) {
	service := wailsapp.NewService(&runtimeClientFake{})
	ctx := context.Background()
	if _, err := service.DaemonStatus(ctx); err == nil {
		t.Fatalf("expected DaemonStatus error for non-HTTP client")
	}
	if _, err := service.StartDaemon(ctx); err == nil {
		t.Fatalf("expected StartDaemon error for non-HTTP client")
	}
	if _, err := service.StopDaemon(ctx); err == nil {
		t.Fatalf("expected StopDaemon error for non-HTTP client")
	}
	if _, err := service.RestartDaemon(ctx); err == nil {
		t.Fatalf("expected RestartDaemon error for non-HTTP client")
	}
}

func TestDaemonStatusReportsRunningDaemon(t *testing.T) {
	srv := daemonHealthCompatServer(t, protocol.DaemonAPIVersion, true)
	service := wailsapp.NewService(client.NewHTTP(srv.URL, nil))

	status, err := service.DaemonStatus(context.Background())
	if err != nil {
		t.Fatalf("daemon status: %v", err)
	}
	if !status.Running {
		t.Fatalf("expected running, status = %#v", status)
	}
	if status.APIVersion != protocol.DaemonAPIVersion || status.GitSHA == "" {
		t.Fatalf("compat fields not populated: %#v", status)
	}
	if status.Version != "dev" || !status.Dirty {
		t.Fatalf("build fields not populated: %#v", status)
	}
	if status.Address != srv.URL || status.Error != "" {
		t.Fatalf("status = %#v", status)
	}
}

func TestDaemonStatusReportsCompatError(t *testing.T) {
	srv := daemonHealthCompatServer(t, protocol.DaemonAPIVersion, false) // no /v1/compat handler
	service := wailsapp.NewService(client.NewHTTP(srv.URL, nil))

	status, err := service.DaemonStatus(context.Background())
	if err != nil {
		t.Fatalf("daemon status: %v", err)
	}
	if !status.Running {
		t.Fatalf("expected running with health OK, status = %#v", status)
	}
	if status.Error == "" {
		t.Fatalf("expected compatibility error, status = %#v", status)
	}
}

func TestDaemonStatusReportsStoppedDaemon(t *testing.T) {
	service := wailsapp.NewService(client.NewHTTP("http://"+freeLoopbackAddr(t), nil))

	status, err := service.DaemonStatus(context.Background())
	if err != nil {
		t.Fatalf("daemon status: %v", err)
	}
	if status.Running || status.Error == "" {
		t.Fatalf("expected stopped daemon with error, status = %#v", status)
	}
}

func TestStartDaemonReusesCompatibleDaemon(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("WHISKD_PATH", "")
	srv := daemonHealthCompatServer(t, protocol.DaemonAPIVersion, true)
	service := wailsapp.NewService(client.NewHTTP(srv.URL, nil))

	status, err := service.StartDaemon(context.Background())
	if err != nil {
		t.Fatalf("start daemon: %v", err)
	}
	if !status.Running {
		t.Fatalf("expected running after start, status = %#v", status)
	}
}

func TestStopDaemonWhenAlreadyDown(t *testing.T) {
	service := wailsapp.NewService(client.NewHTTP("http://"+freeLoopbackAddr(t), nil))

	status, err := service.StopDaemon(context.Background())
	if err != nil {
		t.Fatalf("stop daemon: %v", err)
	}
	if status.Running {
		t.Fatalf("expected stopped daemon, status = %#v", status)
	}
}

func TestRestartDaemonReportsSpawnFailure(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("WHISKD_PATH", "")
	service := wailsapp.NewService(client.NewHTTP("http://"+freeLoopbackAddr(t), nil))

	// Nothing is listening, so Stop is a no-op and Ensure cannot find a daemon binary to spawn.
	// The call should surface that failure while still exercising the restart path.
	if _, err := service.RestartDaemon(context.Background()); err == nil {
		t.Fatalf("expected restart to fail without a daemon binary")
	}
}
