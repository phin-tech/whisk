//go:build !windows

package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// processAlive reports whether a process with the given pid is still running. Signal 0 performs
// the kill(2) permission/existence check without actually delivering a signal.
func processAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func signalProcessTerm(process *os.Process) error {
	return process.Signal(syscall.SIGTERM)
}

func processExitedByInterrupt(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	status, ok := exitErr.ProcessState.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && status.Signal() == syscall.SIGINT
}

func processStartTime(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("pid must be positive")
	}
	psPath, err := psExecutable()
	if err != nil {
		return "", err
	}
	output, err := exec.Command(psPath, "-o", "lstart=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return "", err
	}
	startTime := strings.TrimSpace(string(output))
	if startTime == "" {
		return "", fmt.Errorf("process %d not found", pid)
	}
	parsed, err := time.ParseInLocation("Mon Jan _2 15:04:05 2006", startTime, time.Local)
	if err != nil {
		return startTime, nil
	}
	return parsed.UTC().Format(time.RFC3339), nil
}

func psExecutable() (string, error) {
	if path, err := exec.LookPath("ps"); err == nil {
		return path, nil
	}
	for _, candidate := range []string{"/bin/ps", "/usr/bin/ps"} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("ps executable not found")
}
