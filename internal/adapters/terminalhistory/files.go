package terminalhistory

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	metaVersion       = 1
	checkpointVersion = 1
	outputLogVersion  = 1

	// MaxOutputLogBytes is the append-log payload cap after which callers should
	// write a new terminal checkpoint and truncate the generation log.
	MaxOutputLogBytes uint64 = 5 * 1024 * 1024

	StatusRunning = "running"
	StatusExited  = "exited"
)

var errUnsupportedVersion = errors.New("unsupported terminal history version")

// FileStore stores daemon-owned terminal restore records on disk.
type FileStore struct {
	root string
	now  func() time.Time
}

// PTYMeta is the durable ownership and display metadata for one PTY.
type PTYMeta struct {
	PTYID          string
	SessionID      string
	WindowID       string
	PaneID         string
	OriginWindowID string
	OriginPaneID   string
	WorkingDir     string
	Cols           int
	Rows           int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Status         string
	ExitCode       *int
}

// Checkpoint is a versioned terminal snapshot at a raw PTY byte offset.
type Checkpoint struct {
	PTYID            string
	Generation       uint64
	Offset           uint64
	Cols             int
	Rows             int
	CreatedAt        time.Time
	TerminalSnapshot json.RawMessage
}

// AppendResult reports the durable append-log state after an output append.
type AppendResult struct {
	AppendedOffset   uint64
	AppendedBytes    int
	LogPayloadBytes  uint64
	CheckpointNeeded bool
}

// RestoredPTY is a restorable checkpoint plus matching generation log bytes.
type RestoredPTY struct {
	Meta       PTYMeta
	Checkpoint Checkpoint
	LogBytes   []byte
}

type metaFileV1 struct {
	Version        int       `json:"version"`
	PTYID          string    `json:"ptyId"`
	SessionID      string    `json:"sessionId"`
	WindowID       string    `json:"windowId"`
	PaneID         string    `json:"paneId"`
	OriginWindowID string    `json:"originWindowId"`
	OriginPaneID   string    `json:"originPaneId"`
	WorkingDir     string    `json:"workingDir"`
	Cols           int       `json:"cols"`
	Rows           int       `json:"rows"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Status         string    `json:"status"`
	ExitCode       *int      `json:"exitCode"`
}

type checkpointFileV1 struct {
	Version          int             `json:"version"`
	PTYID            string          `json:"ptyId"`
	Generation       uint64          `json:"generation"`
	Offset           uint64          `json:"offset"`
	Cols             int             `json:"cols"`
	Rows             int             `json:"rows"`
	CreatedAt        time.Time       `json:"createdAt"`
	TerminalSnapshot json.RawMessage `json:"terminalSnapshot"`
}

type outputLogHeaderV1 struct {
	Version    int       `json:"version"`
	PTYID      string    `json:"ptyId"`
	Generation uint64    `json:"generation"`
	BaseOffset uint64    `json:"baseOffset"`
	CreatedAt  time.Time `json:"createdAt"`
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
	return filepath.Join(configDir, "whisk", "terminal-history"), nil
}

func (s *FileStore) RegisterPTY(ctx context.Context, meta PTYMeta) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePTYID(meta.PTYID); err != nil {
		return err
	}
	now := s.now()
	meta = normalizeMeta(meta, now)
	return s.writeMeta(meta)
}

func (s *FileStore) AppendOutput(ctx context.Context, ptyID string, offset uint64, data []byte) (AppendResult, error) {
	if err := ctx.Err(); err != nil {
		return AppendResult{}, err
	}
	if err := validatePTYID(ptyID); err != nil {
		return AppendResult{}, err
	}
	if _, err := s.readMeta(ptyID); err != nil {
		return AppendResult{}, err
	}
	header, headerBytes, payloadBytes, err := s.ensureOutputLog(ptyID)
	if err != nil {
		return AppendResult{}, err
	}
	result, err := appendOutputAtOffset(s.outputLogPath(ptyID), header, headerBytes, payloadBytes, offset, data)
	if err != nil {
		return AppendResult{}, err
	}
	result.CheckpointNeeded = result.LogPayloadBytes >= MaxOutputLogBytes
	return result, nil
}

func (s *FileStore) WriteCheckpoint(ctx context.Context, checkpoint Checkpoint) (Checkpoint, error) {
	if err := ctx.Err(); err != nil {
		return Checkpoint{}, err
	}
	if err := validatePTYID(checkpoint.PTYID); err != nil {
		return Checkpoint{}, err
	}
	if len(checkpoint.TerminalSnapshot) == 0 || !json.Valid(checkpoint.TerminalSnapshot) {
		return Checkpoint{}, fmt.Errorf("terminal snapshot must be valid json")
	}
	meta, err := s.readMeta(checkpoint.PTYID)
	if err != nil {
		return Checkpoint{}, err
	}
	now := s.now()
	checkpoint.Generation = s.nextGeneration(checkpoint.PTYID)
	checkpoint.CreatedAt = now
	checkpoint.TerminalSnapshot = cloneRawMessage(checkpoint.TerminalSnapshot)
	if err := s.writeCheckpoint(checkpoint); err != nil {
		return Checkpoint{}, err
	}
	if err := s.writeOutputLogHeader(outputLogHeaderV1{
		Version:    outputLogVersion,
		PTYID:      checkpoint.PTYID,
		Generation: checkpoint.Generation,
		BaseOffset: checkpoint.Offset,
		CreatedAt:  now,
	}); err != nil {
		return Checkpoint{}, err
	}
	meta.UpdatedAt = now
	if checkpoint.Cols > 0 {
		meta.Cols = checkpoint.Cols
	}
	if checkpoint.Rows > 0 {
		meta.Rows = checkpoint.Rows
	}
	if err := s.writeMeta(meta); err != nil {
		return Checkpoint{}, err
	}
	return checkpoint, nil
}

func (s *FileStore) MarkExit(ctx context.Context, ptyID string, code *int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePTYID(ptyID); err != nil {
		return err
	}
	meta, err := s.readMeta(ptyID)
	if err != nil {
		return err
	}
	meta.Status = StatusExited
	meta.ExitCode = cloneIntPointer(code)
	meta.UpdatedAt = s.now()
	return s.writeMeta(meta)
}

func (s *FileStore) ListRestorable(ctx context.Context) ([]RestoredPTY, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.ptysDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]RestoredPTY, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() {
			continue
		}
		ptyID := entry.Name()
		if validatePTYID(ptyID) != nil {
			continue
		}
		meta, err := s.readMeta(ptyID)
		if err != nil {
			continue
		}
		checkpoint, err := s.readCheckpoint(ptyID)
		if err != nil {
			continue
		}
		logBytes, err := s.readMatchingLogBytes(ptyID, checkpoint)
		if err != nil {
			logBytes = nil
		}
		out = append(out, RestoredPTY{
			Meta:       meta,
			Checkpoint: checkpoint,
			LogBytes:   logBytes,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Meta.PTYID < out[j].Meta.PTYID
	})
	return out, nil
}

func (s *FileStore) DeletePTY(ctx context.Context, ptyID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePTYID(ptyID); err != nil {
		return err
	}
	return os.RemoveAll(s.ptyDir(ptyID))
}

func (s *FileStore) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.RemoveAll(s.root)
}

func normalizeMeta(meta PTYMeta, now time.Time) PTYMeta {
	if meta.OriginWindowID == "" {
		meta.OriginWindowID = meta.WindowID
	}
	if meta.OriginPaneID == "" {
		meta.OriginPaneID = meta.PaneID
	}
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	if meta.UpdatedAt.IsZero() {
		meta.UpdatedAt = now
	}
	if meta.Status == "" {
		meta.Status = StatusRunning
	}
	meta.ExitCode = cloneIntPointer(meta.ExitCode)
	return meta
}

func (s *FileStore) nextGeneration(ptyID string) uint64 {
	var current uint64
	if checkpoint, err := s.readCheckpoint(ptyID); err == nil && checkpoint.Generation > current {
		current = checkpoint.Generation
	}
	if header, _, _, err := readOutputLogState(s.outputLogPath(ptyID)); err == nil && header.Generation > current {
		current = header.Generation
	}
	return current + 1
}

func (s *FileStore) ensureOutputLog(ptyID string) (outputLogHeaderV1, int64, uint64, error) {
	header, headerBytes, payloadBytes, err := readOutputLogState(s.outputLogPath(ptyID))
	if err == nil {
		if header.Version != outputLogVersion {
			return outputLogHeaderV1{}, 0, 0, fmt.Errorf("%w %d", errUnsupportedVersion, header.Version)
		}
		if header.PTYID != ptyID {
			return outputLogHeaderV1{}, 0, 0, fmt.Errorf("output log pty mismatch: %s != %s", header.PTYID, ptyID)
		}
		if checkpoint, err := s.readCheckpoint(ptyID); err == nil {
			if header.Generation != checkpoint.Generation || header.BaseOffset != checkpoint.Offset {
				return outputLogHeaderV1{}, 0, 0, fmt.Errorf(
					"output log generation mismatch for %s: log generation %d baseOffset %d checkpoint generation %d offset %d",
					ptyID,
					header.Generation,
					header.BaseOffset,
					checkpoint.Generation,
					checkpoint.Offset,
				)
			}
		}
		return header, headerBytes, payloadBytes, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return outputLogHeaderV1{}, 0, 0, err
	}

	header = outputLogHeaderV1{
		Version:    outputLogVersion,
		PTYID:      ptyID,
		Generation: 0,
		BaseOffset: 0,
		CreatedAt:  s.now(),
	}
	if checkpoint, err := s.readCheckpoint(ptyID); err == nil {
		header.Generation = checkpoint.Generation
		header.BaseOffset = checkpoint.Offset
		header.CreatedAt = checkpoint.CreatedAt
	}
	if err := s.writeOutputLogHeader(header); err != nil {
		return outputLogHeaderV1{}, 0, 0, err
	}
	return readOutputLogState(s.outputLogPath(ptyID))
}

func appendOutputAtOffset(path string, header outputLogHeaderV1, headerBytes int64, payloadBytes uint64, offset uint64, data []byte) (AppendResult, error) {
	currentOffset := header.BaseOffset + payloadBytes
	if currentOffset < header.BaseOffset {
		return AppendResult{}, fmt.Errorf("output log offset overflow")
	}
	result := AppendResult{
		AppendedOffset:  currentOffset,
		LogPayloadBytes: payloadBytes,
	}
	if len(data) == 0 {
		return result, nil
	}
	dataLen := uint64(len(data))
	if offset > ^uint64(0)-dataLen {
		return AppendResult{}, fmt.Errorf("output append offset overflow")
	}
	endOffset := offset + dataLen
	if offset > currentOffset {
		return AppendResult{}, fmt.Errorf("terminal history gap for %s: offset %d after current offset %d", filepath.Base(path), offset, currentOffset)
	}
	if endOffset <= header.BaseOffset {
		return result, nil
	}
	if offset < header.BaseOffset {
		trim := header.BaseOffset - offset
		data = data[trim:]
		offset = header.BaseOffset
	}
	if offset < currentOffset {
		overlap := currentOffset - offset
		if overlap >= uint64(len(data)) {
			return result, nil
		}
		data = data[overlap:]
		offset = currentOffset
	}
	file, err := os.OpenFile(path, os.O_WRONLY, 0o600)
	if err != nil {
		return AppendResult{}, err
	}
	defer file.Close()
	if _, err := file.Seek(headerBytes+int64(payloadBytes), io.SeekStart); err != nil {
		return AppendResult{}, err
	}
	written, err := file.Write(data)
	if err != nil {
		return AppendResult{}, err
	}
	if written != len(data) {
		return AppendResult{}, io.ErrShortWrite
	}
	result.AppendedOffset = offset
	result.AppendedBytes = written
	result.LogPayloadBytes = payloadBytes + uint64(written)
	return result, nil
}

func (s *FileStore) readMatchingLogBytes(ptyID string, checkpoint Checkpoint) ([]byte, error) {
	file, err := os.Open(s.outputLogPath(ptyID))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	headerLine, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	var header outputLogHeaderV1
	if err := json.Unmarshal(bytesTrimNewline(headerLine), &header); err != nil {
		return nil, err
	}
	if header.Version != outputLogVersion ||
		header.PTYID != ptyID ||
		header.Generation != checkpoint.Generation ||
		header.BaseOffset != checkpoint.Offset {
		return nil, nil
	}
	return io.ReadAll(reader)
}

func (s *FileStore) readMeta(ptyID string) (PTYMeta, error) {
	bytes, err := os.ReadFile(s.metaPath(ptyID))
	if err != nil {
		return PTYMeta{}, err
	}
	var file metaFileV1
	if err := json.Unmarshal(bytes, &file); err != nil {
		return PTYMeta{}, err
	}
	if file.Version != metaVersion {
		return PTYMeta{}, fmt.Errorf("%w %d", errUnsupportedVersion, file.Version)
	}
	if file.PTYID == "" {
		return PTYMeta{}, fmt.Errorf("pty id required")
	}
	if file.PTYID != ptyID {
		return PTYMeta{}, fmt.Errorf("meta pty mismatch: %s != %s", file.PTYID, ptyID)
	}
	return PTYMeta{
		PTYID:          file.PTYID,
		SessionID:      file.SessionID,
		WindowID:       file.WindowID,
		PaneID:         file.PaneID,
		OriginWindowID: file.OriginWindowID,
		OriginPaneID:   file.OriginPaneID,
		WorkingDir:     file.WorkingDir,
		Cols:           file.Cols,
		Rows:           file.Rows,
		CreatedAt:      file.CreatedAt,
		UpdatedAt:      file.UpdatedAt,
		Status:         file.Status,
		ExitCode:       cloneIntPointer(file.ExitCode),
	}, nil
}

func (s *FileStore) writeMeta(meta PTYMeta) error {
	if err := validatePTYID(meta.PTYID); err != nil {
		return err
	}
	file := metaFileV1{
		Version:        metaVersion,
		PTYID:          meta.PTYID,
		SessionID:      meta.SessionID,
		WindowID:       meta.WindowID,
		PaneID:         meta.PaneID,
		OriginWindowID: meta.OriginWindowID,
		OriginPaneID:   meta.OriginPaneID,
		WorkingDir:     meta.WorkingDir,
		Cols:           meta.Cols,
		Rows:           meta.Rows,
		CreatedAt:      meta.CreatedAt,
		UpdatedAt:      meta.UpdatedAt,
		Status:         meta.Status,
		ExitCode:       cloneIntPointer(meta.ExitCode),
	}
	return writeJSONFileAtomic(s.metaPath(meta.PTYID), file)
}

func (s *FileStore) readCheckpoint(ptyID string) (Checkpoint, error) {
	bytes, err := os.ReadFile(s.checkpointPath(ptyID))
	if err != nil {
		return Checkpoint{}, err
	}
	var file checkpointFileV1
	if err := json.Unmarshal(bytes, &file); err != nil {
		return Checkpoint{}, err
	}
	if file.Version != checkpointVersion {
		return Checkpoint{}, fmt.Errorf("%w %d", errUnsupportedVersion, file.Version)
	}
	if file.PTYID == "" {
		return Checkpoint{}, fmt.Errorf("pty id required")
	}
	if file.PTYID != ptyID {
		return Checkpoint{}, fmt.Errorf("checkpoint pty mismatch: %s != %s", file.PTYID, ptyID)
	}
	if len(file.TerminalSnapshot) == 0 || !json.Valid(file.TerminalSnapshot) {
		return Checkpoint{}, fmt.Errorf("terminal snapshot must be valid json")
	}
	return Checkpoint{
		PTYID:            file.PTYID,
		Generation:       file.Generation,
		Offset:           file.Offset,
		Cols:             file.Cols,
		Rows:             file.Rows,
		CreatedAt:        file.CreatedAt,
		TerminalSnapshot: cloneRawMessage(file.TerminalSnapshot),
	}, nil
}

func (s *FileStore) writeCheckpoint(checkpoint Checkpoint) error {
	file := checkpointFileV1{
		Version:          checkpointVersion,
		PTYID:            checkpoint.PTYID,
		Generation:       checkpoint.Generation,
		Offset:           checkpoint.Offset,
		Cols:             checkpoint.Cols,
		Rows:             checkpoint.Rows,
		CreatedAt:        checkpoint.CreatedAt,
		TerminalSnapshot: cloneRawMessage(checkpoint.TerminalSnapshot),
	}
	return writeJSONFileAtomic(s.checkpointPath(checkpoint.PTYID), file)
}

func (s *FileStore) writeOutputLogHeader(header outputLogHeaderV1) error {
	header.Version = outputLogVersion
	bytes, err := json.Marshal(header)
	if err != nil {
		return err
	}
	bytes = append(bytes, '\n')
	return writeFileAtomic(s.outputLogPath(header.PTYID), bytes, 0o600)
}

func readOutputLogState(path string) (outputLogHeaderV1, int64, uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return outputLogHeaderV1{}, 0, 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return outputLogHeaderV1{}, 0, 0, err
	}
	reader := bufio.NewReader(file)
	headerLine, err := reader.ReadBytes('\n')
	if err != nil {
		return outputLogHeaderV1{}, 0, 0, err
	}
	headerBytes := int64(len(headerLine))
	if info.Size() < headerBytes {
		return outputLogHeaderV1{}, 0, 0, fmt.Errorf("output log header longer than file")
	}
	var header outputLogHeaderV1
	if err := json.Unmarshal(bytesTrimNewline(headerLine), &header); err != nil {
		return outputLogHeaderV1{}, 0, 0, err
	}
	return header, headerBytes, uint64(info.Size() - headerBytes), nil
}

func writeJSONFileAtomic(path string, value any) error {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	bytes = append(bytes, '\n')
	return writeFileAtomic(path, bytes, 0o600)
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
	if err := temp.Sync(); err != nil {
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

func validatePTYID(ptyID string) error {
	if ptyID == "" {
		return fmt.Errorf("pty id required")
	}
	if ptyID == "." || ptyID == ".." || strings.ContainsAny(ptyID, `/\`) {
		return fmt.Errorf("invalid pty id %q", ptyID)
	}
	return nil
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func bytesTrimNewline(bytes []byte) []byte {
	return bytes[:len(bytes)-1]
}

func (s *FileStore) ptysDir() string {
	return filepath.Join(s.root, "ptys")
}

func (s *FileStore) ptyDir(ptyID string) string {
	return filepath.Join(s.ptysDir(), ptyID)
}

func (s *FileStore) metaPath(ptyID string) string {
	return filepath.Join(s.ptyDir(ptyID), "meta.json")
}

func (s *FileStore) checkpointPath(ptyID string) string {
	return filepath.Join(s.ptyDir(ptyID), "checkpoint.json")
}

func (s *FileStore) outputLogPath(ptyID string) string {
	return filepath.Join(s.ptyDir(ptyID), "output.log")
}
