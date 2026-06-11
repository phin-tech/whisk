package client_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestLocalHTTPForwarderProxiesThroughDaemon(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nested/asset.js" || r.URL.RawQuery != "v=1" {
			t.Fatalf("target path = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		if r.Header.Get("X-From-Test") != "yes" {
			t.Fatalf("missing forwarded header")
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Target", "hit")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("proxied:" + string(body)))
	}))
	t.Cleanup(target.Close)

	runtime := app.NewRuntime(app.RuntimeConfig{})
	daemonServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(daemonServer.Close)

	daemon := client.NewHTTP(daemonServer.URL, daemonServer.Client())
	forwarder := client.NewLocalForwarder(daemon, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	started, err := forwarder.Start(ctx, protocol.StartHTTPForwardRequest{
		TargetURL: target.URL,
		Name:      "difit",
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = forwarder.Stop(context.Background(), started.ID) })

	forwards, err := daemon.ListHTTPForwards(ctx)
	if err != nil || len(forwards) != 1 || forwards[0].ID != started.ID {
		t.Fatalf("list forwards = %#v, err = %v", forwards, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, started.LocalURL+"/nested/asset.js?v=1", strings.NewReader("body"))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("X-From-Test", "yes")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusCreated || response.Header.Get("X-Target") != "hit" || string(body) != "proxied:body" {
		t.Fatalf("response status=%d header=%q body=%q", response.StatusCode, response.Header.Get("X-Target"), string(body))
	}
}

func TestLocalHTTPForwarderProxiesWebSocketThroughDaemon(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.CloseNow()
		_, message, err := conn.Read(r.Context())
		if err != nil {
			t.Errorf("read: %v", err)
			return
		}
		if err := conn.Write(r.Context(), websocket.MessageText, append([]byte("echo:"), message...)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	t.Cleanup(target.Close)

	runtime := app.NewRuntime(app.RuntimeConfig{})
	daemonServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(daemonServer.Close)

	daemon := client.NewHTTP(daemonServer.URL, daemonServer.Client())
	forwarder := client.NewLocalForwarder(daemon, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	started, err := forwarder.Start(ctx, protocol.StartHTTPForwardRequest{TargetURL: target.URL})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = forwarder.Stop(context.Background(), started.ID) })

	wsURL := "ws" + strings.TrimPrefix(started.LocalURL, "http") + "/socket"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()
	if err := conn.Write(ctx, websocket.MessageText, []byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, message, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(message) != "echo:hello" {
		t.Fatalf("message = %q", string(message))
	}
}

func TestLocalHTTPForwarderRequiresDaemonClient(t *testing.T) {
	forwarder := client.NewLocalForwarder(nil, nil)
	_, err := forwarder.Start(context.Background(), protocol.StartHTTPForwardRequest{TargetURL: "http://127.0.0.1:4966"})
	if err == nil || !strings.Contains(err.Error(), "daemon HTTP client required") {
		t.Fatalf("err = %v", err)
	}
}
