package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
)

func TestRuntimeRecordAgentHookEventDoesNotDeadlockWithDefaultIDGenerator(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{})

	done := make(chan error, 1)
	go func() {
		_, err := runtime.RecordAgentHookEvent(context.Background(), app.AgentBridgeHookRequest{
			Provider:  "claude",
			EventName: "Notification",
			Message:   "manual hook test",
		})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("record hook event: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("record hook event timed out")
	}
}
