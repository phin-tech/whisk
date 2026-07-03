package client_test

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/mailboxstore"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/mailbox"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPClientDrivesMailboxAPI(t *testing.T) {
	store, err := mailboxstore.NewSQLiteStore(filepath.Join(t.TempDir(), "mailbox.sqlite"))
	if err != nil {
		t.Fatalf("new mailbox store: %v", err)
	}
	runtime := app.NewRuntime(app.RuntimeConfig{
		MailboxStore: store,
		EventSink:    newFakeEventBus(),
		IDGenerator:  clientMailIDs("mail_01", "mail_02"),
	})
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)
	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	ctx := context.Background()

	sent, err := daemon.SendMail(ctx, protocol.SendMailRequest{
		From:       protocol.MailAddress{Kind: mailbox.AddressKindPTY, ID: "pty_01"},
		To:         []protocol.MailAddress{{Kind: mailbox.AddressKindRun, ID: "run_01"}},
		Type:       mailbox.TypeDecisionGate,
		Priority:   mailbox.PriorityUrgent,
		Subject:    "Choose target",
		ProjectID:  "proj_01",
		WorkItemID: "wi_01",
		RunID:      "run_01",
	})
	if err != nil {
		t.Fatalf("send mail: %v", err)
	}
	if sent.ID != "mail_01" || sent.Priority != mailbox.PriorityUrgent {
		t.Fatalf("sent = %#v", sent)
	}

	listed, err := daemon.ListMail(ctx, protocol.ListMailRequest{
		To:         []protocol.MailAddress{{Kind: mailbox.AddressKindRun, ID: "run_01"}},
		UnreadOnly: true,
		Types:      []string{mailbox.TypeDecisionGate},
		ProjectID:  "proj_01",
		WorkItemID: "wi_01",
		RunID:      "run_01",
	})
	if err != nil || len(listed) != 1 || listed[0].ID != sent.ID {
		t.Fatalf("listed = %#v, err = %v", listed, err)
	}

	next, err := daemon.NextMail(ctx, protocol.NextMailRequest{
		To:        []protocol.MailAddress{{Kind: mailbox.AddressKindRun, ID: "run_01"}},
		Types:     []string{mailbox.TypeDecisionGate},
		TimeoutMs: 0,
		ProjectID: "proj_01",
	})
	if err != nil || next.Timeout || next.Message == nil || next.Message.ID != sent.ID {
		t.Fatalf("next = %#v, err = %v", next, err)
	}

	read, err := daemon.MarkMailRead(ctx, sent.ID, protocol.MarkMailReadRequest{To: &protocol.MailAddress{Kind: mailbox.AddressKindRun, ID: "run_01"}})
	if err != nil || read.Recipients[0].ReadAt == nil {
		t.Fatalf("read = %#v, err = %v", read, err)
	}
	reply, err := daemon.ReplyMail(ctx, sent.ID, protocol.ReplyMailRequest{
		From: protocol.MailAddress{Kind: mailbox.AddressKindRun, ID: "run_01"},
		Body: "Use staging.",
	})
	if err != nil || reply.ID != "mail_02" || reply.ThreadID != sent.ID || reply.ReplyToID != sent.ID {
		t.Fatalf("reply = %#v, err = %v", reply, err)
	}
}

func clientMailIDs(ids ...string) func() string {
	index := 0
	return func() string {
		if index >= len(ids) {
			return "mail_extra"
		}
		id := ids[index]
		index++
		return id
	}
}
