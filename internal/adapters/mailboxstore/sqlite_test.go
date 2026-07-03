package mailboxstore

import (
	"context"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/domain/mailbox"
)

func TestDefaultSQLitePathUsesXDGConfigHome(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	path, err := DefaultSQLitePath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	want := filepath.Join(configHome, "whisk", "mailbox.sqlite")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestSQLiteStoreRoundTripsMessagesWithRecipientReadState(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "mailbox.sqlite"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	message, err := mailbox.NewMessage(mailbox.Send{
		ID:         "mail_01",
		From:       mailbox.Address{Kind: mailbox.AddressKindPTY, ID: "pty_01"},
		To:         []mailbox.Address{{Kind: mailbox.AddressKindRun, ID: "run_01"}, {Kind: mailbox.AddressKindSession, ID: "sess_01"}},
		Type:       mailbox.TypeDispatch,
		Priority:   mailbox.PriorityHigh,
		Subject:    "Implement mailbox",
		Body:       "Please wire storage.",
		Payload:    []byte(`{"taskId":"task_01"}`),
		ProjectID:  "proj_01",
		WorkItemID: "wi_01",
		RunID:      "run_01",
		ThreadID:   "thread_01",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("new message: %v", err)
	}
	if err := store.SaveMessage(ctx, message); err != nil {
		t.Fatalf("save: %v", err)
	}

	unread, err := store.ListMessages(ctx, mailbox.ListFilter{
		To:         []mailbox.Address{{Kind: mailbox.AddressKindRun, ID: "run_01"}},
		UnreadOnly: true,
		Types:      []string{mailbox.TypeDispatch},
		ProjectID:  "proj_01",
		WorkItemID: "wi_01",
		RunID:      "run_01",
		ThreadID:   "thread_01",
	})
	if err != nil {
		t.Fatalf("list unread: %v", err)
	}
	if len(unread) != 1 || unread[0].ID != message.ID || len(unread[0].Recipients) != 2 || string(unread[0].Payload) != `{"taskId":"task_01"}` {
		t.Fatalf("unread = %#v", unread)
	}

	readAt := now.Add(time.Minute)
	recipient := mailbox.Address{Kind: mailbox.AddressKindRun, ID: "run_01"}
	read, err := store.MarkMessageRead(ctx, mailbox.MarkRead{ID: message.ID, Recipient: &recipient, Now: readAt})
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if read.Recipients[0].ReadAt == nil && read.Recipients[1].ReadAt == nil {
		t.Fatalf("expected one read recipient: %#v", read.Recipients)
	}
	unread, err = store.ListMessages(ctx, mailbox.ListFilter{To: []mailbox.Address{recipient}, UnreadOnly: true})
	if err != nil {
		t.Fatalf("list after read: %v", err)
	}
	if len(unread) != 0 {
		t.Fatalf("unread after read = %#v", unread)
	}
}

func TestSQLiteStoreOrdersNextChronologicallyAndDeletesAll(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "mailbox.sqlite"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	to := mailbox.Address{Kind: mailbox.AddressKindPTY, ID: "pty_01"}
	for _, spec := range []struct {
		id string
		at time.Time
	}{
		{id: "mail_02", at: time.Date(2026, 7, 3, 12, 2, 0, 0, time.UTC)},
		{id: "mail_01", at: time.Date(2026, 7, 3, 12, 1, 0, 0, time.UTC)},
	} {
		message, err := mailbox.NewMessage(mailbox.Send{
			ID:      spec.id,
			From:    mailbox.Address{Kind: mailbox.AddressKindRun, ID: "run_01"},
			To:      []mailbox.Address{to},
			Type:    mailbox.TypeStatus,
			Subject: spec.id,
			Now:     spec.at,
		})
		if err != nil {
			t.Fatalf("message %s: %v", spec.id, err)
		}
		if err := store.SaveMessage(ctx, message); err != nil {
			t.Fatalf("save %s: %v", spec.id, err)
		}
	}
	next, err := store.ListMessages(ctx, mailbox.ListFilter{To: []mailbox.Address{to}, UnreadOnly: true, OldestFirst: true, Limit: 1})
	if err != nil {
		t.Fatalf("next list: %v", err)
	}
	if len(next) != 1 || next[0].ID != "mail_01" {
		t.Fatalf("next = %#v", next)
	}
	if err := store.DeleteAll(ctx); err != nil {
		t.Fatalf("delete all: %v", err)
	}
	remaining, err := store.ListMessages(ctx, mailbox.ListFilter{})
	if err != nil {
		t.Fatalf("list remaining: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining = %#v", remaining)
	}
}

func TestSQLiteStoreSupportsConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "mailbox.sqlite"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})

	to := mailbox.Address{Kind: mailbox.AddressKindRun, ID: "run_01"}
	const messageCount = 24
	var wg sync.WaitGroup
	errCh := make(chan error, messageCount)
	for i := range messageCount {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "mail_" + strconv.Itoa(i)
			message, err := mailbox.NewMessage(mailbox.Send{
				ID:      id,
				From:    mailbox.Address{Kind: mailbox.AddressKindPTY, ID: "pty_" + strconv.Itoa(i)},
				To:      []mailbox.Address{to},
				Type:    mailbox.TypeStatus,
				Subject: id,
				Now:     time.Date(2026, 7, 3, 12, i, 0, 0, time.UTC),
			})
			if err != nil {
				errCh <- err
				return
			}
			if err := store.SaveMessage(ctx, message); err != nil {
				errCh <- err
				return
			}
			if _, err := store.ListMessages(ctx, mailbox.ListFilter{To: []mailbox.Address{to}, UnreadOnly: true, Limit: 1}); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent operation: %v", err)
		}
	}

	messages, err := store.ListMessages(ctx, mailbox.ListFilter{To: []mailbox.Address{to}, UnreadOnly: true})
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != messageCount {
		t.Fatalf("messages len = %d, want %d", len(messages), messageCount)
	}

	errCh = make(chan error, messageCount)
	for _, message := range messages {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if _, err := store.MarkMessageRead(ctx, mailbox.MarkRead{ID: id, Recipient: &to}); err != nil {
				errCh <- err
			}
		}(message.ID)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent mark read: %v", err)
		}
	}

	unread, err := store.ListMessages(ctx, mailbox.ListFilter{To: []mailbox.Address{to}, UnreadOnly: true})
	if err != nil {
		t.Fatalf("list unread: %v", err)
	}
	if len(unread) != 0 {
		t.Fatalf("unread after mark read = %#v", unread)
	}
	if err := store.DeleteAll(ctx); err != nil {
		t.Fatalf("delete all: %v", err)
	}
}

func TestSQLiteStoreEnforcesForeignKeysAndCascadesRecipients(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "mailbox.sqlite"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})

	if _, err := store.db.ExecContext(ctx, `
		insert into message_recipients (message_id, recipient_kind, recipient_id)
		values ('missing_mail', 'run', 'run_01')
	`); err == nil {
		t.Fatalf("expected foreign key violation for orphan recipient")
	}

	message, err := mailbox.NewMessage(mailbox.Send{
		ID:      "mail_01",
		From:    mailbox.Address{Kind: mailbox.AddressKindPTY, ID: "pty_01"},
		To:      []mailbox.Address{{Kind: mailbox.AddressKindRun, ID: "run_01"}},
		Type:    mailbox.TypeStatus,
		Subject: "Cascade",
		Now:     time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("new message: %v", err)
	}
	if err := store.SaveMessage(ctx, message); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `delete from messages where id = ?`, message.ID); err != nil {
		t.Fatalf("delete message: %v", err)
	}
	var recipients int
	if err := store.db.QueryRowContext(ctx, `select count(*) from message_recipients where message_id = ?`, message.ID).Scan(&recipients); err != nil {
		t.Fatalf("count recipients: %v", err)
	}
	if recipients != 0 {
		t.Fatalf("recipients after parent delete = %d", recipients)
	}
}
