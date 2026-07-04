package server

import (
	"net/http"

	"github.com/phin-tech/whisk/internal/adapters/agents"
	"github.com/phin-tech/whisk/internal/protocol"
)

func (s *HTTPServer) listDetectedAgents(w http.ResponseWriter, r *http.Request) {
	detected, err := s.runtime.ListDetectedAgents(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]protocol.DetectedAgent, len(detected))
	for i, agent := range detected {
		out[i] = protocol.DetectedAgent{
			ProfileID:     agent.ProfileID,
			Provider:      string(agent.Provider),
			Label:         agent.Label,
			DetectCommand: agent.DetectCommand,
			Path:          agent.Path,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func agentProfileToProtocol(profile agents.ProfileInfo) protocol.AgentProfile {
	return protocol.AgentProfile{
		ID:                  profile.ID,
		Provider:            string(profile.Provider),
		Label:               profile.Label,
		Description:         profile.Description,
		Source:              string(profile.Source),
		PluginID:            profile.PluginID,
		Launchable:          profile.Launchable,
		LaunchBlockedReason: profile.LaunchBlockedReason,
		DetectCmd:           profile.DetectCmd,
		DetectAliases:       append([]string(nil), profile.DetectAliases...),
		ExpectedProcess:     profile.ExpectedProcess,
		PromptInjectionMode: string(profile.PromptInjectionMode),
		DraftPromptFlag:     profile.DraftPromptFlag,
		DraftPromptEnvVar:   profile.DraftPromptEnvVar,
		PreflightTrust:      string(profile.PreflightTrust),
		ReadySignal:         string(profile.ReadySignal),
		HookProvider:        profile.HookProvider,
	}
}
