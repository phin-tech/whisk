package server

import (
	"testing"
	"time"
)

func TestPTYInputTraceLine(t *testing.T) {
	got := ptyInputTraceLine("daemon.websocket", "pty_01", 3, time.UnixMilli(123))
	want := "pty.input channel=daemon.websocket pty=pty_01 bytes=3 at=1970-01-01T00:00:00.123Z"
	if got != want {
		t.Fatalf("trace line = %q, want %q", got, want)
	}
}
