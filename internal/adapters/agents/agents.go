package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Provider string

const (
	ProviderShell  Provider = "shell"
	ProviderClaude Provider = "claude"
	ProviderCodex  Provider = "codex"
)

// ProfileInfo is the selectable, human-facing view of a builtin agent profile.
// It deliberately omits launch internals (command/args/env) so it can be surfaced
// to clients as a read model.
type ProfileInfo struct {
	ID          string
	Provider    Provider
	Label       string
	Description string
}

// profileCatalog lists the builtin profiles in display order with human labels.
// This is the source of truth for what clients may pick; keep it in sync with
// BuiltinProfiles.
var profileCatalog = []ProfileInfo{
	{ID: "claude", Provider: ProviderClaude, Label: "Claude Code", Description: "Claude Code with default permissions."},
	{ID: "claude-plan", Provider: ProviderClaude, Label: "Claude Code (plan mode)", Description: "Claude Code restricted to plan mode."},
	{ID: "claude-openrouter", Provider: ProviderClaude, Label: "Claude Code (OpenRouter)", Description: "Claude Code routed through OpenRouter with a budget cap."},
	{ID: "codex", Provider: ProviderCodex, Label: "Codex", Description: "OpenAI Codex CLI."},
	{ID: "plain-shell", Provider: ProviderShell, Label: "Shell", Description: "Plain interactive shell, no agent."},
	{ID: "prompt-capture", Provider: ProviderShell, Label: "Prompt capture", Description: "Echoes the prompt via cat for smoke tests."},
}

// ProfileList returns the selectable builtin agent profiles in display order.
func ProfileList() []ProfileInfo {
	out := make([]ProfileInfo, len(profileCatalog))
	copy(out, profileCatalog)
	return out
}

type Profile struct {
	ID       string
	Provider Provider
	Command  string
	Args     []string
	Env      map[string]string
}

type LaunchRequest struct {
	ProfileID    string
	Profile      *Profile
	WorkingDir   string
	SystemPrompt string
	Prompt       string
	Env          map[string]string
}

type Launch struct {
	ProfileID  string
	Provider   Provider
	Command    string
	Args       []string
	WorkingDir string
	Env        map[string]string
	Stdin      string
}

func BuildLaunch(req LaunchRequest) (Launch, error) {
	profile, err := resolveProfile(req)
	if err != nil {
		return Launch{}, err
	}
	args := append([]string(nil), profile.Args...)
	switch profile.Provider {
	case ProviderClaude:
		if req.SystemPrompt != "" {
			args = append(args, "--append-system-prompt", req.SystemPrompt)
		}
		// Pass the prompt as a positional arg so Claude Code auto-runs the first
		// turn on launch — same "just go" behavior as --print, but it stays in the
		// interactive session instead of printing and exiting. No typing into the
		// TUI means no readiness/paste/Enter races to fight.
		if req.Prompt != "" {
			args = append(args, req.Prompt)
		}
	case ProviderCodex:
		if req.SystemPrompt != "" {
			args = append(args, "-c", "instructions="+req.SystemPrompt)
		}
		if req.Prompt != "" {
			args = append(args, req.Prompt)
		}
	}
	stdin := req.Prompt
	if profile.Provider == ProviderCodex || profile.Provider == ProviderClaude {
		stdin = ""
	}
	return Launch{
		ProfileID:  profile.ID,
		Provider:   profile.Provider,
		Command:    profile.Command,
		Args:       args,
		WorkingDir: req.WorkingDir,
		Env:        mergeEnv(profile.Env, req.Env),
		Stdin:      stdin,
	}, nil
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func CommandLine(command string, args []string) string {
	parts := []string{quoteShellArg(command)}
	for _, arg := range args {
		parts = append(parts, quoteShellArg(arg))
	}
	return strings.Join(parts, " ")
}

func quoteShellArg(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return !(r == '/' || r == '-' || r == '_' || r == '.' || r == '=' || r == ':' || r == ',' || r == '+' || r == '@' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z')
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func resolveProfile(req LaunchRequest) (Profile, error) {
	if req.Profile != nil {
		profile := *req.Profile
		profile.Args = append([]string(nil), req.Profile.Args...)
		profile.Env = mergeEnv(req.Profile.Env, nil)
		if profile.Command == "" {
			return Profile{}, fmt.Errorf("profile command required")
		}
		return profile, nil
	}
	id := req.ProfileID
	if id == "" {
		id = "plain-shell"
	}
	profile, ok := BuiltinProfiles()[id]
	if !ok {
		return Profile{}, fmt.Errorf("unknown agent profile %q", id)
	}
	return profile, nil
}

func BuiltinProfiles() map[string]Profile {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	agentEnv := map[string]string{"PATH": commonAgentPath()}
	return map[string]Profile{
		"plain-shell": {
			ID:       "plain-shell",
			Provider: ProviderShell,
			Command:  shell,
		},
		"prompt-capture": {
			ID:       "prompt-capture",
			Provider: ProviderShell,
			Command:  "cat",
		},
		"claude": {
			ID:       "claude",
			Provider: ProviderClaude,
			Command:  "claude",
			Env:      agentEnv,
		},
		"claude-plan": {
			ID:       "claude-plan",
			Provider: ProviderClaude,
			Command:  "claude",
			Args:     []string{"--permission-mode", "plan"},
			Env:      agentEnv,
		},
		"claude-openrouter": {
			ID:       "claude-openrouter",
			Provider: ProviderClaude,
			Command:  "claude",
			Args: []string{
				"--print",
				"--max-budget-usd", "0.05",
				"--allowedTools", "Bash(whisk question ask*)",
			},
			Env: mergeEnv(agentEnv, map[string]string{
				"ANTHROPIC_BASE_URL":             "https://openrouter.ai/api",
				"ANTHROPIC_AUTH_TOKEN":           os.Getenv("OPENROUTER_API_KEY"),
				"ANTHROPIC_API_KEY":              "",
				"ANTHROPIC_DEFAULT_OPUS_MODEL":   envOrDefault("ANTHROPIC_DEFAULT_OPUS_MODEL", "~anthropic/claude-haiku-latest"),
				"ANTHROPIC_DEFAULT_SONNET_MODEL": envOrDefault("ANTHROPIC_DEFAULT_SONNET_MODEL", "~anthropic/claude-haiku-latest"),
				"ANTHROPIC_DEFAULT_HAIKU_MODEL":  envOrDefault("ANTHROPIC_DEFAULT_HAIKU_MODEL", "~anthropic/claude-haiku-latest"),
				"CLAUDE_CODE_SUBAGENT_MODEL":     envOrDefault("CLAUDE_CODE_SUBAGENT_MODEL", "~anthropic/claude-haiku-latest"),
			}),
		},
		"codex": {
			ID:       "codex",
			Provider: ProviderCodex,
			Command:  "codex",
			Env:      agentEnv,
		},
	}
}

func commonAgentPath() string {
	paths := []string{"/opt/homebrew/bin", "/usr/local/bin"}
	if home := os.Getenv("HOME"); home != "" {
		paths = append(paths, filepath.Join(home, ".local", "bin"), filepath.Join(home, "bin"))
	}
	paths = append(paths, filepath.SplitList(os.Getenv("PATH"))...)

	seen := map[string]bool{}
	out := paths[:0]
	for _, path := range paths {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return strings.Join(out, string(os.PathListSeparator))
}

func mergeEnv(left map[string]string, right map[string]string) map[string]string {
	if len(left) == 0 && len(right) == 0 {
		return nil
	}
	merged := map[string]string{}
	for key, value := range left {
		merged[key] = value
	}
	for key, value := range right {
		merged[key] = value
	}
	return merged
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
