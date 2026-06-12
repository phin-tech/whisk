package main

import (
	"os"
	"strings"
	"testing"
)

func TestDevAppTaskRestartsWithFreshDaemonAndCLI(t *testing.T) {
	taskfile, err := os.ReadFile("Taskfile.yml")
	if err != nil {
		t.Fatalf("read Taskfile.yml: %v", err)
	}
	block := taskBlock(string(taskfile), "dev:app")

	requireTaskLine(t, block, "- task: build:daemon")
	requireTaskLine(t, block, "- task: build:cli")
	requireTaskLine(t, block, "- '{{.BIN_DIR}}/whisk daemon stop -url http://{{.DEV_DAEMON_ADDR}} || true'")
	requireTaskLine(t, block, "WHISKD_PATH: \"{{.BIN_DIR}}/whiskd\"")
	requireTaskLine(t, block, "WHISK_CLI: \"{{.BIN_DIR}}/whisk\"")
	requireTaskLine(t, block, "WHISKD_URL: \"http://{{.DEV_DAEMON_ADDR}}\"")
}

func taskBlock(taskfile string, name string) string {
	lines := strings.Split(taskfile, "\n")
	var block []string
	inBlock := false
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(line, ":") {
			if inBlock {
				break
			}
			inBlock = strings.TrimSpace(line) == name+":"
		}
		if inBlock {
			block = append(block, line)
		}
	}
	return strings.Join(block, "\n")
}

func requireTaskLine(t *testing.T, block string, want string) {
	t.Helper()
	if block == "" {
		t.Fatalf("dev:app task block not found")
	}
	if !strings.Contains(block, want) {
		t.Fatalf("dev:app task missing %q\nblock:\n%s", want, block)
	}
}
