package app

import (
	"context"

	"github.com/phin-tech/whisk/internal/adapters/agents"
)

func (r *Runtime) ListDetectedAgents(context.Context) ([]agents.DetectedProfile, error) {
	return agents.DetectProfiles(agents.ProfileList(), nil), nil
}
