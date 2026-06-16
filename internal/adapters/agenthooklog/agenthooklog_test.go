package agenthooklog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendWritesRedactedJSONL(t *testing.T) {
	logger := NewWithOptions(Paths{LogPath: filepath.Join(t.TempDir(), "hooks.jsonl")}, 5, 1024, nil)

	err := logger.Append(Entry{
		Timestamp: time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC),
		Provider:  "claude",
		EventName: "Notification",
		Message:   "Need input",
		Result:    "logged",
		Raw: map[string]any{
			"message": "Need input",
			"token":   "secret",
			"nested":  map[string]any{"api_key": "key"},
		},
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	file, err := os.Open(logger.Path())
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("missing log line")
	}
	var entry Entry
	if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
		t.Fatalf("parse log line: %v", err)
	}
	if entry.Raw["token"] != "[redacted]" {
		t.Fatalf("token was not redacted: %#v", entry.Raw)
	}
	nested := entry.Raw["nested"].(map[string]any)
	if nested["api_key"] != "[redacted]" {
		t.Fatalf("nested api key was not redacted: %#v", nested)
	}
}

func TestAppendRotatesAndClearRemovesLogs(t *testing.T) {
	logger := NewWithOptions(Paths{LogPath: filepath.Join(t.TempDir(), "hooks.jsonl")}, 2, 1, nil)

	for idx := 0; idx < 3; idx++ {
		if err := logger.Append(Entry{Provider: "claude", EventName: "Stop", Result: "logged"}); err != nil {
			t.Fatalf("append %d: %v", idx, err)
		}
	}
	if _, err := os.Stat(logger.rotatedPath(1)); err != nil {
		t.Fatalf("rotated log missing: %v", err)
	}
	if err := logger.Clear(); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, err := os.Stat(logger.Path()); !os.IsNotExist(err) {
		t.Fatalf("current log remains: %v", err)
	}
	if _, err := os.Stat(logger.rotatedPath(1)); !os.IsNotExist(err) {
		t.Fatalf("rotated log remains: %v", err)
	}
}

func TestOpenCreatesMissingLogBeforeOpener(t *testing.T) {
	var opened string
	logger := NewWithOptions(Paths{LogPath: filepath.Join(t.TempDir(), "hooks.jsonl")}, 5, 1024, func(path string) error {
		opened = path
		return nil
	})

	if err := logger.Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	if opened != logger.Path() {
		t.Fatalf("opened = %q, want %q", opened, logger.Path())
	}
	if _, err := os.Stat(logger.Path()); err != nil {
		t.Fatalf("log file was not created: %v", err)
	}
}
