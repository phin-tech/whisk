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
	ManifestVersion int              `json:"manifestVersion,omitempty"`
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Version         string           `json:"version"`
	Resolvers       []Resolver       `json:"resolvers"`
	Events          []EventHandler   `json:"events,omitempty"`
	Hooks           []HookHandler    `json:"hooks,omitempty"`
	Gates           []WorkflowGate   `json:"gates,omitempty"`
	WorkflowActions []WorkflowAction `json:"workflowActions,omitempty"`
	Permissions     Permissions      `json:"permissions,omitempty"`
	UI              UI               `json:"ui"`
}

type Resolver struct {
	Provider string   `json:"provider"`
	Kinds    []string `json:"kinds"`
	Command  string   `json:"command"`
}

type UI struct {
	ProjectAttachments []ProjectAttachmentAction `json:"projectAttachments"`
	ReviewActions      []ReviewAction            `json:"reviewActions,omitempty"`
}

type ProjectAttachmentAction struct {
	ID       string                    `json:"id"`
	Label    string                    `json:"label"`
	Provider string                    `json:"provider"`
	Kind     string                    `json:"kind"`
	Command  string                    `json:"command"`
	Fields   []app.PluginTemplateField `json:"fields"`
}

// ReviewAction is the legacy in-the-wild shape that manifest v2 replaces with
// typed workflow gates/actions. It is parsed for visibility to daemon-side code,
// but is not executed by this adapter.
type ReviewAction struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	URLTemplate   string `json:"urlTemplate,omitempty"`
	SubmitCommand string `json:"submitCommand,omitempty"`
	Blocking      bool   `json:"blocking,omitempty"`
}

type EventHandler struct {
	ID             string         `json:"id"`
	Subjects       []string       `json:"subjects"`
	Filter         map[string]any `json:"filter,omitempty"`
	Command        string         `json:"command"`
	TimeoutMs      int            `json:"timeoutMs,omitempty"`
	OutputCapBytes int            `json:"outputCapBytes,omitempty"`
}

type HookHandler struct {
	ID             string `json:"id"`
	Point          string `json:"point"`
	Command        string `json:"command"`
	TimeoutMs      int    `json:"timeoutMs,omitempty"`
	OutputCapBytes int    `json:"outputCapBytes,omitempty"`
}

type WorkflowGate struct {
	ID             string        `json:"id"`
	Label          string        `json:"label"`
	AppliesTo      GateAppliesTo `json:"appliesTo,omitempty"`
	Open           GateOpen      `json:"open,omitempty"`
	Resolve        GateResolve   `json:"resolve,omitempty"`
	Blocking       bool          `json:"blocking,omitempty"`
	TimeoutMs      int           `json:"timeoutMs,omitempty"`
	OutputCapBytes int           `json:"outputCapBytes,omitempty"`
}

type GateAppliesTo struct {
	GateKinds []string `json:"gateKinds,omitempty"`
	Phases    []string `json:"phases,omitempty"`
}

type GateOpen struct {
	URLTemplate string `json:"urlTemplate,omitempty"`
}

type GateResolve struct {
	Command string `json:"command,omitempty"`
}

type WorkflowAction struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	Command        string   `json:"command"`
	Phases         []string `json:"phases,omitempty"`
	TimeoutMs      int      `json:"timeoutMs,omitempty"`
	OutputCapBytes int      `json:"outputCapBytes,omitempty"`
}

type Permissions struct {
	PTYOutput   bool     `json:"ptyOutput,omitempty"`
	EnvPrefixes []string `json:"envPrefixes,omitempty"`
	Network     []string `json:"network,omitempty"`
}

type CommandLimits struct {
	TimeoutMs      int
	OutputCapBytes int
}

const (
	defaultManifestVersion   = 1
	supportedManifestVersion = 2

	manifestEventDefaultTimeoutMs          = 10000
	manifestEventMaxTimeoutMs              = 30000
	manifestHookDefaultTimeoutMs           = 3000
	manifestHookMaxTimeoutMs               = 5000
	manifestWorkflowActionDefaultTimeoutMs = 10000
	manifestWorkflowActionMaxTimeoutMs     = 30000
	manifestGateDefaultTimeoutMs           = 10000
	manifestGateMaxTimeoutMs               = 30000

	manifestCommandDefaultOutputCapBytes = 1 << 20
	manifestCommandMaxOutputCapBytes     = 4 << 20
)

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
	return parseManifest(data)
}

func parseManifest(data []byte) (Manifest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Manifest{}, err
	}
	version, err := manifestVersion(raw)
	if err != nil {
		return Manifest{}, err
	}
	if version == supportedManifestVersion {
		if unknown := unknownManifestV2Fields(raw); len(unknown) > 0 {
			return Manifest{}, fmt.Errorf("manifestVersion 2 contains unsupported top-level field %q", unknown[0])
		}
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	manifest.ManifestVersion = version
	if err := validateManifest(&manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func manifestVersion(raw map[string]json.RawMessage) (int, error) {
	if _, ok := raw["manifestVersion"]; !ok {
		return defaultManifestVersion, nil
	}
	var version int
	if err := json.Unmarshal(raw["manifestVersion"], &version); err != nil {
		return 0, fmt.Errorf("manifestVersion must be an integer")
	}
	if version < defaultManifestVersion || version > supportedManifestVersion {
		return 0, fmt.Errorf("unsupported manifestVersion %d", version)
	}
	return version, nil
}

func unknownManifestV2Fields(raw map[string]json.RawMessage) []string {
	known := map[string]bool{
		"$schema":         true,
		"manifestVersion": true,
		"id":              true,
		"name":            true,
		"version":         true,
		"resolvers":       true,
		"events":          true,
		"hooks":           true,
		"gates":           true,
		"workflowActions": true,
		"permissions":     true,
		"ui":              true,
	}
	var unknown []string
	for key := range raw {
		if !known[key] {
			unknown = append(unknown, key)
		}
	}
	sort.Strings(unknown)
	return unknown
}

func validateManifest(manifest *Manifest) error {
	if strings.TrimSpace(manifest.ID) == "" {
		return fmt.Errorf("plugin id required")
	}
	if manifest.ManifestVersion == defaultManifestVersion {
		return nil
	}
	return validateManifestV2(manifest)
}

func validateManifestV2(manifest *Manifest) error {
	seenContributionIDs := map[string]string{}
	for i := range manifest.Events {
		event := &manifest.Events[i]
		id := strings.TrimSpace(event.ID)
		if id == "" {
			return fmt.Errorf("events[%d].id required", i)
		}
		if err := recordManifestContributionID(seenContributionIDs, id, "events"); err != nil {
			return err
		}
		if strings.TrimSpace(event.Command) == "" {
			return fmt.Errorf("events[%s].command required", id)
		}
		if len(event.Subjects) == 0 {
			return fmt.Errorf("events[%s].subjects required", id)
		}
		for _, subject := range event.Subjects {
			if err := validateManifestSubjectPattern(subject); err != nil {
				return fmt.Errorf("events[%s]: %w", id, err)
			}
		}
		for key := range event.Filter {
			if strings.TrimSpace(key) == "" {
				return fmt.Errorf("events[%s].filter contains empty key", id)
			}
		}
		limits, err := normalizeManifestCommandLimits("event", event.TimeoutMs, event.OutputCapBytes)
		if err != nil {
			return fmt.Errorf("events[%s]: %w", id, err)
		}
		event.TimeoutMs = limits.TimeoutMs
		event.OutputCapBytes = limits.OutputCapBytes
	}
	for i := range manifest.Hooks {
		hook := &manifest.Hooks[i]
		id := strings.TrimSpace(hook.ID)
		if id == "" {
			return fmt.Errorf("hooks[%d].id required", i)
		}
		if err := recordManifestContributionID(seenContributionIDs, id, "hooks"); err != nil {
			return err
		}
		if strings.TrimSpace(hook.Point) == "" {
			return fmt.Errorf("hooks[%s].point required", id)
		}
		if strings.TrimSpace(hook.Command) == "" {
			return fmt.Errorf("hooks[%s].command required", id)
		}
		limits, err := normalizeManifestCommandLimits("hook", hook.TimeoutMs, hook.OutputCapBytes)
		if err != nil {
			return fmt.Errorf("hooks[%s]: %w", id, err)
		}
		hook.TimeoutMs = limits.TimeoutMs
		hook.OutputCapBytes = limits.OutputCapBytes
	}
	for i := range manifest.Gates {
		gate := &manifest.Gates[i]
		id := strings.TrimSpace(gate.ID)
		if id == "" {
			return fmt.Errorf("gates[%d].id required", i)
		}
		if err := recordManifestContributionID(seenContributionIDs, id, "gates"); err != nil {
			return err
		}
		if strings.TrimSpace(gate.Label) == "" {
			return fmt.Errorf("gates[%s].label required", id)
		}
		if strings.TrimSpace(gate.Open.URLTemplate) == "" && strings.TrimSpace(gate.Resolve.Command) == "" {
			return fmt.Errorf("gates[%s].open.urlTemplate or resolve.command required", id)
		}
		if strings.TrimSpace(gate.Resolve.Command) != "" {
			limits, err := normalizeManifestCommandLimits("gate", gate.TimeoutMs, gate.OutputCapBytes)
			if err != nil {
				return fmt.Errorf("gates[%s]: %w", id, err)
			}
			gate.TimeoutMs = limits.TimeoutMs
			gate.OutputCapBytes = limits.OutputCapBytes
		}
	}
	for i := range manifest.WorkflowActions {
		action := &manifest.WorkflowActions[i]
		id := strings.TrimSpace(action.ID)
		if id == "" {
			return fmt.Errorf("workflowActions[%d].id required", i)
		}
		if err := recordManifestContributionID(seenContributionIDs, id, "workflowActions"); err != nil {
			return err
		}
		if strings.TrimSpace(action.Label) == "" {
			return fmt.Errorf("workflowActions[%s].label required", id)
		}
		if strings.TrimSpace(action.Command) == "" {
			return fmt.Errorf("workflowActions[%s].command required", id)
		}
		for _, phase := range action.Phases {
			if strings.TrimSpace(phase) == "" {
				return fmt.Errorf("workflowActions[%s].phases contains empty value", id)
			}
		}
		limits, err := normalizeManifestCommandLimits("workflowAction", action.TimeoutMs, action.OutputCapBytes)
		if err != nil {
			return fmt.Errorf("workflowActions[%s]: %w", id, err)
		}
		action.TimeoutMs = limits.TimeoutMs
		action.OutputCapBytes = limits.OutputCapBytes
	}
	if err := validateManifestPermissions(&manifest.Permissions); err != nil {
		return err
	}
	return nil
}

func recordManifestContributionID(seen map[string]string, id, kind string) error {
	if previous, ok := seen[id]; ok {
		return fmt.Errorf("duplicate manifest contribution id %q in %s and %s", id, previous, kind)
	}
	seen[id] = kind
	return nil
}

func validateManifestSubjectPattern(subject string) error {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return fmt.Errorf("subject required")
	}
	if !strings.Contains(subject, "*") {
		return nil
	}
	if strings.Count(subject, "*") > 1 || !strings.HasSuffix(subject, ".*") || strings.TrimSuffix(subject, ".*") == "" {
		return fmt.Errorf("subject %q uses unsupported wildcard", subject)
	}
	return nil
}

func validateManifestPermissions(permissions *Permissions) error {
	seenEnvPrefixes := map[string]bool{}
	for _, prefix := range permissions.EnvPrefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix == "" {
			return fmt.Errorf("permissions.envPrefixes contains empty value")
		}
		if seenEnvPrefixes[prefix] {
			return fmt.Errorf("permissions.envPrefixes contains duplicate value %q", prefix)
		}
		seenEnvPrefixes[prefix] = true
	}
	seenNetwork := map[string]bool{}
	for _, host := range permissions.Network {
		host = strings.TrimSpace(host)
		if host == "" {
			return fmt.Errorf("permissions.network contains empty value")
		}
		if seenNetwork[host] {
			return fmt.Errorf("permissions.network contains duplicate value %q", host)
		}
		seenNetwork[host] = true
	}
	return nil
}

func normalizeManifestCommandLimits(kind string, timeoutMs, outputCapBytes int) (CommandLimits, error) {
	defaultTimeout, maxTimeout, ok := manifestCommandTimeoutBounds(kind)
	if !ok {
		return CommandLimits{}, fmt.Errorf("unknown command limit kind %q", kind)
	}
	if timeoutMs < 0 {
		return CommandLimits{}, fmt.Errorf("timeoutMs must be non-negative")
	}
	if outputCapBytes < 0 {
		return CommandLimits{}, fmt.Errorf("outputCapBytes must be non-negative")
	}
	if timeoutMs == 0 {
		timeoutMs = defaultTimeout
	}
	if timeoutMs > maxTimeout {
		timeoutMs = maxTimeout
	}
	if outputCapBytes == 0 {
		outputCapBytes = manifestCommandDefaultOutputCapBytes
	}
	if outputCapBytes > manifestCommandMaxOutputCapBytes {
		outputCapBytes = manifestCommandMaxOutputCapBytes
	}
	return CommandLimits{TimeoutMs: timeoutMs, OutputCapBytes: outputCapBytes}, nil
}

func manifestCommandTimeoutBounds(kind string) (int, int, bool) {
	switch kind {
	case "event":
		return manifestEventDefaultTimeoutMs, manifestEventMaxTimeoutMs, true
	case "hook":
		return manifestHookDefaultTimeoutMs, manifestHookMaxTimeoutMs, true
	case "gate":
		return manifestGateDefaultTimeoutMs, manifestGateMaxTimeoutMs, true
	case "workflowAction":
		return manifestWorkflowActionDefaultTimeoutMs, manifestWorkflowActionMaxTimeoutMs, true
	default:
		return 0, 0, false
	}
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
