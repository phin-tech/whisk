package native

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/creack/pty"
	"github.com/phin-tech/whisk/internal/app"
)

const outputLimitBytes = 256 * 1024

type Backend struct {
	mu   sync.Mutex
	ptys map[string]*proc
}

type proc struct {
	record app.PTYRecord
	cmd    *exec.Cmd
	master *os.File
	buffer *outputBuffer
	subs   map[chan app.PTYEvent]struct{}
}

func NewBackend() *Backend {
	return &Backend{ptys: map[string]*proc{}}
}

func (b *Backend) Spawn(_ context.Context, req app.SpawnPTYRequest) (app.PTYRecord, error) {
	if req.ID == "" {
		return app.PTYRecord{}, fmt.Errorf("pty id required")
	}
	workingDir, err := resolveWorkingDir(req.WorkingDir)
	if err != nil {
		return app.PTYRecord{}, err
	}
	cols := req.Cols
	if cols <= 0 {
		cols = 80
	}
	rows := req.Rows
	if rows <= 0 {
		rows = 24
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	cmd := exec.Command(shell)
	cmd.Dir = workingDir

	record := app.PTYRecord{
		ID:         req.ID,
		WorkingDir: workingDir,
		Cols:       cols,
		Rows:       rows,
		Running:    true,
	}
	p := &proc{
		record: record,
		cmd:    cmd,
		buffer: newOutputBuffer(outputLimitBytes),
		subs:   map[chan app.PTYEvent]struct{}{},
	}

	b.mu.Lock()
	if _, exists := b.ptys[req.ID]; exists {
		b.mu.Unlock()
		return app.PTYRecord{}, fmt.Errorf("pty %s already exists", req.ID)
	}
	b.ptys[req.ID] = p
	b.mu.Unlock()

	master, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		b.mu.Lock()
		delete(b.ptys, req.ID)
		b.mu.Unlock()
		return app.PTYRecord{}, err
	}
	p.master = master
	go b.capture(req.ID, master)
	go b.wait(req.ID, cmd)
	return record, nil
}

func (b *Backend) Write(_ context.Context, ptyID string, data []byte) error {
	b.mu.Lock()
	p, ok := b.ptys[ptyID]
	b.mu.Unlock()
	if !ok {
		return fmt.Errorf("pty %s not found", ptyID)
	}
	_, err := p.master.Write(data)
	return err
}

func (b *Backend) Resize(_ context.Context, ptyID string, size app.PTYSize) error {
	if size.Cols <= 0 {
		return fmt.Errorf("pty cols must be positive")
	}
	if size.Rows <= 0 {
		return fmt.Errorf("pty rows must be positive")
	}

	b.mu.Lock()
	p, ok := b.ptys[ptyID]
	if !ok {
		b.mu.Unlock()
		return fmt.Errorf("pty %s not found", ptyID)
	}
	master := p.master
	p.record.Cols = size.Cols
	p.record.Rows = size.Rows
	b.mu.Unlock()

	return pty.Setsize(master, &pty.Winsize{Cols: uint16(size.Cols), Rows: uint16(size.Rows)})
}

func (b *Backend) Attach(ctx context.Context, req app.AttachPTYRequest) (*app.PTYAttach, error) {
	ch := make(chan app.PTYEvent, 64)
	b.mu.Lock()
	p, ok := b.ptys[req.PtyID]
	if !ok {
		b.mu.Unlock()
		return nil, fmt.Errorf("pty %s not found", req.PtyID)
	}
	replayOffset, replayBytes := p.buffer.snapshotFrom(req.ReplayFromOffset)
	p.subs[ch] = struct{}{}
	record := p.record
	b.mu.Unlock()

	cancelOnce := sync.Once{}
	cancel := func() {
		cancelOnce.Do(func() {
			b.mu.Lock()
			if p, ok := b.ptys[req.PtyID]; ok {
				delete(p.subs, ch)
			}
			b.mu.Unlock()
			close(ch)
		})
	}
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return &app.PTYAttach{
		Record:       record,
		ReplayBytes:  replayBytes,
		ReplayOffset: replayOffset,
		Events:       ch,
		CloseFunc:    cancel,
	}, nil
}

func (b *Backend) Output(_ context.Context, ptyID string, fromOffset uint64) (app.PTYOutputSnapshot, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	p, ok := b.ptys[ptyID]
	if !ok {
		return app.PTYOutputSnapshot{}, fmt.Errorf("pty %s not found", ptyID)
	}
	offset, bytes := p.buffer.snapshotFrom(fromOffset)
	return app.PTYOutputSnapshot{
		Record:      p.record,
		Offset:      offset,
		OutputBytes: bytes,
	}, nil
}

func (b *Backend) List(context.Context) ([]app.PTYRecord, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]app.PTYRecord, 0, len(b.ptys))
	for _, p := range b.ptys {
		out = append(out, p.record)
	}
	return out, nil
}

func (b *Backend) Shutdown(_ context.Context) error {
	b.mu.Lock()
	ptys := make([]*proc, 0, len(b.ptys))
	for _, p := range b.ptys {
		ptys = append(ptys, p)
	}
	b.ptys = map[string]*proc{}
	b.mu.Unlock()
	for _, p := range ptys {
		_ = p.master.Close()
		if p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
		}
	}
	return nil
}

func (b *Backend) capture(id string, master *os.File) {
	buf := make([]byte, 4096)
	for {
		n, err := master.Read(buf)
		if n > 0 {
			b.broadcastOutput(id, append([]byte(nil), buf[:n]...))
		}
		if err != nil {
			if err != io.EOF {
				return
			}
			return
		}
	}
}

func (b *Backend) wait(id string, cmd *exec.Cmd) {
	err := cmd.Wait()
	var code int
	if err == nil {
		code = 0
	} else if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else {
		code = -1
	}
	b.mu.Lock()
	p, ok := b.ptys[id]
	if !ok {
		b.mu.Unlock()
		return
	}
	p.record.Running = false
	event := app.PTYEvent{Kind: app.PTYExit, Code: &code}
	for ch := range p.subs {
		select {
		case ch <- event:
		default:
		}
	}
	b.mu.Unlock()
}

func (b *Backend) broadcastOutput(id string, data []byte) {
	b.mu.Lock()
	p, ok := b.ptys[id]
	if !ok {
		b.mu.Unlock()
		return
	}
	offset := p.buffer.append(data)
	event := app.PTYEvent{Kind: app.PTYOutput, Offset: offset, Bytes: data}
	for ch := range p.subs {
		select {
		case ch <- event:
		default:
		}
	}
	b.mu.Unlock()
}

func resolveWorkingDir(path string) (string, error) {
	if path == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("working dir is not a directory: %s", abs)
	}
	return abs, nil
}

type outputBuffer struct {
	bytes      []byte
	start      uint64
	total      uint64
	limitBytes int
}

func newOutputBuffer(limitBytes int) *outputBuffer {
	return &outputBuffer{limitBytes: limitBytes}
}

func (b *outputBuffer) append(data []byte) uint64 {
	offset := b.total
	b.bytes = append(b.bytes, data...)
	b.total += uint64(len(data))
	if len(b.bytes) > b.limitBytes {
		drop := len(b.bytes) - b.limitBytes
		b.bytes = append([]byte(nil), b.bytes[drop:]...)
		b.start += uint64(drop)
	}
	return offset
}

func (b *outputBuffer) snapshotFrom(offset uint64) (uint64, []byte) {
	if offset < b.start {
		offset = b.start
	}
	if offset > b.total {
		offset = b.total
	}
	index := int(offset - b.start)
	return offset, append([]byte(nil), b.bytes[index:]...)
}
