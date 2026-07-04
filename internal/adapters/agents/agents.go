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

type ProfileSource string

const (
	ProfileSourceBuiltin ProfileSource = "builtin"
	ProfileSourcePlugin  ProfileSource = "plugin"
)

type PromptInjectionMode string

const (
	PromptInjectionArgv                  PromptInjectionMode = "argv"
	PromptInjectionFlagPrompt            PromptInjectionMode = "flag-prompt"
	PromptInjectionFlagPromptInteractive PromptInjectionMode = "flag-prompt-interactive"
	PromptInjectionFlagInteractive       PromptInjectionMode = "flag-interactive"
	PromptInjectionStdinAfterStart       PromptInjectionMode = "stdin-after-start"
)

type PreflightTrust string

const (
	PreflightTrustCursor  PreflightTrust = "cursor"
	PreflightTrustCopilot PreflightTrust = "copilot"
	PreflightTrustCodex   PreflightTrust = "codex"
)

type ReadySignal string

const (
	ReadySignalRenderQuietAfterBracketedPaste  ReadySignal = "render-quiet-after-bracketed-paste"
	ReadySignalCodexComposerPrompt             ReadySignal = "codex-composer-prompt"
	ReadySignalRenderCursorAfterBracketedPaste ReadySignal = "render-cursor-after-bracketed-paste"
)

// ProfileInfo is the selectable, human-facing view of a daemon agent profile.
// It deliberately omits launch internals (command/args/env) so it can be surfaced
// to clients as a read model.
type ProfileInfo struct {
	ID                  string
	Provider            Provider
	Label               string
	Description         string
	Source              ProfileSource
	PluginID            string
	Launchable          bool
	LaunchBlockedReason string
	DetectCmd           string
	DetectAliases       []string
	ExpectedProcess     string
	PromptInjectionMode PromptInjectionMode
	DraftPromptFlag     string
	DraftPromptEnvVar   string
	PreflightTrust      PreflightTrust
	ReadySignal         ReadySignal
	HookProvider        string
}

// profileCatalog lists the builtin profiles in display order with human labels.
// This is the source of truth for what clients may pick; keep it in sync with
// BuiltinProfiles.
var profileCatalog = []ProfileInfo{
	{ID: "claude", Provider: ProviderClaude, Label: "Claude Code", Description: "Claude Code with default permissions.", DetectCmd: "claude", ExpectedProcess: "claude", PromptInjectionMode: PromptInjectionArgv, DraftPromptFlag: "--prefill"},
	{ID: "claude-plan", Provider: ProviderClaude, Label: "Claude Code (plan mode)", Description: "Claude Code restricted to plan mode.", DetectCmd: "claude", ExpectedProcess: "claude", PromptInjectionMode: PromptInjectionArgv, DraftPromptFlag: "--prefill"},
	{ID: "claude-openrouter", Provider: ProviderClaude, Label: "Claude Code (OpenRouter)", Description: "Claude Code routed through OpenRouter with a budget cap.", DetectCmd: "claude", ExpectedProcess: "claude", PromptInjectionMode: PromptInjectionArgv, DraftPromptFlag: "--prefill"},
	{ID: "codex", Provider: ProviderCodex, Label: "Codex", Description: "OpenAI Codex CLI.", DetectCmd: "codex", ExpectedProcess: "codex", PromptInjectionMode: PromptInjectionArgv, PreflightTrust: PreflightTrustCodex, ReadySignal: ReadySignalCodexComposerPrompt},
	{ID: "codex-plan", Provider: ProviderCodex, Label: "Codex (plan mode)", Description: "Codex CLI in read-only planning mode.", DetectCmd: "codex", ExpectedProcess: "codex", PromptInjectionMode: PromptInjectionArgv, PreflightTrust: PreflightTrustCodex, ReadySignal: ReadySignalCodexComposerPrompt},
	{ID: "plain-shell", Provider: ProviderShell, Label: "Shell", Description: "Plain interactive shell, no agent.", PromptInjectionMode: PromptInjectionStdinAfterStart},
	{ID: "prompt-capture", Provider: ProviderShell, Label: "Prompt capture", Description: "Echoes the prompt via cat for smoke tests.", PromptInjectionMode: PromptInjectionStdinAfterStart},
}

// ProfileList returns the selectable builtin agent profiles in display order.
func ProfileList() []ProfileInfo {
	out := make([]ProfileInfo, len(profileCatalog))
	copy(out, profileCatalog)
	for i := range out {
		out[i].Source = ProfileSourceBuiltin
		out[i].Launchable = true
		out[i].DetectAliases = append([]string(nil), out[i].DetectAliases...)
	}
	return out
}

type Profile struct {
	ID                  string              `json:"id"`
	Provider            Provider            `json:"provider"`
	Label               string              `json:"label,omitempty"`
	Description         string              `json:"description,omitempty"`
	Command             string              `json:"command"`
	Args                []string            `json:"args,omitempty"`
	Env                 map[string]string   `json:"env,omitempty"`
	DetectCmd           string              `json:"detectCmd,omitempty"`
	DetectAliases       []string            `json:"detectAliases,omitempty"`
	ExpectedProcess     string              `json:"expectedProcess,omitempty"`
	PromptInjectionMode PromptInjectionMode `json:"promptInjectionMode,omitempty"`
	DraftPromptFlag     string              `json:"draftPromptFlag,omitempty"`
	DraftPromptEnvVar   string              `json:"draftPromptEnvVar,omitempty"`
	PreflightTrust      PreflightTrust      `json:"preflightTrust,omitempty"`
	ReadySignal         ReadySignal         `json:"readySignal,omitempty"`
	HookProvider        string              `json:"hookProvider,omitempty"`
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
		profile.DetectAliases = append([]string(nil), req.Profile.DetectAliases...)
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
	profile := func(id string, command string, args []string, env map[string]string) Profile {
		info, ok := profileInfoByID(id)
		if !ok {
			panic("builtin profile missing catalog metadata: " + id)
		}
		return Profile{
			ID:                  info.ID,
			Provider:            info.Provider,
			Label:               info.Label,
			Description:         info.Description,
			Command:             command,
			Args:                append([]string(nil), args...),
			Env:                 env,
			DetectCmd:           info.DetectCmd,
			DetectAliases:       append([]string(nil), info.DetectAliases...),
			ExpectedProcess:     info.ExpectedProcess,
			PromptInjectionMode: info.PromptInjectionMode,
			DraftPromptFlag:     info.DraftPromptFlag,
			DraftPromptEnvVar:   info.DraftPromptEnvVar,
			PreflightTrust:      info.PreflightTrust,
			ReadySignal:         info.ReadySignal,
			HookProvider:        info.HookProvider,
		}
	}
	return map[string]Profile{
		"plain-shell":    profile("plain-shell", shell, nil, nil),
		"prompt-capture": profile("prompt-capture", "cat", nil, nil),
		"claude":         profile("claude", "claude", nil, agentEnv),
		"claude-plan":    profile("claude-plan", "claude", []string{"--permission-mode", "plan"}, agentEnv),
		"claude-openrouter": profile("claude-openrouter", "claude", []string{
			"--print",
			"--max-budget-usd", "0.05",
			"--allowedTools", "Bash(whisk question ask*)",
		}, mergeEnv(agentEnv, map[string]string{
			"ANTHROPIC_BASE_URL":             "https://openrouter.ai/api",
			"ANTHROPIC_AUTH_TOKEN":           os.Getenv("OPENROUTER_API_KEY"),
			"ANTHROPIC_API_KEY":              "",
			"ANTHROPIC_DEFAULT_OPUS_MODEL":   envOrDefault("ANTHROPIC_DEFAULT_OPUS_MODEL", "~anthropic/claude-haiku-latest"),
			"ANTHROPIC_DEFAULT_SONNET_MODEL": envOrDefault("ANTHROPIC_DEFAULT_SONNET_MODEL", "~anthropic/claude-haiku-latest"),
			"ANTHROPIC_DEFAULT_HAIKU_MODEL":  envOrDefault("ANTHROPIC_DEFAULT_HAIKU_MODEL", "~anthropic/claude-haiku-latest"),
			"CLAUDE_CODE_SUBAGENT_MODEL":     envOrDefault("CLAUDE_CODE_SUBAGENT_MODEL", "~anthropic/claude-haiku-latest"),
		})),
		"codex":      profile("codex", "codex", nil, agentEnv),
		"codex-plan": profile("codex-plan", "codex", []string{"--sandbox", "read-only"}, agentEnv),
	}
}

func IsBuiltinProfileID(id string) bool {
	_, ok := profileInfoByID(id)
	return ok
}

func profileInfoByID(id string) (ProfileInfo, bool) {
	for _, profile := range profileCatalog {
		if profile.ID == id {
			profile.DetectAliases = append([]string(nil), profile.DetectAliases...)
			profile.Source = ProfileSourceBuiltin
			profile.Launchable = true
			return profile, true
		}
	}
	return ProfileInfo{}, false
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
