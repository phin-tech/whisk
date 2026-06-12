package workitemstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phin-tech/whisk/internal/domain/workitem"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	path string
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if path == "" {
		defaultPath, err := DefaultSQLitePath()
		if err != nil {
			return nil, err
		}
		path = defaultPath
	}
	store := &SQLiteStore{path: filepath.Clean(path)}
	db, err := store.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := migrateSQLite(db); err != nil {
		return nil, err
	}
	return store, nil
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
	return filepath.Join(configDir, "whisk", "work-items.sqlite"), nil
}

func (s *SQLiteStore) LoadWorkItems(ctx context.Context) (workitem.Snapshot, error) {
	db, err := s.open()
	if err != nil {
		return workitem.Snapshot{}, err
	}
	defer db.Close()
	if err := migrateSQLite(db); err != nil {
		return workitem.Snapshot{}, err
	}
	var payload []byte
	err = db.QueryRowContext(ctx, `select payload from snapshots where id = 1`).Scan(&payload)
	if err == sql.ErrNoRows {
		return workitem.Snapshot{}, nil
	}
	if err != nil {
		return workitem.Snapshot{}, err
	}
	var snapshot workitem.Snapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return workitem.Snapshot{}, err
	}
	return snapshot, nil
}

func (s *SQLiteStore) SaveWorkItems(ctx context.Context, snapshot workitem.Snapshot) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := migrateSQLite(db); err != nil {
		return err
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		insert into snapshots (id, payload, updated_at)
		values (1, ?, datetime('now'))
		on conflict(id) do update set payload = excluded.payload, updated_at = excluded.updated_at
	`, payload)
	return err
}

func (s *SQLiteStore) open() (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return nil, err
	}
	return sql.Open("sqlite", s.path)
}

func migrateSQLite(db *sql.DB) error {
	_, err := db.Exec(`
		create table if not exists snapshots (
			id integer primary key check (id = 1),
			payload blob not null,
			updated_at text not null
		)
	`)
	return err
}
