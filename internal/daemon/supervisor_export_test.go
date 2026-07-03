package daemon

import "github.com/phin-tech/whisk/internal/protocol"

func DaemonPathForTest(executable string) (string, error) {
	return daemonPathForExecutable(executable)
}

func ProcessStartTimeForTest(pid int) (string, error) {
	return processStartTime(pid)
}

func WriteStateForTest(baseURL string, pid int, processStartTime string, binaryPath string) error {
	addr, err := addrFromURL(baseURL)
	if err != nil {
		return err
	}
	return writeStateFile(baseURL, daemonStateFile{
		Version:          daemonStateVersion,
		PID:              pid,
		ProcessStartTime: processStartTime,
		ListenAddress:    addr,
		APIVersion:       protocol.DaemonAPIVersion,
		BinaryPath:       binaryPath,
	})
}

func StatePathForTest(baseURL string) (string, error) {
	return statePath(baseURL)
}
