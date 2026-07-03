package mailbox

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseAddressNormalizesConcreteRuntimeAddresses(t *testing.T) {
	address, err := ParseAddress(" workitem:wi_01 ")
	if err != nil {
		t.Fatalf("parse address: %v", err)
	}
	if address.Kind != AddressKindWorkItem || address.ID != "wi_01" || address.String() != "work-item:wi_01" {
		t.Fatalf("address = %#v", address)
	}

	if _, err := ParseAddress("@idle"); err == nil || !strings.Contains(err.Error(), "group selectors") {
		t.Fatalf("expected unsupported group selector error, got %v", err)
	}
	if _, err := ParseAddress("pty:bad id"); err == nil {
		t.Fatalf("expected invalid id error")
	}
}

func TestNewMessageValidatesLifecycleMail(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	message, err := NewMessage(Send{
		ID:       "mail_01",
		From:     Address{Kind: AddressKindPTY, ID: "pty_01"},
		To:       []Address{{Kind: AddressKindRun, ID: "run_01"}, {Kind: AddressKindRun, ID: "run_01"}},
		Type:     TypeDispatch,
		Priority: "",
		Subject:  "Implement mailbox",
		Body:     "Please wire the daemon API.",
		Payload:  json.RawMessage(`{"taskId":"task_01"}`),
		Now:      now,
	})
	if err != nil {
		t.Fatalf("new message: %v", err)
	}
	if message.Type != TypeDispatch || message.Priority != PriorityNormal || len(message.Recipients) != 1 || !message.CreatedAt.Equal(now) {
		t.Fatalf("message = %#v", message)
	}

	if _, err := NewMessage(Send{ID: "mail_02", From: message.From, To: []Address{message.From}, Type: TypeWorkerDone}); err == nil {
		t.Fatalf("expected subject requirement")
	}
	if _, err := NewMessage(Send{ID: "mail_03", From: message.From, To: []Address{message.From}, Type: TypeHeartbeat}); err != nil {
		t.Fatalf("heartbeat should not require subject: %v", err)
	}
	if _, err := NewMessage(Send{ID: "mail_04", From: message.From, To: []Address{message.From}, Type: TypeStatus, Subject: "x", Payload: json.RawMessage(`{`)}); err == nil {
		t.Fatalf("expected invalid payload error")
	}
}

func TestNewReplyThreadsToOriginalSenderAndInheritsContext(t *testing.T) {
	original, err := NewMessage(Send{
		ID:         "mail_01",
		From:       Address{Kind: AddressKindRun, ID: "run_01"},
		To:         []Address{{Kind: AddressKindPTY, ID: "pty_01"}},
		Type:       TypeDecisionGate,
		Priority:   PriorityHigh,
		Subject:    "Pick release target",
		ProjectID:  "proj_01",
		WorkItemID: "wi_01",
		RunID:      "run_01",
	})
	if err != nil {
		t.Fatalf("original: %v", err)
	}
	reply, err := NewReply(Reply{
		ID:       "mail_02",
		Original: original,
		From:     Address{Kind: AddressKindPTY, ID: "pty_01"},
		Type:     TypeStatus,
		Body:     "Use staging.",
	})
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if reply.ThreadID != original.ID || reply.ReplyToID != original.ID || reply.Recipients[0].Address != original.From {
		t.Fatalf("reply thread/recipient = %#v", reply)
	}
	if reply.Subject != "Re: Pick release target" || reply.ProjectID != "proj_01" || reply.WorkItemID != "wi_01" || reply.RunID != "run_01" {
		t.Fatalf("reply context = %#v", reply)
	}
}

func TestMarkMessageReadUpdatesOneRecipientOrAll(t *testing.T) {
	message, err := NewMessage(Send{
		ID:      "mail_01",
		From:    Address{Kind: AddressKindPTY, ID: "pty_01"},
		To:      []Address{{Kind: AddressKindRun, ID: "run_01"}, {Kind: AddressKindSession, ID: "sess_01"}},
		Type:    TypeStatus,
		Subject: "Heads up",
	})
	if err != nil {
		t.Fatalf("new message: %v", err)
	}
	readAt := time.Date(2026, 7, 3, 12, 30, 0, 0, time.UTC)
	recipient := Address{Kind: AddressKindRun, ID: "run_01"}
	updated, err := MarkMessageRead(message, &recipient, readAt)
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if updated.Recipients[0].ReadAt == nil || !updated.Recipients[0].ReadAt.Equal(readAt) || updated.Recipients[1].ReadAt != nil {
		t.Fatalf("updated recipients = %#v", updated.Recipients)
	}
	if message.Recipients[0].ReadAt != nil {
		t.Fatalf("original message mutated")
	}
	allRead, err := MarkMessageRead(updated, nil, readAt.Add(time.Minute))
	if err != nil {
		t.Fatalf("mark all read: %v", err)
	}
	if allRead.Recipients[0].ReadAt == nil || allRead.Recipients[1].ReadAt == nil {
		t.Fatalf("all read recipients = %#v", allRead.Recipients)
	}
}
