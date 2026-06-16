package main

import (
	"os"
	"strings"
	"testing"
)

func TestDevAppTaskPreservesRunningDaemonAndCLI(t *testing.T) {
	taskfile, err := os.ReadFile("Taskfile.yml")
	if err != nil {
		t.Fatalf("read Taskfile.yml: %v", err)
	}
	block := taskBlock(string(taskfile), "dev:app")

	requireTaskLine(t, block, "- task: build:daemon")
	requireTaskLine(t, block, "- task: build:cli")
	requireTaskLine(t, block, "WHISKD_PATH: \"{{.BIN_DIR}}/whisk\"")
	requireTaskLine(t, block, "WHISK_CLI: \"{{.BIN_DIR}}/whisk\"")
	requireTaskLine(t, block, "WHISKD_URL: \"http://{{.DEV_DAEMON_ADDR}}\"")
	requireTaskLineAbsent(t, block, "whisk daemon stop")
	requireTaskLineAbsent(t, block, "{{.BIN_DIR}}/whiskd")
}

func TestBuildDaemonTaskBuildsWhiskDaemonMode(t *testing.T) {
	taskfile, err := os.ReadFile("Taskfile.yml")
	if err != nil {
		t.Fatalf("read Taskfile.yml: %v", err)
	}
	block := taskBlock(string(taskfile), "build:daemon")

	requireTaskLine(t, block, "go build -o {{.BIN_DIR}}/whisk ./cmd/whisk")
	requireTaskLineAbsent(t, block, "./cmd/whiskd")
	requireTaskLineAbsent(t, block, "{{.BIN_DIR}}/whiskd")
}

func TestDevDaemonTaskRunsWhiskDaemonMode(t *testing.T) {
	taskfile, err := os.ReadFile("Taskfile.yml")
	if err != nil {
		t.Fatalf("read Taskfile.yml: %v", err)
	}
	block := taskBlock(string(taskfile), "dev:daemon")

	requireTaskLine(t, block, "{{.BIN_DIR}}/whisk daemon run -addr {{.DAEMON_ADDR}}")
	requireTaskLineAbsent(t, block, "whiskd")
}

func TestSDKIntegrationTasksUseWhiskDaemonMode(t *testing.T) {
	taskfile, err := os.ReadFile("Taskfile.yml")
	if err != nil {
		t.Fatalf("read Taskfile.yml: %v", err)
	}
	for _, taskName := range []string{"sdk:test:python", "sdk:test:ts"} {
		block := taskBlock(string(taskfile), taskName)

		requireTaskLine(t, block, "go build -o {{.BIN_DIR}}/whisk ./cmd/whisk")
		requireTaskLine(t, block, "WHISKD_BIN=\"$PWD/{{.BIN_DIR}}/whisk\"")
		requireTaskLineAbsent(t, block, "./cmd/whiskd")
		requireTaskLineAbsent(t, block, "{{.BIN_DIR}}/whiskd")
	}
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

func requireTaskLineAbsent(t *testing.T, block string, unwanted string) {
	t.Helper()
	if block == "" {
		t.Fatalf("dev:app task block not found")
	}
	if strings.Contains(block, unwanted) {
		t.Fatalf("dev:app task should not contain %q\nblock:\n%s", unwanted, block)
	}
}
