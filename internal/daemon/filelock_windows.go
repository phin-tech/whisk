//go:build windows

package daemon

import (
	"context"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

type stateFileLock struct {
	file       *os.File
	overlapped windows.Overlapped
}

func lockFile(ctx context.Context, path string) (*stateFileLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	lock := &stateFileLock{file: file}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		err := windows.LockFileEx(
			windows.Handle(file.Fd()),
			windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
			0,
			1,
			0,
			&lock.overlapped,
		)
		if err == nil {
			return lock, nil
		}
		if err != windows.ERROR_LOCK_VIOLATION {
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
	unlockErr := windows.UnlockFileEx(windows.Handle(l.file.Fd()), 0, 1, 0, &l.overlapped)
	closeErr := l.file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
