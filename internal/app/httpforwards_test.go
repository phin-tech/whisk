package app_test

import (
	"context"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/app"
)

func TestRuntimeHTTPForwardFlow(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{})
	ctx := context.Background()

	created, err := runtime.CreateHTTPForward(ctx, app.CreateHTTPForwardRequest{
		Name:      "difit",
		TargetURL: "http://127.0.0.1:4966",
		SessionID: "session_01",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" || created.Name != "difit" || created.SessionID != "session_01" {
		t.Fatalf("created = %#v", created)
	}

	listed, err := runtime.ListHTTPForwards(ctx)
	if err != nil || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("list = %#v, err = %v", listed, err)
	}
	got, err := runtime.GetHTTPForward(ctx, created.ID)
	if err != nil || got.TargetURL != "http://127.0.0.1:4966" {
		t.Fatalf("get = %#v, err = %v", got, err)
	}
	if err := runtime.DeleteHTTPForward(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := runtime.GetHTTPForward(ctx, created.ID); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing get err = %v", err)
	}
	if err := runtime.DeleteHTTPForward(ctx, created.ID); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing delete err = %v", err)
	}
}

func TestRuntimeHTTPForwardRejectsInvalidTarget(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{})
	_, err := runtime.CreateHTTPForward(context.Background(), app.CreateHTTPForwardRequest{
		TargetURL: "http://example.com:4966",
	})
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("err = %v", err)
	}
}
