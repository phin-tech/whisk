package notification

import (
	"testing"
	"time"
)

func TestApplyCooldownAllowsFirstDisplayAndSuppressesSameKeyWithinWindow(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	state, ok := ApplyCooldown(CooldownState{}, "status|session:sess_01|pane:pane_01|actor:codex|kind:question", now, 5*time.Second)
	if !ok {
		t.Fatalf("first display suppressed")
	}
	if got := state.LastShown["status|session:sess_01|pane:pane_01|actor:codex|kind:question"]; !got.Equal(now) {
		t.Fatalf("last shown = %s, want %s", got, now)
	}

	next, ok := ApplyCooldown(state, "status|session:sess_01|pane:pane_01|actor:codex|kind:question", now.Add(4*time.Second), 5*time.Second)
	if ok {
		t.Fatalf("same key inside cooldown allowed")
	}
	if got := next.LastShown["status|session:sess_01|pane:pane_01|actor:codex|kind:question"]; !got.Equal(now) {
		t.Fatalf("suppressed display mutated last shown = %s, want %s", got, now)
	}
}

func TestApplyCooldownAllowsDifferentKeysAndExpiredWindowsWithoutMutatingInput(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	input := CooldownState{LastShown: map[string]time.Time{
		"status|session:sess_01|pane:pane_01|actor:codex|kind:question": now,
	}}

	next, ok := ApplyCooldown(input, "status|session:sess_01|pane:pane_02|actor:codex|kind:question", now.Add(time.Second), 5*time.Second)
	if !ok {
		t.Fatalf("different key suppressed")
	}
	if len(input.LastShown) != 1 {
		t.Fatalf("input state mutated = %#v", input.LastShown)
	}

	expired, ok := ApplyCooldown(next, "status|session:sess_01|pane:pane_01|actor:codex|kind:question", now.Add(5*time.Second), 5*time.Second)
	if !ok {
		t.Fatalf("expired key suppressed")
	}
	if got := expired.LastShown["status|session:sess_01|pane:pane_01|actor:codex|kind:question"]; !got.Equal(now.Add(5 * time.Second)) {
		t.Fatalf("expired last shown = %s", got)
	}
}
