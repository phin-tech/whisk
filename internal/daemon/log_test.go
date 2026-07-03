package daemon_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/daemon"
)

func TestLogPathUsesStateDirAndListenAddress(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	tempDir := filepath.Join(t.TempDir(), "tmp")
	t.Setenv("WHISK_STATE_DIR", stateDir)
	t.Setenv("TMPDIR", tempDir)

	first, err := daemon.LogPath("http://127.0.0.1:19996")
	if err != nil {
		t.Fatalf("log path: %v", err)
	}
	second, err := daemon.LogPath("http://127.0.0.1:19997")
	if err != nil {
		t.Fatalf("second log path: %v", err)
	}

	if !strings.HasPrefix(first, stateDir+string(filepath.Separator)) {
		t.Fatalf("log path = %q, want under %q", first, stateDir)
	}
	if strings.HasPrefix(first, tempDir+string(filepath.Separator)) {
		t.Fatalf("log path = %q, should not use system temp dir %q", first, tempDir)
	}
	if !strings.HasSuffix(first, "daemon-127_0_0_1_19996.log") {
		t.Fatalf("log path = %q, want per-address daemon log name", first)
	}
	if first == second {
		t.Fatalf("different listen addresses should not share log path: %q", first)
	}
}

func TestRotatingLogWriterCapsFootprint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon-127_0_0_1_19998.log")
	writer, err := daemon.NewRotatingLogWriter(path, daemon.LogRotation{
		MaxBytes:   32,
		MaxBackups: 2,
	})
	if err != nil {
		t.Fatalf("new rotating log writer: %v", err)
	}

	for i := 0; i < 8; i++ {
		if _, err := writer.Write(bytes.Repeat([]byte{byte('a' + i)}, 12)); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	var total int64
	for _, suffix := range []string{"", ".1", ".2"} {
		info, err := os.Stat(path + suffix)
		if err != nil {
			t.Fatalf("stat rotated log %q: %v", path+suffix, err)
		}
		if info.Size() > 32 {
			t.Fatalf("rotated log %q size = %d, want <= 32", path+suffix, info.Size())
		}
		total += info.Size()
	}
	if _, err := os.Stat(path + ".3"); !os.IsNotExist(err) {
		t.Fatalf("unexpected uncapped backup %q, stat err=%v", path+".3", err)
	}
	if total > 96 {
		t.Fatalf("total log footprint = %d, want <= 96", total)
	}
}
