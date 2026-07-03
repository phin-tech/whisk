package notification

import (
	"strings"
	"time"
)

type CooldownState struct {
	LastShown map[string]time.Time
}

func ApplyCooldown(state CooldownState, key string, now time.Time, cooldown time.Duration) (CooldownState, bool) {
	cleanKey := strings.TrimSpace(key)
	next := CooldownState{LastShown: cloneLastShown(state.LastShown)}
	if cleanKey == "" {
		return next, true
	}
	if cooldown > 0 {
		if last, ok := next.LastShown[cleanKey]; ok && now.Sub(last) < cooldown {
			return next, false
		}
	}
	next.LastShown[cleanKey] = now
	return next, true
}

func cloneLastShown(in map[string]time.Time) map[string]time.Time {
	out := make(map[string]time.Time, len(in)+1)
	for key, value := range in {
		out[key] = value
	}
	return out
}
