package agents

import (
	"fmt"
	"os"
)

type Provider string

const (
	ProviderShell  Provider = "shell"
	ProviderClaude Provider = "claude"
	ProviderCodex  Provider = "codex"
)

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
		if req.Prompt != "" && containsArg(args, "--print") {
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
	if profile.Provider == ProviderCodex || (profile.Provider == ProviderClaude && containsArg(args, "--print")) {
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
		},
		"claude-plan": {
			ID:       "claude-plan",
			Provider: ProviderClaude,
			Command:  "claude",
			Args:     []string{"--permission-mode", "plan"},
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
			Env: map[string]string{
				"ANTHROPIC_BASE_URL":             "https://openrouter.ai/api",
				"ANTHROPIC_AUTH_TOKEN":           os.Getenv("OPENROUTER_API_KEY"),
				"ANTHROPIC_API_KEY":              "",
				"ANTHROPIC_DEFAULT_OPUS_MODEL":   envOrDefault("ANTHROPIC_DEFAULT_OPUS_MODEL", "~anthropic/claude-haiku-latest"),
				"ANTHROPIC_DEFAULT_SONNET_MODEL": envOrDefault("ANTHROPIC_DEFAULT_SONNET_MODEL", "~anthropic/claude-haiku-latest"),
				"ANTHROPIC_DEFAULT_HAIKU_MODEL":  envOrDefault("ANTHROPIC_DEFAULT_HAIKU_MODEL", "~anthropic/claude-haiku-latest"),
				"CLAUDE_CODE_SUBAGENT_MODEL":     envOrDefault("CLAUDE_CODE_SUBAGENT_MODEL", "~anthropic/claude-haiku-latest"),
			},
		},
		"codex": {
			ID:       "codex",
			Provider: ProviderCodex,
			Command:  "codex",
		},
	}
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
