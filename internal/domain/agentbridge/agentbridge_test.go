package agentbridge_test

import (
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/domain/agentbridge"
)

func TestHookPayloadToEvaluationRequestMapsToolCallAndResult(t *testing.T) {
	pre, ok := agentbridge.HookPayloadToEvaluationRequest(agentbridge.HookPayload{
		Provider:  agentbridge.ProviderClaude,
		EventName: "PreToolUse",
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "rm -rf /tmp/x"},
	})
	if !ok {
		t.Fatalf("expected PreToolUse to produce an evaluation request")
	}
	if pre.Phase != agentbridge.PhaseToolCall || pre.ToolName != "Bash" {
		t.Fatalf("pre request = %#v", pre)
	}
	if pre.ToolInput["command"] != "rm -rf /tmp/x" {
		t.Fatalf("pre tool input = %#v", pre.ToolInput)
	}

	post, ok := agentbridge.HookPayloadToEvaluationRequest(agentbridge.HookPayload{
		Provider:   agentbridge.ProviderCodex,
		EventName:  "PostToolUse",
		ToolName:   "Read",
		ToolInput:  map[string]any{"file_path": "AGENTS.md"},
		ToolOutput: "contents",
	})
	if !ok {
		t.Fatalf("expected PostToolUse to produce an evaluation request")
	}
	if post.Phase != agentbridge.PhaseToolResult || post.ToolOutput != "contents" {
		t.Fatalf("post request = %#v", post)
	}
}

func TestHookPayloadSkipsWhiskRelayTools(t *testing.T) {
	_, ok := agentbridge.HookPayloadToEvaluationRequest(agentbridge.HookPayload{
		Provider:  agentbridge.ProviderClaude,
		EventName: "PreToolUse",
		ToolName:  "mcp__whisk__report_status",
		ToolInput: map[string]any{"message": "working"},
	})
	if ok {
		t.Fatalf("expected Whisk relay tools to be skipped because daemon routes already own them")
	}
}

func TestEvaluationDecisionToProviderOutputDoesNotAutoApproveAllows(t *testing.T) {
	out, ok := agentbridge.EvaluationDecisionToProviderOutput(agentbridge.ProviderClaude, "PreToolUse", agentbridge.EvaluationDecision{
		Action: agentbridge.PolicyAllow,
	})
	if ok || len(out) != 0 {
		t.Fatalf("allow should return no provider override, got ok=%v out=%#v", ok, out)
	}

	deny, ok := agentbridge.EvaluationDecisionToProviderOutput(agentbridge.ProviderCodex, "PreToolUse", agentbridge.EvaluationDecision{
		Action: agentbridge.PolicyDeny,
		Reason: "blocked by workspace policy",
	})
	if !ok {
		t.Fatalf("expected deny output")
	}
	hookSpecific, ok := deny["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("deny output = %#v", deny)
	}
	if hookSpecific["permissionDecision"] != "deny" ||
		hookSpecific["permissionDecisionReason"] != "blocked by workspace policy" {
		t.Fatalf("deny hookSpecificOutput = %#v", hookSpecific)
	}
}

func TestBridgeStateValidatesHookTokenAndTracksPendingQuestion(t *testing.T) {
	state, err := agentbridge.NewState(agentbridge.Bridge{
		ID:        "bridge_01",
		SessionID: "sess_01",
		PTYID:     "pty_01",
		Provider:  agentbridge.ProviderClaude,
		TokenHash: agentbridge.HashHookToken("secret-token"),
	})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	if !state.ValidateHookToken("bridge_01", "secret-token") {
		t.Fatalf("expected valid hook token")
	}
	if state.ValidateHookToken("bridge_01", "wrong-token") {
		t.Fatalf("wrong hook token validated")
	}

	updated, question, err := state.RecordPendingQuestion(agentbridge.RecordPendingQuestion{
		ID:       "question_01",
		BridgeID: "bridge_01",
		RunID:    "run_01",
		Prompt:   "Need approval to run command?",
	})
	if err != nil {
		t.Fatalf("record question: %v", err)
	}
	if question.Status != agentbridge.QuestionPending || question.Prompt == "" {
		t.Fatalf("question = %#v", question)
	}
	resolved, err := updated.ResolveQuestion(agentbridge.ResolveQuestion{
		ID:     "question_01",
		Answer: "approved",
	})
	if err != nil {
		t.Fatalf("resolve question: %v", err)
	}
	if resolvedQuestion, ok := resolved.Question("question_01"); !ok ||
		resolvedQuestion.Status != agentbridge.QuestionResolved ||
		resolvedQuestion.Answer != "approved" {
		t.Fatalf("resolved question = %#v ok=%v", resolvedQuestion, ok)
	}
}

func TestBridgeStateTracksPromptLifecycle(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	state, err := agentbridge.NewState(agentbridge.Bridge{
		ID:        "bridge_01",
		SessionID: "sess_01",
		PTYID:     "pty_01",
		Provider:  agentbridge.ProviderClaude,
		TokenHash: agentbridge.HashHookToken("secret-token"),
	})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}

	pendingState, prompt, err := state.RecordPendingPrompt(agentbridge.RecordPendingPrompt{
		ID:            "prompt_01",
		BridgeID:      "bridge_01",
		Kind:          agentbridge.PromptKindQuestion,
		EventName:     "Elicitation",
		Message:       "Pick one",
		ElicitationID: "ask_01",
		Options:       []agentbridge.PromptOption{{Label: "One", Value: "one"}},
		Now:           now,
	})
	if err != nil {
		t.Fatalf("record prompt: %v", err)
	}
	if prompt.Status != agentbridge.PromptPending ||
		prompt.SessionID != "sess_01" ||
		prompt.PTYID != "pty_01" ||
		prompt.Message != "Pick one" ||
		len(prompt.Options) != 1 {
		t.Fatalf("prompt = %#v", prompt)
	}

	resolvedState, resolved, err := pendingState.ResolvePrompt(agentbridge.ResolvePrompt{
		ID:     "prompt_01",
		Answer: "one",
		Now:    now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("resolve prompt: %v", err)
	}
	if resolved.Status != agentbridge.PromptResolved || resolved.Answer != "one" || resolved.ResolvedAt == nil {
		t.Fatalf("resolved prompt = %#v", resolved)
	}
	if pending := resolvedState.ListPrompts(agentbridge.ListPrompts{Status: agentbridge.PromptPending}); len(pending) != 0 {
		t.Fatalf("pending prompts = %#v", pending)
	}
	if _, _, err := resolvedState.ResolvePrompt(agentbridge.ResolvePrompt{ID: "prompt_01", Answer: "one"}); err == nil {
		t.Fatalf("expected already resolved prompt error")
	}
}

func TestResolvePromptRejectsUnknownOption(t *testing.T) {
	state, err := agentbridge.NewState(agentbridge.Bridge{
		ID:        "bridge_01",
		SessionID: "sess_01",
		PTYID:     "pty_01",
		Provider:  agentbridge.ProviderClaude,
		TokenHash: agentbridge.HashHookToken("secret-token"),
	})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	pendingState, _, err := state.RecordPendingPrompt(agentbridge.RecordPendingPrompt{
		ID:        "prompt_01",
		BridgeID:  "bridge_01",
		Kind:      agentbridge.PromptKindQuestion,
		EventName: "Elicitation",
		Message:   "Pick one",
		Options:   []agentbridge.PromptOption{{Label: "One", Value: "one"}},
	})
	if err != nil {
		t.Fatalf("record prompt: %v", err)
	}
	if _, _, err := pendingState.ResolvePrompt(agentbridge.ResolvePrompt{ID: "prompt_01", Answer: "two"}); err == nil {
		t.Fatalf("expected invalid option error")
	}
}

func TestPromptAnswerToProviderOutput(t *testing.T) {
	out, ok := agentbridge.PromptAnswerToProviderOutput(agentbridge.ProviderClaude, "Elicitation", "ask_01", "one")
	if !ok {
		t.Fatalf("expected elicitation answer output")
	}
	hookSpecific, ok := out["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("output = %#v", out)
	}
	if hookSpecific["hookEventName"] != "Elicitation" ||
		hookSpecific["elicitationId"] != "ask_01" ||
		hookSpecific["decision"] != "one" {
		t.Fatalf("hookSpecificOutput = %#v", hookSpecific)
	}

	out, ok = agentbridge.PromptAnswerToProviderOutput(agentbridge.ProviderClaude, "PreToolUse", "", "Uranus", map[string]any{
		"questions": []any{
			map[string]any{
				"question": "Which planet rotates on its side?",
				"options":  []any{map[string]any{"label": "Uranus"}},
			},
		},
	})
	if !ok {
		t.Fatalf("expected AskUserQuestion output")
	}
	hookSpecific, ok = out["hookSpecificOutput"].(map[string]any)
	if !ok || hookSpecific["hookEventName"] != "PreToolUse" || hookSpecific["permissionDecision"] != "allow" {
		t.Fatalf("AskUserQuestion hookSpecificOutput = %#v", hookSpecific)
	}
	updatedInput, ok := hookSpecific["updatedInput"].(map[string]any)
	if !ok {
		t.Fatalf("updatedInput = %#v", hookSpecific["updatedInput"])
	}
	answers, ok := updatedInput["answers"].(map[string]any)
	if !ok || answers["Which planet rotates on its side?"] != "Uranus" {
		t.Fatalf("answers = %#v", updatedInput["answers"])
	}
}

func TestBridgeStateTracksApprovalLifecycle(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	state, err := agentbridge.NewState(agentbridge.Bridge{
		ID:        "bridge_01",
		SessionID: "sess_01",
		PTYID:     "pty_01",
		Provider:  agentbridge.ProviderClaude,
		TokenHash: agentbridge.HashHookToken("secret-token"),
	})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}

	pendingState, approval, err := state.RecordPendingApproval(agentbridge.RecordPendingApproval{
		ID:        "approval_01",
		BridgeID:  "bridge_01",
		RunID:     "run_01",
		EventName: "PreToolUse",
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "pwd"},
		Now:       now,
	})
	if err != nil {
		t.Fatalf("record approval: %v", err)
	}
	if approval.Status != agentbridge.ApprovalPending ||
		approval.SessionID != "sess_01" ||
		approval.PTYID != "pty_01" ||
		approval.Provider != agentbridge.ProviderClaude ||
		approval.ToolInput["command"] != "pwd" {
		t.Fatalf("approval = %#v", approval)
	}
	if approvals := pendingState.ListApprovals(agentbridge.ListApprovals{Status: agentbridge.ApprovalPending}); len(approvals) != 1 {
		t.Fatalf("pending approvals = %#v", approvals)
	}

	resolvedState, resolved, err := pendingState.ResolveApproval(agentbridge.ResolveApproval{
		ID: "approval_01",
		Decision: agentbridge.EvaluationDecision{
			Action: agentbridge.PolicyAllow,
		},
		Now: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("resolve approval: %v", err)
	}
	if resolved.Status != agentbridge.ApprovalResolved || resolved.Decision.Action != agentbridge.PolicyAllow || resolved.ResolvedAt == nil {
		t.Fatalf("resolved approval = %#v", resolved)
	}
	if approvals := resolvedState.ListApprovals(agentbridge.ListApprovals{Status: agentbridge.ApprovalPending}); len(approvals) != 0 {
		t.Fatalf("pending approvals after resolve = %#v", approvals)
	}
}

func TestBridgeStateMarksPassiveEventRead(t *testing.T) {
	state, err := agentbridge.NewState()
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	pending, event, err := state.RecordEvent(agentbridge.RecordEvent{
		ID:        "event_01",
		Provider:  agentbridge.ProviderClaude,
		EventName: "Notification",
		Message:   "Need input.",
	})
	if err != nil {
		t.Fatalf("record event: %v", err)
	}
	if event.Status != agentbridge.EventPending {
		t.Fatalf("event = %#v", event)
	}

	readState, read, err := pending.MarkEventRead(agentbridge.MarkEventRead{ID: "event_01"})
	if err != nil {
		t.Fatalf("mark event read: %v", err)
	}
	if read.Status != agentbridge.EventRead {
		t.Fatalf("read = %#v", read)
	}
	if pendingEvents := readState.ListEvents(agentbridge.ListEvents{Status: agentbridge.EventPending}); len(pendingEvents) != 0 {
		t.Fatalf("pending events = %#v", pendingEvents)
	}
	if _, _, err := readState.MarkEventRead(agentbridge.MarkEventRead{ID: "missing"}); err == nil {
		t.Fatalf("expected missing event error")
	}
}

func TestBridgeStateTimeoutApprovalDenies(t *testing.T) {
	state, err := agentbridge.NewState(agentbridge.Bridge{
		ID:        "bridge_01",
		SessionID: "sess_01",
		PTYID:     "pty_01",
		Provider:  agentbridge.ProviderClaude,
		TokenHash: agentbridge.HashHookToken("secret-token"),
	})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	pendingState, _, err := state.RecordPendingApproval(agentbridge.RecordPendingApproval{
		ID:        "approval_01",
		BridgeID:  "bridge_01",
		EventName: "PreToolUse",
		ToolName:  "Bash",
	})
	if err != nil {
		t.Fatalf("record approval: %v", err)
	}

	timedOutState, timedOut, err := pendingState.TimeoutApproval(agentbridge.TimeoutApproval{ID: "approval_01"})
	if err != nil {
		t.Fatalf("timeout approval: %v", err)
	}
	if timedOut.Status != agentbridge.ApprovalTimedOut ||
		timedOut.Decision.Action != agentbridge.PolicyDeny ||
		timedOut.Decision.Reason != "Approval timed out" {
		t.Fatalf("timed out approval = %#v", timedOut)
	}
	if _, _, err := timedOutState.ResolveApproval(agentbridge.ResolveApproval{
		ID:       "approval_01",
		Decision: agentbridge.EvaluationDecision{Action: agentbridge.PolicyAllow},
	}); err == nil {
		t.Fatalf("expected timed out approval to reject later resolve")
	}
}
