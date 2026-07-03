package agenthooks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	ProviderClaude = "claude"
	ProviderCodex  = "codex"

	StatusMissing     = "missing"
	StatusCurrent     = "current"
	StatusOutdated    = "outdated"
	StatusModified    = "modified"
	StatusUntrusted   = "untrusted"
	StatusUnavailable = "unavailable"

	StateInstalled    = "installed"
	StateNotInstalled = "not_installed"
	StatePartial      = "partial"
	StateError        = "error"

	SchemaVersion    = 1
	InstallerVersion = "1.1.0"
)

type HookMode string

const (
	HookModePassive  HookMode = "passive"
	HookModeDecision HookMode = "decision"
)

type EventSpec struct {
	Event         string
	Matcher       string
	Mode          HookMode
	StatusMessage string
}

func ProviderEventSpecs(provider string) []EventSpec {
	switch provider {
	case ProviderClaude:
		return []EventSpec{
			{Event: "PreToolUse", Mode: HookModeDecision, StatusMessage: "Whisk checking tool use"},
			{Event: "PermissionRequest", Mode: HookModeDecision, StatusMessage: "Whisk checking permission request"},
			{Event: "Elicitation", Mode: HookModeDecision, StatusMessage: "Whisk recording question"},
			{Event: "PostToolUse", Mode: HookModePassive, StatusMessage: "Whisk recording tool result"},
			{Event: "Notification", Matcher: "permission_prompt|elicitation_dialog|elicitation_complete|elicitation_response", Mode: HookModePassive, StatusMessage: "Whisk recording notification"},
			{Event: "ElicitationResult", Mode: HookModePassive, StatusMessage: "Whisk recording question result"},
			{Event: "PostToolUseFailure", Mode: HookModePassive, StatusMessage: "Whisk recording tool failure"},
			{Event: "Stop", Mode: HookModePassive, StatusMessage: "Whisk recording stop"},
			{Event: "StopFailure", Mode: HookModePassive, StatusMessage: "Whisk recording stop failure"},
			{Event: "SessionEnd", Mode: HookModePassive, StatusMessage: "Whisk recording session end"},
			{Event: "PreCompact", Mode: HookModePassive, StatusMessage: "Whisk recording compaction"},
			{Event: "PostCompact", Mode: HookModePassive, StatusMessage: "Whisk recording compaction"},
		}
	case ProviderCodex:
		return []EventSpec{
			{Event: "PreToolUse", Mode: HookModeDecision, StatusMessage: "Whisk checking tool use"},
			{Event: "PermissionRequest", Mode: HookModeDecision, StatusMessage: "Whisk checking permission request"},
			{Event: "PostToolUse", Mode: HookModePassive, StatusMessage: "Whisk recording tool result"},
			{Event: "SessionStart", Mode: HookModePassive, StatusMessage: "Whisk recording session start"},
			{Event: "UserPromptSubmit", Mode: HookModePassive, StatusMessage: "Whisk recording prompt"},
			{Event: "PreCompact", Mode: HookModePassive, StatusMessage: "Whisk recording compaction"},
			{Event: "PostCompact", Mode: HookModePassive, StatusMessage: "Whisk recording compaction"},
			{Event: "SubagentStart", Mode: HookModePassive, StatusMessage: "Whisk recording subagent start"},
			{Event: "SubagentStop", Mode: HookModePassive, StatusMessage: "Whisk recording subagent stop"},
			{Event: "Stop", Mode: HookModePassive, StatusMessage: "Whisk recording stop"},
		}
	default:
		return nil
	}
}

func providerEvents(provider string) []string {
	specs := ProviderEventSpecs(provider)
	events := make([]string, 0, len(specs))
	for _, spec := range specs {
		events = append(events, spec.Event)
	}
	return events
}

type Paths struct {
	ConfigRoot         string
	HelperSourcePath   string
	ClaudeSettingsPath string
	CodexHooksPath     string
}

type Installer struct {
	paths Paths
}

type Integration struct {
	Provider         string `json:"provider"`
	State            string `json:"state"`
	Status           string `json:"status"`
	InstalledVersion string `json:"installedVersion,omitempty"`
	LatestVersion    string `json:"latestVersion"`
	HelperPath       string `json:"helperPath"`
	ConfigPath       string `json:"configPath"`
	ManifestPath     string `json:"manifestPath"`
	Detail           string `json:"detail,omitempty"`
}

type Manifest struct {
	SchemaVersion    int                         `json:"schemaVersion"`
	InstallerVersion string                      `json:"installerVersion"`
	HelperPath       string                      `json:"helperPath"`
	HelperHash       string                      `json:"helperHash"`
	Providers        map[string]ProviderManifest `json:"providers"`
	UpdatedAt        string                      `json:"updatedAt"`
}

type ProviderManifest struct {
	ConfigPath     string   `json:"configPath"`
	Command        string   `json:"command"`
	CommandHash    string   `json:"commandHash"`
	Events         []string `json:"events"`
	TrustVerified  bool     `json:"trustVerified,omitempty"`
	TrustDetail    string   `json:"trustDetail,omitempty"`
	InstalledAt    string   `json:"installedAt"`
	LastVerifiedAt string   `json:"lastVerifiedAt,omitempty"`
}

func DefaultPaths(helperSourcePath string) (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	if helperSourcePath == "" {
		helperSourcePath, err = os.Executable()
		if err != nil {
			return Paths{}, err
		}
	}
	return Paths{
		ConfigRoot:         filepath.Join(home, ".config", "whisk"),
		HelperSourcePath:   helperSourcePath,
		ClaudeSettingsPath: filepath.Join(home, ".claude", "settings.json"),
		CodexHooksPath:     filepath.Join(home, ".codex", "hooks.json"),
	}, nil
}

func NewInstaller(paths Paths) *Installer {
	return &Installer{paths: normalizePaths(paths)}
}

func (i *Installer) List(ctx context.Context) ([]Integration, error) {
	providers := []string{ProviderClaude, ProviderCodex}
	out := make([]Integration, 0, len(providers))
	for _, provider := range providers {
		status, err := i.Check(ctx, provider)
		if err != nil {
			return nil, err
		}
		out = append(out, status)
	}
	return out, nil
}

func (i *Installer) Check(_ context.Context, provider string) (Integration, error) {
	provider = strings.TrimSpace(provider)
	if err := validateProvider(provider); err != nil {
		return Integration{}, err
	}
	base := i.baseIntegration(provider)
	manifest, manifestErr := i.readManifest()
	if manifestErr != nil {
		base.Status = StatusUnavailable
		base.Detail = manifestErr.Error()
		return finalizeIntegration(base), nil
	}
	providerManifest, hasManifest := manifest.Providers[provider]
	if hasManifest {
		base.InstalledVersion = manifest.InstallerVersion
	}
	helperHash, helperErr := fileSHA256(i.helperPath())
	if helperErr != nil && !os.IsNotExist(helperErr) {
		base.Status = StatusUnavailable
		base.Detail = helperErr.Error()
		return finalizeIntegration(base), nil
	}
	configCommand, configComplete, configErr := i.findProviderCommand(provider)
	if configErr != nil {
		base.Status = StatusUnavailable
		base.Detail = configErr.Error()
		return finalizeIntegration(base), nil
	}
	if !hasManifest && configCommand == "" {
		base.Status = StatusMissing
		return finalizeIntegration(base), nil
	}
	expectedCommand := i.hookCommand(provider)
	if configCommand == "" {
		base.Status = StatusModified
		base.Detail = "managed manifest exists but provider hook is missing"
		return finalizeIntegration(base), nil
	}
	if configCommand != expectedCommand || !configComplete {
		base.Status = StatusModified
		base.Detail = "provider hook command differs from manifest"
		return finalizeIntegration(base), nil
	}
	if !hasManifest {
		base.Status = StatusModified
		base.Detail = "provider hook exists but Whisk manifest is missing"
		return finalizeIntegration(base), nil
	}
	if manifest.SchemaVersion != SchemaVersion || manifest.InstallerVersion != InstallerVersion {
		base.Status = StatusOutdated
		base.Detail = "managed manifest version is outdated"
		return finalizeIntegration(base), nil
	}
	if helperErr != nil {
		base.Status = StatusModified
		base.Detail = "helper binary is missing"
		return finalizeIntegration(base), nil
	}
	if manifest.HelperHash != helperHash {
		base.Status = StatusModified
		base.Detail = "helper binary hash differs from manifest"
		return finalizeIntegration(base), nil
	}
	if providerManifest.CommandHash != hashText(expectedCommand) ||
		providerManifest.ConfigPath != i.configPath(provider) ||
		!sameStrings(providerManifest.Events, providerEvents(provider)) {
		base.Status = StatusModified
		base.Detail = "provider manifest does not match expected hook config"
		return finalizeIntegration(base), nil
	}
	if provider == ProviderCodex && !providerManifest.TrustVerified {
		base.Status = StatusUntrusted
		if providerManifest.TrustDetail != "" {
			base.Detail = providerManifest.TrustDetail
		} else {
			base.Detail = "Codex hook trust has not been verified"
		}
		return finalizeIntegration(base), nil
	}
	base.Status = StatusCurrent
	return finalizeIntegration(base), nil
}

func (i *Installer) Install(ctx context.Context, provider string) (Integration, error) {
	provider = strings.TrimSpace(provider)
	if err := validateProvider(provider); err != nil {
		return Integration{}, err
	}
	if err := i.installHelper(); err != nil {
		return Integration{}, err
	}
	if err := i.upsertProviderCommand(provider, i.hookCommand(provider)); err != nil {
		return Integration{}, err
	}
	helperHash, err := fileSHA256(i.helperPath())
	if err != nil {
		return Integration{}, err
	}
	manifest, err := i.readManifest()
	if err != nil {
		return Integration{}, err
	}
	if manifest.Providers == nil {
		manifest.Providers = map[string]ProviderManifest{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	providerManifest := ProviderManifest{
		ConfigPath:  i.configPath(provider),
		Command:     i.hookCommand(provider),
		CommandHash: hashText(i.hookCommand(provider)),
		Events:      providerEvents(provider),
		InstalledAt: now,
	}
	if existing, ok := manifest.Providers[provider]; ok && existing.InstalledAt != "" {
		providerManifest.InstalledAt = existing.InstalledAt
	}
	if provider == ProviderCodex {
		providerManifest.TrustVerified = false
		providerManifest.TrustDetail = "Codex hook installed; trust verification is required before it is current"
	}
	manifest.SchemaVersion = SchemaVersion
	manifest.InstallerVersion = InstallerVersion
	manifest.HelperPath = i.helperPath()
	manifest.HelperHash = helperHash
	manifest.Providers[provider] = providerManifest
	manifest.UpdatedAt = now
	if err := i.writeManifest(manifest); err != nil {
		return Integration{}, err
	}
	return i.Check(ctx, provider)
}

func (i *Installer) Remove(ctx context.Context, provider string) (Integration, error) {
	provider = strings.TrimSpace(provider)
	if err := validateProvider(provider); err != nil {
		return Integration{}, err
	}
	if err := i.removeProviderCommand(provider); err != nil {
		return Integration{}, err
	}
	if _, err := os.Stat(i.manifestPath()); os.IsNotExist(err) {
		return i.Check(ctx, provider)
	} else if err != nil {
		return Integration{}, err
	}
	manifest, err := i.readManifest()
	if err != nil {
		return Integration{}, err
	}
	delete(manifest.Providers, provider)
	manifest.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := i.writeManifest(manifest); err != nil {
		return Integration{}, err
	}
	return i.Check(ctx, provider)
}

func (i *Installer) baseIntegration(provider string) Integration {
	return Integration{
		Provider:      provider,
		LatestVersion: InstallerVersion,
		HelperPath:    i.helperPath(),
		ConfigPath:    i.configPath(provider),
		ManifestPath:  i.manifestPath(),
	}
}

func finalizeIntegration(integration Integration) Integration {
	switch integration.Status {
	case StatusCurrent:
		integration.State = StateInstalled
	case StatusMissing:
		integration.State = StateNotInstalled
	case StatusUnavailable:
		integration.State = StateError
	case StatusOutdated, StatusModified, StatusUntrusted:
		integration.State = StatePartial
	default:
		integration.State = StateNotInstalled
	}
	return integration
}

func (i *Installer) installHelper() error {
	if strings.TrimSpace(i.paths.HelperSourcePath) == "" {
		return fmt.Errorf("helper source path required")
	}
	if err := os.MkdirAll(filepath.Dir(i.helperPath()), 0o755); err != nil {
		return err
	}
	src, err := os.Open(i.paths.HelperSourcePath)
	if err != nil {
		return err
	}
	defer src.Close()
	tmp, err := os.CreateTemp(filepath.Dir(i.helperPath()), ".whisk-hook-helper-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := io.Copy(tmp, src); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	return os.Rename(tmpName, i.helperPath())
}

func (i *Installer) findProviderCommand(provider string) (string, bool, error) {
	cfg, err := readHookConfig(i.configPath(provider))
	if err != nil {
		return "", false, err
	}
	expected := i.hookCommand(provider)
	foundAny := ""
	for _, event := range providerEvents(provider) {
		foundEvent := false
		for _, entry := range cfg.Hooks[event] {
			for _, hook := range entry.Hooks {
				if hook.Command == expected {
					foundAny = hook.Command
					foundEvent = true
				}
			}
		}
		if !foundEvent {
			return foundAny, false, nil
		}
	}
	return foundAny, foundAny != "", nil
}

func (i *Installer) upsertProviderCommand(provider string, command string) error {
	cfg, err := readHookConfig(i.configPath(provider))
	if err != nil {
		return err
	}
	if cfg.Hooks == nil {
		cfg.Hooks = map[string][]HookMatcher{}
	}
	for _, spec := range ProviderEventSpecs(provider) {
		cfg.Hooks[spec.Event] = upsertManagedHook(cfg.Hooks[spec.Event], command, spec)
	}
	return writeHookConfig(i.configPath(provider), cfg)
}

func (i *Installer) removeProviderCommand(provider string) error {
	configPath := i.configPath(provider)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	cfg, err := readHookConfig(configPath)
	if err != nil {
		return err
	}
	for _, event := range providerEvents(provider) {
		cfg.Hooks[event] = removeManagedHook(cfg.Hooks[event], i.hookCommand(provider))
	}
	return writeHookConfig(configPath, cfg)
}

func (i *Installer) hookCommand(provider string) string {
	return shellQuote(i.helperPath()) + " agent-bridge hook -provider " + provider
}

func (i *Installer) helperPath() string {
	return filepath.Join(i.paths.ConfigRoot, "bin", "whisk-hook-helper")
}

func (i *Installer) manifestPath() string {
	return filepath.Join(i.paths.ConfigRoot, "agent-hooks", "manifest.json")
}

func (i *Installer) configPath(provider string) string {
	if provider == ProviderClaude {
		return i.paths.ClaudeSettingsPath
	}
	return i.paths.CodexHooksPath
}

func (i *Installer) readManifest() (Manifest, error) {
	raw, err := os.ReadFile(i.manifestPath())
	if os.IsNotExist(err) {
		return Manifest{
			SchemaVersion:    SchemaVersion,
			InstallerVersion: InstallerVersion,
			Providers:        map[string]ProviderManifest{},
		}, nil
	}
	if err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{Providers: map[string]ProviderManifest{}}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, err
	}
	if manifest.Providers == nil {
		manifest.Providers = map[string]ProviderManifest{}
	}
	return manifest, nil
}

func (i *Installer) writeManifest(manifest Manifest) error {
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(i.manifestPath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(i.manifestPath(), append(raw, '\n'), 0o600)
}

type HookConfig struct {
	Hooks map[string][]HookMatcher   `json:"hooks,omitempty"`
	Rest  map[string]json.RawMessage `json:"-"`
}

func (cfg *HookConfig) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	cfg.Rest = cloneRawFields(fields)
	if hooksRaw, ok := fields["hooks"]; ok {
		if err := json.Unmarshal(hooksRaw, &cfg.Hooks); err != nil {
			return err
		}
		delete(cfg.Rest, "hooks")
	}
	if cfg.Hooks == nil {
		cfg.Hooks = map[string][]HookMatcher{}
	}
	return nil
}

func (cfg HookConfig) MarshalJSON() ([]byte, error) {
	fields := cloneRawFields(cfg.Rest)
	if len(cfg.Hooks) > 0 {
		hooksRaw, err := json.Marshal(cfg.Hooks)
		if err != nil {
			return nil, err
		}
		fields["hooks"] = hooksRaw
	}
	return json.Marshal(fields)
}

type HookMatcher struct {
	Matcher string                     `json:"matcher,omitempty"`
	Hooks   []CommandHook              `json:"hooks"`
	Rest    map[string]json.RawMessage `json:"-"`
}

func (matcher *HookMatcher) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	matcher.Rest = cloneRawFields(fields)
	if matcherRaw, ok := fields["matcher"]; ok {
		if err := json.Unmarshal(matcherRaw, &matcher.Matcher); err != nil {
			return err
		}
		delete(matcher.Rest, "matcher")
	}
	if hooksRaw, ok := fields["hooks"]; ok {
		if err := json.Unmarshal(hooksRaw, &matcher.Hooks); err != nil {
			return err
		}
		delete(matcher.Rest, "hooks")
	}
	return nil
}

func (matcher HookMatcher) MarshalJSON() ([]byte, error) {
	fields := cloneRawFields(matcher.Rest)
	if matcher.Matcher != "" {
		matcherRaw, err := json.Marshal(matcher.Matcher)
		if err != nil {
			return nil, err
		}
		fields["matcher"] = matcherRaw
	}
	hooksRaw, err := json.Marshal(matcher.Hooks)
	if err != nil {
		return nil, err
	}
	fields["hooks"] = hooksRaw
	return json.Marshal(fields)
}

type CommandHook struct {
	Type          string                     `json:"type"`
	Command       string                     `json:"command"`
	Timeout       int                        `json:"timeout,omitempty"`
	StatusMessage string                     `json:"statusMessage,omitempty"`
	Rest          map[string]json.RawMessage `json:"-"`
}

func (hook *CommandHook) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	hook.Rest = cloneRawFields(fields)
	if typeRaw, ok := fields["type"]; ok {
		if err := json.Unmarshal(typeRaw, &hook.Type); err != nil {
			return err
		}
		delete(hook.Rest, "type")
	}
	if commandRaw, ok := fields["command"]; ok {
		if err := json.Unmarshal(commandRaw, &hook.Command); err != nil {
			return err
		}
		delete(hook.Rest, "command")
	}
	if timeoutRaw, ok := fields["timeout"]; ok {
		if err := json.Unmarshal(timeoutRaw, &hook.Timeout); err != nil {
			return err
		}
		delete(hook.Rest, "timeout")
	}
	if statusRaw, ok := fields["statusMessage"]; ok {
		if err := json.Unmarshal(statusRaw, &hook.StatusMessage); err != nil {
			return err
		}
		delete(hook.Rest, "statusMessage")
	}
	return nil
}

func (hook CommandHook) MarshalJSON() ([]byte, error) {
	fields := cloneRawFields(hook.Rest)
	typeRaw, err := json.Marshal(hook.Type)
	if err != nil {
		return nil, err
	}
	fields["type"] = typeRaw
	commandRaw, err := json.Marshal(hook.Command)
	if err != nil {
		return nil, err
	}
	fields["command"] = commandRaw
	if hook.Timeout != 0 {
		timeoutRaw, err := json.Marshal(hook.Timeout)
		if err != nil {
			return nil, err
		}
		fields["timeout"] = timeoutRaw
	}
	if hook.StatusMessage != "" {
		statusRaw, err := json.Marshal(hook.StatusMessage)
		if err != nil {
			return nil, err
		}
		fields["statusMessage"] = statusRaw
	}
	return json.Marshal(fields)
}

func readHookConfig(path string) (HookConfig, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return HookConfig{Hooks: map[string][]HookMatcher{}}, nil
	}
	if err != nil {
		return HookConfig{}, err
	}
	var cfg HookConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return HookConfig{}, err
	}
	if cfg.Hooks == nil {
		cfg.Hooks = map[string][]HookMatcher{}
	}
	return cfg, nil
}

func writeHookConfig(path string, cfg HookConfig) error {
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

func upsertManagedHook(entries []HookMatcher, command string, spec EventSpec) []HookMatcher {
	entries = removeManagedHook(entries, command)
	hook := CommandHook{
		Type:          "command",
		Command:       command,
		Timeout:       86400,
		StatusMessage: spec.StatusMessage,
	}
	return append(entries, HookMatcher{Matcher: spec.Matcher, Hooks: []CommandHook{hook}})
}

func removeManagedHook(entries []HookMatcher, command string) []HookMatcher {
	out := make([]HookMatcher, 0, len(entries))
	for _, entry := range entries {
		hooks := entry.Hooks[:0]
		for _, hook := range entry.Hooks {
			if hook.Command == command || (strings.Contains(hook.Command, "whisk-hook-helper") && strings.Contains(hook.Command, "agent-bridge hook")) {
				continue
			}
			hooks = append(hooks, hook)
		}
		entry.Hooks = hooks
		if len(entry.Hooks) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func normalizePaths(paths Paths) Paths {
	if paths.ConfigRoot == "" {
		if defaults, err := DefaultPaths(paths.HelperSourcePath); err == nil {
			paths.ConfigRoot = defaults.ConfigRoot
			if paths.HelperSourcePath == "" {
				paths.HelperSourcePath = defaults.HelperSourcePath
			}
			if paths.ClaudeSettingsPath == "" {
				paths.ClaudeSettingsPath = defaults.ClaudeSettingsPath
			}
			if paths.CodexHooksPath == "" {
				paths.CodexHooksPath = defaults.CodexHooksPath
			}
		}
	}
	return paths
}

func validateProvider(provider string) error {
	switch provider {
	case ProviderClaude, ProviderCodex:
		return nil
	default:
		return fmt.Errorf("unsupported agent hook provider %q", provider)
	}
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sameStrings(a []string, b []string) bool {
	return slices.Equal(a, b)
}

func cloneRawFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	if len(fields) == 0 {
		return map[string]json.RawMessage{}
	}
	cloned := make(map[string]json.RawMessage, len(fields))
	for key, value := range fields {
		cloned[key] = slices.Clone(value)
	}
	return cloned
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune("/._-:", r))
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
