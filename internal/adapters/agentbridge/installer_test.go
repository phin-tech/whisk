package agentbridge_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/agentbridge"
)

func TestInstallWritesBridgeConfigAndExecutableHookWithoutPersistingToken(t *testing.T) {
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
	if strings.Contains(string(rawConfig), "secret-token") {
		t.Fatalf("config persisted token: %s", rawConfig)
	}
	var config map[string]string
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		t.Fatalf("config json: %v", err)
	}
	if config["bridge_id"] != "bridge_01" || config["provider"] != "claude" || config["hook_url"] == "" {
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
}
