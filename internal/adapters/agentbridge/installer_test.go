package agentbridge_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/agentbridge"
	domainbridge "github.com/phin-tech/whisk/internal/domain/agentbridge"
)

func TestInstallWritesVersionedBridgeConfigAndExecutableHook(t *testing.T) {
	root := t.TempDir()
	installed, err := agentbridge.Install(agentbridge.InstallRequest{
		RootDir:   root,
		BridgeID:  "bridge_01",
		RunID:     "run_01",
		Provider:  "claude",
		HookURL:   "http://127.0.0.1:8787/v1/agent-bridges/bridge_01/hooks",
		Token:     "secret-token",
		WhiskCLI:  "/usr/local/bin/whisk",
		WhiskdURL: "http://127.0.0.1:8787",
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if installed.Dir != filepath.Join(root, ".whisk", "agent-bridges", "run_01", "bridge_01") {
		t.Fatalf("dir = %q", installed.Dir)
	}

	rawConfig, err := os.ReadFile(filepath.Join(installed.Dir, "bridge.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var config map[string]string
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		t.Fatalf("config json: %v", err)
	}
	if config["bridge_id"] != "bridge_01" ||
		config["provider"] != "claude" ||
		config["hook_url"] == "" ||
		config["token"] != "secret-token" ||
		config["whisk_hook_protocol"] != "1" {
		t.Fatalf("config = %#v", config)
	}

	info, err := os.Stat(installed.HookScript)
	if err != nil {
		t.Fatalf("stat hook: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("hook script not executable: %s", info.Mode())
	}
	rawHook, err := os.ReadFile(installed.HookScript)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if strings.Contains(string(rawHook), "secret-token") {
		t.Fatalf("hook persisted token: %s", rawHook)
	}
	if !strings.Contains(string(rawHook), "agent-bridge hook") {
		t.Fatalf("hook script = %s", rawHook)
	}
	if !strings.Contains(string(rawHook), "# whisk_hook_protocol=1") {
		t.Fatalf("hook protocol comment missing: %s", rawHook)
	}
}

func TestInstalledHookFallsBackToBesideBridgeConfig(t *testing.T) {
	root := t.TempDir()
	capture := filepath.Join(root, "args.txt")
	fakeCLI := filepath.Join(root, "fake-whisk")
	if err := os.WriteFile(fakeCLI, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$WHISK_CAPTURE\"\n"), 0o700); err != nil {
		t.Fatalf("write fake cli: %v", err)
	}

	installed, err := agentbridge.Install(agentbridge.InstallRequest{
		RootDir:   root,
		BridgeID:  "bridge_01",
		RunID:     "run_01",
		Provider:  "claude",
		HookURL:   "http://127.0.0.1:8787/v1/agent-bridges/bridge_01/hooks",
		Token:     "secret-token",
		WhiskCLI:  fakeCLI,
		WhiskdURL: "http://127.0.0.1:8787",
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	cmd := exec.Command(installed.HookScript)
	cmd.Stdin = strings.NewReader(`{"hook_event_name":"Notification"}`)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"WHISK_CAPTURE=" + capture,
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hook script: %v\n%s", err, out)
	}

	rawArgs, err := os.ReadFile(capture)
	if err != nil {
		t.Fatalf("read captured args: %v", err)
	}
	args := strings.Split(strings.TrimSpace(string(rawArgs)), "\n")
	want := []string{
		"agent-bridge",
		"hook",
		"-url",
		"http://127.0.0.1:8787",
		"-bridge",
		"bridge_01",
		"-token",
		"secret-token",
		"-provider",
		"claude",
		"-hook-protocol",
		"1",
	}
	if strings.Join(args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
	if domainbridge.HookProtocolVersion != 1 {
		t.Fatalf("unexpected hook protocol version %d", domainbridge.HookProtocolVersion)
	}
}
