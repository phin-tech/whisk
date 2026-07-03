package agentbridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	domainbridge "github.com/phin-tech/whisk/internal/domain/agentbridge"
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
		"bridge_id":           req.BridgeID,
		"provider":            req.Provider,
		"hook_url":            req.HookURL,
		"token":               req.Token,
		"whisk_cli":           req.WhiskCLI,
		"whisk_hook_protocol": strconv.Itoa(domainbridge.HookProtocolVersion),
		"whiskd_url":          req.WhiskdURL,
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return InstalledConfig{}, err
	}
	if err := os.WriteFile(configPath, append(raw, '\n'), 0o600); err != nil {
		return InstalledConfig{}, err
	}

	hookPath := filepath.Join(dir, "hook.sh")
	script := fmt.Sprintf(`#!/bin/sh
# whisk_hook_protocol=%d

config_dir=${WHISK_AGENT_BRIDGE_CONFIG_DIR:-$(cd "$(dirname "$0")" && pwd)}
config_path="$config_dir/bridge.json"

json_value() {
	key="$1"
	sed -n 's/^[[:space:]]*"'"$key"'"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$config_path" | head -n 1
}

if [ -f "$config_path" ]; then
	: "${WHISK_CLI:=$(json_value whisk_cli)}"
	: "${WHISKD_URL:=$(json_value whiskd_url)}"
	: "${WHISK_AGENT_BRIDGE_ID:=$(json_value bridge_id)}"
	: "${WHISK_AGENT_BRIDGE_TOKEN:=$(json_value token)}"
	: "${WHISK_AGENT_BRIDGE_PROVIDER:=$(json_value provider)}"
	: "${WHISK_AGENT_BRIDGE_HOOK_PROTOCOL:=$(json_value whisk_hook_protocol)}"
fi

exec "${WHISK_CLI:-whisk}" agent-bridge hook \
	-url "$WHISKD_URL" \
	-bridge "$WHISK_AGENT_BRIDGE_ID" \
	-token "$WHISK_AGENT_BRIDGE_TOKEN" \
	-provider "$WHISK_AGENT_BRIDGE_PROVIDER" \
	-hook-protocol "$WHISK_AGENT_BRIDGE_HOOK_PROTOCOL"
`, domainbridge.HookProtocolVersion)
	if err := os.WriteFile(hookPath, []byte(script), 0o700); err != nil {
		return InstalledConfig{}, err
	}
	return InstalledConfig{Dir: dir, HookScript: hookPath}, nil
}
