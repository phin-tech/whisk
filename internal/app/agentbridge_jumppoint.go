package app

import (
	"context"
	"strings"

	"github.com/phin-tech/whisk/internal/domain/agentbridge"
)

const agentHookJumpPointBookmarkKind = "jump_point"

type agentHookJumpPointTrigger string

const (
	agentHookJumpPointPrompt   agentHookJumpPointTrigger = "prompt"
	agentHookJumpPointApproval agentHookJumpPointTrigger = "approval"
)

func agentHookJumpPointForHook(req AgentBridgeHookRequest, needsApproval bool) (agentHookJumpPointTrigger, bool) {
	if isAgentPromptHook(req) {
		return agentHookJumpPointPrompt, true
	}
	if needsApproval {
		return agentHookJumpPointApproval, true
	}
	return "", false
}

func (r *Runtime) createAgentHookJumpPoint(ctx context.Context, bridge agentbridge.Bridge, req AgentBridgeHookRequest) {
	ptyID := strings.TrimSpace(firstNonEmpty(req.PTYID, bridge.PTYID))
	if ptyID == "" {
		return
	}
	if r.ptys == nil {
		return
	}
	offset := uint64(0)
	if snapshot, err := r.PTYOutput(ctx, ptyID, 0); err == nil {
		offset = snapshot.Offset + uint64(len(snapshot.OutputBytes))
	}
	_, _ = r.AddPTYBookmark(ctx, AddPTYBookmarkRequest{
		PTYID:  ptyID,
		Offset: offset,
		Kind:   agentHookJumpPointBookmarkKind,
	})
}
