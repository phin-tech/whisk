package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/protocol"
)

func runAgentBridge(args []string) error {
	if len(args) == 0 || args[0] != "hook" {
		return fmt.Errorf("usage: whisk agent-bridge hook [-url http://127.0.0.1:8787] [-bridge id] [-token token] [-provider claude|codex] [-event name]")
	}
	return runAgentBridgeHook(args[1:], os.Stdin, os.Stdout)
}

func runAgentBridgeHook(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("agent-bridge hook", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	bridgeID := flags.String("bridge", envOrDefault("WHISK_AGENT_BRIDGE_ID", ""), "agent bridge id")
	token := flags.String("token", envOrDefault("WHISK_AGENT_BRIDGE_TOKEN", ""), "agent bridge hook token")
	provider := flags.String("provider", envOrDefault("WHISK_AGENT_BRIDGE_PROVIDER", ""), "agent provider")
	eventName := flags.String("event", "", "provider hook event name")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk agent-bridge hook [-url http://127.0.0.1:8787] [-bridge id] [-token token] [-provider claude|codex] [-event name]")
	}

	payload, err := readHookPayload(stdin)
	if err != nil {
		return nil
	}
	req := hookRequestFromPayload(payload)
	req.Token = *token
	if req.Provider == "" {
		req.Provider = *provider
	}
	if *eventName != "" {
		req.EventName = *eventName
	}
	if req.SessionID == "" {
		req.SessionID = os.Getenv("WHISK_SESSION_ID")
	}
	if req.PTYID == "" {
		req.PTYID = os.Getenv("WHISK_PTY_ID")
	}
	addWhiskHookMetadata(req.RawPayload, req.Provider)
	daemon := client.NewHTTP(*baseURL, nil)
	if *bridgeID == "" || req.Token == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = daemon.RecordAgentHookEvent(ctx, req)
		return nil
	}

	ctx := context.Background()
	response, err := daemon.AgentBridgeHook(ctx, *bridgeID, req)
	if err != nil {
		return nil
	}
	if response.Output == nil {
		return nil
	}
	return json.NewEncoder(stdout).Encode(response.Output)
}

func addWhiskHookMetadata(payload map[string]any, provider string) {
	if payload == nil {
		return
	}
	meta := map[string]any{}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		meta["cwd"] = cwd
	}
	addEnvMetadata(meta, "sessionId", "WHISK_SESSION_ID")
	addEnvMetadata(meta, "ptyId", "WHISK_PTY_ID")
	addEnvMetadata(meta, "projectId", "WHISK_PROJECT_ID")
	addEnvMetadata(meta, "projectRoot", "WHISK_PROJECT_ROOT")
	addEnvMetadata(meta, "workItemId", "WHISK_WORK_ITEM_ID")
	addEnvMetadata(meta, "runId", "WHISK_RUN_ID")
	addEnvMetadata(meta, "actor", "WHISK_ACTOR")
	if provider != "" {
		meta["provider"] = provider
	}
	if len(meta) > 0 {
		payload["whisk"] = meta
	}
}

func addEnvMetadata(meta map[string]any, key string, env string) {
	if value := os.Getenv(env); value != "" {
		meta[key] = value
	}
}

func readHookPayload(stdin io.Reader) (map[string]any, error) {
	var payload map[string]any
	if err := json.NewDecoder(stdin).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func hookRequestFromPayload(payload map[string]any) protocol.AgentBridgeHookRequest {
	req := protocol.AgentBridgeHookRequest{
		EventName:        firstStringField(payload, "hook_event_name", "eventName", "event_name"),
		ToolName:         firstStringField(payload, "tool_name", "toolName"),
		ToolInput:        objectField(payload, "tool_input"),
		ToolOutput:       firstStringField(payload, "tool_output", "toolOutput"),
		Message:          firstStringField(payload, "message", "prompt"),
		NotificationType: firstStringField(payload, "notification_type", "notificationType", "type"),
		ElicitationID:    firstStringField(payload, "elicitation_id", "elicitationId"),
		Action:           firstStringField(payload, "action"),
		SessionID:        firstStringField(payload, "session_id", "sessionId"),
		PTYID:            firstStringField(payload, "pty_id", "ptyId"),
		RawPayload:       payload,
	}
	if provider := stringField(payload, "provider"); provider != "" {
		req.Provider = provider
	}
	return req
}

func stringField(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}

func firstStringField(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringField(payload, key); value != "" {
			return value
		}
	}
	return ""
}

func objectField(payload map[string]any, key string) map[string]any {
	value, _ := payload[key].(map[string]any)
	return value
}
