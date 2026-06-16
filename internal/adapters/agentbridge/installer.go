package agentbridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type InstallRequest struct {
	RootDir   string
	BridgeID  string
	RunID     string
	Provider  string
	HookURL   string
	Token     string
	WhiskCLI  string
	WhiskdURL string
}

type InstalledConfig struct {
	Dir        string
	HookScript string
}

func Install(req InstallRequest) (InstalledConfig, error) {
	if req.RootDir == "" {
		return InstalledConfig{}, fmt.Errorf("agent bridge root dir required")
	}
	if req.BridgeID == "" {
		return InstalledConfig{}, fmt.Errorf("agent bridge id required")
	}
	if req.RunID == "" {
		return InstalledConfig{}, fmt.Errorf("agent bridge run id required")
	}
	dir := filepath.Join(req.RootDir, ".whisk", "agent-bridges", req.RunID, req.BridgeID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return InstalledConfig{}, err
	}
	configPath := filepath.Join(dir, "bridge.json")
	config := map[string]string{
		"bridge_id":  req.BridgeID,
		"provider":   req.Provider,
		"hook_url":   req.HookURL,
		"whisk_cli":  req.WhiskCLI,
		"whiskd_url": req.WhiskdURL,
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return InstalledConfig{}, err
	}
	if err := os.WriteFile(configPath, append(raw, '\n'), 0o600); err != nil {
		return InstalledConfig{}, err
	}

	hookPath := filepath.Join(dir, "hook.sh")
	script := "#!/bin/sh\n" +
		"exec \"$WHISK_CLI\" agent-bridge hook" +
		" -url \"$WHISKD_URL\"" +
		" -bridge \"$WHISK_AGENT_BRIDGE_ID\"" +
		" -token \"$WHISK_AGENT_BRIDGE_TOKEN\"" +
		" -provider \"$WHISK_AGENT_BRIDGE_PROVIDER\"\n"
	if err := os.WriteFile(hookPath, []byte(script), 0o700); err != nil {
		return InstalledConfig{}, err
	}
	return InstalledConfig{Dir: dir, HookScript: hookPath}, nil
}
