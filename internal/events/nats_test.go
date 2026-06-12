package events

import (
	"testing"

	"github.com/phin-tech/whisk/internal/app"
)

func TestSubjectForRuntimeEvents(t *testing.T) {
	tests := []struct {
		eventType app.RuntimeEventType
		want      string
	}{
		{eventType: app.EventSessionChanged, want: "whisk.session.changed"},
		{eventType: app.EventPTYChanged, want: "whisk.pty.changed"},
		{eventType: app.EventPTYOutput, want: "whisk.pty.output"},
		{eventType: app.EventWorkItemsChanged, want: "whisk.workitems.changed"},
		{eventType: app.EventStatusChanged, want: "whisk.status.changed"},
	}

	for _, test := range tests {
		if got := subjectFor(test.eventType); got != test.want {
			t.Fatalf("subjectFor(%q) = %q, want %q", test.eventType, got, test.want)
		}
	}
}
