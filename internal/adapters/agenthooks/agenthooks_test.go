package agenthooks

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestClaudeInstallCheckAndRemovePreservesUnrelatedHooks(t *testing.T) {
	paths := testPaths(t)
	writeFile(t, paths.ClaudeSettingsPath, `{
  "statusLine": {
    "type": "command",
    "command": "existing-status"
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "unknownMatcherField": "keep",
        "hooks": [
          {
            "type": "command",
            "command": "echo existing",
            "unknownHookField": {"keep": true}
          }
        ]
      }
    ]
  }
}`)
	installer := NewInstaller(paths)

	missing, err := installer.Check(context.Background(), ProviderClaude)
	if err != nil {
		t.Fatalf("check missing: %v", err)
	}
	if missing.Status != StatusMissing {
		t.Fatalf("missing status = %#v", missing)
	}

	installed, err := installer.Install(context.Background(), ProviderClaude)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if installed.Status != StatusCurrent {
		t.Fatalf("installed status = %#v", installed)
	}

	cfg := readConfigFile(t, paths.ClaudeSettingsPath)
	pre := cfg.Hooks["PreToolUse"]
	if len(pre) != 2 {
		t.Fatalf("PreToolUse entries = %#v", pre)
	}
	if pre[0].Hooks[0].Command != "echo existing" {
		t.Fatalf("existing hook was not preserved: %#v", pre)
	}
	if string(pre[0].Rest["unknownMatcherField"]) != `"keep"` {
		t.Fatalf("unknown matcher field was not preserved: %#v", pre[0].Rest)
	}
	if compactJSON(t, pre[0].Hooks[0].Rest["unknownHookField"]) != `{"keep":true}` {
		t.Fatalf("unknown hook field was not preserved: %#v", pre[0].Hooks[0].Rest)
	}
	if pre[1].Hooks[0].Command != installedCommand(paths, ProviderClaude) {
		t.Fatalf("managed command = %#v", pre[1].Hooks[0])
	}
	assertRawJSONField(t, paths.ClaudeSettingsPath, "statusLine", `{"command":"existing-status","type":"command"}`)

	removed, err := installer.Remove(context.Background(), ProviderClaude)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if removed.Status != StatusMissing {
		t.Fatalf("removed status = %#v", removed)
	}
	cfg = readConfigFile(t, paths.ClaudeSettingsPath)
	pre = cfg.Hooks["PreToolUse"]
	if len(pre) != 1 || pre[0].Hooks[0].Command != "echo existing" {
		t.Fatalf("remove did not preserve only existing hook: %#v", pre)
	}
	assertRawJSONField(t, paths.ClaudeSettingsPath, "statusLine", `{"command":"existing-status","type":"command"}`)
}

func TestCodexInstallReportsUntrustedUntilTrustVerified(t *testing.T) {
	paths := testPaths(t)
	installer := NewInstaller(paths)

	status, err := installer.Install(context.Background(), ProviderCodex)
	if err != nil {
		t.Fatalf("install codex: %v", err)
	}
	if status.Status != StatusUntrusted {
		t.Fatalf("codex status = %#v", status)
	}

	cfg := readConfigFile(t, paths.CodexHooksPath)
	if got := cfg.Hooks["PreToolUse"][0].Hooks[0].Command; got != installedCommand(paths, ProviderCodex) {
		t.Fatalf("codex command = %q", got)
	}
}

func TestInstallWritesProviderSpecificHookEvents(t *testing.T) {
	paths := testPaths(t)
	installer := NewInstaller(paths)
	if _, err := installer.Install(context.Background(), ProviderClaude); err != nil {
		t.Fatalf("install claude: %v", err)
	}
	claude := readConfigFile(t, paths.ClaudeSettingsPath)
	for _, event := range []string{"PreToolUse", "PermissionRequest", "Notification", "Elicitation", "ElicitationResult", "Stop"} {
		if len(claude.Hooks[event]) == 0 {
			t.Fatalf("claude event %s missing from %#v", event, claude.Hooks)
		}
	}

	if _, err := installer.Install(context.Background(), ProviderCodex); err != nil {
		t.Fatalf("install codex: %v", err)
	}
	codex := readConfigFile(t, paths.CodexHooksPath)
	for _, event := range []string{"PreToolUse", "PermissionRequest", "UserPromptSubmit", "SubagentStart", "Stop"} {
		if len(codex.Hooks[event]) == 0 {
			t.Fatalf("codex event %s missing from %#v", event, codex.Hooks)
		}
	}
	if len(codex.Hooks["Elicitation"]) != 0 {
		t.Fatalf("codex should not install unsupported Elicitation hook: %#v", codex.Hooks["Elicitation"])
	}
}

func TestCheckDetectsOutdatedManifestAndModifiedCommand(t *testing.T) {
	paths := testPaths(t)
	installer := NewInstaller(paths)
	if _, err := installer.Install(context.Background(), ProviderClaude); err != nil {
		t.Fatalf("install: %v", err)
	}

	manifest := readManifestFile(t, installer.manifestPath())
	manifest.InstallerVersion = "0.9.0"
	writeManifestFile(t, installer.manifestPath(), manifest)
	status, err := installer.Check(context.Background(), ProviderClaude)
	if err != nil {
		t.Fatalf("check outdated: %v", err)
	}
	if status.Status != StatusOutdated {
		t.Fatalf("outdated status = %#v", status)
	}

	manifest.InstallerVersion = InstallerVersion
	writeManifestFile(t, installer.manifestPath(), manifest)
	cfg := readConfigFile(t, paths.ClaudeSettingsPath)
	cfg.Hooks["PreToolUse"][0].Hooks[0].Command = "echo tampered"
	writeConfigFile(t, paths.ClaudeSettingsPath, cfg)
	status, err = installer.Check(context.Background(), ProviderClaude)
	if err != nil {
		t.Fatalf("check modified: %v", err)
	}
	if status.Status != StatusModified {
		t.Fatalf("modified status = %#v", status)
	}
}

func TestCheckDetectsMissingHelperAfterInstall(t *testing.T) {
	paths := testPaths(t)
	installer := NewInstaller(paths)
	if _, err := installer.Install(context.Background(), ProviderClaude); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := os.Remove(installer.helperPath()); err != nil {
		t.Fatalf("remove helper: %v", err)
	}
	status, err := installer.Check(context.Background(), ProviderClaude)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if status.Status != StatusModified {
		t.Fatalf("status = %#v", status)
	}
}

func TestRemoveMissingProviderDoesNotCreateFiles(t *testing.T) {
	paths := testPaths(t)
	installer := NewInstaller(paths)

	status, err := installer.Remove(context.Background(), ProviderClaude)
	if err != nil {
		t.Fatalf("remove missing: %v", err)
	}
	if status.Status != StatusMissing {
		t.Fatalf("status = %#v", status)
	}
	if _, err := os.Stat(paths.ClaudeSettingsPath); !os.IsNotExist(err) {
		t.Fatalf("claude settings exists after missing remove: %v", err)
	}
	if _, err := os.Stat(installer.manifestPath()); !os.IsNotExist(err) {
		t.Fatalf("manifest exists after missing remove: %v", err)
	}
}

func testPaths(t *testing.T) Paths {
	t.Helper()
	root := t.TempDir()
	helperSource := filepath.Join(root, "source-whisk")
	writeFile(t, helperSource, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(helperSource, 0o755); err != nil {
		t.Fatalf("chmod helper source: %v", err)
	}
	return Paths{
		ConfigRoot:         filepath.Join(root, ".config", "whisk"),
		HelperSourcePath:   helperSource,
		ClaudeSettingsPath: filepath.Join(root, ".claude", "settings.json"),
		CodexHooksPath:     filepath.Join(root, ".codex", "hooks.json"),
	}
}

func installedCommand(paths Paths, provider string) string {
	return filepath.Join(paths.ConfigRoot, "bin", "whisk-hook-helper") + " agent-bridge hook -provider " + provider
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readConfigFile(t *testing.T, path string) HookConfig {
	t.Helper()
	cfg, err := readHookConfig(path)
	if err != nil {
		t.Fatalf("read config %s: %v", path, err)
	}
	return cfg
}

func writeConfigFile(t *testing.T, path string, cfg HookConfig) {
	t.Helper()
	if err := writeHookConfig(path, cfg); err != nil {
		t.Fatalf("write config %s: %v", path, err)
	}
}

func readManifestFile(t *testing.T, path string) Manifest {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	return manifest
}

func writeManifestFile(t *testing.T, path string, manifest Manifest) {
	t.Helper()
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func assertRawJSONField(t *testing.T, path string, field string, expected string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	var gotValue any
	if err := json.Unmarshal(fields[field], &gotValue); err != nil {
		t.Fatalf("parse %s: %v", field, err)
	}
	var expectedValue any
	if err := json.Unmarshal([]byte(expected), &expectedValue); err != nil {
		t.Fatalf("parse expected %s: %v", field, err)
	}
	if !reflect.DeepEqual(gotValue, expectedValue) {
		t.Fatalf("%s = %#v, want %#v", field, gotValue, expectedValue)
	}
}

func compactJSON(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		t.Fatalf("compact json: %v", err)
	}
	return buf.String()
}
