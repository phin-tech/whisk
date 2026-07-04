package browser_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adapter "github.com/phin-tech/whisk/internal/adapters/browser"
)

func TestCDPProbeReadsVersionAndTargets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/json/version":
			_, _ = w.Write([]byte(`{"Browser":"Chrome/126.0","Protocol-Version":"1.3","webSocketDebuggerUrl":"ws://127.0.0.1/devtools/browser/1"}`))
		case "/json/list":
			_, _ = w.Write([]byte(`[{"id":"page_1","type":"page","url":"https://example.test/","title":"Example","webSocketDebuggerUrl":"ws://127.0.0.1/devtools/page/1"}]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	result, err := adapter.NewCDPProbe(server.Client()).ProbeCDP(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("ProbeCDP: %v", err)
	}
	if result.Browser != "Chrome/126.0" || result.ProtocolVersion != "1.3" {
		t.Fatalf("version result = %#v", result)
	}
	if len(result.Targets) != 1 || result.Targets[0].ID != "page_1" || result.Targets[0].Title != "Example" {
		t.Fatalf("targets = %#v", result.Targets)
	}
}

func TestCDPProbeRejectsNonLoopbackEndpoint(t *testing.T) {
	_, err := adapter.NewCDPProbe(nil).ProbeCDP(context.Background(), "http://example.com:9222")
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("err = %v", err)
	}
}

func TestCDPProbeReturnsTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := adapter.NewCDPProbe(nil).ProbeCDP(ctx, "http://127.0.0.1:9222")
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context canceled", err)
	}
}

func TestCDPProbeReportsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "missing browser", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := adapter.NewCDPProbe(server.Client()).ProbeCDP(context.Background(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("err = %v", err)
	}
}

func TestCDPProbeRejectsRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		http.Redirect(w, r, "http://example.com/json/version", http.StatusFound)
	}))
	defer server.Close()

	_, err := adapter.NewCDPProbe(server.Client()).ProbeCDP(context.Background(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "refused redirect") {
		t.Fatalf("err = %v", err)
	}
}
