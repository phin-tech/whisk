package app_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/mailboxstore"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/domain/mailbox"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/events"
)

func TestRuntimeMailboxSendListReadReplyPublishesEvents(t *testing.T) {
	ctx := context.Background()
	store := newMailboxStore(t)
	sink := newRecordingEventSink()
	runtime := app.NewRuntime(app.RuntimeConfig{
		MailboxStore: store,
		EventSink:    sink,
		IDGenerator:  sequentialIDs("mail_01", "mail_02"),
	})

	message, err := runtime.SendMail(ctx, app.SendMailRequest{
		From:       mailbox.Address{Kind: mailbox.AddressKindPTY, ID: "pty_01"},
		Recipients: []mailbox.Address{{Kind: mailbox.AddressKindRun, ID: "run_01"}},
		Type:       mailbox.TypeDispatch,
		Priority:   mailbox.PriorityHigh,
		Subject:    "Implement mailbox",
		Body:       "Wire the daemon API.",
		ProjectID:  "proj_01",
		RunID:      "run_01",
	})
	if err != nil {
		t.Fatalf("send mail: %v", err)
	}
	if message.ID != "mail_01" {
		t.Fatalf("message = %#v", message)
	}
	sink.waitFor(t, ctx, app.EventMailboxChanged, "")

	listed, err := runtime.ListMail(ctx, app.ListMailRequest{
		To:         []mailbox.Address{{Kind: mailbox.AddressKindRun, ID: "run_01"}},
		UnreadOnly: true,
		Types:      []string{mailbox.TypeDispatch},
		ProjectID:  "proj_01",
		RunID:      "run_01",
	})
	if err != nil {
		t.Fatalf("list mail: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != message.ID {
		t.Fatalf("listed = %#v", listed)
	}

	read, err := runtime.MarkMailRead(ctx, app.MarkMailReadRequest{
		ID:        message.ID,
		Recipient: &mailbox.Address{Kind: mailbox.AddressKindRun, ID: "run_01"},
	})
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if read.Recipients[0].ReadAt == nil {
		t.Fatalf("read = %#v", read)
	}
	sink.waitFor(t, ctx, app.EventMailboxChanged, "")

	reply, err := runtime.ReplyMail(ctx, app.ReplyMailRequest{
		ID:   message.ID,
		From: mailbox.Address{Kind: mailbox.AddressKindRun, ID: "run_01"},
		Body: "Done.",
	})
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if reply.ID != "mail_02" || reply.ThreadID != message.ID || reply.ReplyToID != message.ID || reply.Recipients[0].Address != message.From {
		t.Fatalf("reply = %#v", reply)
	}
	sink.waitFor(t, ctx, app.EventMailboxChanged, "")
}

func TestRuntimeNextMailWaitsForMailboxChanged(t *testing.T) {
	ctx := context.Background()
	store := newMailboxStore(t)
	sink := newRecordingEventSink()
	runtime := app.NewRuntime(app.RuntimeConfig{
		MailboxStore: store,
		EventSink:    sink,
		IDGenerator:  sequentialIDs("mail_01"),
	})
	to := mailbox.Address{Kind: mailbox.AddressKindPTY, ID: "pty_01"}

	resultCh := make(chan app.NextMailResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := runtime.NextMail(ctx, app.NextMailRequest{
			To:      []mailbox.Address{to},
			Timeout: time.Second,
		})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()
	time.Sleep(20 * time.Millisecond)
	if _, err := runtime.SendMail(ctx, app.SendMailRequest{
		From:       mailbox.Address{Kind: mailbox.AddressKindRun, ID: "run_01"},
		Recipients: []mailbox.Address{to},
		Type:       mailbox.TypeStatus,
		Subject:    "Ready",
	}); err != nil {
		t.Fatalf("send mail: %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("next mail: %v", err)
	case result := <-resultCh:
		if result.Timeout || result.Message == nil || result.Message.ID != "mail_01" {
			t.Fatalf("result = %#v", result)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for next mail")
	}
}

func TestRuntimeNextMailWakesMultipleWaitersForMailboxChanged(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	store := newMailboxStore(t)
	eventBus, err := events.NewNATSBus()
	if err != nil {
		t.Fatalf("new event bus: %v", err)
	}
	t.Cleanup(eventBus.Close)
	runtime := app.NewRuntime(app.RuntimeConfig{
		MailboxStore: store,
		EventSink:    eventBus,
		IDGenerator:  sequentialIDs("mail_01"),
	})
	to := mailbox.Address{Kind: mailbox.AddressKindPTY, ID: "pty_01"}

	type nextResult struct {
		result app.NextMailResult
		err    error
	}
	const waiterCount = 2
	start := make(chan struct{})
	results := make(chan nextResult, waiterCount)
	for range waiterCount {
		go func() {
			<-start
			result, err := runtime.NextMail(ctx, app.NextMailRequest{
				To:      []mailbox.Address{to},
				Timeout: time.Second,
			})
			results <- nextResult{result: result, err: err}
		}()
	}
	close(start)
	time.Sleep(20 * time.Millisecond)
	if _, err := runtime.SendMail(ctx, app.SendMailRequest{
		From:       mailbox.Address{Kind: mailbox.AddressKindRun, ID: "run_01"},
		Recipients: []mailbox.Address{to},
		Type:       mailbox.TypeStatus,
		Subject:    "Ready",
	}); err != nil {
		t.Fatalf("send mail: %v", err)
	}

	for range waiterCount {
		select {
		case got := <-results:
			if got.err != nil {
				t.Fatalf("next mail: %v", got.err)
			}
			if got.result.Timeout || got.result.Message == nil || got.result.Message.ID != "mail_01" {
				t.Fatalf("result = %#v", got.result)
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for next mail waiters: %v", ctx.Err())
		}
	}
}

func TestRuntimeSendMailExpandsGroupSelectorsAgainstDaemonReadModels(t *testing.T) {
	ctx := context.Background()
	store := newMailboxStore(t)
	runtime := app.NewRuntime(app.RuntimeConfig{
		MailboxStore: store,
		EventSink:    newRecordingEventSink(),
	})
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
	projectSession, err := runtime.CreateSession(ctx, app.CreateSessionRequest{
		Name:      "Project session",
		RootDir:   t.TempDir(),
		ProjectID: project.ID,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	message, err := runtime.SendMail(ctx, app.SendMailRequest{
		From: mailbox.Address{Kind: mailbox.AddressKindPTY, ID: "coordinator"},
		Recipients: []mailbox.Address{
			{Kind: mailbox.AddressKindProjectGroup, ID: project.ID},
			{Kind: mailbox.AddressKindWorkItemGroup, ID: item.ID},
			{Kind: mailbox.AddressKindPTY, ID: "pty_run"},
		},
		Type:    mailbox.TypeDispatch,
		Subject: "Fan out",
	})
	if err != nil {
		t.Fatalf("send mail: %v", err)
	}
	assertMailRecipients(t, message.Recipients, []mailbox.Address{
		{Kind: mailbox.AddressKindSession, ID: projectSession.Session.ID},
		{Kind: mailbox.AddressKindRun, ID: run.ID},
		{Kind: mailbox.AddressKindSession, ID: "sess_run"},
		{Kind: mailbox.AddressKindPTY, ID: "pty_run"},
	})
}

func TestRuntimeSendMailRejectsInvalidGroupSelectors(t *testing.T) {
	ctx := context.Background()
	store := newMailboxStore(t)
	runtime := app.NewRuntime(app.RuntimeConfig{
		MailboxStore: store,
		EventSink:    newRecordingEventSink(),
	})
	project, err := runtime.CreateProject(ctx, app.CreateProjectRequest{Name: "App", RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	from := mailbox.Address{Kind: mailbox.AddressKindPTY, ID: "coordinator"}
	for _, tc := range []struct {
		name       string
		recipients []mailbox.Address
		want       string
	}{
		{
			name:       "idle deferred",
			recipients: []mailbox.Address{{Kind: mailbox.AddressKindIdleGroup}},
			want:       "agent status",
		},
		{
			name:       "missing project",
			recipients: []mailbox.Address{{Kind: mailbox.AddressKindProjectGroup, ID: "missing"}},
			want:       "project missing not found",
		},
		{
			name:       "no current project recipients",
			recipients: []mailbox.Address{{Kind: mailbox.AddressKindProjectGroup, ID: project.ID}},
			want:       "resolved to no recipients",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runtime.SendMail(ctx, app.SendMailRequest{
				From:       from,
				Recipients: tc.recipients,
				Type:       mailbox.TypeStatus,
				Subject:    "Fan out",
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestRuntimeClearDaemonClearsMailboxStore(t *testing.T) {
	ctx := context.Background()
	store := newMailboxStore(t)
	sink := newRecordingEventSink()
	runtime := app.NewRuntime(app.RuntimeConfig{
		MailboxStore: store,
		EventSink:    sink,
		IDGenerator:  sequentialIDs("mail_01"),
	})
	if _, err := runtime.SendMail(ctx, app.SendMailRequest{
		From:       mailbox.Address{Kind: mailbox.AddressKindRun, ID: "run_01"},
		Recipients: []mailbox.Address{{Kind: mailbox.AddressKindPTY, ID: "pty_01"}},
		Type:       mailbox.TypeStatus,
		Subject:    "Clear me",
	}); err != nil {
		t.Fatalf("send mail: %v", err)
	}
	sink.waitFor(t, ctx, app.EventMailboxChanged, "")

	if _, err := runtime.ClearDaemon(ctx); err != nil {
		t.Fatalf("clear daemon: %v", err)
	}
	remaining, err := runtime.ListMail(ctx, app.ListMailRequest{})
	if err != nil {
		t.Fatalf("list mail: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining = %#v", remaining)
	}
	sink.waitFor(t, ctx, app.EventMailboxChanged, "")
}

func newMailboxStore(t *testing.T) *mailboxstore.SQLiteStore {
	t.Helper()
	store, err := mailboxstore.NewSQLiteStore(filepath.Join(t.TempDir(), "mailbox.sqlite"))
	if err != nil {
		t.Fatalf("new mailbox store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close mailbox store: %v", err)
		}
	})
	return store
}

func sequentialIDs(ids ...string) func() string {
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

func assertMailRecipients(t *testing.T, recipients []mailbox.Recipient, want []mailbox.Address) {
	t.Helper()
	got := map[string]int{}
	for _, recipient := range recipients {
		got[recipient.Address.String()]++
	}
	for address, count := range got {
		if count != 1 {
			t.Fatalf("recipient %s appeared %d times in %#v", address, count, recipients)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("recipients = %#v, want %#v", recipients, want)
	}
	for _, address := range want {
		if got[address.String()] != 1 {
			t.Fatalf("missing recipient %s in %#v", address, recipients)
		}
	}
}
