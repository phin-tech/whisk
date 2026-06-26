package app

import "testing"

func TestAgentHookJumpPointTrigger(t *testing.T) {
	tests := []struct {
		name          string
		req           AgentBridgeHookRequest
		needsApproval bool
		want          agentHookJumpPointTrigger
		wantOK        bool
	}{
		{
			name:   "elicitation prompt",
			req:    AgentBridgeHookRequest{EventName: "Elicitation"},
			want:   agentHookJumpPointPrompt,
			wantOK: true,
		},
		{
			name:   "ask user question prompt",
			req:    AgentBridgeHookRequest{EventName: "PermissionRequest", ToolName: "AskUserQuestion"},
			want:   agentHookJumpPointPrompt,
			wantOK: true,
		},
		{
			name:          "blocking approval",
			req:           AgentBridgeHookRequest{EventName: "PermissionRequest", ToolName: "Bash"},
			needsApproval: true,
			want:          agentHookJumpPointApproval,
			wantOK:        true,
		},
		{
			name:   "passive notification",
			req:    AgentBridgeHookRequest{EventName: "Notification"},
			wantOK: false,
		},
		{
			name:          "predecided tool hook",
			req:           AgentBridgeHookRequest{EventName: "PermissionRequest", ToolName: "Bash"},
			needsApproval: false,
			wantOK:        false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := agentHookJumpPointForHook(test.req, test.needsApproval)
			if ok != test.wantOK || got != test.want {
				t.Fatalf("trigger = %q, %v; want %q, %v", got, ok, test.want, test.wantOK)
			}
		})
	}
}
