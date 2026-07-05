package terminalhistory

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultRootUsesXDGConfigHomeOrHomeDotConfig(t *testing.T) {
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	root, err := DefaultRoot()
	if err != nil {
		t.Fatalf("default root: %v", err)
	}
	if root != filepath.Join(xdg, "whisk", "terminal-history") {
		t.Fatalf("root = %q", root)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	root, err = DefaultRoot()
	if err != nil {
		t.Fatalf("default root with home: %v", err)
	}
	if root != filepath.Join(home, ".config", "whisk", "terminal-history") {
		t.Fatalf("root = %q", root)
	}
}

func TestRegisterPTYWritesVersionedMetadataAndPrivatePermissions(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := newTestStore(t, root)

	if err := store.RegisterPTY(ctx, PTYMeta{
		PTYID:      "pty_01",
		SessionID:  "sess_01",
		WindowID:   "win_01",
		PaneID:     "pane_01",
		WorkingDir: "/repo",
		Cols:       120,
		Rows:       36,
	}); err != nil {
		t.Fatalf("register pty: %v", err)
	}

	var meta metaFileV1
	readJSONFile(t, filepath.Join(root, "ptys", "pty_01", "meta.json"), &meta)
	if meta.Version != 1 ||
		meta.PTYID != "pty_01" ||
		meta.SessionID != "sess_01" ||
		meta.WindowID != "win_01" ||
		meta.PaneID != "pane_01" ||
		meta.OriginWindowID != "win_01" ||
		meta.OriginPaneID != "pane_01" ||
		meta.WorkingDir != "/repo" ||
		meta.Cols != 120 ||
		meta.Rows != 36 ||
		meta.Status != StatusRunning ||
		meta.ExitCode != nil {
		t.Fatalf("meta = %#v", meta)
	}
	assertMode(t, filepath.Join(root, "ptys"), 0o700)
	assertMode(t, filepath.Join(root, "ptys", "pty_01"), 0o700)
	assertMode(t, filepath.Join(root, "ptys", "pty_01", "meta.json"), 0o600)
}

func TestAppendOutputCreatesGenerationZeroLogAndHandlesOverlap(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := newTestStore(t, root)
	registerTestPTY(t, store, "pty_01")

	result, err := store.AppendOutput(ctx, "pty_01", 0, []byte("hello"))
	if err != nil {
		t.Fatalf("append output: %v", err)
	}
	if result.AppendedOffset != 0 || result.AppendedBytes != 5 || result.LogPayloadBytes != 5 || result.CheckpointNeeded {
		t.Fatalf("first append result = %#v", result)
	}

	result, err = store.AppendOutput(ctx, "pty_01", 3, []byte("lo world"))
	if err != nil {
		t.Fatalf("append partial overlap: %v", err)
	}
	if result.AppendedOffset != 5 || result.AppendedBytes != 6 || result.LogPayloadBytes != 11 || result.CheckpointNeeded {
		t.Fatalf("overlap append result = %#v", result)
	}

	result, err = store.AppendOutput(ctx, "pty_01", 0, []byte("hello"))
	if err != nil {
		t.Fatalf("append duplicate overlap: %v", err)
	}
	if result.AppendedBytes != 0 || result.LogPayloadBytes != 11 {
		t.Fatalf("duplicate append result = %#v", result)
	}

	header, payload := readOutputLog(t, root, "pty_01")
	if header.Version != 1 || header.PTYID != "pty_01" || header.Generation != 0 || header.BaseOffset != 0 {
		t.Fatalf("header = %#v", header)
	}
	if string(payload) != "hello world" {
		t.Fatalf("payload = %q", string(payload))
	}
}

func TestAppendOutputRejectsGaps(t *testing.T) {
	store := newTestStore(t, t.TempDir())
	registerTestPTY(t, store, "pty_01")

	_, err := store.AppendOutput(context.Background(), "pty_01", 4, []byte("gap"))
	if err == nil {
		t.Fatalf("expected gap error")
	}
	if !strings.Contains(err.Error(), "gap") {
		t.Fatalf("error = %v", err)
	}
}

func TestWriteCheckpointIncrementsGenerationTruncatesLogAndRestoresMatchingGeneration(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := newTestStore(t, root)
	registerTestPTY(t, store, "pty_01")

	if _, err := store.AppendOutput(ctx, "pty_01", 0, []byte("before")); err != nil {
		t.Fatalf("append before checkpoint: %v", err)
	}
	written, err := store.WriteCheckpoint(ctx, Checkpoint{
		PTYID:            "pty_01",
		Offset:           6,
		Cols:             120,
		Rows:             36,
		TerminalSnapshot: json.RawMessage(`{"cursor":[1,2]}`),
	})
	if err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
	if written.Generation != 1 || written.Offset != 6 || !json.Valid(written.TerminalSnapshot) {
		t.Fatalf("written checkpoint = %#v", written)
	}

	var checkpoint checkpointFileV1
	readJSONFile(t, filepath.Join(root, "ptys", "pty_01", "checkpoint.json"), &checkpoint)
	if checkpoint.Version != 1 || checkpoint.Generation != 1 || checkpoint.Offset != 6 ||
		compactJSON(t, checkpoint.TerminalSnapshot) != `{"cursor":[1,2]}` {
		t.Fatalf("checkpoint file = %#v", checkpoint)
	}
	header, payload := readOutputLog(t, root, "pty_01")
	if header.Generation != 1 || header.BaseOffset != 6 || len(payload) != 0 {
		t.Fatalf("log after checkpoint header=%#v payload=%q", header, string(payload))
	}

	if _, err := store.AppendOutput(ctx, "pty_01", 6, []byte("after")); err != nil {
		t.Fatalf("append after checkpoint: %v", err)
	}
	restored, err := store.ListRestorable(ctx)
	if err != nil {
		t.Fatalf("list restorable: %v", err)
	}
	if len(restored) != 1 {
		t.Fatalf("restored = %#v", restored)
	}
	if restored[0].Checkpoint.Generation != 1 ||
		restored[0].Checkpoint.Offset != 6 ||
		string(restored[0].LogBytes) != "after" ||
		compactJSON(t, restored[0].Checkpoint.TerminalSnapshot) != `{"cursor":[1,2]}` {
		t.Fatalf("restored record = %#v", restored[0])
	}

	written, err = store.WriteCheckpoint(ctx, Checkpoint{
		PTYID:            "pty_01",
		Offset:           11,
		Cols:             120,
		Rows:             36,
		TerminalSnapshot: json.RawMessage(`{"screen":"next"}`),
	})
	if err != nil {
		t.Fatalf("write second checkpoint: %v", err)
	}
	if written.Generation != 2 {
		t.Fatalf("second checkpoint generation = %d", written.Generation)
	}
	readBack, err := store.ReadCheckpoint(ctx, "pty_01")
	if err != nil {
		t.Fatalf("read checkpoint: %v", err)
	}
	if readBack.Generation != 2 ||
		readBack.Offset != 11 ||
		compactJSON(t, readBack.TerminalSnapshot) != `{"screen":"next"}` {
		t.Fatalf("read checkpoint = %#v", readBack)
	}
	header, payload = readOutputLog(t, root, "pty_01")
	if header.Generation != 2 || header.BaseOffset != 11 || len(payload) != 0 {
		t.Fatalf("second log header=%#v payload=%q", header, string(payload))
	}
}

func TestAppendOutputTrimsBytesBeforeCheckpointBaseOffset(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := newTestStore(t, root)
	registerTestPTY(t, store, "pty_01")

	if _, err := store.WriteCheckpoint(ctx, Checkpoint{
		PTYID:            "pty_01",
		Offset:           10,
		Cols:             80,
		Rows:             24,
		TerminalSnapshot: json.RawMessage(`{"base":true}`),
	}); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
	result, err := store.AppendOutput(ctx, "pty_01", 8, []byte("xxnew"))
	if err != nil {
		t.Fatalf("append crossing base offset: %v", err)
	}
	if result.AppendedOffset != 10 || result.AppendedBytes != 3 || result.LogPayloadBytes != 3 {
		t.Fatalf("append result = %#v", result)
	}
	_, payload := readOutputLog(t, root, "pty_01")
	if string(payload) != "new" {
		t.Fatalf("payload = %q", string(payload))
	}
}

func TestListRestorableIgnoresMismatchedAndCorruptOutputLogs(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := newTestStore(t, root)
	registerTestPTY(t, store, "pty_01")

	if _, err := store.WriteCheckpoint(ctx, Checkpoint{
		PTYID:            "pty_01",
		Offset:           5,
		Cols:             80,
		Rows:             24,
		TerminalSnapshot: json.RawMessage(`{"ok":true}`),
	}); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
	writeOutputLogForTest(t, root, "pty_01", outputLogHeaderV1{
		Version:    1,
		PTYID:      "pty_01",
		Generation: 99,
		BaseOffset: 5,
		CreatedAt:  testNow,
	}, []byte("ignored"))

	restored, err := store.ListRestorable(ctx)
	if err != nil {
		t.Fatalf("list restorable after mismatch: %v", err)
	}
	if len(restored) != 1 || len(restored[0].LogBytes) != 0 {
		t.Fatalf("restored after mismatch = %#v", restored)
	}

	logPath := filepath.Join(root, "ptys", "pty_01", "output.log")
	if err := os.WriteFile(logPath, []byte("{not-json\nignored"), 0o600); err != nil {
		t.Fatalf("write corrupt log: %v", err)
	}
	restored, err = store.ListRestorable(ctx)
	if err != nil {
		t.Fatalf("list restorable after corrupt log: %v", err)
	}
	if len(restored) != 1 || len(restored[0].LogBytes) != 0 {
		t.Fatalf("restored after corrupt log = %#v", restored)
	}
}

func TestListRestorableSkipsCorruptRecords(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := newTestStore(t, root)
	registerTestPTY(t, store, "good")
	if _, err := store.WriteCheckpoint(ctx, Checkpoint{
		PTYID:            "good",
		Offset:           0,
		Cols:             80,
		Rows:             24,
		TerminalSnapshot: json.RawMessage(`{"ok":true}`),
	}); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}

	badMetaDir := filepath.Join(root, "ptys", "bad-meta")
	if err := os.MkdirAll(badMetaDir, 0o700); err != nil {
		t.Fatalf("mkdir bad meta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(badMetaDir, "meta.json"), []byte(`{`), 0o600); err != nil {
		t.Fatalf("write bad meta: %v", err)
	}

	badCheckpointDir := filepath.Join(root, "ptys", "bad-checkpoint")
	if err := os.MkdirAll(badCheckpointDir, 0o700); err != nil {
		t.Fatalf("mkdir bad checkpoint: %v", err)
	}
	writeJSONFileForTest(t, filepath.Join(badCheckpointDir, "meta.json"), metaFileV1{
		Version:   1,
		CreatedAt: testNow,
		UpdatedAt: testNow,
		PTYID:     "bad-checkpoint",
		Status:    StatusRunning,
	})
	if err := os.WriteFile(filepath.Join(badCheckpointDir, "checkpoint.json"), []byte(`{"version":99}`), 0o600); err != nil {
		t.Fatalf("write bad checkpoint: %v", err)
	}

	restored, err := store.ListRestorable(ctx)
	if err != nil {
		t.Fatalf("list restorable: %v", err)
	}
	if len(restored) != 1 || restored[0].Meta.PTYID != "good" {
		t.Fatalf("restored = %#v", restored)
	}
}

func TestListRestorableSortsByDurableMetadataCreationTime(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := newTestStore(t, root)
	createdFirst := testNow.Add(-2 * time.Hour)
	createdSecond := testNow.Add(-1 * time.Hour)

	for _, meta := range []PTYMeta{
		{PTYID: "pty_late", CreatedAt: createdSecond},
		{PTYID: "pty_tie_b", CreatedAt: createdFirst},
		{PTYID: "pty_tie_a", CreatedAt: createdFirst},
	} {
		if err := store.RegisterPTY(ctx, PTYMeta{
			PTYID:      meta.PTYID,
			SessionID:  "sess_01",
			WindowID:   "win_01",
			PaneID:     "pane_01",
			WorkingDir: "/repo",
			Cols:       80,
			Rows:       24,
			CreatedAt:  meta.CreatedAt,
		}); err != nil {
			t.Fatalf("register %s: %v", meta.PTYID, err)
		}
		if _, err := store.WriteCheckpoint(ctx, Checkpoint{
			PTYID:            meta.PTYID,
			Cols:             80,
			Rows:             24,
			TerminalSnapshot: json.RawMessage(`{"ok":true}`),
		}); err != nil {
			t.Fatalf("write checkpoint for %s: %v", meta.PTYID, err)
		}
	}

	restored, err := store.ListRestorable(ctx)
	if err != nil {
		t.Fatalf("list restorable: %v", err)
	}
	got := restoredPTYIDs(restored)
	want := []string{"pty_tie_a", "pty_tie_b", "pty_late"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("restored order = %#v, want %#v", got, want)
	}
}

func TestMarkExitUpdatesMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := newTestStore(t, root)
	registerTestPTY(t, store, "pty_01")

	code := 42
	if err := store.MarkExit(ctx, "pty_01", &code); err != nil {
		t.Fatalf("mark exit: %v", err)
	}
	var meta metaFileV1
	readJSONFile(t, filepath.Join(root, "ptys", "pty_01", "meta.json"), &meta)
	if meta.Status != StatusExited || meta.ExitCode == nil || *meta.ExitCode != 42 {
		t.Fatalf("meta = %#v", meta)
	}
}

func TestDeletePTYAndClearPruneRestoreFiles(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := newTestStore(t, root)
	registerTestPTY(t, store, "pty_01")
	registerTestPTY(t, store, "pty_02")

	if err := store.DeletePTY(ctx, "pty_01"); err != nil {
		t.Fatalf("delete pty: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "ptys", "pty_01")); !os.IsNotExist(err) {
		t.Fatalf("pty_01 dir err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "ptys", "pty_02")); err != nil {
		t.Fatalf("pty_02 dir missing: %v", err)
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("root err = %v", err)
	}
}

func TestAppendResultFlagsCheckpointNeededAtFiveMB(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, t.TempDir())
	registerTestPTY(t, store, "pty_01")

	almostLimit := bytes.Repeat([]byte("a"), int(MaxOutputLogBytes)-1)
	result, err := store.AppendOutput(ctx, "pty_01", 0, almostLimit)
	if err != nil {
		t.Fatalf("append almost limit: %v", err)
	}
	if result.LogPayloadBytes != MaxOutputLogBytes-1 || result.CheckpointNeeded {
		t.Fatalf("almost limit result = %#v", result)
	}

	result, err = store.AppendOutput(ctx, "pty_01", MaxOutputLogBytes-1, []byte("b"))
	if err != nil {
		t.Fatalf("append limit byte: %v", err)
	}
	if result.LogPayloadBytes != MaxOutputLogBytes || !result.CheckpointNeeded {
		t.Fatalf("limit result = %#v", result)
	}
}

func TestWriteFileAtomicCleansTemporaryFileOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := writeFileAtomic(target, []byte("replacement"), 0o600); err == nil {
		t.Fatalf("expected rename failure")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tmp-") {
			t.Fatalf("temporary file was not cleaned up: %s", entry.Name())
		}
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("target should remain a directory")
	}
}

var testNow = time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)

func newTestStore(t *testing.T, root string) *FileStore {
	t.Helper()
	store, err := NewFileStore(root)
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}
	store.now = func() time.Time { return testNow }
	return store
}

func registerTestPTY(t *testing.T, store *FileStore, ptyID string) {
	t.Helper()
	if err := store.RegisterPTY(context.Background(), PTYMeta{
		PTYID:      ptyID,
		SessionID:  "sess_01",
		WindowID:   "win_01",
		PaneID:     "pane_01",
		WorkingDir: "/repo",
		Cols:       80,
		Rows:       24,
	}); err != nil {
		t.Fatalf("register %s: %v", ptyID, err)
	}
}

func readOutputLog(t *testing.T, root, ptyID string) (outputLogHeaderV1, []byte) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "ptys", ptyID, "output.log"))
	if err != nil {
		t.Fatalf("read output log: %v", err)
	}
	parts := bytes.SplitN(raw, []byte("\n"), 2)
	if len(parts) != 2 {
		t.Fatalf("output log missing header newline: %q", string(raw))
	}
	var header outputLogHeaderV1
	if err := json.Unmarshal(parts[0], &header); err != nil {
		t.Fatalf("decode output log header: %v", err)
	}
	return header, parts[1]
}

func writeOutputLogForTest(t *testing.T, root, ptyID string, header outputLogHeaderV1, payload []byte) {
	t.Helper()
	headerBytes, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	path := filepath.Join(root, "ptys", ptyID, "output.log")
	if err := os.WriteFile(path, append(append(headerBytes, '\n'), payload...), 0o600); err != nil {
		t.Fatalf("write output log: %v", err)
	}
}

func readJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(bytes, value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func writeJSONFileForTest(t *testing.T, path string, value any) {
	t.Helper()
	bytes, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, bytes, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %o, want %o", path, got, want)
	}
}

func compactJSON(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		t.Fatalf("compact json: %v", err)
	}
	return out.String()
}

func restoredPTYIDs(restored []RestoredPTY) []string {
	out := make([]string, 0, len(restored))
	for _, item := range restored {
		out = append(out, item.Meta.PTYID)
	}
	return out
}
