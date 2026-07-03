//go:build !windows

package daemon

import (
	"context"
	"os"
	"syscall"
	"time"
)

type stateFileLock struct {
	file *os.File
}

func lockFile(ctx context.Context, path string) (*stateFileLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return &stateFileLock{file: file}, nil
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			_ = file.Close()
			return nil, err
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			_ = file.Close()
			return nil, ctx.Err()
		}
	}
}

func (l *stateFileLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	unlockErr := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	closeErr := l.file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
