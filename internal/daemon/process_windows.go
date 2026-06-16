//go:build windows

package daemon

import (
	"os/exec"
	"syscall"
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
