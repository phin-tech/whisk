package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/appsettings"
)

type Manifest struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Version   string     `json:"version"`
	Resolvers []Resolver `json:"resolvers"`
	UI        UI         `json:"ui"`
}

type Resolver struct {
	Provider string   `json:"provider"`
	Kinds    []string `json:"kinds"`
	Command  string   `json:"command"`
}

type UI struct {
	ProjectAttachments []ProjectAttachmentAction `json:"projectAttachments"`
}

type ProjectAttachmentAction struct {
	ID       string                    `json:"id"`
	Label    string                    `json:"label"`
	Provider string                    `json:"provider"`
	Kind     string                    `json:"kind"`
	Command  string                    `json:"command"`
	Fields   []app.PluginTemplateField `json:"fields"`
}

type CommandResolver struct {
	PluginID string
	Dir      string
	Command  string
}

type SettingsStore interface {
	Load(context.Context) (appsettings.Settings, error)
	Save(context.Context, appsettings.Settings) (appsettings.Settings, error)
}

type Manager struct {
	mu        sync.RWMutex
	envDirs   []string
	settings  SettingsStore
	statuses  []app.PluginStatus
	manifests map[string]Manifest
	dirs      map[string]string
	trusted   map[string]bool
	resolvers map[string]app.ProjectContextResolver
	installer *Installer
}

func NewManager(envDirs []string, settings SettingsStore) (*Manager, error) {
	manager := &Manager{envDirs: envDirs, settings: settings}
	if _, err := manager.RescanPlugins(context.Background()); err != nil {
		return nil, err
	}
	return manager, nil
}

// buildInstaller resolves the configured registries (settings list, else
// WHISK_PLUGIN_REGISTRY, else a built-in default) into an Installer that
// installs into the shared config plugin directory. It returns nil if the
// environment can't be resolved; the registry endpoints then report that
// installation is unavailable rather than crashing the daemon.
func buildInstaller(settings appsettings.Settings) *Installer {
	configDir, err := configPluginDir()
	if err != nil {
		return nil
	}
	registries, err := resolveRegistries(settings.PluginRegistries, os.Getenv("WHISK_PLUGIN_REGISTRY"), registryCacheDir())
	if err != nil {
		return nil
	}
	lockPath := filepath.Join(filepath.Dir(configDir), "plugins.lock.json")
	return NewInstaller(registries, configDir, lockPath)
}

// registryCacheDir is where git-transport registries keep their shallow clones.
func registryCacheDir() string {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "whisk", "registries")
		}
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "whisk", "registries")
}

func (m *Manager) ListPlugins(context.Context) ([]app.PluginStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]app.PluginStatus(nil), m.statuses...), nil
}

func (m *Manager) RescanPlugins(ctx context.Context) ([]app.PluginStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.rescanLocked(ctx); err != nil {
		return nil, err
	}
	return append([]app.PluginStatus(nil), m.statuses...), nil
}

func (m *Manager) TrustPlugin(ctx context.Context, id string) (app.PluginStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.manifests[id]; !ok {
		return app.PluginStatus{}, fmt.Errorf("plugin %s not found", id)
	}
	if m.settings == nil {
		return app.PluginStatus{}, fmt.Errorf("plugin trust store is not configured")
	}
	settings, err := m.loadSettings(ctx)
	if err != nil {
		return app.PluginStatus{}, err
	}
	settings.TrustedPlugins = append(settings.TrustedPlugins, id)
	if _, err := m.settings.Save(ctx, settings); err != nil {
		return app.PluginStatus{}, err
	}
	if err := m.rescanLocked(ctx); err != nil {
		return app.PluginStatus{}, err
	}
	return m.statusFor(id)
}

func (m *Manager) UntrustPlugin(ctx context.Context, id string) (app.PluginStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.settings == nil {
		return app.PluginStatus{}, fmt.Errorf("plugin trust store is not configured")
	}
	settings, err := m.loadSettings(ctx)
	if err != nil {
		return app.PluginStatus{}, err
	}
	next := settings.TrustedPlugins[:0]
	for _, trusted := range settings.TrustedPlugins {
		if trusted != id {
			next = append(next, trusted)
		}
	}
	settings.TrustedPlugins = next
	if _, err := m.settings.Save(ctx, settings); err != nil {
		return app.PluginStatus{}, err
	}
	if err := m.rescanLocked(ctx); err != nil {
		return app.PluginStatus{}, err
	}
	return m.statusFor(id)
}

func (m *Manager) RunProjectAttachmentTemplate(ctx context.Context, req app.RunPluginProjectAttachmentTemplateRequest) (app.AddProjectAttachmentRequest, error) {
	m.mu.RLock()
	manifest, ok := m.manifests[req.PluginID]
	dir := m.dirs[req.PluginID]
	trusted := m.trusted[req.PluginID]
	m.mu.RUnlock()
	if !ok {
		return app.AddProjectAttachmentRequest{}, fmt.Errorf("plugin %s not found", req.PluginID)
	}
	if !trusted {
		return app.AddProjectAttachmentRequest{}, fmt.Errorf("plugin %s is not trusted", req.PluginID)
	}
	var action ProjectAttachmentAction
	for _, candidate := range manifest.UI.ProjectAttachments {
		if candidate.ID == req.TemplateID {
			action = candidate
			break
		}
	}
	if action.ID == "" {
		return app.AddProjectAttachmentRequest{}, fmt.Errorf("project attachment template %s not found", req.TemplateID)
	}
	for _, field := range action.Fields {
		if field.Required && strings.TrimSpace(req.Values[field.ID]) == "" {
			return app.AddProjectAttachmentRequest{}, fmt.Errorf("field %s required", field.ID)
		}
	}
	input, err := json.Marshal(struct {
		PluginID   string            `json:"pluginId"`
		TemplateID string            `json:"templateId"`
		ProjectID  string            `json:"projectId"`
		Values     map[string]string `json:"values,omitempty"`
	}{PluginID: req.PluginID, TemplateID: req.TemplateID, ProjectID: req.ProjectID, Values: req.Values})
	if err != nil {
		return app.AddProjectAttachmentRequest{}, err
	}
	cmd := shellCommand(ctx, action.Command)
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return app.AddProjectAttachmentRequest{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return app.AddProjectAttachmentRequest{}, err
	}
	var out app.PluginProjectAttachmentOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return app.AddProjectAttachmentRequest{}, err
	}
	if out.Kind == "" {
		out.Kind = action.Kind
	}
	if out.Provider == "" {
		out.Provider = action.Provider
	}
	return app.AddProjectAttachmentRequest{
		ProjectID:        req.ProjectID,
		Kind:             out.Kind,
		Scope:            out.Scope,
		Title:            out.Title,
		Path:             out.Path,
		URL:              out.URL,
		Note:             out.Note,
		Provider:         out.Provider,
		Target:           out.Target,
		IncludeInContext: out.IncludeInContext,
		Meta:             out.Meta,
	}, nil
}

func (m *Manager) ResolveProjectAttachmentProvider(provider string) app.ProjectContextResolver {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resolvers[provider]
}

func (m *Manager) loadSettings(ctx context.Context) (appsettings.Settings, error) {
	if m.settings == nil {
		return appsettings.Default(), nil
	}
	return m.settings.Load(ctx)
}

func (m *Manager) rescanLocked(ctx context.Context) error {
	settings, err := m.loadSettings(ctx)
	if err != nil {
		return err
	}
	trusted := map[string]bool{}
	for _, id := range settings.TrustedPlugins {
		trusted[id] = true
	}
	discovered, err := discoverPlugins(m.envDirs)
	if err != nil {
		return err
	}
	statuses := make([]app.PluginStatus, 0, len(discovered))
	manifests := map[string]Manifest{}
	manifestDirs := map[string]string{}
	resolvers := map[string]app.ProjectContextResolver{}
	for _, found := range discovered {
		manifest, err := ReadManifest(found.Dir)
		if err != nil {
			statuses = append(statuses, app.PluginStatus{
				ID:           qualifiedID(found.Registry, filepath.Base(found.Dir)),
				Registry:     found.Registry,
				Dir:          found.Dir,
				ManifestPath: filepath.Join(found.Dir, "plugin.json"),
				Valid:        false,
				Error:        err.Error(),
			})
			continue
		}
		id := qualifiedID(found.Registry, manifest.ID)
		status := statusFromManifest(found.Registry, found.Dir, manifest, trusted[id])
		statuses = append(statuses, status)
		manifests[id] = manifest
		manifestDirs[id] = found.Dir
		if trusted[id] {
			for _, resolver := range manifest.Resolvers {
				provider := strings.TrimSpace(resolver.Provider)
				if provider == "" || strings.TrimSpace(resolver.Command) == "" {
					continue
				}
				resolvers[provider] = CommandResolver{PluginID: id, Dir: found.Dir, Command: resolver.Command}
			}
		}
	}
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].ID < statuses[j].ID })
	m.statuses = statuses
	m.manifests = manifests
	m.dirs = manifestDirs
	m.trusted = trusted
	m.resolvers = resolvers
	m.installer = buildInstaller(settings)
	return nil
}

// qualifiedID namespaces an installed plugin's id by its registry. Flat plugins
// (manually placed or from WHISK_PLUGIN_DIRS) have no registry and keep their
// bare id for backward compatibility.
func qualifiedID(registry, id string) string {
	if registry == "" {
		return id
	}
	return registry + "/" + id
}

func (m *Manager) statusFor(id string) (app.PluginStatus, error) {
	for _, status := range m.statuses {
		if status.ID == id {
			return status, nil
		}
	}
	return app.PluginStatus{}, fmt.Errorf("plugin %s not found", id)
}

func statusFromManifest(registry, dir string, manifest Manifest, trusted bool) app.PluginStatus {
	resolvers := make([]app.PluginResolver, 0, len(manifest.Resolvers))
	for _, resolver := range manifest.Resolvers {
		if strings.TrimSpace(resolver.Provider) == "" {
			continue
		}
		resolvers = append(resolvers, app.PluginResolver{Provider: strings.TrimSpace(resolver.Provider), Kinds: resolver.Kinds})
	}
	templates := make([]app.ProjectAttachmentTemplate, 0, len(manifest.UI.ProjectAttachments))
	for _, action := range manifest.UI.ProjectAttachments {
		if strings.TrimSpace(action.ID) == "" {
			continue
		}
		templates = append(templates, app.ProjectAttachmentTemplate{
			ID:       action.ID,
			Label:    action.Label,
			Provider: action.Provider,
			Kind:     action.Kind,
			Fields:   action.Fields,
		})
	}
	return app.PluginStatus{
		ID:                         qualifiedID(registry, manifest.ID),
		Registry:                   registry,
		Name:                       manifest.Name,
		Version:                    manifest.Version,
		Dir:                        dir,
		ManifestPath:               filepath.Join(dir, "plugin.json"),
		Trusted:                    trusted,
		Valid:                      true,
		Resolvers:                  resolvers,
		ProjectAttachmentTemplates: templates,
	}
}

// discoveredPlugin is a plugin directory found on disk, with the registry
// namespace it was installed under (empty for flat/manual plugins).
type discoveredPlugin struct {
	Dir      string
	Registry string
}

// discoverPlugins finds plugin directories. WHISK_PLUGIN_DIRS entries are flat
// plugins. Inside the config plugin dir, a child holding a plugin.json is a flat
// plugin (manually placed); a child without one is treated as a registry
// namespace whose own children are namespaced plugins.
func discoverPlugins(envDirs []string) ([]discoveredPlugin, error) {
	seen := map[string]bool{}
	var out []discoveredPlugin
	add := func(dir, registry string) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			return
		}
		clean := filepath.Clean(dir)
		if seen[clean] {
			return
		}
		seen[clean] = true
		out = append(out, discoveredPlugin{Dir: clean, Registry: registry})
	}
	for _, dir := range envDirs {
		add(dir, "")
	}
	configDir, err := configPluginDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(configDir)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		child := filepath.Join(configDir, entry.Name())
		if hasManifest(child) {
			add(child, "") // flat, manually-placed plugin
			continue
		}
		// Registry namespace: scan one level deeper for installed plugins.
		grandchildren, err := os.ReadDir(child)
		if err != nil {
			continue
		}
		for _, grandchild := range grandchildren {
			if grandchild.IsDir() && hasManifest(filepath.Join(child, grandchild.Name())) {
				add(filepath.Join(child, grandchild.Name()), entry.Name())
			}
		}
	}
	return out, nil
}

func hasManifest(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "plugin.json"))
	return err == nil
}

func configPluginDir() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "whisk", "plugins"), nil
}

func ScanTrustedResolvers(dirs []string, trustedIDs map[string]bool) (map[string]app.ProjectContextResolver, error) {
	out := map[string]app.ProjectContextResolver{}
	for _, dir := range dirs {
		manifest, err := ReadManifest(dir)
		if err != nil {
			return nil, err
		}
		if !trustedIDs[manifest.ID] {
			continue
		}
		for _, resolver := range manifest.Resolvers {
			provider := strings.TrimSpace(resolver.Provider)
			if provider == "" || strings.TrimSpace(resolver.Command) == "" {
				continue
			}
			out[provider] = CommandResolver{PluginID: manifest.ID, Dir: dir, Command: resolver.Command}
		}
	}
	return out, nil
}

func ReadManifest(dir string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "plugin.json"))
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	if strings.TrimSpace(manifest.ID) == "" {
		return Manifest{}, fmt.Errorf("plugin id required")
	}
	return manifest, nil
}

func (r CommandResolver) ResolveProjectAttachment(ctx context.Context, req app.ResolveProjectAttachmentRequest) (app.ResolvedProjectAttachment, error) {
	input, err := json.Marshal(req)
	if err != nil {
		return app.ResolvedProjectAttachment{}, err
	}
	cmd := shellCommand(ctx, r.Command)
	cmd.Dir = r.Dir
	cmd.Stdin = bytes.NewReader(input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return app.ResolvedProjectAttachment{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return app.ResolvedProjectAttachment{}, err
	}
	var out app.ResolvedProjectAttachment
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return app.ResolvedProjectAttachment{}, err
	}
	return out, nil
}

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/c", command)
	}
	return exec.CommandContext(ctx, "sh", "-lc", command)
}
