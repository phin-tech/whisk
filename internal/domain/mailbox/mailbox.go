package mailbox

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	AddressKindPTY           = "pty"
	AddressKindRun           = "run"
	AddressKindSession       = "session"
	AddressKindWorkItem      = "work-item"
	AddressKindProject       = "project"
	AddressKindProjectGroup  = "@project"
	AddressKindWorkItemGroup = "@work-item"
	AddressKindIdleGroup     = "@idle"

	TypeStatus       = "status"
	TypeDispatch     = "dispatch"
	TypeWorkerDone   = "worker_done"
	TypeEscalation   = "escalation"
	TypeHandoff      = "handoff"
	TypeDecisionGate = "decision_gate"
	TypeHeartbeat    = "heartbeat"

	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"

	MaxSubjectBytes = 240
	MaxBodyBytes    = 64 * 1024
	MaxPayloadBytes = 64 * 1024
)

var supportedTypes = map[string]struct{}{
	TypeStatus:       {},
	TypeDispatch:     {},
	TypeWorkerDone:   {},
	TypeEscalation:   {},
	TypeHandoff:      {},
	TypeDecisionGate: {},
	TypeHeartbeat:    {},
}

var supportedPriorities = map[string]struct{}{
	PriorityLow:    {},
	PriorityNormal: {},
	PriorityHigh:   {},
	PriorityUrgent: {},
}

type Address struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func ParseAddress(raw string) (Address, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return Address{}, fmt.Errorf("mail address required")
	}
	kind, id, ok := strings.Cut(value, ":")
	if !ok {
		if selectorKind := normalizeAddressKind(value); selectorKind == AddressKindIdleGroup {
			return Address{}, Address{Kind: selectorKind}.Validate()
		}
		return Address{}, fmt.Errorf("mail address must be kind:id")
	}
	address := Address{Kind: normalizeAddressKind(kind), ID: strings.TrimSpace(id)}
	if err := address.Validate(); err != nil {
		return Address{}, err
	}
	return address, nil
}

func ParseAddressList(raw string) ([]Address, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	addresses := make([]Address, 0, len(parts))
	for _, part := range parts {
		address, err := ParseAddress(part)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, address)
	}
	return DeduplicateAddresses(addresses), nil
}

func (a Address) String() string {
	if a.Kind == "" && a.ID == "" {
		return ""
	}
	return a.Kind + ":" + a.ID
}

func (a Address) Validate() error {
	if a.Kind == "" {
		return fmt.Errorf("mail address kind required")
	}
	if normalizeAddressKind(a.Kind) != a.Kind {
		return fmt.Errorf("unsupported mail address kind %q", a.Kind)
	}
	if a.Kind == AddressKindIdleGroup {
		return fmt.Errorf("mail group selector @idle is deferred until agent status exists")
	}
	if a.ID == "" {
		return fmt.Errorf("mail address id required")
	}
	for _, r := range a.ID {
		if unicode.IsSpace(r) || r == ',' || r == ':' {
			return fmt.Errorf("invalid mail address id %q", a.ID)
		}
	}
	return nil
}

func (a Address) ValidateConcrete() error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.IsGroupSelector() {
		return fmt.Errorf("mail group selector %s must be expanded before storage", a.Kind)
	}
	return nil
}

func (a Address) IsGroupSelector() bool {
	return strings.HasPrefix(a.Kind, "@")
}

func DeduplicateAddresses(addresses []Address) []Address {
	if len(addresses) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]Address, 0, len(addresses))
	for _, address := range addresses {
		key := address.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, address)
	}
	return out
}

type Recipient struct {
	Address Address    `json:"address"`
	ReadAt  *time.Time `json:"readAt,omitempty"`
}

type Message struct {
	ID         string          `json:"id"`
	ThreadID   string          `json:"threadId,omitempty"`
	ReplyToID  string          `json:"replyToId,omitempty"`
	From       Address         `json:"from"`
	Recipients []Recipient     `json:"recipients"`
	Type       string          `json:"type"`
	Priority   string          `json:"priority"`
	Subject    string          `json:"subject,omitempty"`
	Body       string          `json:"body,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	ProjectID  string          `json:"projectId,omitempty"`
	WorkItemID string          `json:"workItemId,omitempty"`
	RunID      string          `json:"runId,omitempty"`
	SessionID  string          `json:"sessionId,omitempty"`
	PTYID      string          `json:"ptyId,omitempty"`
	DispatchID string          `json:"dispatchId,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
}

type Send struct {
	ID         string
	ThreadID   string
	ReplyToID  string
	From       Address
	To         []Address
	Type       string
	Priority   string
	Subject    string
	Body       string
	Payload    json.RawMessage
	ProjectID  string
	WorkItemID string
	RunID      string
	SessionID  string
	PTYID      string
	DispatchID string
	Now        time.Time
}

type Reply struct {
	ID       string
	Original Message
	From     Address
	Type     string
	Priority string
	Subject  string
	Body     string
	Payload  json.RawMessage
	Now      time.Time
}

type ListFilter struct {
	ID          string
	To          []Address
	UnreadOnly  bool
	Types       []string
	ProjectID   string
	WorkItemID  string
	RunID       string
	ThreadID    string
	Limit       int
	OldestFirst bool
}

type MarkRead struct {
	ID        string
	Recipient *Address
	Now       time.Time
}

func NewMessage(req Send) (Message, error) {
	if req.ID == "" {
		return Message{}, fmt.Errorf("mail id required")
	}
	messageType, err := NormalizeType(req.Type)
	if err != nil {
		return Message{}, err
	}
	priority, err := NormalizePriority(req.Priority)
	if err != nil {
		return Message{}, err
	}
	if err := req.From.ValidateConcrete(); err != nil {
		return Message{}, fmt.Errorf("from: %w", err)
	}
	to := DeduplicateAddresses(req.To)
	if len(to) == 0 {
		return Message{}, fmt.Errorf("mail recipient required")
	}
	recipients := make([]Recipient, 0, len(to))
	for _, address := range to {
		if err := address.ValidateConcrete(); err != nil {
			return Message{}, fmt.Errorf("recipient: %w", err)
		}
		recipients = append(recipients, Recipient{Address: address})
	}
	if messageType != TypeHeartbeat && strings.TrimSpace(req.Subject) == "" {
		return Message{}, fmt.Errorf("mail subject required")
	}
	if len([]byte(req.Subject)) > MaxSubjectBytes {
		return Message{}, fmt.Errorf("mail subject exceeds %d bytes", MaxSubjectBytes)
	}
	if len([]byte(req.Body)) > MaxBodyBytes {
		return Message{}, fmt.Errorf("mail body exceeds %d bytes", MaxBodyBytes)
	}
	payload, err := normalizePayload(req.Payload)
	if err != nil {
		return Message{}, err
	}
	createdAt := req.Now
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	return Message{
		ID:         req.ID,
		ThreadID:   req.ThreadID,
		ReplyToID:  req.ReplyToID,
		From:       req.From,
		Recipients: recipients,
		Type:       messageType,
		Priority:   priority,
		Subject:    req.Subject,
		Body:       req.Body,
		Payload:    payload,
		ProjectID:  req.ProjectID,
		WorkItemID: req.WorkItemID,
		RunID:      req.RunID,
		SessionID:  req.SessionID,
		PTYID:      req.PTYID,
		DispatchID: req.DispatchID,
		CreatedAt:  createdAt.UTC(),
	}, nil
}

func NewReply(req Reply) (Message, error) {
	if req.Original.ID == "" {
		return Message{}, fmt.Errorf("reply target required")
	}
	threadID := req.Original.ThreadID
	if threadID == "" {
		threadID = req.Original.ID
	}
	subject := req.Subject
	if strings.TrimSpace(subject) == "" && req.Type != TypeHeartbeat {
		if req.Original.Subject == "" {
			subject = "Re: " + req.Original.ID
		} else if strings.HasPrefix(strings.ToLower(req.Original.Subject), "re: ") {
			subject = req.Original.Subject
		} else {
			subject = "Re: " + req.Original.Subject
		}
	}
	return NewMessage(Send{
		ID:         req.ID,
		ThreadID:   threadID,
		ReplyToID:  req.Original.ID,
		From:       req.From,
		To:         []Address{req.Original.From},
		Type:       req.Type,
		Priority:   req.Priority,
		Subject:    subject,
		Body:       req.Body,
		Payload:    req.Payload,
		ProjectID:  req.Original.ProjectID,
		WorkItemID: req.Original.WorkItemID,
		RunID:      req.Original.RunID,
		SessionID:  req.Original.SessionID,
		PTYID:      req.Original.PTYID,
		DispatchID: req.Original.DispatchID,
		Now:        req.Now,
	})
}

func MarkMessageRead(message Message, recipient *Address, at time.Time) (Message, error) {
	if message.ID == "" {
		return Message{}, fmt.Errorf("mail id required")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	at = at.UTC()
	updated := copyMessage(message)
	marked := false
	for i := range updated.Recipients {
		if recipient != nil && updated.Recipients[i].Address != *recipient {
			continue
		}
		if updated.Recipients[i].ReadAt == nil {
			readAt := at
			updated.Recipients[i].ReadAt = &readAt
		}
		marked = true
	}
	if !marked {
		return Message{}, fmt.Errorf("mail recipient not found")
	}
	return updated, nil
}

func NormalizeType(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "", fmt.Errorf("mail type required")
	}
	if _, ok := supportedTypes[normalized]; !ok {
		return "", fmt.Errorf("unsupported mail type %q", value)
	}
	return normalized, nil
}

func NormalizePriority(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = PriorityNormal
	}
	if _, ok := supportedPriorities[normalized]; !ok {
		return "", fmt.Errorf("unsupported mail priority %q", value)
	}
	return normalized, nil
}

func NormalizeListFilter(filter ListFilter) (ListFilter, error) {
	out := filter
	out.To = DeduplicateAddresses(out.To)
	for _, address := range out.To {
		if err := address.ValidateConcrete(); err != nil {
			return ListFilter{}, err
		}
	}
	if len(out.Types) > 0 {
		types := make([]string, 0, len(out.Types))
		for _, value := range out.Types {
			normalized, err := NormalizeType(value)
			if err != nil {
				return ListFilter{}, err
			}
			types = append(types, normalized)
		}
		sort.Strings(types)
		out.Types = compactStrings(types)
	}
	if out.Limit < 0 {
		return ListFilter{}, fmt.Errorf("mail limit must be non-negative")
	}
	return out, nil
}

func copyMessage(message Message) Message {
	out := message
	if message.Payload != nil {
		out.Payload = append(json.RawMessage(nil), message.Payload...)
	}
	if message.Recipients != nil {
		out.Recipients = make([]Recipient, len(message.Recipients))
		copy(out.Recipients, message.Recipients)
		for i := range out.Recipients {
			if out.Recipients[i].ReadAt != nil {
				readAt := *out.Recipients[i].ReadAt
				out.Recipients[i].ReadAt = &readAt
			}
		}
	}
	return out
}

func normalizePayload(payload json.RawMessage) (json.RawMessage, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	if len(payload) > MaxPayloadBytes {
		return nil, fmt.Errorf("mail payload exceeds %d bytes", MaxPayloadBytes)
	}
	if !json.Valid(payload) {
		return nil, fmt.Errorf("mail payload must be valid JSON")
	}
	return append(json.RawMessage(nil), payload...), nil
}

func normalizeAddressKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case AddressKindPTY:
		return AddressKindPTY
	case AddressKindRun:
		return AddressKindRun
	case AddressKindSession:
		return AddressKindSession
	case AddressKindProject:
		return AddressKindProject
	case AddressKindWorkItem, "workitem", "work_item":
		return AddressKindWorkItem
	case AddressKindProjectGroup:
		return AddressKindProjectGroup
	case AddressKindWorkItemGroup, "@workitem", "@work_item":
		return AddressKindWorkItemGroup
	case AddressKindIdleGroup:
		return AddressKindIdleGroup
	default:
		return ""
	}
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := values[:0]
	for _, value := range values {
		if value == "" {
			continue
		}
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}
