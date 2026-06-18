package appsettings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	StartupViewSessions = "sessions"
	StartupViewKanban   = "kanban"
)

type Settings struct {
	StartupView string `json:"startupView"`
	// KeepDaemonAlive controls whether a daemon the app started is left running after the app
	// quits. It defaults to true so sessions persist across app restarts; setting it false makes
	// the app stop the daemon it started on quit.
	KeepDaemonAlive          bool  `json:"keepDaemonAlive"`
	HookLogEnabled           *bool `json:"hookLogEnabled,omitempty"`
	ClearHookLogAfterSession bool  `json:"clearHookLogAfterSession,omitempty"`
	// Keybindings holds user overrides for editable keyboard shortcuts, keyed by command id
	// (e.g. "open-preferences") with an accelerator value (e.g. "Cmd+Shift+P"). Commands absent
	// from the map use their built-in default; an empty map means all defaults.
	Keybindings    map[string]string `json:"keybindings,omitempty"`
	TrustedPlugins []string          `json:"trustedPlugins,omitempty"`
	// PluginRegistries configures the plugin registries the daemon can install
	// from. When empty, the daemon falls back to the WHISK_PLUGIN_REGISTRY env
	// var and then a built-in default. Each registry installs into its own
	// namespace, so two registries may advertise the same plugin id.
	PluginRegistries []PluginRegistryConfig `json:"pluginRegistries,omitempty"`
}

// PluginRegistryConfig describes one plugin registry source.
type PluginRegistryConfig struct {
	// Name is the registry namespace (e.g. "phin-tech", "acme"). Plugins from
	// this registry install under it, so names must be unique.
	Name string `json:"name"`
	// Source locates the registry: "owner/repo[@ref]", a GitHub/SSH URL, or a
	// local directory.
	Source string `json:"source"`
	// Transport forces a fetch mechanism: "tarball" (GitHub HTTP, default for
	// owner/repo), "git" (shell out to git, reusing local SSH/credential auth),
	// or "local" (a directory). Empty auto-detects from Source.
	Transport string `json:"transport,omitempty"`
	// TokenEnv names an environment variable holding a token for private
	// tarball/API access. The secret is never persisted in settings.
	TokenEnv string `json:"tokenEnv,omitempty"`
}

type Store struct {
	path string
}

func Default() Settings {
	enabled := true
	return Settings{
		StartupView:     StartupViewSessions,
		KeepDaemonAlive: true,
		HookLogEnabled:  &enabled,
	}
}

func Normalize(settings Settings) (Settings, error) {
	if settings.StartupView == "" {
		settings.StartupView = StartupViewSessions
	}
	if settings.HookLogEnabled == nil {
		enabled := true
		settings.HookLogEnabled = &enabled
	}
	settings.Keybindings = normalizeKeybindings(settings.Keybindings)
	settings.TrustedPlugins = normalizeTrustedPlugins(settings.TrustedPlugins)
	registries, err := normalizePluginRegistries(settings.PluginRegistries)
	if err != nil {
		return Settings{}, err
	}
	settings.PluginRegistries = registries
	switch settings.StartupView {
	case StartupViewSessions, StartupViewKanban:
		return settings, nil
	default:
		return Settings{}, fmt.Errorf("invalid startup view %q", settings.StartupView)
	}
}

// normalizeKeybindings drops entries with a blank command id or blank accelerator and trims
// surrounding whitespace. It returns nil when no usable overrides remain so the persisted JSON
// omits the field entirely (the map uses the "omitempty" tag).
func normalizeKeybindings(bindings map[string]string) map[string]string {
	if len(bindings) == 0 {
		return nil
	}
	cleaned := make(map[string]string, len(bindings))
	for id, accelerator := range bindings {
		id = strings.TrimSpace(id)
		accelerator = strings.TrimSpace(accelerator)
		if id == "" || accelerator == "" {
			continue
		}
		cleaned[id] = accelerator
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

// normalizePluginRegistries trims entries, drops blanks, and rejects duplicate
// names (the namespace must be unique). Order is preserved so the UI lists
// registries as configured.
func normalizePluginRegistries(registries []PluginRegistryConfig) ([]PluginRegistryConfig, error) {
	if len(registries) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	out := make([]PluginRegistryConfig, 0, len(registries))
	for _, registry := range registries {
		registry.Name = strings.TrimSpace(registry.Name)
		registry.Source = strings.TrimSpace(registry.Source)
		registry.Transport = strings.TrimSpace(registry.Transport)
		registry.TokenEnv = strings.TrimSpace(registry.TokenEnv)
		if registry.Name == "" || registry.Source == "" {
			continue
		}
		if strings.ContainsAny(registry.Name, "/\\") {
			return nil, fmt.Errorf("plugin registry name %q must not contain path separators", registry.Name)
		}
		if seen[registry.Name] {
			return nil, fmt.Errorf("duplicate plugin registry name %q", registry.Name)
		}
		switch registry.Transport {
		case "", "tarball", "git", "local":
		default:
			return nil, fmt.Errorf("plugin registry %q has unknown transport %q", registry.Name, registry.Transport)
		}
		seen[registry.Name] = true
		out = append(out, registry)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func normalizeTrustedPlugins(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func DefaultPath() (string, error) {
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return filepath.Join(configDir, "whisk.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "whisk.json"), nil
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func NewDefaultStore() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return NewStore(path), nil
}

func (s *Store) Load(context.Context) (Settings, error) {
	bytes, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return Default(), nil
	}
	if err != nil {
		return Settings{}, err
	}

	settings := Default()
	if err := json.Unmarshal(bytes, &settings); err != nil {
		return Settings{}, err
	}
	return Normalize(settings)
}

func (s *Store) Save(_ context.Context, settings Settings) (Settings, error) {
	normalized, err := Normalize(settings)
	if err != nil {
		return Settings{}, err
	}
	bytes, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return Settings{}, err
	}
	bytes = append(bytes, '\n')

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return Settings{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".whisk-*.json")
	if err != nil {
		return Settings{}, err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(bytes); err != nil {
		_ = tmp.Close()
		return Settings{}, err
	}
	if err := tmp.Close(); err != nil {
		return Settings{}, err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return Settings{}, err
	}
	return normalized, nil
}
