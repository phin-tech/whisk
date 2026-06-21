package app

import (
	"context"
	"fmt"
	"time"

	"github.com/phin-tech/whisk/internal/domain/agentbridge"
)

type AgentBridgeHookRequest struct {
	BridgeID         string
	Token            string
	Provider         string
	EventName        string
	ToolName         string
	ToolInput        map[string]any
	ToolOutput       string
	Message          string
	NotificationType string
	ElicitationID    string
	Action           string
	SessionID        string
	PTYID            string
	RawPayload       map[string]any
	Decision         AgentBridgeHookDecision
}

type AgentBridgeHookDecision struct {
	Action string
	Reason string
}

type AgentBridgeHookResponse struct {
	Output map[string]any
}

type ListAgentBridgeApprovalsRequest struct {
	Status string
}

type ListAgentBridgeEventsRequest struct {
	Status string
}

type ListAgentPromptsRequest struct {
	Status string
}

type MarkAgentBridgeEventReadRequest struct {
	ID string
}

type ResolveAgentBridgeApprovalRequest struct {
	ID     string
	Action string
	Reason string
}

type ResolveAgentPromptRequest struct {
	ID     string
	Answer string
}

func (r *Runtime) HandleAgentBridgeHook(ctx context.Context, req AgentBridgeHookRequest) (AgentBridgeHookResponse, error) {
	r.mu.Lock()
	validToken := r.agentBridges.ValidateHookToken(req.BridgeID, req.Token)
	bridge, bridgeOK := r.agentBridges.Bridge(req.BridgeID)
	r.mu.Unlock()
	if !validToken || !bridgeOK {
		return AgentBridgeHookResponse{}, ErrUnauthorizedAgentBridgeHook
	}

	provider := agentbridge.Provider(req.Provider)
	if provider == "" {
		provider = bridge.Provider
	}
	if isAgentPromptHook(req) {
		answer, err := r.requestAgentPrompt(ctx, bridge, req)
		if err != nil {
			return AgentBridgeHookResponse{}, err
		}
		output, _ := agentbridge.PromptAnswerToProviderOutput(provider, req.EventName, req.ElicitationID, answer, req.ToolInput)
		return AgentBridgeHookResponse{Output: output}, nil
	}
	evaluation, ok := agentbridge.HookPayloadToEvaluationRequest(agentbridge.HookPayload{
		Provider:   provider,
		EventName:  req.EventName,
		ToolName:   req.ToolName,
		ToolInput:  req.ToolInput,
		ToolOutput: req.ToolOutput,
	})
	if !ok {
		if _, err := r.recordAgentBridgeEvent(ctx, req, bridge, "logged"); err != nil {
			return AgentBridgeHookResponse{}, err
		}
		return AgentBridgeHookResponse{}, nil
	}

	decision := agentbridge.EvaluationDecision{
		Action: agentbridge.PolicyAction(req.Decision.Action),
		Reason: req.Decision.Reason,
	}
	if decision.Action == "" && evaluation.Phase == agentbridge.PhaseToolCall {
		var err error
		decision, err = r.requestAgentBridgeApproval(ctx, bridge, req)
		if err != nil {
			return AgentBridgeHookResponse{}, err
		}
	}
	result := "logged"
	if decision.Action != "" {
		result = "approval_created"
	}
	_, _ = r.recordAgentBridgeEvent(ctx, req, bridge, result)
	output, _ := agentbridge.EvaluationDecisionToProviderOutput(provider, req.EventName, decision)
	return AgentBridgeHookResponse{Output: output}, nil
}

func isAgentPromptHook(req AgentBridgeHookRequest) bool {
	return req.EventName == "Elicitation" || (req.EventName == "PreToolUse" && req.ToolName == "AskUserQuestion")
}

var ErrUnauthorizedAgentBridgeHook = fmt.Errorf("unauthorized agent bridge hook")

func (r *Runtime) ListAgentBridgeApprovals(_ context.Context, req ListAgentBridgeApprovalsRequest) ([]agentbridge.Approval, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.agentBridges.ListApprovals(agentbridge.ListApprovals{Status: agentbridge.ApprovalStatus(req.Status)}), nil
}

func (r *Runtime) ListAgentBridgeEvents(_ context.Context, req ListAgentBridgeEventsRequest) ([]agentbridge.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.agentBridges.ListEvents(agentbridge.ListEvents{Status: agentbridge.EventStatus(req.Status)}), nil
}

func (r *Runtime) ListAgentPrompts(_ context.Context, req ListAgentPromptsRequest) ([]agentbridge.Prompt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	prompts := r.agentBridges.ListPrompts(agentbridge.ListPrompts{Status: agentbridge.PromptStatus(req.Status)})
	for _, approval := range r.agentBridges.ListApprovals(agentbridge.ListApprovals{Status: approvalStatusForPromptStatus(req.Status)}) {
		prompts = append(prompts, promptFromApproval(approval))
	}
	return prompts, nil
}

func (r *Runtime) MarkAgentBridgeEventRead(ctx context.Context, req MarkAgentBridgeEventReadRequest) (agentbridge.Event, error) {
	r.mu.Lock()
	next, event, err := r.agentBridges.MarkEventRead(agentbridge.MarkEventRead{ID: req.ID})
	if err != nil {
		r.mu.Unlock()
		return agentbridge.Event{}, err
	}
	r.agentBridges = next
	r.mu.Unlock()
	r.publish(ctx, RuntimeEvent{Type: EventAgentHookEventsChanged})
	return event, nil
}

func (r *Runtime) RecordAgentHookEvent(ctx context.Context, req AgentBridgeHookRequest) (agentbridge.Event, error) {
	event, err := r.recordAgentBridgeEvent(ctx, req, agentbridge.Bridge{}, "logged")
	if err != nil {
		return agentbridge.Event{}, err
	}
	return event, nil
}

func (r *Runtime) ResolveAgentBridgeApproval(ctx context.Context, req ResolveAgentBridgeApprovalRequest) (agentbridge.Approval, error) {
	decision := agentbridge.EvaluationDecision{
		Action: agentbridge.PolicyAction(req.Action),
		Reason: req.Reason,
	}
	r.mu.Lock()
	next, approval, err := r.agentBridges.ResolveApproval(agentbridge.ResolveApproval{
		ID:       req.ID,
		Decision: decision,
		Now:      time.Now().UTC(),
	})
	if err != nil {
		r.mu.Unlock()
		return agentbridge.Approval{}, err
	}
	r.agentBridges = next
	waiter := r.agentBridgeApprovalWaiters[req.ID]
	r.mu.Unlock()

	if waiter != nil {
		select {
		case waiter <- decision:
		default:
		}
	}
	r.publish(ctx, RuntimeEvent{Type: EventAgentBridgeApprovalsChanged})
	return approval, nil
}

func (r *Runtime) ResolveAgentPrompt(ctx context.Context, req ResolveAgentPromptRequest) (agentbridge.Prompt, error) {
	r.mu.Lock()
	next, prompt, err := r.agentBridges.ResolvePrompt(agentbridge.ResolvePrompt{
		ID:     req.ID,
		Answer: req.Answer,
		Now:    time.Now().UTC(),
	})
	if err != nil {
		r.mu.Unlock()
		approval, approvalErr := r.ResolveAgentBridgeApproval(ctx, ResolveAgentBridgeApprovalRequest{
			ID:     req.ID,
			Action: req.Answer,
		})
		if approvalErr != nil {
			return agentbridge.Prompt{}, approvalErr
		}
		return promptFromApproval(approval), nil
	}
	r.agentBridges = next
	waiter := r.agentPromptWaiters[req.ID]
	r.mu.Unlock()

	if waiter != nil {
		select {
		case waiter <- prompt.Answer:
		default:
		}
	}
	r.publish(ctx, RuntimeEvent{Type: EventAgentPromptsChanged})
	return prompt, nil
}

func approvalStatusForPromptStatus(status string) agentbridge.ApprovalStatus {
	switch agentbridge.PromptStatus(status) {
	case agentbridge.PromptPending:
		return agentbridge.ApprovalPending
	case agentbridge.PromptResolved:
		return agentbridge.ApprovalResolved
	case agentbridge.PromptTimedOut:
		return agentbridge.ApprovalTimedOut
	default:
		return ""
	}
}

func promptFromApproval(approval agentbridge.Approval) agentbridge.Prompt {
	message := approval.ToolName
	if command, ok := approval.ToolInput["command"].(string); ok && command != "" {
		message += ": " + command
	}
	status := agentbridge.PromptStatus(approval.Status)
	if approval.Status == agentbridge.ApprovalTimedOut {
		status = agentbridge.PromptTimedOut
	}
	return agentbridge.Prompt{
		ID:        approval.ID,
		BridgeID:  approval.BridgeID,
		SessionID: approval.SessionID,
		PTYID:     approval.PTYID,
		RunID:     approval.RunID,
		Provider:  approval.Provider,
		Kind:      agentbridge.PromptKindApproval,
		EventName: approval.EventName,
		ToolName:  approval.ToolName,
		ToolInput: approval.ToolInput,
		Message:   message,
		Options: []agentbridge.PromptOption{
			{Label: "Allow", Value: string(agentbridge.PolicyAllow)},
			{Label: "Deny", Value: string(agentbridge.PolicyDeny)},
		},
		Status:     status,
		Answer:     string(approval.Decision.Action),
		CreatedAt:  approval.CreatedAt,
		ResolvedAt: approval.ResolvedAt,
	}
}

func (r *Runtime) requestAgentBridgeApproval(ctx context.Context, bridge agentbridge.Bridge, req AgentBridgeHookRequest) (agentbridge.EvaluationDecision, error) {
	approvalID := r.ids()
	waiter := make(chan agentbridge.EvaluationDecision, 1)
	r.mu.Lock()
	next, approval, err := r.agentBridges.RecordPendingApproval(agentbridge.RecordPendingApproval{
		ID:        approvalID,
		BridgeID:  bridge.ID,
		EventName: req.EventName,
		ToolName:  req.ToolName,
		ToolInput: req.ToolInput,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		r.mu.Unlock()
		return agentbridge.EvaluationDecision{}, err
	}
	r.agentBridges = next
	r.agentBridgeApprovalWaiters[approval.ID] = waiter
	timeout := r.agentBridgeApprovalTimeout
	r.mu.Unlock()

	r.publish(ctx, RuntimeEvent{Type: EventAgentBridgeApprovalsChanged})

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	defer func() {
		r.mu.Lock()
		delete(r.agentBridgeApprovalWaiters, approval.ID)
		r.mu.Unlock()
	}()

	select {
	case decision := <-waiter:
		return decision, nil
	case <-timer.C:
		decision := agentbridge.EvaluationDecision{Action: agentbridge.PolicyDeny, Reason: "Approval timed out"}
		r.mu.Lock()
		next, _, err := r.agentBridges.TimeoutApproval(agentbridge.TimeoutApproval{
			ID:     approval.ID,
			Reason: decision.Reason,
			Now:    time.Now().UTC(),
		})
		if err == nil {
			r.agentBridges = next
		}
		r.mu.Unlock()
		r.publish(ctx, RuntimeEvent{Type: EventAgentBridgeApprovalsChanged})
		return decision, nil
	case <-ctx.Done():
		decision := agentbridge.EvaluationDecision{Action: agentbridge.PolicyDeny, Reason: "Approval cancelled"}
		r.mu.Lock()
		next, _, err := r.agentBridges.TimeoutApproval(agentbridge.TimeoutApproval{
			ID:     approval.ID,
			Reason: decision.Reason,
			Now:    time.Now().UTC(),
		})
		if err == nil {
			r.agentBridges = next
		}
		r.mu.Unlock()
		r.publish(ctx, RuntimeEvent{Type: EventAgentBridgeApprovalsChanged})
		return decision, nil
	}
}

func (r *Runtime) requestAgentPrompt(ctx context.Context, bridge agentbridge.Bridge, req AgentBridgeHookRequest) (string, error) {
	promptID := r.ids()
	waiter := make(chan string, 1)
	promptReq, ok := promptRequestFromHook(promptID, bridge, req)
	if !ok {
		if _, err := r.recordAgentBridgeEvent(ctx, req, bridge, "logged"); err != nil {
			return "", err
		}
		return "", nil
	}
	r.mu.Lock()
	next, prompt, err := r.agentBridges.RecordPendingPrompt(promptReq)
	if err != nil {
		r.mu.Unlock()
		return "", err
	}
	r.agentBridges = next
	r.agentPromptWaiters[prompt.ID] = waiter
	timeout := r.agentBridgeApprovalTimeout
	r.mu.Unlock()

	r.publish(ctx, RuntimeEvent{Type: EventAgentPromptsChanged})

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	defer func() {
		r.mu.Lock()
		delete(r.agentPromptWaiters, prompt.ID)
		r.mu.Unlock()
	}()

	select {
	case answer := <-waiter:
		return answer, nil
	case <-timer.C:
		r.mu.Lock()
		next, _, err := r.agentBridges.TimeoutPrompt(agentbridge.TimeoutPrompt{
			ID:     prompt.ID,
			Reason: "Prompt timed out",
			Now:    time.Now().UTC(),
		})
		if err == nil {
			r.agentBridges = next
		}
		r.mu.Unlock()
		r.publish(ctx, RuntimeEvent{Type: EventAgentPromptsChanged})
		return "", nil
	case <-ctx.Done():
		r.mu.Lock()
		next, _, err := r.agentBridges.TimeoutPrompt(agentbridge.TimeoutPrompt{
			ID:     prompt.ID,
			Reason: "Prompt cancelled",
			Now:    time.Now().UTC(),
		})
		if err == nil {
			r.agentBridges = next
		}
		r.mu.Unlock()
		r.publish(ctx, RuntimeEvent{Type: EventAgentPromptsChanged})
		return "", nil
	}
}

func promptRequestFromHook(id string, bridge agentbridge.Bridge, req AgentBridgeHookRequest) (agentbridge.RecordPendingPrompt, bool) {
	event := agentbridge.Event{
		ID:            id,
		BridgeID:      bridge.ID,
		SessionID:     bridge.SessionID,
		PTYID:         bridge.PTYID,
		Provider:      bridge.Provider,
		EventName:     req.EventName,
		ToolName:      req.ToolName,
		Message:       req.Message,
		ElicitationID: req.ElicitationID,
		Raw:           req.RawPayload,
	}
	normalized := agentbridge.NormalizeEvent(event)
	if !normalized.Answerable || normalized.Message == "" {
		return agentbridge.RecordPendingPrompt{}, false
	}
	options := make([]agentbridge.PromptOption, 0, len(normalized.Options))
	for _, option := range normalized.Options {
		options = append(options, agentbridge.PromptOption{Label: option.Label, Value: option.Value})
	}
	return agentbridge.RecordPendingPrompt{
		ID:            id,
		BridgeID:      bridge.ID,
		Kind:          agentbridge.PromptKindQuestion,
		EventName:     req.EventName,
		ToolName:      req.ToolName,
		ToolInput:     req.ToolInput,
		Message:       normalized.Message,
		CWD:           normalized.CWD,
		ElicitationID: req.ElicitationID,
		Options:       options,
		Now:           time.Now().UTC(),
	}, true
}

func (r *Runtime) recordAgentBridgeEvent(ctx context.Context, req AgentBridgeHookRequest, bridge agentbridge.Bridge, result string) (agentbridge.Event, error) {
	provider := agentbridge.Provider(req.Provider)
	if provider == "" {
		provider = bridge.Provider
	}
	eventID := r.ids()
	now := time.Now().UTC()
	r.mu.Lock()
	next, event, err := r.agentBridges.RecordEvent(agentbridge.RecordEvent{
		ID:               eventID,
		BridgeID:         firstNonEmpty(req.BridgeID, bridge.ID),
		SessionID:        firstNonEmpty(req.SessionID, bridge.SessionID),
		PTYID:            firstNonEmpty(req.PTYID, bridge.PTYID),
		Provider:         provider,
		EventName:        req.EventName,
		ToolName:         req.ToolName,
		Message:          req.Message,
		NotificationType: req.NotificationType,
		ElicitationID:    req.ElicitationID,
		Action:           req.Action,
		Result:           result,
		Raw:              req.RawPayload,
		Now:              now,
	})
	if err != nil {
		r.mu.Unlock()
		return agentbridge.Event{}, err
	}
	r.agentBridges = next
	logEnabled := r.agentHookLogEnabled
	r.mu.Unlock()

	if logEnabled {
		_ = r.appendAgentHookLog(event)
	}
	r.publish(ctx, RuntimeEvent{Type: EventAgentHookEventsChanged})
	return event, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (r *Runtime) registerAgentBridge(bridge agentbridge.Bridge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	next, err := r.agentBridges.AddBridge(bridge)
	if err != nil {
		return err
	}
	r.agentBridges = next
	return nil
}
