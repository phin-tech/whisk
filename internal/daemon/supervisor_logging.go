package daemon

import (
	"io"
	"os"
	"strings"
	"sync"
)

const (
	supervisorStderrCaptureBytes = 32 * 1024
	supervisorDaemonLogTailBytes = 16 * 1024
)

type limitedCapture struct {
	mu        sync.Mutex
	maxBytes  int
	recording bool
	data      []byte
}

func newLimitedCapture(maxBytes int) *limitedCapture {
	return &limitedCapture{
		maxBytes:  maxBytes,
		recording: true,
	}
}

func (capture *limitedCapture) Write(p []byte) (int, error) {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	if !capture.recording || capture.maxBytes <= 0 {
		return len(p), nil
	}
	capture.data = append(capture.data, p...)
	if len(capture.data) > capture.maxBytes {
		capture.data = append([]byte(nil), capture.data[len(capture.data)-capture.maxBytes:]...)
	}
	return len(p), nil
}

func (capture *limitedCapture) StopRecording() {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	capture.recording = false
}

func (capture *limitedCapture) String() string {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return string(append([]byte(nil), capture.data...))
}

func daemonStartDiagnostics(baseURL string, stderr *limitedCapture) string {
	var parts []string
	if tail, err := readDaemonLogTail(baseURL, supervisorDaemonLogTailBytes); err == nil && strings.TrimSpace(tail) != "" {
		parts = append(parts, "daemon log tail:\n"+strings.TrimSpace(tail))
	}
	if stderr != nil {
		if tail := strings.TrimSpace(stderr.String()); tail != "" {
			parts = append(parts, "early stderr:\n"+tail)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "\n" + strings.Join(parts, "\n")
}

func readDaemonLogTail(baseURL string, maxBytes int64) (string, error) {
	path, err := LogPath(baseURL)
	if err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if maxBytes > 0 && info.Size() > maxBytes {
		if _, err := file.Seek(info.Size()-maxBytes, io.SeekStart); err != nil {
			return "", err
		}
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
