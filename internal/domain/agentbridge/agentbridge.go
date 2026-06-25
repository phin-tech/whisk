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

type PromptKind string

const (
	PromptKindQuestion PromptKind = "question"
	PromptKindApproval PromptKind = "approval"
)

type PromptStatus string

const (
	PromptPending  PromptStatus = "pending"
	PromptResolved PromptStatus = "resolved"
	PromptTimedOut PromptStatus = "timed_out"
	PromptCanceled PromptStatus = "cancelled"
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
	RunID     string
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

type PromptOption struct {
	Label string
	Value string
}

type Prompt struct {
	ID            string
	BridgeID      string
	SessionID     string
	PTYID         string
	RunID         string
	Provider      Provider
	Kind          PromptKind
	EventName     string
	ToolName      string
	ToolInput     map[string]any
	Message       string
	CWD           string
	ElicitationID string
	Options       []PromptOption
	Status        PromptStatus
	Answer        string
	CreatedAt     time.Time
	ResolvedAt    *time.Time
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

type MarkEventRead struct {
	ID string
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

type RecordPendingPrompt struct {
	ID            string
	BridgeID      string
	RunID         string
	Kind          PromptKind
	EventName     string
	ToolName      string
	ToolInput     map[string]any
	Message       string
	CWD           string
	ElicitationID string
	Options       []PromptOption
	Now           time.Time
}

type ResolvePrompt struct {
	ID     string
	Answer string
	Now    time.Time
}

type TimeoutPrompt struct {
	ID     string
	Reason string
	Now    time.Time
}

type ListPrompts struct {
	Status PromptStatus
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
	prompts   map[string]Prompt
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
	case "PreToolUse":
		if payload.Provider == ProviderClaude {
			return EvaluationRequest{}, false
		}
		return EvaluationRequest{
			Phase:     PhaseToolCall,
			Provider:  payload.Provider,
			ToolName:  payload.ToolName,
			ToolInput: cloneMap(input),
		}, true
	case "PermissionRequest":
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

func PromptAnswerToProviderOutput(_ Provider, eventName string, elicitationID string, answer string, toolInputs ...map[string]any) (map[string]any, bool) {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return nil, false
	}
	if eventName == "PreToolUse" || eventName == "PermissionRequest" {
		if len(toolInputs) == 0 {
			return nil, false
		}
		question := firstAskUserQuestion(toolInputs[0])
		if question == "" {
			return nil, false
		}
		updatedInput := cloneMap(toolInputs[0])
		updatedInput["answers"] = map[string]any{question: answer}
		return map[string]any{
			"hookSpecificOutput": map[string]any{
				"hookEventName":      eventName,
				"permissionDecision": "allow",
				"updatedInput":       updatedInput,
			},
		}, true
	}
	if eventName != "Elicitation" {
		return nil, false
	}
	output := map[string]any{
		"hookEventName": eventName,
		"decision":      answer,
	}
	if strings.TrimSpace(elicitationID) != "" {
		output["elicitationId"] = strings.TrimSpace(elicitationID)
	}
	return map[string]any{"hookSpecificOutput": output}, true
}

func firstAskUserQuestion(toolInput map[string]any) string {
	values, _ := toolInput["questions"].([]any)
	if len(values) == 0 {
		return ""
	}
	question, _ := values[0].(map[string]any)
	value, _ := question["question"].(string)
	return strings.TrimSpace(value)
}

func HashHookToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func NewState(bridges ...Bridge) (State, error) {
	state := State{
		bridges:   map[string]Bridge{},
		questions: map[string]Question{},
		prompts:   map[string]Prompt{},
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

func (s State) MarkEventRead(req MarkEventRead) (State, Event, error) {
	event, ok := s.events[req.ID]
	if !ok {
		return State{}, Event{}, fmt.Errorf("event %s not found", req.ID)
	}
	next := s.clone()
	event.Status = EventRead
	next.events[event.ID] = event
	return next, cloneEvent(event), nil
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
	bridge, ok := s.bridges[req.BridgeID]
	if !ok {
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
		RunID:    firstNonEmptyString(bridge.RunID, req.RunID),
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
		RunID:     firstNonEmptyString(bridge.RunID, req.RunID),
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

func (s State) RecordPendingPrompt(req RecordPendingPrompt) (State, Prompt, error) {
	bridge, ok := s.bridges[req.BridgeID]
	if !ok {
		return State{}, Prompt{}, fmt.Errorf("bridge %s not found", req.BridgeID)
	}
	if strings.TrimSpace(req.ID) == "" {
		return State{}, Prompt{}, fmt.Errorf("prompt id required")
	}
	if req.Kind == "" {
		return State{}, Prompt{}, fmt.Errorf("prompt kind required")
	}
	if strings.TrimSpace(req.EventName) == "" {
		return State{}, Prompt{}, fmt.Errorf("prompt event name required")
	}
	if strings.TrimSpace(req.Message) == "" {
		return State{}, Prompt{}, fmt.Errorf("prompt message required")
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	next := s.clone()
	prompt := Prompt{
		ID:            strings.TrimSpace(req.ID),
		BridgeID:      strings.TrimSpace(req.BridgeID),
		SessionID:     bridge.SessionID,
		PTYID:         bridge.PTYID,
		RunID:         firstNonEmptyString(bridge.RunID, req.RunID),
		Provider:      bridge.Provider,
		Kind:          req.Kind,
		EventName:     strings.TrimSpace(req.EventName),
		ToolName:      strings.TrimSpace(req.ToolName),
		ToolInput:     cloneMap(req.ToolInput),
		Message:       strings.TrimSpace(req.Message),
		CWD:           strings.TrimSpace(req.CWD),
		ElicitationID: strings.TrimSpace(req.ElicitationID),
		Options:       clonePromptOptions(req.Options),
		Status:        PromptPending,
		CreatedAt:     now,
	}
	next.prompts[prompt.ID] = prompt
	return next, prompt, nil
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

func (s State) ListPrompts(filter ListPrompts) []Prompt {
	out := make([]Prompt, 0, len(s.prompts))
	for _, prompt := range s.prompts {
		if filter.Status != "" && prompt.Status != filter.Status {
			continue
		}
		out = append(out, clonePrompt(prompt))
	}
	return out
}

func (s State) ResolvePrompt(req ResolvePrompt) (State, Prompt, error) {
	prompt, ok := s.prompts[req.ID]
	if !ok {
		return State{}, Prompt{}, fmt.Errorf("prompt %s not found", req.ID)
	}
	if prompt.Status != PromptPending {
		return State{}, Prompt{}, fmt.Errorf("prompt %s is not pending", req.ID)
	}
	answer := strings.TrimSpace(req.Answer)
	if answer == "" {
		return State{}, Prompt{}, fmt.Errorf("prompt answer required")
	}
	if len(prompt.Options) > 0 && !promptHasOption(prompt, answer) {
		return State{}, Prompt{}, fmt.Errorf("prompt answer %q is not an option", answer)
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	next := s.clone()
	prompt.Status = PromptResolved
	prompt.Answer = answer
	prompt.ResolvedAt = &now
	next.prompts[prompt.ID] = prompt
	return next, clonePrompt(prompt), nil
}

func (s State) TimeoutPrompt(req TimeoutPrompt) (State, Prompt, error) {
	prompt, ok := s.prompts[req.ID]
	if !ok {
		return State{}, Prompt{}, fmt.Errorf("prompt %s not found", req.ID)
	}
	if prompt.Status != PromptPending {
		return State{}, Prompt{}, fmt.Errorf("prompt %s is not pending", req.ID)
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "Prompt timed out"
	}
	next := s.clone()
	prompt.Status = PromptTimedOut
	prompt.Answer = reason
	prompt.ResolvedAt = &now
	next.prompts[prompt.ID] = prompt
	return next, clonePrompt(prompt), nil
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
		prompts:   map[string]Prompt{},
		approvals: map[string]Approval{},
		events:    map[string]Event{},
	}
	for id, bridge := range s.bridges {
		next.bridges[id] = bridge
	}
	for id, question := range s.questions {
		next.questions[id] = question
	}
	for id, prompt := range s.prompts {
		next.prompts[id] = clonePrompt(prompt)
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

func clonePrompt(prompt Prompt) Prompt {
	prompt.ToolInput = cloneMap(prompt.ToolInput)
	prompt.Options = clonePromptOptions(prompt.Options)
	if prompt.ResolvedAt != nil {
		resolvedAt := *prompt.ResolvedAt
		prompt.ResolvedAt = &resolvedAt
	}
	return prompt
}

func clonePromptOptions(options []PromptOption) []PromptOption {
	if len(options) == 0 {
		return nil
	}
	out := make([]PromptOption, len(options))
	copy(out, options)
	return out
}

func promptHasOption(prompt Prompt, answer string) bool {
	for _, option := range prompt.Options {
		if option.Value == answer {
			return true
		}
	}
	return false
}

func cloneEvent(event Event) Event {
	event.Raw = cloneMap(event.Raw)
	return event
}
