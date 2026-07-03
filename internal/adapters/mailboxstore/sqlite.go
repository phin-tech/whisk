package mailboxstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/phin-tech/whisk/internal/domain/mailbox"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	path string
	db   *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if path == "" {
		defaultPath, err := DefaultSQLitePath()
		if err != nil {
			return nil, err
		}
		path = defaultPath
	}
	cleaned := filepath.Clean(path)
	db, err := openSQLite(cleaned)
	if err != nil {
		return nil, err
	}
	if err := configureSQLite(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrateSQLite(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteStore{path: cleaned, db: db}, nil
}

func DefaultSQLitePath() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "whisk", "mailbox.sqlite"), nil
}

func (s *SQLiteStore) SaveMessage(ctx context.Context, message mailbox.Message) error {
	if err := validateMessage(message); err != nil {
		return err
	}
	db, err := s.database()
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var payload any
	if len(message.Payload) > 0 {
		payload = string(message.Payload)
	}
	_, err = tx.ExecContext(ctx, `
		insert into messages (
			id, thread_id, reply_to_id, from_kind, from_id, type, priority,
			subject, body, payload_json, project_id, work_item_id, run_id,
			session_id, pty_id, dispatch_id, created_at
		)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, message.ID, message.ThreadID, message.ReplyToID, message.From.Kind, message.From.ID,
		message.Type, message.Priority, message.Subject, message.Body, payload,
		message.ProjectID, message.WorkItemID, message.RunID, message.SessionID,
		message.PTYID, message.DispatchID, formatTime(message.CreatedAt))
	if err != nil {
		return err
	}
	for _, recipient := range message.Recipients {
		var readAt any
		if recipient.ReadAt != nil {
			readAt = formatTime(*recipient.ReadAt)
		}
		if _, err := tx.ExecContext(ctx, `
			insert into message_recipients (message_id, recipient_kind, recipient_id, read_at)
			values (?, ?, ?, ?)
		`, message.ID, recipient.Address.Kind, recipient.Address.ID, readAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) ListMessages(ctx context.Context, filter mailbox.ListFilter) ([]mailbox.Message, error) {
	normalized, err := mailbox.NormalizeListFilter(filter)
	if err != nil {
		return nil, err
	}
	db, err := s.database()
	if err != nil {
		return nil, err
	}

	ids, err := queryMessageIDs(ctx, db, normalized)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return loadMessages(ctx, db, ids)
}

func (s *SQLiteStore) MarkMessageRead(ctx context.Context, req mailbox.MarkRead) (mailbox.Message, error) {
	if strings.TrimSpace(req.ID) == "" {
		return mailbox.Message{}, fmt.Errorf("mail id required")
	}
	if req.Recipient != nil {
		if err := req.Recipient.Validate(); err != nil {
			return mailbox.Message{}, err
		}
	}
	readAt := req.Now
	if readAt.IsZero() {
		readAt = time.Now().UTC()
	}
	db, err := s.database()
	if err != nil {
		return mailbox.Message{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return mailbox.Message{}, err
	}
	defer tx.Rollback()

	var result sql.Result
	if req.Recipient == nil {
		result, err = tx.ExecContext(ctx, `
			update message_recipients
			set read_at = coalesce(read_at, ?)
			where message_id = ?
		`, formatTime(readAt), req.ID)
	} else {
		result, err = tx.ExecContext(ctx, `
			update message_recipients
			set read_at = coalesce(read_at, ?)
			where message_id = ? and recipient_kind = ? and recipient_id = ?
		`, formatTime(readAt), req.ID, req.Recipient.Kind, req.Recipient.ID)
	}
	if err != nil {
		return mailbox.Message{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return mailbox.Message{}, err
	}
	if rows == 0 {
		return mailbox.Message{}, fmt.Errorf("mail %s recipient not found", req.ID)
	}
	if err := tx.Commit(); err != nil {
		return mailbox.Message{}, err
	}
	messages, err := s.ListMessages(ctx, mailbox.ListFilter{ID: req.ID, Limit: 1})
	if err != nil {
		return mailbox.Message{}, err
	}
	if len(messages) == 0 {
		return mailbox.Message{}, fmt.Errorf("mail %s not found", req.ID)
	}
	return messages[0], nil
}

func (s *SQLiteStore) DeleteAll(ctx context.Context) error {
	db, err := s.database()
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `delete from message_recipients`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `delete from messages`); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) database() (*sql.DB, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("mailbox sqlite store is closed")
	}
	return s.db, nil
}

func openSQLite(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}

func configureSQLite(db *sql.DB) error {
	if _, err := db.Exec(`pragma busy_timeout = 5000`); err != nil {
		return fmt.Errorf("enable sqlite busy_timeout: %w", err)
	}
	var journalMode string
	if err := db.QueryRow(`pragma journal_mode = WAL`).Scan(&journalMode); err != nil {
		return fmt.Errorf("enable sqlite WAL: %w", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		return fmt.Errorf("enable sqlite WAL: journal_mode is %s", journalMode)
	}
	if _, err := db.Exec(`pragma foreign_keys = ON`); err != nil {
		return fmt.Errorf("enable sqlite foreign_keys: %w", err)
	}
	var foreignKeys int
	if err := db.QueryRow(`pragma foreign_keys`).Scan(&foreignKeys); err != nil {
		return fmt.Errorf("verify sqlite foreign_keys: %w", err)
	}
	if foreignKeys != 1 {
		return fmt.Errorf("enable sqlite foreign_keys: pragma remained disabled")
	}
	return nil
}

func migrateSQLite(db *sql.DB) error {
	_, err := db.Exec(`
		create table if not exists messages (
			id text primary key,
			thread_id text not null default '',
			reply_to_id text not null default '',
			from_kind text not null,
			from_id text not null,
			type text not null,
			priority text not null,
			subject text not null default '',
			body text not null default '',
			payload_json text,
			project_id text not null default '',
			work_item_id text not null default '',
			run_id text not null default '',
			session_id text not null default '',
			pty_id text not null default '',
			dispatch_id text not null default '',
			created_at text not null
		);
		create table if not exists message_recipients (
			message_id text not null,
			recipient_kind text not null,
			recipient_id text not null,
			read_at text,
			primary key (message_id, recipient_kind, recipient_id),
			foreign key (message_id) references messages(id) on delete cascade
		);
		create index if not exists idx_mail_recipients_read
			on message_recipients (recipient_kind, recipient_id, read_at, message_id);
		create index if not exists idx_mail_messages_type_created
			on messages (type, created_at);
		create index if not exists idx_mail_messages_thread_created
			on messages (thread_id, created_at);
		create index if not exists idx_mail_messages_project_created
			on messages (project_id, created_at);
		create index if not exists idx_mail_messages_work_item_created
			on messages (work_item_id, created_at);
		create index if not exists idx_mail_messages_run_created
			on messages (run_id, created_at);
	`)
	return err
}

func queryMessageIDs(ctx context.Context, db *sql.DB, filter mailbox.ListFilter) ([]string, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if filter.ID != "" {
		where = append(where, "m.id = ?")
		args = append(args, filter.ID)
	}
	if len(filter.To) > 0 {
		recipientWhere, recipientArgs := recipientExistsClause(filter.To, filter.UnreadOnly)
		where = append(where, recipientWhere)
		args = append(args, recipientArgs...)
	} else if filter.UnreadOnly {
		where = append(where, `exists (
			select 1 from message_recipients r
			where r.message_id = m.id and r.read_at is null
		)`)
	}
	if len(filter.Types) > 0 {
		placeholders := make([]string, len(filter.Types))
		for i, messageType := range filter.Types {
			placeholders[i] = "?"
			args = append(args, messageType)
		}
		where = append(where, "m.type in ("+strings.Join(placeholders, ", ")+")")
	}
	if filter.ProjectID != "" {
		where = append(where, "m.project_id = ?")
		args = append(args, filter.ProjectID)
	}
	if filter.WorkItemID != "" {
		where = append(where, "m.work_item_id = ?")
		args = append(args, filter.WorkItemID)
	}
	if filter.RunID != "" {
		where = append(where, "m.run_id = ?")
		args = append(args, filter.RunID)
	}
	if filter.ThreadID != "" {
		where = append(where, "m.thread_id = ?")
		args = append(args, filter.ThreadID)
	}

	direction := "desc"
	if filter.OldestFirst {
		direction = "asc"
	}
	query := "select m.id from messages m where " + strings.Join(where, " and ") +
		" order by m.created_at " + direction + ", m.id " + direction
	if filter.Limit > 0 {
		query += " limit ?"
		args = append(args, filter.Limit)
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func recipientExistsClause(to []mailbox.Address, unreadOnly bool) (string, []any) {
	parts := make([]string, 0, len(to))
	args := make([]any, 0, len(to)*2)
	for _, address := range to {
		parts = append(parts, "(r.recipient_kind = ? and r.recipient_id = ?)")
		args = append(args, address.Kind, address.ID)
	}
	unread := ""
	if unreadOnly {
		unread = " and r.read_at is null"
	}
	return `exists (
		select 1 from message_recipients r
		where r.message_id = m.id and (` + strings.Join(parts, " or ") + `)` + unread + `
	)`, args
}

func loadMessages(ctx context.Context, db *sql.DB, ids []string) ([]mailbox.Message, error) {
	messages := make([]mailbox.Message, 0, len(ids))
	for _, id := range ids {
		message, err := loadMessage(ctx, db, id)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func loadMessage(ctx context.Context, db *sql.DB, id string) (mailbox.Message, error) {
	var message mailbox.Message
	var payload sql.NullString
	var createdAt string
	err := db.QueryRowContext(ctx, `
		select id, thread_id, reply_to_id, from_kind, from_id, type, priority,
			subject, body, payload_json, project_id, work_item_id, run_id,
			session_id, pty_id, dispatch_id, created_at
		from messages
		where id = ?
	`, id).Scan(&message.ID, &message.ThreadID, &message.ReplyToID,
		&message.From.Kind, &message.From.ID, &message.Type, &message.Priority,
		&message.Subject, &message.Body, &payload, &message.ProjectID,
		&message.WorkItemID, &message.RunID, &message.SessionID, &message.PTYID,
		&message.DispatchID, &createdAt)
	if err != nil {
		return mailbox.Message{}, err
	}
	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return mailbox.Message{}, err
	}
	message.CreatedAt = parsedCreatedAt
	if payload.Valid {
		message.Payload = json.RawMessage(payload.String)
	}
	recipients, err := loadRecipients(ctx, db, id)
	if err != nil {
		return mailbox.Message{}, err
	}
	message.Recipients = recipients
	return message, nil
}

func loadRecipients(ctx context.Context, db *sql.DB, id string) ([]mailbox.Recipient, error) {
	rows, err := db.QueryContext(ctx, `
		select recipient_kind, recipient_id, read_at
		from message_recipients
		where message_id = ?
		order by recipient_kind asc, recipient_id asc
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recipients []mailbox.Recipient
	for rows.Next() {
		var recipient mailbox.Recipient
		var readAt sql.NullString
		if err := rows.Scan(&recipient.Address.Kind, &recipient.Address.ID, &readAt); err != nil {
			return nil, err
		}
		if readAt.Valid {
			parsed, err := parseTime(readAt.String)
			if err != nil {
				return nil, err
			}
			recipient.ReadAt = &parsed
		}
		recipients = append(recipients, recipient)
	}
	return recipients, rows.Err()
}

func validateMessage(message mailbox.Message) error {
	if message.ID == "" {
		return fmt.Errorf("mail id required")
	}
	if _, err := mailbox.NormalizeType(message.Type); err != nil {
		return err
	}
	if _, err := mailbox.NormalizePriority(message.Priority); err != nil {
		return err
	}
	if err := message.From.Validate(); err != nil {
		return fmt.Errorf("from: %w", err)
	}
	if len(message.Recipients) == 0 {
		return fmt.Errorf("mail recipient required")
	}
	for _, recipient := range message.Recipients {
		if err := recipient.Address.Validate(); err != nil {
			return fmt.Errorf("recipient: %w", err)
		}
	}
	if len(message.Payload) > 0 && !json.Valid(message.Payload) {
		return fmt.Errorf("mail payload must be valid JSON")
	}
	return nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}
