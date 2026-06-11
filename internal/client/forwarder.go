package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/phin-tech/whisk/internal/domain/httpforward"
	"github.com/phin-tech/whisk/internal/protocol"
)

type LocalForwarder struct {
	daemon *HTTPClient

	mu       sync.Mutex
	forwards map[string]*localForward
}

type localForward struct {
	server *http.Server
}

func NewLocalForwarder(daemon *HTTPClient, _ *http.Client) *LocalForwarder {
	return &LocalForwarder{
		daemon:   daemon,
		forwards: map[string]*localForward{},
	}
}

func (f *LocalForwarder) Start(ctx context.Context, req protocol.StartHTTPForwardRequest) (protocol.StartedHTTPForward, error) {
	if f.daemon == nil {
		return protocol.StartedHTTPForward{}, fmt.Errorf("daemon HTTP client required")
	}
	forward, err := f.daemon.CreateHTTPForward(ctx, protocol.CreateHTTPForwardRequest{
		Name:      req.Name,
		TargetURL: req.TargetURL,
		SessionID: req.SessionID,
	})
	if err != nil {
		return protocol.StartedHTTPForward{}, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = f.daemon.DeleteHTTPForward(ctx, forward.ID)
		return protocol.StartedHTTPForward{}, err
	}

	target, err := url.Parse(f.daemon.forwardProxyURL(forward.ID))
	if err != nil {
		_ = listener.Close()
		_ = f.daemon.DeleteHTTPForward(ctx, forward.ID)
		return protocol.StartedHTTPForward{}, err
	}
	server := &http.Server{
		Handler:           f.localProxy(target),
		ReadHeaderTimeout: 5 * time.Second,
	}

	f.mu.Lock()
	f.forwards[forward.ID] = &localForward{server: server}
	f.mu.Unlock()

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			_ = f.Stop(context.Background(), forward.ID)
		}
	}()

	return protocol.StartedHTTPForward{
		ID:       forward.ID,
		LocalURL: "http://" + listener.Addr().String(),
		Forward:  forward,
	}, nil
}

func (f *LocalForwarder) Stop(ctx context.Context, id string) error {
	f.mu.Lock()
	forward, ok := f.forwards[id]
	if ok {
		delete(f.forwards, id)
	}
	f.mu.Unlock()

	var shutdownErr error
	if ok {
		shutdownErr = forward.server.Shutdown(ctx)
	}
	deleteErr := f.daemon.DeleteHTTPForward(ctx, id)
	if shutdownErr != nil {
		return shutdownErr
	}
	return deleteErr
}

func (f *LocalForwarder) localProxy(target *url.URL) http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(proxyReq *httputil.ProxyRequest) {
			in := proxyReq.In
			proxyReq.SetURL(target)
			proxyReq.SetXForwarded()
			proxyReq.Out.Host = target.Host
			proxyReq.Out.URL.Path = localProxyPath(target.Path, in.URL.Path)
			proxyReq.Out.URL.RawPath = ""
			proxyReq.Out.URL.RawQuery = mergeRawQuery(target.RawQuery, in.URL.RawQuery)
		},
		FlushInterval: -1 * time.Nanosecond,
	}
	return proxy
}

func localProxyPath(targetBasePath string, incomingPath string) string {
	path, err := httpforward.ProxyPath(targetBasePath, incomingPath)
	if err != nil {
		return "/"
	}
	return path
}

func mergeRawQuery(base string, incoming string) string {
	if base == "" {
		return incoming
	}
	if incoming == "" {
		return base
	}
	return base + "&" + incoming
}
