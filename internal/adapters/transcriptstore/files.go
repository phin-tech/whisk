package transcriptstore

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/phin-tech/whisk/internal/app"
)

type FileStore struct {
	root string
	now  func() time.Time
}

type ptyMetaFile struct {
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"createdAt"`
	PTYID      string    `json:"ptyId"`
	SessionID  string    `json:"sessionId"`
	WindowID   string    `json:"windowId"`
	PaneID     string    `json:"paneId"`
	WorkingDir string    `json:"workingDir"`
	Cols       int       `json:"cols"`
	Rows       int       `json:"rows"`
}

type eventFileRecord struct {
	Time   time.Time `json:"time"`
	Type   string    `json:"type"`
	PTYID  string    `json:"ptyId"`
	Offset uint64    `json:"offset,omitempty"`
	Length int       `json:"length,omitempty"`
	Code   *int      `json:"code,omitempty"`
}

func NewFileStore(root string) (*FileStore, error) {
	if root == "" {
		defaultRoot, err := DefaultRoot()
		if err != nil {
			return nil, err
		}
		root = defaultRoot
	}
	return &FileStore{root: filepath.Clean(root), now: func() time.Time { return time.Now().UTC() }}, nil
}

func DefaultRoot() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "whisk", "transcripts"), nil
}

func (s *FileStore) RegisterPTY(_ context.Context, meta app.PTYTranscriptMeta) error {
	if meta.PTYID == "" {
		return fmt.Errorf("pty id required")
	}
	if err := os.MkdirAll(s.ptyDir(), 0o700); err != nil {
		return err
	}
	file := ptyMetaFile{
		Version:    1,
		CreatedAt:  s.now(),
		PTYID:      meta.PTYID,
		SessionID:  meta.SessionID,
		WindowID:   meta.WindowID,
		PaneID:     meta.PaneID,
		WorkingDir: meta.WorkingDir,
		Cols:       meta.Cols,
		Rows:       meta.Rows,
	}
	bytes, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	bytes = append(bytes, '\n')
	return writeFileAtomic(s.metaPath(meta.PTYID), bytes, 0o600)
}

func (s *FileStore) AppendPTYOutput(ctx context.Context, event app.PTYTranscriptOutput) error {
	if event.PTYID == "" {
		return fmt.Errorf("pty id required")
	}
	if len(event.Bytes) == 0 {
		return nil
	}
	if err := os.MkdirAll(s.ptyDir(), 0o700); err != nil {
		return err
	}
	offset, length, err := appendRawAtOffset(s.rawPath(event.PTYID), event.Offset, event.Bytes)
	if err != nil {
		return err
	}
	if length == 0 {
		return nil
	}
	return s.appendEvent(ctx, eventFileRecord{
		Time:   s.now(),
		Type:   "pty.output",
		PTYID:  event.PTYID,
		Offset: offset,
		Length: length,
	})
}

func (s *FileStore) MarkPTYExit(ctx context.Context, event app.PTYTranscriptExit) error {
	if event.PTYID == "" {
		return fmt.Errorf("pty id required")
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	return s.appendEvent(ctx, eventFileRecord{
		Time:  s.now(),
		Type:  "pty.exit",
		PTYID: event.PTYID,
		Code:  event.Code,
	})
}

func (s *FileStore) ListPTYHistory(_ context.Context) ([]app.PTYHistorySummary, error) {
	metas, err := filepath.Glob(filepath.Join(s.ptyDir(), "*.json"))
	if err != nil {
		return nil, err
	}
	exits, err := s.exitCodes()
	if err != nil {
		return nil, err
	}
	out := make([]app.PTYHistorySummary, 0, len(metas))
	for _, path := range metas {
		meta, err := readPTYMeta(path)
		if err != nil {
			return nil, err
		}
		summary := ptyHistorySummary(meta)
		summary.ExitCode = exits[summary.PTYID]
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *FileStore) ReadPTYHistory(_ context.Context, ptyID string) (app.PTYHistory, error) {
	if ptyID == "" {
		return app.PTYHistory{}, fmt.Errorf("pty id required")
	}
	meta, err := readPTYMeta(s.metaPath(ptyID))
	if err != nil {
		return app.PTYHistory{}, err
	}
	raw, err := os.ReadFile(s.rawPath(ptyID))
	if err != nil && !os.IsNotExist(err) {
		return app.PTYHistory{}, err
	}
	exits, err := s.exitCodes()
	if err != nil {
		return app.PTYHistory{}, err
	}
	summary := ptyHistorySummary(meta)
	summary.ExitCode = exits[ptyID]
	return app.PTYHistory{PTYHistorySummary: summary, Output: string(raw)}, nil
}

func (s *FileStore) appendEvent(_ context.Context, event eventFileRecord) error {
	file, err := os.OpenFile(filepath.Join(s.root, "events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(bytes, '\n')); err != nil {
		return err
	}
	return nil
}

func readPTYMeta(path string) (ptyMetaFile, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return ptyMetaFile{}, err
	}
	var meta ptyMetaFile
	if err := json.Unmarshal(bytes, &meta); err != nil {
		return ptyMetaFile{}, err
	}
	return meta, nil
}

func ptyHistorySummary(meta ptyMetaFile) app.PTYHistorySummary {
	return app.PTYHistorySummary{
		PTYID:      meta.PTYID,
		SessionID:  meta.SessionID,
		WindowID:   meta.WindowID,
		PaneID:     meta.PaneID,
		WorkingDir: meta.WorkingDir,
		CreatedAt:  meta.CreatedAt,
	}
}

func (s *FileStore) exitCodes() (map[string]*int, error) {
	out := map[string]*int{}
	file, err := os.Open(filepath.Join(s.root, "events.jsonl"))
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event eventFileRecord
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, err
		}
		if event.Type == "pty.exit" && event.PTYID != "" {
			out[event.PTYID] = event.Code
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func appendRawAtOffset(path string, offset uint64, bytes []byte) (uint64, int, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, 0, err
	}
	size := uint64(info.Size())
	if offset > size {
		return 0, 0, fmt.Errorf("transcript gap for %s: offset %d after size %d", filepath.Base(path), offset, size)
	}
	if offset < size {
		overlap := size - offset
		if overlap >= uint64(len(bytes)) {
			return size, 0, nil
		}
		bytes = bytes[overlap:]
		offset = size
	}
	if _, err := file.Seek(int64(offset), 0); err != nil {
		return 0, 0, err
	}
	if _, err := file.Write(bytes); err != nil {
		return 0, 0, err
	}
	return offset, len(bytes), nil
}

func writeFileAtomic(path string, bytes []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := temp.Write(bytes); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Chmod(perm); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (s *FileStore) ptyDir() string {
	return filepath.Join(s.root, "ptys")
}

func (s *FileStore) metaPath(ptyID string) string {
	return filepath.Join(s.ptyDir(), ptyID+".json")
}

func (s *FileStore) rawPath(ptyID string) string {
	return filepath.Join(s.ptyDir(), ptyID+".raw")
}
