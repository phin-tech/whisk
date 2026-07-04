package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/phin-tech/whisk/internal/domain/mailbox"
	"github.com/phin-tech/whisk/internal/domain/workitem"
)

type SendMailRequest struct {
	From       mailbox.Address
	Recipients []mailbox.Address
	Type       string
	Priority   string
	Subject    string
	Body       string
	Payload    json.RawMessage
	ThreadID   string
	ReplyToID  string
	ProjectID  string
	WorkItemID string
	RunID      string
	SessionID  string
	PTYID      string
	DispatchID string
}

type ListMailRequest struct {
	To          []mailbox.Address
	UnreadOnly  bool
	Types       []string
	ProjectID   string
	WorkItemID  string
	RunID       string
	ThreadID    string
	Limit       int
	OldestFirst bool
}

type NextMailRequest struct {
	To        []mailbox.Address
	Types     []string
	Timeout   time.Duration
	ProjectID string
}

type NextMailResult struct {
	Message *mailbox.Message
	Timeout bool
}

type MarkMailReadRequest struct {
	ID        string
	Recipient *mailbox.Address
}

type ReplyMailRequest struct {
	ID       string
	From     mailbox.Address
	Type     string
	Priority string
	Subject  string
	Body     string
	Payload  json.RawMessage
}

func (r *Runtime) SendMail(ctx context.Context, req SendMailRequest) (mailbox.Message, error) {
	if r.mailboxStore == nil {
		return mailbox.Message{}, fmt.Errorf("mailbox store unavailable")
	}
	recipients, err := r.expandMailRecipients(req.Recipients)
	if err != nil {
		return mailbox.Message{}, err
	}
	message, err := mailbox.NewMessage(mailbox.Send{
		ID:         r.ids(),
		ThreadID:   req.ThreadID,
		ReplyToID:  req.ReplyToID,
		From:       req.From,
		To:         recipients,
		Type:       req.Type,
		Priority:   req.Priority,
		Subject:    req.Subject,
		Body:       req.Body,
		Payload:    req.Payload,
		ProjectID:  req.ProjectID,
		WorkItemID: req.WorkItemID,
		RunID:      req.RunID,
		SessionID:  req.SessionID,
		PTYID:      req.PTYID,
		DispatchID: req.DispatchID,
		Now:        time.Now().UTC(),
	})
	if err != nil {
		return mailbox.Message{}, err
	}
	if err := r.mailboxStore.SaveMessage(ctx, message); err != nil {
		return mailbox.Message{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventMailboxChanged})
	return message, nil
}

func (r *Runtime) ListMail(ctx context.Context, req ListMailRequest) ([]mailbox.Message, error) {
	if r.mailboxStore == nil {
		return nil, fmt.Errorf("mailbox store unavailable")
	}
	return r.mailboxStore.ListMessages(ctx, mailbox.ListFilter{
		To:          req.To,
		UnreadOnly:  req.UnreadOnly,
		Types:       req.Types,
		ProjectID:   req.ProjectID,
		WorkItemID:  req.WorkItemID,
		RunID:       req.RunID,
		ThreadID:    req.ThreadID,
		Limit:       req.Limit,
		OldestFirst: req.OldestFirst,
	})
}

func (r *Runtime) NextMail(ctx context.Context, req NextMailRequest) (NextMailResult, error) {
	if r.mailboxStore == nil {
		return NextMailResult{}, fmt.Errorf("mailbox store unavailable")
	}
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}
	filter := mailbox.ListFilter{
		To:          req.To,
		UnreadOnly:  true,
		Types:       req.Types,
		ProjectID:   req.ProjectID,
		Limit:       1,
		OldestFirst: true,
	}
	afterSeq := r.currentEventSeq()
	message, err := r.nextUnreadMail(ctx, filter)
	if err != nil {
		return NextMailResult{}, err
	}
	if message != nil {
		return NextMailResult{Message: message}, nil
	}
	if req.Timeout <= 0 {
		return NextMailResult{Timeout: true}, nil
	}

	source, ok := r.eventSink.(EventSource)
	if !ok || source == nil {
		return NextMailResult{}, fmt.Errorf("runtime event source unavailable")
	}
	for {
		event, err := source.Next(ctx, afterSeq)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return NextMailResult{Timeout: true}, nil
			}
			return NextMailResult{}, err
		}
		afterSeq = event.Event.Seq
		if !event.Missed && event.Event.Type != EventMailboxChanged {
			continue
		}
		message, err := r.nextUnreadMail(ctx, filter)
		if err != nil {
			return NextMailResult{}, err
		}
		if message != nil {
			return NextMailResult{Message: message}, nil
		}
	}
}

func (r *Runtime) MarkMailRead(ctx context.Context, req MarkMailReadRequest) (mailbox.Message, error) {
	if r.mailboxStore == nil {
		return mailbox.Message{}, fmt.Errorf("mailbox store unavailable")
	}
	message, err := r.mailboxStore.MarkMessageRead(ctx, mailbox.MarkRead{
		ID:        req.ID,
		Recipient: req.Recipient,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return mailbox.Message{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventMailboxChanged})
	return message, nil
}

func (r *Runtime) ReplyMail(ctx context.Context, req ReplyMailRequest) (mailbox.Message, error) {
	if r.mailboxStore == nil {
		return mailbox.Message{}, fmt.Errorf("mailbox store unavailable")
	}
	messages, err := r.mailboxStore.ListMessages(ctx, mailbox.ListFilter{ID: req.ID, Limit: 1})
	if err != nil {
		return mailbox.Message{}, err
	}
	if len(messages) == 0 {
		return mailbox.Message{}, fmt.Errorf("mail %s not found", req.ID)
	}
	messageType := req.Type
	if messageType == "" {
		messageType = mailbox.TypeStatus
	}
	reply, err := mailbox.NewReply(mailbox.Reply{
		ID:       r.ids(),
		Original: messages[0],
		From:     req.From,
		Type:     messageType,
		Priority: req.Priority,
		Subject:  req.Subject,
		Body:     req.Body,
		Payload:  req.Payload,
		Now:      time.Now().UTC(),
	})
	if err != nil {
		return mailbox.Message{}, err
	}
	if err := r.mailboxStore.SaveMessage(ctx, reply); err != nil {
		return mailbox.Message{}, err
	}
	r.publish(ctx, RuntimeEvent{Type: EventMailboxChanged})
	return reply, nil
}

func (r *Runtime) nextUnreadMail(ctx context.Context, filter mailbox.ListFilter) (*mailbox.Message, error) {
	messages, err := r.mailboxStore.ListMessages(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return &messages[0], nil
}

func (r *Runtime) expandMailRecipients(recipients []mailbox.Address) ([]mailbox.Address, error) {
	if len(recipients) == 0 {
		return nil, nil
	}
	out := make([]mailbox.Address, 0, len(recipients))
	for _, recipient := range recipients {
		if err := recipient.Validate(); err != nil {
			return nil, err
		}
		if !recipient.IsGroupSelector() {
			out = append(out, recipient)
			continue
		}
		expanded, err := r.expandMailGroupSelector(recipient)
		if err != nil {
			return nil, err
		}
		if len(expanded) == 0 {
			return nil, fmt.Errorf("mail selector %s resolved to no recipients", recipient.String())
		}
		out = append(out, expanded...)
	}
	return mailbox.DeduplicateAddresses(out), nil
}

func (r *Runtime) expandMailGroupSelector(selector mailbox.Address) ([]mailbox.Address, error) {
	switch selector.Kind {
	case mailbox.AddressKindProjectGroup:
		return r.projectMailRecipients(selector.ID)
	case mailbox.AddressKindWorkItemGroup:
		return r.workItemMailRecipients(selector.ID)
	case mailbox.AddressKindIdleGroup:
		return nil, selector.Validate()
	default:
		return nil, fmt.Errorf("unsupported mail group selector %s", selector.Kind)
	}
}

func (r *Runtime) projectMailRecipients(projectID string) ([]mailbox.Address, error) {
	if _, ok := r.workItems.GetProject(projectID); !ok {
		return nil, fmt.Errorf("project %s not found", projectID)
	}
	recipients := []mailbox.Address{}
	for _, current := range r.state.List() {
		if current.ProjectID != projectID {
			continue
		}
		recipients = append(recipients, mailbox.Address{Kind: mailbox.AddressKindSession, ID: current.ID})
		for _, pane := range current.Panes {
			if pane.CurrentPTYID == nil {
				continue
			}
			recipients = append(recipients, mailbox.Address{Kind: mailbox.AddressKindPTY, ID: *pane.CurrentPTYID})
		}
	}
	for _, run := range r.workItems.ListRuns("") {
		if run.ProjectID != projectID || !mailRunAddressable(run) {
			continue
		}
		recipients = appendRunMailRecipients(recipients, run)
	}
	return mailbox.DeduplicateAddresses(recipients), nil
}

func (r *Runtime) workItemMailRecipients(workItemID string) ([]mailbox.Address, error) {
	if _, ok := r.workItems.GetWorkItem(workItemID); !ok {
		return nil, fmt.Errorf("work item %s not found", workItemID)
	}
	recipients := []mailbox.Address{}
	for _, run := range r.workItems.ListRuns(workItemID) {
		if !mailRunAddressable(run) {
			continue
		}
		recipients = appendRunMailRecipients(recipients, run)
	}
	return mailbox.DeduplicateAddresses(recipients), nil
}

func appendRunMailRecipients(recipients []mailbox.Address, run workitem.WorkItemRun) []mailbox.Address {
	if run.ID != "" {
		recipients = append(recipients, mailbox.Address{Kind: mailbox.AddressKindRun, ID: run.ID})
	}
	if run.SessionID != "" {
		recipients = append(recipients, mailbox.Address{Kind: mailbox.AddressKindSession, ID: run.SessionID})
	}
	if run.PTYID != "" {
		recipients = append(recipients, mailbox.Address{Kind: mailbox.AddressKindPTY, ID: run.PTYID})
	}
	return recipients
}

func mailRunAddressable(run workitem.WorkItemRun) bool {
	switch run.Status {
	case workitem.RunStateCompleted, workitem.RunStateCancelled, workitem.RunStateFailed:
		return false
	default:
		return true
	}
}
