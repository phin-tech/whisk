package agents

import (
	"path/filepath"
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
	// The prompt rides in argv (positional) so Claude auto-runs the first turn on
	// launch, and Stdin stays empty — nothing is typed into the TUI.
	wantArgs := []string{"--permission-mode", "plan", "--append-system-prompt", "Follow AGENTS.md", "Plan the work"}
	if launch.Command != "claude" || !reflect.DeepEqual(launch.Args, wantArgs) || launch.WorkingDir != "/repo" {
		t.Fatalf("launch = %#v", launch)
	}
	if launch.Stdin != "" || launch.Provider != ProviderClaude {
		t.Fatalf("launch metadata = %#v", launch)
	}
}

func TestClaudeOpenRouterLaunchCopiesOpenRouterEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("OPENROUTER_API_KEY", "or-test-key")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "")
	t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "")
	t.Setenv("ANTHROPIC_DEFAULT_HAIKU_MODEL", "")
	t.Setenv("CLAUDE_CODE_SUBAGENT_MODEL", "")

	launch, err := BuildLaunch(LaunchRequest{
		ProfileID:    "claude-openrouter",
		WorkingDir:   "/repo",
		SystemPrompt: "Keep it cheap.",
		Prompt:       "Say ok.",
	})
	if err != nil {
		t.Fatalf("BuildLaunch error: %v", err)
	}
	if launch.Command != "claude" || launch.Provider != ProviderClaude {
		t.Fatalf("launch = %#v", launch)
	}
	if !reflect.DeepEqual(launch.Args[:5], []string{"--print", "--max-budget-usd", "0.05", "--allowedTools", "Bash(whisk question ask*)"}) {
		t.Fatalf("args = %#v", launch.Args)
	}
	if launch.Args[len(launch.Args)-1] != "Say ok." || launch.Stdin != "" {
		t.Fatalf("prompt delivery = args %#v, stdin %q", launch.Args, launch.Stdin)
	}
	wantEnv := map[string]string{
		"PATH":                           "/opt/homebrew/bin:/usr/local/bin:" + filepath.Join(home, ".local", "bin") + ":" + filepath.Join(home, "bin") + ":/usr/bin",
		"ANTHROPIC_BASE_URL":             "https://openrouter.ai/api",
		"ANTHROPIC_AUTH_TOKEN":           "or-test-key",
		"ANTHROPIC_API_KEY":              "",
		"ANTHROPIC_DEFAULT_OPUS_MODEL":   "~anthropic/claude-haiku-latest",
		"ANTHROPIC_DEFAULT_SONNET_MODEL": "~anthropic/claude-haiku-latest",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "~anthropic/claude-haiku-latest",
		"CLAUDE_CODE_SUBAGENT_MODEL":     "~anthropic/claude-haiku-latest",
	}
	if !reflect.DeepEqual(launch.Env, wantEnv) {
		t.Fatalf("env = %#v", launch.Env)
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
	wantArgs := []string{"-c", "instructions=Be terse", "Implement it"}
	if launch.Command != "codex" || !reflect.DeepEqual(launch.Args, wantArgs) || launch.Stdin != "" {
		t.Fatalf("launch = %#v", launch)
	}
}

func TestCodexPlanLaunchUsesReadOnlySandboxAndInstructionsConfig(t *testing.T) {
	launch, err := BuildLaunch(LaunchRequest{
		ProfileID:    "codex-plan",
		WorkingDir:   "/repo",
		SystemPrompt: "Be terse",
		Prompt:       "Plan it",
	})
	if err != nil {
		t.Fatalf("BuildLaunch error: %v", err)
	}
	wantArgs := []string{"--sandbox", "read-only", "-c", "instructions=Be terse", "Plan it"}
	if launch.Command != "codex" || !reflect.DeepEqual(launch.Args, wantArgs) || launch.Stdin != "" {
		t.Fatalf("launch = %#v", launch)
	}
}

func TestCodexLaunchAddsCommonAgentPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin")

	launch, err := BuildLaunch(LaunchRequest{ProfileID: "codex"})
	if err != nil {
		t.Fatalf("BuildLaunch error: %v", err)
	}

	want := []string{
		"/opt/homebrew/bin",
		"/usr/local/bin",
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "bin"),
		"/usr/bin",
	}
	if got := filepath.SplitList(launch.Env["PATH"]); !reflect.DeepEqual(got, want) {
		t.Fatalf("PATH = %#v, want %#v", got, want)
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

func TestCommandLineQuotesArgs(t *testing.T) {
	got := CommandLine("codex", []string{"-c", "instructions=Don't overbuild.", "say $PATH"})
	want := "codex -c 'instructions=Don'\\''t overbuild.' 'say $PATH'"
	if got != want {
		t.Fatalf("command line = %q, want %q", got, want)
	}
}

func TestUnknownProfileFails(t *testing.T) {
	if _, err := BuildLaunch(LaunchRequest{ProfileID: "missing"}); err == nil {
		t.Fatalf("expected unknown profile error")
	}
}

func TestProfileListReturnsOrderedBuiltinProfilesWithLabels(t *testing.T) {
	profiles := ProfileList()

	gotIDs := make([]string, len(profiles))
	for i, p := range profiles {
		gotIDs[i] = p.ID
	}
	wantIDs := []string{"claude", "claude-plan", "claude-openrouter", "codex", "codex-plan", "plain-shell", "prompt-capture"}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("profile ids = %#v, want %#v", gotIDs, wantIDs)
	}

	if len(profiles) != len(BuiltinProfiles()) {
		t.Fatalf("ProfileList covers %d profiles, BuiltinProfiles has %d", len(profiles), len(BuiltinProfiles()))
	}

	byID := map[string]ProfileInfo{}
	for _, p := range profiles {
		if p.Label == "" {
			t.Fatalf("profile %q missing label", p.ID)
		}
		byID[p.ID] = p
	}
	if byID["claude"].Label != "Claude Code" || byID["claude"].Provider != ProviderClaude {
		t.Fatalf("claude profile = %#v", byID["claude"])
	}
	if byID["codex"].Provider != ProviderCodex {
		t.Fatalf("codex profile = %#v", byID["codex"])
	}
	if byID["plain-shell"].Provider != ProviderShell {
		t.Fatalf("plain-shell profile = %#v", byID["plain-shell"])
	}
}
