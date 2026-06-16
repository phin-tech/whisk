package agentbridge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Provider string

const (
	ProviderClaude Provider = "claude"
	ProviderCodex  Provider = "codex"
)

type Phase string

const (
	PhaseToolCall   Phase = "tool_call"
	PhaseToolResult Phase = "tool_result"
)

type PolicyAction string

const (
	PolicyAllow PolicyAction = "allow"
	PolicyDeny  PolicyAction = "deny"
	PolicyAsk   PolicyAction = "ask"
)

type QuestionStatus string

const (
	QuestionPending  QuestionStatus = "pending"
	QuestionResolved QuestionStatus = "resolved"
)

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalResolved ApprovalStatus = "resolved"
	ApprovalTimedOut ApprovalStatus = "timed_out"
)

type HookPayload struct {
	Provider   Provider
	EventName  string
	ToolName   string
	ToolInput  map[string]any
	ToolOutput string
}

type EvaluationRequest struct {
	Phase      Phase
	Provider   Provider
	ToolName   string
	ToolInput  map[string]any
	ToolOutput string
}

type EvaluationDecision struct {
	Action PolicyAction
	Reason string
}

type Bridge struct {
	ID        string
	SessionID string
	PTYID     string
	Provider  Provider
	TokenHash string
}

type Question struct {
	ID       string
	BridgeID string
	RunID    string
	Prompt   string
	Answer   string
	Status   QuestionStatus
}

type Approval struct {
	ID         string
	BridgeID   string
	SessionID  string
	PTYID      string
	RunID      string
	Provider   Provider
	EventName  string
	ToolName   string
	ToolInput  map[string]any
	Status     ApprovalStatus
	Decision   EvaluationDecision
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

type EventStatus string

const (
	EventPending EventStatus = "pending"
	EventRead    EventStatus = "read"
)

type Event struct {
	ID               string
	BridgeID         string
	SessionID        string
	PTYID            string
	Provider         Provider
	EventName        string
	ToolName         string
	Message          string
	NotificationType string
	ElicitationID    string
	Action           string
	Result           string
	Status           EventStatus
	CreatedAt        time.Time
	Raw              map[string]any
}

type RecordEvent struct {
	ID               string
	BridgeID         string
	SessionID        string
	PTYID            string
	Provider         Provider
	EventName        string
	ToolName         string
	Message          string
	NotificationType string
	ElicitationID    string
	Action           string
	Result           string
	Raw              map[string]any
	Now              time.Time
}

type ListEvents struct {
	Status EventStatus
}

type RecordPendingQuestion struct {
	ID       string
	BridgeID string
	RunID    string
	Prompt   string
}

type ResolveQuestion struct {
	ID     string
	Answer string
}

type RecordPendingApproval struct {
	ID        string
	BridgeID  string
	RunID     string
	EventName string
	ToolName  string
	ToolInput map[string]any
	Now       time.Time
}

type ResolveApproval struct {
	ID       string
	Decision EvaluationDecision
	Now      time.Time
}

type TimeoutApproval struct {
	ID     string
	Reason string
	Now    time.Time
}

type ListApprovals struct {
	Status ApprovalStatus
}

type State struct {
	bridges   map[string]Bridge
	questions map[string]Question
	approvals map[string]Approval
	events    map[string]Event
}

func HookPayloadToEvaluationRequest(payload HookPayload) (EvaluationRequest, bool) {
	if strings.HasPrefix(payload.ToolName, "mcp__whisk__") {
		return EvaluationRequest{}, false
	}
	input := payload.ToolInput
	if input == nil {
		input = map[string]any{}
	}
	switch payload.EventName {
	case "PreToolUse", "PermissionRequest":
		return EvaluationRequest{
			Phase:     PhaseToolCall,
			Provider:  payload.Provider,
			ToolName:  payload.ToolName,
			ToolInput: cloneMap(input),
		}, true
	case "PostToolUse":
		return EvaluationRequest{
			Phase:      PhaseToolResult,
			Provider:   payload.Provider,
			ToolName:   payload.ToolName,
			ToolInput:  cloneMap(input),
			ToolOutput: payload.ToolOutput,
		}, true
	default:
		return EvaluationRequest{}, false
	}
}

func EvaluationDecisionToProviderOutput(_ Provider, eventName string, decision EvaluationDecision) (map[string]any, bool) {
	switch eventName {
	case "PreToolUse", "PermissionRequest":
		switch decision.Action {
		case PolicyDeny:
			output := map[string]any{
				"hookEventName":      eventName,
				"permissionDecision": "deny",
			}
			if strings.TrimSpace(decision.Reason) != "" {
				output["permissionDecisionReason"] = decision.Reason
			}
			return map[string]any{"hookSpecificOutput": output}, true
		case PolicyAsk:
			output := map[string]any{
				"hookEventName":      eventName,
				"permissionDecision": "deny",
			}
			if strings.TrimSpace(decision.Reason) != "" {
				output["permissionDecisionReason"] = decision.Reason
			}
			return map[string]any{"hookSpecificOutput": output}, true
		default:
			return nil, false
		}
	case "PostToolUse":
		if decision.Action != PolicyDeny || strings.TrimSpace(decision.Reason) == "" {
			return nil, false
		}
		return map[string]any{
			"hookSpecificOutput": map[string]any{
				"hookEventName":     "PostToolUse",
				"additionalContext": "[Policy violation] " + decision.Reason,
			},
		}, true
	default:
		return nil, false
	}
}

func HashHookToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func NewState(bridges ...Bridge) (State, error) {
	state := State{
		bridges:   map[string]Bridge{},
		questions: map[string]Question{},
		approvals: map[string]Approval{},
		events:    map[string]Event{},
	}
	for _, bridge := range bridges {
		if strings.TrimSpace(bridge.ID) == "" {
			return State{}, fmt.Errorf("bridge id required")
		}
		if bridge.Provider != ProviderClaude && bridge.Provider != ProviderCodex {
			return State{}, fmt.Errorf("unsupported bridge provider %q", bridge.Provider)
		}
		if strings.TrimSpace(bridge.TokenHash) == "" {
			return State{}, fmt.Errorf("bridge token hash required")
		}
		state.bridges[bridge.ID] = bridge
	}
	return state, nil
}

func (s State) RecordEvent(req RecordEvent) (State, Event, error) {
	if strings.TrimSpace(req.ID) == "" {
		return State{}, Event{}, fmt.Errorf("event id required")
	}
	if strings.TrimSpace(req.EventName) == "" {
		return State{}, Event{}, fmt.Errorf("event name required")
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	provider := req.Provider
	sessionID := strings.TrimSpace(req.SessionID)
	ptyID := strings.TrimSpace(req.PTYID)
	if bridge, ok := s.bridges[req.BridgeID]; ok {
		provider = bridge.Provider
		sessionID = bridge.SessionID
		ptyID = bridge.PTYID
	}
	next := s.clone()
	event := Event{
		ID:               strings.TrimSpace(req.ID),
		BridgeID:         strings.TrimSpace(req.BridgeID),
		SessionID:        sessionID,
		PTYID:            ptyID,
		Provider:         provider,
		EventName:        strings.TrimSpace(req.EventName),
		ToolName:         strings.TrimSpace(req.ToolName),
		Message:          strings.TrimSpace(req.Message),
		NotificationType: strings.TrimSpace(req.NotificationType),
		ElicitationID:    strings.TrimSpace(req.ElicitationID),
		Action:           strings.TrimSpace(req.Action),
		Result:           strings.TrimSpace(req.Result),
		Status:           EventPending,
		CreatedAt:        now,
		Raw:              cloneMap(req.Raw),
	}
	next.events[event.ID] = event
	return next, event, nil
}

func (s State) ListEvents(filter ListEvents) []Event {
	out := make([]Event, 0, len(s.events))
	for _, event := range s.events {
		if filter.Status != "" && event.Status != filter.Status {
			continue
		}
		out = append(out, cloneEvent(event))
	}
	return out
}

func (s State) AddBridge(bridge Bridge) (State, error) {
	if strings.TrimSpace(bridge.ID) == "" {
		return State{}, fmt.Errorf("bridge id required")
	}
	if bridge.Provider != ProviderClaude && bridge.Provider != ProviderCodex {
		return State{}, fmt.Errorf("unsupported bridge provider %q", bridge.Provider)
	}
	if strings.TrimSpace(bridge.TokenHash) == "" {
		return State{}, fmt.Errorf("bridge token hash required")
	}
	next := s.clone()
	next.bridges[bridge.ID] = bridge
	return next, nil
}

func (s State) ValidateHookToken(bridgeID string, token string) bool {
	bridge, ok := s.bridges[bridgeID]
	return ok && bridge.TokenHash == HashHookToken(token)
}

func (s State) Bridge(id string) (Bridge, bool) {
	bridge, ok := s.bridges[id]
	return bridge, ok
}

func (s State) RecordPendingQuestion(req RecordPendingQuestion) (State, Question, error) {
	if _, ok := s.bridges[req.BridgeID]; !ok {
		return State{}, Question{}, fmt.Errorf("bridge %s not found", req.BridgeID)
	}
	if strings.TrimSpace(req.ID) == "" {
		return State{}, Question{}, fmt.Errorf("question id required")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return State{}, Question{}, fmt.Errorf("question prompt required")
	}
	next := s.clone()
	question := Question{
		ID:       req.ID,
		BridgeID: req.BridgeID,
		RunID:    strings.TrimSpace(req.RunID),
		Prompt:   strings.TrimSpace(req.Prompt),
		Status:   QuestionPending,
	}
	next.questions[question.ID] = question
	return next, question, nil
}

func (s State) RecordPendingApproval(req RecordPendingApproval) (State, Approval, error) {
	bridge, ok := s.bridges[req.BridgeID]
	if !ok {
		return State{}, Approval{}, fmt.Errorf("bridge %s not found", req.BridgeID)
	}
	if strings.TrimSpace(req.ID) == "" {
		return State{}, Approval{}, fmt.Errorf("approval id required")
	}
	if strings.TrimSpace(req.EventName) == "" {
		return State{}, Approval{}, fmt.Errorf("approval event name required")
	}
	if strings.TrimSpace(req.ToolName) == "" {
		return State{}, Approval{}, fmt.Errorf("approval tool name required")
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	next := s.clone()
	approval := Approval{
		ID:        strings.TrimSpace(req.ID),
		BridgeID:  req.BridgeID,
		SessionID: bridge.SessionID,
		PTYID:     bridge.PTYID,
		RunID:     strings.TrimSpace(req.RunID),
		Provider:  bridge.Provider,
		EventName: strings.TrimSpace(req.EventName),
		ToolName:  strings.TrimSpace(req.ToolName),
		ToolInput: cloneMap(req.ToolInput),
		Status:    ApprovalPending,
		CreatedAt: now,
	}
	next.approvals[approval.ID] = approval
	return next, approval, nil
}

func (s State) ResolveApproval(req ResolveApproval) (State, Approval, error) {
	approval, ok := s.approvals[req.ID]
	if !ok {
		return State{}, Approval{}, fmt.Errorf("approval %s not found", req.ID)
	}
	if approval.Status != ApprovalPending {
		return State{}, Approval{}, fmt.Errorf("approval %s is not pending", req.ID)
	}
	switch req.Decision.Action {
	case PolicyAllow, PolicyDeny:
	default:
		return State{}, Approval{}, fmt.Errorf("approval decision action required")
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	next := s.clone()
	approval.Status = ApprovalResolved
	approval.Decision = req.Decision
	approval.ResolvedAt = &now
	next.approvals[approval.ID] = approval
	return next, approval, nil
}

func (s State) TimeoutApproval(req TimeoutApproval) (State, Approval, error) {
	approval, ok := s.approvals[req.ID]
	if !ok {
		return State{}, Approval{}, fmt.Errorf("approval %s not found", req.ID)
	}
	if approval.Status != ApprovalPending {
		return State{}, Approval{}, fmt.Errorf("approval %s is not pending", req.ID)
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "Approval timed out"
	}
	next := s.clone()
	approval.Status = ApprovalTimedOut
	approval.Decision = EvaluationDecision{Action: PolicyDeny, Reason: reason}
	approval.ResolvedAt = &now
	next.approvals[approval.ID] = approval
	return next, approval, nil
}

func (s State) ListApprovals(filter ListApprovals) []Approval {
	out := make([]Approval, 0, len(s.approvals))
	for _, approval := range s.approvals {
		if filter.Status != "" && approval.Status != filter.Status {
			continue
		}
		out = append(out, cloneApproval(approval))
	}
	return out
}

func (s State) ResolveQuestion(req ResolveQuestion) (State, error) {
	question, ok := s.questions[req.ID]
	if !ok {
		return State{}, fmt.Errorf("question %s not found", req.ID)
	}
	answer := strings.TrimSpace(req.Answer)
	if answer == "" {
		return State{}, fmt.Errorf("question answer required")
	}
	next := s.clone()
	question.Answer = answer
	question.Status = QuestionResolved
	next.questions[question.ID] = question
	return next, nil
}

func (s State) Question(id string) (Question, bool) {
	question, ok := s.questions[id]
	return question, ok
}

func (s State) clone() State {
	next := State{
		bridges:   map[string]Bridge{},
		questions: map[string]Question{},
		approvals: map[string]Approval{},
		events:    map[string]Event{},
	}
	for id, bridge := range s.bridges {
		next.bridges[id] = bridge
	}
	for id, question := range s.questions {
		next.questions[id] = question
	}
	for id, approval := range s.approvals {
		next.approvals[id] = cloneApproval(approval)
	}
	for id, event := range s.events {
		next.events[id] = cloneEvent(event)
	}
	return next
}

func cloneMap(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func cloneApproval(approval Approval) Approval {
	approval.ToolInput = cloneMap(approval.ToolInput)
	if approval.ResolvedAt != nil {
		resolvedAt := *approval.ResolvedAt
		approval.ResolvedAt = &resolvedAt
	}
	return approval
}

func cloneEvent(event Event) Event {
	event.Raw = cloneMap(event.Raw)
	return event
}
