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
		{eventType: app.EventPluginsChanged, want: "whisk.plugins.changed"},
		{eventType: app.EventMailboxChanged, want: "whisk.mailbox.changed"},
		{eventType: app.EventAgentBridgeApprovalsChanged, want: "whisk.agent_bridge_approvals.changed"},
		{eventType: app.EventAgentPromptsChanged, want: "whisk.agent_prompts.changed"},
		{eventType: app.EventAgentHookEventsChanged, want: "whisk.agent_hook_events.changed"},
	}

	for _, test := range tests {
		if got := subjectFor(test.eventType); got != test.want {
			t.Fatalf("subjectFor(%q) = %q, want %q", test.eventType, got, test.want)
		}
	}
}

func TestNATSBusNextRetainedReportsMissedCursor(t *testing.T) {
	bus := &NATSBus{
		retained: []app.RuntimeEvent{
			{Seq: 3, Type: app.EventSessionChanged},
			{Seq: 4, Type: app.EventPTYChanged, PtyID: "pty_01"},
		},
		notify: make(chan struct{}, 1),
	}

	result, ok := bus.nextRetainedLocked(1)
	if !ok || !result.Missed || result.Event.Seq != 3 {
		t.Fatalf("old cursor result = %#v, ok=%v", result, ok)
	}

	result, ok = bus.nextRetainedLocked(3)
	if !ok || result.Missed || result.Event.Seq != 4 {
		t.Fatalf("next cursor result = %#v, ok=%v", result, ok)
	}

	if result, ok = bus.nextRetainedLocked(4); ok {
		t.Fatalf("current cursor result = %#v, ok=%v", result, ok)
	}

	result, ok = bus.nextRetainedLocked(99)
	if !ok || !result.Missed || result.Event.Seq != 3 {
		t.Fatalf("future cursor result = %#v, ok=%v", result, ok)
	}
}
