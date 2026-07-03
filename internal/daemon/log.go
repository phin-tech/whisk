package daemon

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	defaultLogMaxBytes = int64(1024 * 1024)
	defaultLogBackups  = 4
)

// LogRotation bounds the daemon log footprint to MaxBytes times the current file plus backups.
type LogRotation struct {
	MaxBytes   int64
	MaxBackups int
}

func DefaultLogRotation() LogRotation {
	return LogRotation{
		MaxBytes:   defaultLogMaxBytes,
		MaxBackups: defaultLogBackups,
	}
}

func (rotation LogRotation) normalized() LogRotation {
	defaults := DefaultLogRotation()
	if rotation.MaxBytes <= 0 {
		rotation.MaxBytes = defaults.MaxBytes
	}
	if rotation.MaxBackups < 0 {
		rotation.MaxBackups = 0
	}
	return rotation
}

func LogPath(baseURL string) (string, error) {
	addr, err := addrFromURL(baseURL)
	if err != nil {
		return "", err
	}
	return LogPathForListenAddress(addr)
}

func LogPathForListenAddress(addr string) (string, error) {
	root, err := daemonStateRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "daemon-"+sanitizeDaemonAddr(addr)+".log"), nil
}

type RotatingLogWriter struct {
	mu       sync.Mutex
	path     string
	rotation LogRotation
	file     *os.File
	size     int64
}

func NewRotatingLogWriter(path string, rotation LogRotation) (*RotatingLogWriter, error) {
	writer := &RotatingLogWriter{
		path:     path,
		rotation: rotation.normalized(),
	}
	if err := writer.openLocked(); err != nil {
		return nil, err
	}
	return writer, nil
}

func (writer *RotatingLogWriter) Write(p []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	originalLen := len(p)
	if originalLen == 0 {
		return 0, nil
	}
	if int64(len(p)) > writer.rotation.MaxBytes {
		p = p[len(p)-int(writer.rotation.MaxBytes):]
	}
	if err := writer.openLocked(); err != nil {
		return 0, err
	}
	if writer.size > 0 && writer.size+int64(len(p)) > writer.rotation.MaxBytes {
		if err := writer.rotateLocked(); err != nil {
			return 0, err
		}
	}
	n, err := writer.file.Write(p)
	writer.size += int64(n)
	if err != nil {
		return n, err
	}
	if n != len(p) {
		return n, io.ErrShortWrite
	}
	return originalLen, nil
}

func (writer *RotatingLogWriter) Close() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.file == nil {
		return nil
	}
	err := writer.file.Close()
	writer.file = nil
	return err
}

func (writer *RotatingLogWriter) openLocked() error {
	if writer.file != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(writer.path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(writer.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	writer.file = file
	writer.size = info.Size()
	if writer.size > writer.rotation.MaxBytes {
		return writer.rotateLocked()
	}
	return nil
}

func (writer *RotatingLogWriter) rotateLocked() error {
	if writer.file != nil {
		if err := writer.file.Close(); err != nil {
			return err
		}
		writer.file = nil
	}
	if writer.rotation.MaxBackups == 0 {
		if err := os.Remove(writer.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		writer.size = 0
		return writer.openLocked()
	}
	if err := os.Remove(backupLogPath(writer.path, writer.rotation.MaxBackups)); err != nil && !os.IsNotExist(err) {
		return err
	}
	for i := writer.rotation.MaxBackups - 1; i >= 1; i-- {
		src := backupLogPath(writer.path, i)
		dst := backupLogPath(writer.path, i+1)
		if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(src, dst); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Rename(writer.path, backupLogPath(writer.path, 1)); err != nil && !os.IsNotExist(err) {
		return err
	}
	writer.size = 0
	return writer.openLocked()
}

func backupLogPath(path string, index int) string {
	return fmt.Sprintf("%s.%d", path, index)
}
