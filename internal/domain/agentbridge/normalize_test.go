package agentbridge_test

import (
	"testing"

	"github.com/phin-tech/whisk/internal/domain/agentbridge"
)

func TestNormalizeEventMapsCodexPromptMetadata(t *testing.T) {
	normalized := agentbridge.NormalizeEvent(agentbridge.Event{
		Provider:  agentbridge.ProviderCodex,
		EventName: "UserPromptSubmit",
		Message:   "Implement the feature.",
		Raw: map[string]any{
			"session_id": "codex_session_01",
			"whisk": map[string]any{
				"sessionId": "whisk_session_01",
				"ptyId":     "pty_01",
				"cwd":       "/repo",
				"agent":     "codex",
			},
		},
	})

	if normalized.Kind != agentbridge.EventKindPrompt ||
		normalized.Title != "Codex prompt" ||
		normalized.ProviderSessionID != "codex_session_01" ||
		normalized.SessionID != "whisk_session_01" ||
		normalized.PTYID != "pty_01" ||
		normalized.CWD != "/repo" ||
		normalized.Agent != "codex" ||
		normalized.Answerable {
		t.Fatalf("normalized codex prompt = %#v", normalized)
	}
}

func TestNormalizeEventMapsClaudeQuestionOptions(t *testing.T) {
	normalized := agentbridge.NormalizeEvent(agentbridge.Event{
		Provider:      agentbridge.ProviderClaude,
		EventName:     "Elicitation",
		Message:       "Pick a workflow.",
		ElicitationID: "ask_01",
		Raw: map[string]any{
			"options": []any{
				map[string]any{"label": "Fix a bug", "value": "fix"},
				map[string]any{"label": "Build a feature", "value": "build"},
			},
			"whisk": map[string]any{
				"cwd":   "/repo",
				"agent": "claude",
			},
		},
	})

	if normalized.Kind != agentbridge.EventKindQuestion ||
		normalized.Title != "Claude question" ||
		!normalized.Answerable ||
		normalized.CWD != "/repo" ||
		normalized.Agent != "claude" ||
		len(normalized.Options) != 2 ||
		normalized.Options[0].Label != "Fix a bug" ||
		normalized.Options[0].Value != "fix" ||
		normalized.Options[1].Label != "Build a feature" ||
		normalized.Options[1].Value != "build" {
		t.Fatalf("normalized claude question = %#v", normalized)
	}
}

func TestClaudeElicitationFixtureNormalizesToAnswerablePrompt(t *testing.T) {
	event := agentbridge.Event{
		Provider:      agentbridge.ProviderClaude,
		EventName:     "Elicitation",
		ElicitationID: "ask_01",
		Raw: map[string]any{
			"tool_input": map[string]any{
				"questions": []any{
					map[string]any{
						"question": "Pick a route",
						"options": []any{
							map[string]any{"label": "Fast", "value": "fast"},
							map[string]any{"label": "Safe", "value": "safe"},
						},
					},
				},
			},
		},
	}

	normalized := agentbridge.NormalizeEvent(event)
	if normalized.Kind != agentbridge.EventKindQuestion ||
		!normalized.Answerable ||
		normalized.Message != "Pick a route" ||
		len(normalized.Options) != 2 ||
		normalized.Options[1].Value != "safe" {
		t.Fatalf("normalized elicitation = %#v", normalized)
	}
	out, ok := agentbridge.PromptAnswerToProviderOutput(agentbridge.ProviderClaude, "Elicitation", "ask_01", "safe")
	if !ok {
		t.Fatalf("expected provider output")
	}
	hookSpecific, ok := out["hookSpecificOutput"].(map[string]any)
	if !ok || hookSpecific["decision"] != "safe" || hookSpecific["elicitationId"] != "ask_01" {
		t.Fatalf("provider output = %#v", out)
	}
}

func TestNormalizeEventMapsClaudeAskUserQuestionToolInput(t *testing.T) {
	event := agentbridge.Event{
		Provider:  agentbridge.ProviderClaude,
		EventName: "PermissionRequest",
		ToolName:  "AskUserQuestion",
		Raw: map[string]any{
			"tool_input": map[string]any{
				"questions": []any{
					map[string]any{
						"question": "What would you like to work on today?",
						"options": []any{
							map[string]any{"label": "Fix a bug"},
							map[string]any{"label": "Build a feature"},
						},
					},
				},
			},
		},
	}
	normalized := agentbridge.NormalizeEvent(event)

	if normalized.Kind != agentbridge.EventKindQuestion ||
		normalized.Title != "Claude question" ||
		normalized.Message != "What would you like to work on today?" ||
		len(normalized.Options) != 2 ||
		normalized.Options[0].Label != "Fix a bug" ||
		normalized.Options[0].Value != "Fix a bug" ||
		normalized.Options[1].Label != "Build a feature" ||
		normalized.Options[1].Value != "Build a feature" ||
		!normalized.Answerable {
		t.Fatalf("normalized AskUserQuestion = %#v", normalized)
	}

	event.EventName = "PreToolUse"
	normalized = agentbridge.NormalizeEvent(event)
	if !normalized.Answerable {
		t.Fatalf("PreToolUse AskUserQuestion should be answerable: %#v", normalized)
	}
}
