package main

import (
	"os"
	"os/exec"
	"path/filepath"
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
	requireTaskLine(t, block, "WHISK_PLUGIN_DIRS: \"{{.ROOT_DIR}}/../whisk-plugins/registry/plugins/github-issues\"")
	requireTaskLineAbsent(t, block, "{{.ROOT_DIR}}/plugins/github-issues")
	requireTaskLineAbsent(t, block, "whisk daemon stop")
	requireTaskLineAbsent(t, block, "{{.BIN_DIR}}/whiskd")
}

func TestSeedPluginsLiveOutsideThisRepo(t *testing.T) {
	for _, path := range []string{
		"plugins/github-issues/plugin.json",
		"registry/plugins/github-issues/plugin.json",
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("seed plugin copy belongs in ../whisk-plugins/registry/plugins/github-issues, not %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat seed plugin: %v", err)
		}
	}
}

func TestDeletedScaffoldingStaysDeleted(t *testing.T) {
	for _, path := range []string{
		"build/android",
		"build/ios",
		"internal/adapters/ghcli",
		"internal/adapters/gitcli",
		"internal/adapters/hooks",
		"internal/adapters/processrunner",
		"internal/adapters/workitemstore/json.go",
		"internal/adapters/workitemstore/json_test.go",
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("delete stale scaffolding: %s still exists", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}

func TestRootTaskfileOmitsDeletedScaffolding(t *testing.T) {
	taskfile, err := os.ReadFile("Taskfile.yml")
	if err != nil {
		t.Fatalf("read Taskfile.yml: %v", err)
	}
	for _, unwanted := range []string{
		"ios: ./build/ios/Taskfile.yml",
		"android: ./build/android/Taskfile.yml",
		"./internal/adapters/ghcli",
		"./internal/adapters/gitcli",
		"./internal/adapters/hooks",
		"./internal/adapters/processrunner",
	} {
		if strings.Contains(string(taskfile), unwanted) {
			t.Fatalf("Taskfile.yml should not contain %q", unwanted)
		}
	}
}

func TestSigningScriptRejectsUnexpectedMacOSExecutablesBeforeNotary(t *testing.T) {
	tmp := t.TempDir()
	app := filepath.Join(tmp, "Whisk.app")
	macos := filepath.Join(app, "Contents", "MacOS")
	bin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(macos, 0o755); err != nil {
		t.Fatalf("mkdir app: %v", err)
	}
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	for _, name := range []string{"whisk-app", "whisk", "whisk.disabled-kill-loop"} {
		path := filepath.Join(macos, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write executable: %v", err)
		}
	}
	for _, name := range []string{"codesign", "xcrun", "ditto", "spctl"} {
		path := filepath.Join(bin, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\necho unexpected $0 >&2\nexit 99\n"), 0o755); err != nil {
			t.Fatalf("write fake tool: %v", err)
		}
	}

	out, err := runBashScript(t, map[string]string{
		"PATH":             bin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"SIGN_IDENTITY":    "Developer ID Application: Test",
		"KEYCHAIN_PROFILE": "notary-profile",
	}, "scripts/sign-notarize-macos-app.sh", app)
	if err == nil {
		t.Fatalf("signing script succeeded, output:\n%s", out)
	}
	if !strings.Contains(out, "unexpected executable in app bundle") || !strings.Contains(out, "whisk.disabled-kill-loop") {
		t.Fatalf("output did not explain unexpected executable:\n%s", out)
	}
	if strings.Contains(out, "unexpected "+filepath.Join(bin, "xcrun")) {
		t.Fatalf("script reached notary tool before rejecting bundle:\n%s", out)
	}
}

func TestDevAppRestartTaskStopsDevDaemonThenRunsApp(t *testing.T) {
	taskfile, err := os.ReadFile("Taskfile.yml")
	if err != nil {
		t.Fatalf("read Taskfile.yml: %v", err)
	}
	block := taskBlock(string(taskfile), "dev:app:restart")

	requireTaskLine(t, block, "- task: build:daemon")
	requireTaskLine(t, block, "{{.BIN_DIR}}/whisk daemon stop -url http://{{.DEV_DAEMON_ADDR}}")
	requireTaskLine(t, block, "- task: dev:app")
}

func TestBuildDaemonTaskBuildsWhiskDaemonMode(t *testing.T) {
	taskfile, err := os.ReadFile("Taskfile.yml")
	if err != nil {
		t.Fatalf("read Taskfile.yml: %v", err)
	}
	block := taskBlock(string(taskfile), "build:daemon")

	requireTaskLine(t, block, "go build -ldflags=\"{{.BUILDINFO_LDFLAGS}}\" -o {{.BIN_DIR}}/whisk ./cmd/whisk")
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

func runBashScript(t *testing.T, env map[string]string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("bash", args...)
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
