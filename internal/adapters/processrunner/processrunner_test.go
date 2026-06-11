package processrunner

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStartOutputListAndExit(t *testing.T) {
	service := NewService()
	record, err := service.Start(StartRequest{Command: commandForOutput()})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if record.ID != "process_000001" || !record.Running || record.WorkingDir == "" {
		t.Fatalf("record = %#v", record)
	}
	waitForExit(t, service, record.ID)

	output, current, err := service.Output(record.ID, 1024)
	if err != nil {
		t.Fatalf("Output error: %v", err)
	}
	if current.Running || !strings.Contains(output, "hello") || current.RetainedOutputBytes != len(output) {
		t.Fatalf("output=%q current=%#v", output, current)
	}
	if len(service.List()) != 1 {
		t.Fatalf("List length mismatch")
	}
}

func TestKillAndShutdown(t *testing.T) {
	service := NewService()
	record, err := service.Start(StartRequest{Command: commandForSleep()})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	killed, err := service.Kill(record.ID)
	if err != nil {
		t.Fatalf("Kill error: %v", err)
	}
	if killed.Running {
		t.Fatalf("killed record = %#v", killed)
	}

	record, err = service.Start(StartRequest{Command: commandForSleep()})
	if err != nil {
		t.Fatalf("Start second error: %v", err)
	}
	service.Shutdown()
	_, current, err := service.Output(record.ID, 1024)
	if err != nil {
		t.Fatalf("Output after shutdown error: %v", err)
	}
	if current.Running {
		t.Fatalf("shutdown left process running: %#v", current)
	}
}

func TestOutputBufferRetainsTail(t *testing.T) {
	buffer := newOutputBuffer(5)
	buffer.append([]byte("abc"))
	buffer.append([]byte("def"))
	if got := buffer.snapshot(10); got != "bcdef" {
		t.Fatalf("snapshot = %q", got)
	}
	if !buffer.truncated || buffer.len() != 5 {
		t.Fatalf("buffer metadata mismatch")
	}
}

func commandForOutput() string {
	if runtime.GOOS == "windows" {
		return "echo hello"
	}
	return "printf 'hello\\n'"
}

func commandForSleep() string {
	if runtime.GOOS == "windows" {
		return "ping -n 10 127.0.0.1 >NUL"
	}
	return "sleep 5"
}

func waitForExit(t *testing.T, service *Service, id string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, record, err := service.Output(id, 1024)
		if err == nil && !record.Running {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("process did not exit")
}
