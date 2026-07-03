//go:build windows

package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// processAlive cannot rely on signal 0 on Windows, where os.Process.Signal only supports Kill.
// Report not-alive so callers fall back to the health-based liveness checks rather than blocking;
// this preserves the pre-existing Windows behaviour.
func processAlive(_ int) bool {
	return false
}

func signalProcessTerm(process *os.Process) error {
	return process.Kill()
}

func processStartTime(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("pid must be positive")
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)
	var created windows.Filetime
	var exited windows.Filetime
	var kernel windows.Filetime
	var user windows.Filetime
	if err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return "", err
	}
	return time.Unix(0, created.Nanoseconds()).UTC().Format(time.RFC3339Nano), nil
}
