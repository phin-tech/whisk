package main

import (
	"context"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/protocol"
)

func TestRunForwardCreateStartsLocalForwardAndWaitsForContext(t *testing.T) {
	started := make(chan protocol.StartedHTTPForward, 1)
	forwardDone := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	deps := runDeps{
		context: func() (context.Context, context.CancelFunc) {
			return ctx, cancel
		},
		startForward: func(ctx context.Context, baseURL string, req protocol.StartHTTPForwardRequest) (protocol.StartedHTTPForward, func(context.Context) error, error) {
			if baseURL != "http://127.0.0.1:8787" {
				t.Fatalf("baseURL = %q", baseURL)
			}
			if req.TargetURL != "http://127.0.0.1:4966" || req.Name != "difit" {
				t.Fatalf("req = %#v", req)
			}
			result := protocol.StartedHTTPForward{ID: "fwd_01", LocalURL: "http://127.0.0.1:50001"}
			started <- result
			return result, func(context.Context) error {
				close(forwardDone)
				return nil
			}, nil
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- runWithDeps([]string{"forward", "create", "http://127.0.0.1:4966", "-name", "difit"}, deps)
	}()

	got := <-started
	if got.LocalURL != "http://127.0.0.1:50001" {
		t.Fatalf("started = %#v", got)
	}
	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("run: %v", err)
	}
	<-forwardDone
}

func TestRunForwardCreateRequiresTarget(t *testing.T) {
	err := runWithDeps([]string{"forward", "create"}, runDeps{})
	if err == nil || !strings.Contains(err.Error(), "usage: whisk forward create") {
		t.Fatalf("err = %v", err)
	}
}
