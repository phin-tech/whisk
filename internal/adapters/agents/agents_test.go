package agents

import (
	"reflect"
	"testing"
)

func TestClaudePlanLaunchUsesPlanModeAndSystemPrompt(t *testing.T) {
	launch, err := BuildLaunch(LaunchRequest{
		ProfileID:    "claude-plan",
		WorkingDir:   "/repo",
		SystemPrompt: "Follow AGENTS.md",
		Prompt:       "Plan the work",
	})
	if err != nil {
		t.Fatalf("BuildLaunch error: %v", err)
	}
	wantArgs := []string{"--permission-mode", "plan", "--append-system-prompt", "Follow AGENTS.md"}
	if launch.Command != "claude" || !reflect.DeepEqual(launch.Args, wantArgs) || launch.WorkingDir != "/repo" {
		t.Fatalf("launch = %#v", launch)
	}
	if launch.Stdin != "Plan the work" || launch.Provider != ProviderClaude {
		t.Fatalf("launch metadata = %#v", launch)
	}
}

func TestCodexLaunchUsesInstructionsConfig(t *testing.T) {
	launch, err := BuildLaunch(LaunchRequest{
		ProfileID:    "codex",
		WorkingDir:   "/repo",
		SystemPrompt: "Be terse",
		Prompt:       "Implement it",
	})
	if err != nil {
		t.Fatalf("BuildLaunch error: %v", err)
	}
	wantArgs := []string{"-c", "instructions=Be terse"}
	if launch.Command != "codex" || !reflect.DeepEqual(launch.Args, wantArgs) || launch.Stdin != "Implement it" {
		t.Fatalf("launch = %#v", launch)
	}
}

func TestPromptCaptureLaunchEchoesPromptThroughCat(t *testing.T) {
	launch, err := BuildLaunch(LaunchRequest{
		ProfileID:  "prompt-capture",
		WorkingDir: "/repo",
		Prompt:     "Smoke prompt",
	})
	if err != nil {
		t.Fatalf("BuildLaunch error: %v", err)
	}
	if launch.Command != "cat" || len(launch.Args) != 0 || launch.Stdin != "Smoke prompt" || launch.WorkingDir != "/repo" {
		t.Fatalf("launch = %#v", launch)
	}
}

func TestInlineProfileMergesArgsAndEnv(t *testing.T) {
	launch, err := BuildLaunch(LaunchRequest{
		Profile: &Profile{
			ID:       "review",
			Provider: ProviderClaude,
			Command:  "/opt/bin/claude",
			Args:     []string{"--model", "opus"},
			Env:      map[string]string{"A": "B"},
		},
		Env: map[string]string{"C": "D"},
	})
	if err != nil {
		t.Fatalf("BuildLaunch error: %v", err)
	}
	if launch.Command != "/opt/bin/claude" || !reflect.DeepEqual(launch.Args, []string{"--model", "opus"}) {
		t.Fatalf("launch = %#v", launch)
	}
	if !reflect.DeepEqual(launch.Env, map[string]string{"A": "B", "C": "D"}) {
		t.Fatalf("env = %#v", launch.Env)
	}
}

func TestUnknownProfileFails(t *testing.T) {
	if _, err := BuildLaunch(LaunchRequest{ProfileID: "missing"}); err == nil {
		t.Fatalf("expected unknown profile error")
	}
}
