package daemon

func DaemonPathForTest(executable string) (string, error) {
	return daemonPathForExecutable(executable)
}
