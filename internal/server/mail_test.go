package server_test

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/mailboxstore"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/mailbox"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPServerMailRoutes(t *testing.T) {
	store, err := mailboxstore.NewSQLiteStore(filepath.Join(t.TempDir(), "mailbox.sqlite"))
	if err != nil {
		t.Fatalf("new mailbox store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close mailbox store: %v", err)
		}
	})
	runtime := app.NewRuntime(app.RuntimeConfig{
		MailboxStore: store,
		EventSink:    newFakeEventBus(),
		IDGenerator:  serverMailIDs("mail_01", "mail_02"),
	})
	handler := server.NewHTTP(runtime)

	created := postJSON[protocol.MailMessage](t, handler, "/v1/mail", protocol.SendMailRequest{
		From:       protocol.MailAddress{Kind: mailbox.AddressKindPTY, ID: "pty_01"},
		To:         []protocol.MailAddress{{Kind: mailbox.AddressKindRun, ID: "run_01"}},
		Type:       mailbox.TypeDispatch,
		Priority:   mailbox.PriorityHigh,
		Subject:    "Implement mailbox",
		Body:       "Wire route handlers.",
		ProjectID:  "proj_01",
		WorkItemID: "wi_01",
		RunID:      "run_01",
	}, http.StatusCreated)
	if created.ID != "mail_01" || created.From.ID != "pty_01" || len(created.Recipients) != 1 {
		t.Fatalf("created = %#v", created)
	}

	listed := getJSON[[]protocol.MailMessage](t, handler, "/v1/mail?to=run:run_01&unread=true&types=dispatch&projectId=proj_01&workItemId=wi_01&runId=run_01", http.StatusOK)
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("listed = %#v", listed)
	}

	next := getJSON[protocol.NextMailResponse](t, handler, "/v1/mail/next?to=run:run_01&types=dispatch&timeoutMs=0&projectId=proj_01", http.StatusOK)
	if next.Timeout || next.Message == nil || next.Message.ID != created.ID {
		t.Fatalf("next = %#v", next)
	}

	read := postJSON[protocol.MailMessage](t, handler, "/v1/mail/"+created.ID+"/read", protocol.MarkMailReadRequest{
		To: &protocol.MailAddress{Kind: mailbox.AddressKindRun, ID: "run_01"},
	}, http.StatusOK)
	if read.Recipients[0].ReadAt == nil {
		t.Fatalf("read = %#v", read)
	}
	next = getJSON[protocol.NextMailResponse](t, handler, "/v1/mail/next?to=run:run_01&types=dispatch&timeoutMs=0", http.StatusOK)
	if !next.Timeout || next.Message != nil {
		t.Fatalf("next after read = %#v", next)
	}

	reply := postJSON[protocol.MailMessage](t, handler, "/v1/mail/"+created.ID+"/reply", protocol.ReplyMailRequest{
		From: protocol.MailAddress{Kind: mailbox.AddressKindRun, ID: "run_01"},
		Body: "Done.",
	}, http.StatusCreated)
	if reply.ID != "mail_02" || reply.ThreadID != created.ID || reply.ReplyToID != created.ID || reply.Recipients[0].Address.ID != "pty_01" {
		t.Fatalf("reply = %#v", reply)
	}
}

func TestHTTPServerSendMailExpandsGroupSelectors(t *testing.T) {
	store, err := mailboxstore.NewSQLiteStore(filepath.Join(t.TempDir(), "mailbox.sqlite"))
	if err != nil {
		t.Fatalf("new mailbox store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close mailbox store: %v", err)
		}
	})
	runtime := app.NewRuntime(app.RuntimeConfig{
		MailboxStore: store,
		EventSink:    newFakeEventBus(),
	})
	ctx := context.Background()
	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	item, err := runtime.CreateWorkItem(ctx, app.CreateWorkItemRequest{ProjectID: project.ID, Title: "Wire mailbox"})
	if err != nil {
		t.Fatalf("create work item: %v", err)
	}
	run, err := runtime.StartWorkItemRun(ctx, app.StartWorkItemRunRequest{
		WorkItemID:       item.ID,
		Preset:           workitem.RunPresetWriter,
		PromptTemplateID: workitem.PromptTemplateImplement,
		SessionID:        "sess_run",
		PTYID:            "pty_run",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	handler := server.NewHTTP(runtime)

	created := postJSON[protocol.MailMessage](t, handler, "/v1/mail", protocol.SendMailRequest{
		From:    protocol.MailAddress{Kind: mailbox.AddressKindPTY, ID: "coordinator"},
		To:      []protocol.MailAddress{{Kind: mailbox.AddressKindWorkItemGroup, ID: item.ID}},
		Type:    mailbox.TypeDispatch,
		Subject: "Implement",
	}, http.StatusCreated)
	want := map[protocol.MailAddress]bool{
		{Kind: mailbox.AddressKindRun, ID: run.ID}: false,
	}
	if len(created.Recipients) != len(want) {
		t.Fatalf("recipients = %#v", created.Recipients)
	}
	for _, recipient := range created.Recipients {
		if _, ok := want[recipient.Address]; !ok {
			t.Fatalf("unexpected recipient %#v in %#v", recipient.Address, created.Recipients)
		}
		want[recipient.Address] = true
	}
	for address, seen := range want {
		if !seen {
			t.Fatalf("missing recipient %#v in %#v", address, created.Recipients)
		}
	}
}

func serverMailIDs(ids ...string) func() string {
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
