package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	defaultPluginCommandTimeout        = 10 * time.Second
	maxPluginCommandTimeout            = 30 * time.Second
	defaultPluginCommandStdoutCapBytes = manifestCommandDefaultOutputCapBytes
	maxPluginCommandStdoutCapBytes     = manifestCommandMaxOutputCapBytes
	defaultPluginCommandStderrCapBytes = 64 << 10
	maxPluginCommandStderrCapBytes     = 256 << 10
	pluginCommandWaitDelay             = 250 * time.Millisecond
)

type PluginCommandRequest struct {
	PluginID       string
	Dir            string
	Command        string
	Input          any
	Timeout        time.Duration
	StdoutCapBytes int
	StderrCapBytes int
}

type PluginCommandResult struct {
	Stdout          []byte
	Stderr          []byte
	StderrTruncated bool
}

type PluginCommandError struct {
	PluginID         string
	Kind             string
	Err              error
	Timeout          time.Duration
	StdoutCapBytes   int
	Stderr           string
	StderrTruncated  bool
	StderrCapBytes   int
	WorkingDirectory string
	Command          string
}

func (e *PluginCommandError) Error() string {
	subject := "plugin command"
	if e.PluginID != "" {
		subject = fmt.Sprintf("plugin %s command", e.PluginID)
	}
	switch e.Kind {
	case "timeout":
		return fmt.Sprintf("%s timed out after %s", subject, e.Timeout)
	case "stdout_cap":
		return fmt.Sprintf("%s stdout exceeded %d bytes", subject, e.StdoutCapBytes)
	case "canceled":
		if e.Err != nil {
			return fmt.Sprintf("%s canceled: %v", subject, e.Err)
		}
		return fmt.Sprintf("%s canceled", subject)
	default:
		msg := fmt.Sprintf("%s failed", subject)
		if e.Err != nil {
			msg += ": " + e.Err.Error()
		}
		if e.Stderr != "" {
			msg += ": " + e.Stderr
		}
		if e.StderrTruncated {
			msg += fmt.Sprintf(" (stderr truncated to %d bytes)", e.StderrCapBytes)
		}
		return msg
	}
}

func (e *PluginCommandError) Unwrap() error {
	return e.Err
}

func runPluginCommand(ctx context.Context, req PluginCommandRequest) (PluginCommandResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	pluginID := strings.TrimSpace(req.PluginID)
	command := strings.TrimSpace(req.Command)
	if command == "" {
		return PluginCommandResult{}, &PluginCommandError{PluginID: pluginID, Kind: "invalid", Err: errors.New("command required")}
	}
	dir := strings.TrimSpace(req.Dir)
	if dir == "" {
		return PluginCommandResult{}, &PluginCommandError{PluginID: pluginID, Kind: "invalid", Err: errors.New("working directory required")}
	}
	absDir, err := filepath.Abs(filepath.Clean(dir))
	if err != nil {
		return PluginCommandResult{}, &PluginCommandError{PluginID: pluginID, Kind: "invalid", Err: fmt.Errorf("resolve working directory: %w", err)}
	}
	timeout, stdoutCap, stderrCap, err := normalizePluginCommandLimits(req)
	if err != nil {
		return PluginCommandResult{}, &PluginCommandError{PluginID: pluginID, Kind: "invalid", Err: err}
	}
	input, err := json.Marshal(req.Input)
	if err != nil {
		return PluginCommandResult{}, fmt.Errorf("marshal plugin %s command input: %w", pluginID, err)
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := pluginShellCommand(runCtx, command)
	configurePluginCommandProcess(cmd)
	cmd.Dir = absDir
	cmd.WaitDelay = pluginCommandWaitDelay
	cmd.Stdin = bytes.NewReader(input)
	stdout := newCappedCommandBuffer(stdoutCap)
	stderr := newCappedCommandBuffer(stderrCap)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	runErr := cmd.Run()
	result := PluginCommandResult{
		Stdout:          stdout.Bytes(),
		Stderr:          stderr.Bytes(),
		StderrTruncated: stderr.Exceeded(),
	}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return result, &PluginCommandError{
			PluginID:         pluginID,
			Kind:             "timeout",
			Err:              runCtx.Err(),
			Timeout:          timeout,
			WorkingDirectory: absDir,
			Command:          command,
		}
	}
	if errors.Is(runCtx.Err(), context.Canceled) && runErr != nil {
		return result, &PluginCommandError{
			PluginID:         pluginID,
			Kind:             "canceled",
			Err:              runErr,
			WorkingDirectory: absDir,
			Command:          command,
		}
	}
	if stdout.Exceeded() {
		return result, &PluginCommandError{
			PluginID:         pluginID,
			Kind:             "stdout_cap",
			StdoutCapBytes:   stdoutCap,
			WorkingDirectory: absDir,
			Command:          command,
		}
	}
	if runErr != nil {
		return result, &PluginCommandError{
			PluginID:         pluginID,
			Kind:             "exit",
			Err:              runErr,
			Stderr:           strings.TrimSpace(stderr.String()),
			StderrTruncated:  stderr.Exceeded(),
			StderrCapBytes:   stderrCap,
			WorkingDirectory: absDir,
			Command:          command,
		}
	}
	return result, nil
}

func normalizePluginCommandLimits(req PluginCommandRequest) (time.Duration, int, int, error) {
	timeout := req.Timeout
	if timeout < 0 {
		return 0, 0, 0, fmt.Errorf("timeout must be non-negative")
	}
	if timeout == 0 {
		timeout = defaultPluginCommandTimeout
	}
	if timeout > maxPluginCommandTimeout {
		timeout = maxPluginCommandTimeout
	}
	stdoutCap := req.StdoutCapBytes
	if stdoutCap < 0 {
		return 0, 0, 0, fmt.Errorf("stdout cap must be non-negative")
	}
	if stdoutCap == 0 {
		stdoutCap = defaultPluginCommandStdoutCapBytes
	}
	if stdoutCap > maxPluginCommandStdoutCapBytes {
		stdoutCap = maxPluginCommandStdoutCapBytes
	}
	stderrCap := req.StderrCapBytes
	if stderrCap < 0 {
		return 0, 0, 0, fmt.Errorf("stderr cap must be non-negative")
	}
	if stderrCap == 0 {
		stderrCap = defaultPluginCommandStderrCapBytes
	}
	if stderrCap > maxPluginCommandStderrCapBytes {
		stderrCap = maxPluginCommandStderrCapBytes
	}
	return timeout, stdoutCap, stderrCap, nil
}

func pluginShellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/c", command)
	}
	return exec.CommandContext(ctx, "sh", "-lc", command)
}

type cappedCommandBuffer struct {
	limit    int
	buf      bytes.Buffer
	exceeded bool
}

func newCappedCommandBuffer(limit int) *cappedCommandBuffer {
	return &cappedCommandBuffer{limit: limit}
}

func (b *cappedCommandBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		if len(p) > 0 {
			b.exceeded = true
		}
		return len(p), nil
	}
	remaining := b.limit - b.buf.Len()
	if remaining > 0 {
		if len(p) <= remaining {
			_, _ = b.buf.Write(p)
			return len(p), nil
		}
		_, _ = b.buf.Write(p[:remaining])
	}
	if len(p) > remaining {
		b.exceeded = true
	}
	return len(p), nil
}

func (b *cappedCommandBuffer) Bytes() []byte {
	return append([]byte(nil), b.buf.Bytes()...)
}

func (b *cappedCommandBuffer) String() string {
	return b.buf.String()
}

func (b *cappedCommandBuffer) Exceeded() bool {
	return b.exceeded
}
