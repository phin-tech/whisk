package ptytrace

import (
	"fmt"
	"log"
	"os"
	"time"
)

func Enabled() bool {
	return os.Getenv("WHISK_PTY_TRACE") != "" || os.Getenv("WHISK_PTY_TRACE_FILE") != ""
}

func Line(channel string, ptyID string, bytes int, at time.Time) string {
	return fmt.Sprintf("pty.input channel=%s pty=%s bytes=%d at=%s", channel, ptyID, bytes, at.UTC().Format("2006-01-02T15:04:05.000Z"))
}

func Write(line string) {
	if line == "" || !Enabled() {
		return
	}
	if path := os.Getenv("WHISK_PTY_TRACE_FILE"); path != "" {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = file.WriteString(line + "\n")
			_ = file.Close()
			return
		}
	}
	log.Print(line)
}
