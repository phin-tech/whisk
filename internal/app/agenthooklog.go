package app

import (
	"context"

	"github.com/phin-tech/whisk/internal/adapters/agenthooklog"
	"github.com/phin-tech/whisk/internal/domain/agentbridge"
)

type AgentHookLogStatus = agenthooklog.Status

type SetAgentHookLogSettingsRequest struct {
	Enabled           *bool
	ClearAfterSession *bool
}

func (r *Runtime) AgentHookLogStatus(context.Context) (AgentHookLogStatus, error) {
	logger, err := r.agentHookLogger()
	if err != nil {
		return AgentHookLogStatus{}, err
	}
	size, err := logger.Size()
	if err != nil {
		return AgentHookLogStatus{}, err
	}
	r.mu.Lock()
	enabled := r.agentHookLogEnabled
	clearAfterSession := r.clearHookLogAfterSession
	r.mu.Unlock()
	return AgentHookLogStatus{
		Enabled:           enabled,
		ClearAfterSession: clearAfterSession,
		Path:              logger.Path(),
		SizeBytes:         size,
	}, nil
}

func (r *Runtime) SetAgentHookLogSettings(ctx context.Context, req SetAgentHookLogSettingsRequest) (AgentHookLogStatus, error) {
	r.mu.Lock()
	if req.Enabled != nil {
		r.agentHookLogEnabled = *req.Enabled
	}
	if req.ClearAfterSession != nil {
		r.clearHookLogAfterSession = *req.ClearAfterSession
	}
	r.mu.Unlock()
	return r.AgentHookLogStatus(ctx)
}

func (r *Runtime) ClearAgentHookLog(ctx context.Context) (AgentHookLogStatus, error) {
	logger, err := r.agentHookLogger()
	if err != nil {
		return AgentHookLogStatus{}, err
	}
	if err := logger.Clear(); err != nil {
		return AgentHookLogStatus{}, err
	}
	return r.AgentHookLogStatus(ctx)
}

func (r *Runtime) OpenAgentHookLog(ctx context.Context) (AgentHookLogStatus, error) {
	logger, err := r.agentHookLogger()
	if err != nil {
		return AgentHookLogStatus{}, err
	}
	if err := logger.Open(); err != nil {
		return AgentHookLogStatus{}, err
	}
	return r.AgentHookLogStatus(ctx)
}

func (r *Runtime) appendAgentHookLog(event agentbridge.Event) error {
	logger, err := r.agentHookLogger()
	if err != nil {
		return err
	}
	return logger.Append(agenthooklog.Entry{
		Timestamp:        event.CreatedAt,
		Provider:         string(event.Provider),
		EventName:        event.EventName,
		BridgeID:         event.BridgeID,
		SessionID:        event.SessionID,
		PTYID:            event.PTYID,
		ToolName:         event.ToolName,
		Message:          event.Message,
		NotificationType: event.NotificationType,
		ElicitationID:    event.ElicitationID,
		Action:           event.Action,
		Result:           event.Result,
		Raw:              event.Raw,
	})
}

func (r *Runtime) agentHookLogger() (*agenthooklog.Logger, error) {
	if r.agentHookLogPaths != nil {
		return agenthooklog.New(*r.agentHookLogPaths), nil
	}
	paths, err := agenthooklog.DefaultPaths()
	if err != nil {
		return nil, err
	}
	return agenthooklog.New(paths), nil
}
