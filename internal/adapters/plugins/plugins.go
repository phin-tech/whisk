package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/phin-tech/whisk/internal/adapters/agents"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/appsettings"
)

type Manifest struct {
	ManifestVersion int              `json:"manifestVersion,omitempty"`
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Version         string           `json:"version"`
	Resolvers       []Resolver       `json:"resolvers"`
	UsageResolvers  []UsageResolver  `json:"usageResolvers,omitempty"`
	AgentProfiles   []agents.Profile `json:"agentProfiles,omitempty"`
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

type UsageResolver struct {
	ID             string   `json:"id"`
	Provider       string   `json:"provider"`
	Label          string   `json:"label"`
	Profiles       []string `json:"profiles,omitempty"`
	Command        string   `json:"command"`
	TimeoutMs      int      `json:"timeoutMs,omitempty"`
	OutputCapBytes int      `json:"outputCapBytes,omitempty"`
	MinRefreshMs   int      `json:"minRefreshMs,omitempty"`
	StaleAfterMs   int      `json:"staleAfterMs,omitempty"`
}

type UI struct {
	ProjectAttachments []ProjectAttachmentAction `json:"projectAttachments,omitempty"`
	ReviewActions      []ReviewAction            `json:"reviewActions,omitempty"`
	Panels             []UIPanel                 `json:"panels,omitempty"`
	Commands           []UICommand               `json:"commands,omitempty"`
}

type ProjectAttachmentAction struct {
	ID       string                    `json:"id"`
	Label    string                    `json:"label"`
	Provider string                    `json:"provider"`
	Kind     string                    `json:"kind"`
	Command  string                    `json:"command"`
	Fields   []app.PluginTemplateField `json:"fields"`
}

// ReviewAction is the in-the-wild review UI shape formalized for manifest v2.
// It is cataloged for visibility, but is not executed by this adapter.
type ReviewAction struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Scope          string `json:"scope,omitempty"`
	URLTemplate    string `json:"urlTemplate,omitempty"`
	SubmitCommand  string `json:"submitCommand,omitempty"`
	Blocking       bool   `json:"blocking,omitempty"`
	TimeoutMs      int    `json:"timeoutMs,omitempty"`
	OutputCapBytes int    `json:"outputCapBytes,omitempty"`
}

type UIPanel struct {
	ID      string          `json:"id"`
	Title   string          `json:"title"`
	Scope   string          `json:"scope"`
	Kind    string          `json:"kind,omitempty"`
	Read    UICommandRef    `json:"read,omitempty"`
	Entry   UIPanelEntry    `json:"entry,omitempty"`
	Actions []UIPanelAction `json:"actions,omitempty"`
}

type UICommand struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Scope          string `json:"scope"`
	Command        string `json:"command"`
	TimeoutMs      int    `json:"timeoutMs,omitempty"`
	OutputCapBytes int    `json:"outputCapBytes,omitempty"`
}

type UIPanelAction struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Command        string `json:"command"`
	TimeoutMs      int    `json:"timeoutMs,omitempty"`
	OutputCapBytes int    `json:"outputCapBytes,omitempty"`
}

type UICommandRef struct {
	Command        string `json:"command,omitempty"`
	TimeoutMs      int    `json:"timeoutMs,omitempty"`
	OutputCapBytes int    `json:"outputCapBytes,omitempty"`
}

type UIPanelEntry struct {
	Path    string `json:"path,omitempty"`
	Forward string `json:"forward,omitempty"`
}

func (e *UIPanelEntry) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*e = UIPanelEntry{}
		return nil
	}
	var path string
	if err := json.Unmarshal(data, &path); err == nil {
		e.Path = path
		e.Forward = ""
		return nil
	}
	type entry UIPanelEntry
	var decoded entry
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*e = UIPanelEntry(decoded)
	return nil
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
	manifestUsageResolverDefaultTimeoutMs  = 10000
	manifestUsageResolverMaxTimeoutMs      = 30000
	manifestUICommandDefaultTimeoutMs      = 10000
	manifestUICommandMaxTimeoutMs          = 30000

	manifestCommandDefaultOutputCapBytes = 1 << 20
	manifestCommandMaxOutputCapBytes     = 4 << 20

	pluginUIPanelKindView = "view"
	pluginUIPanelKindHTML = "html"

	pluginUIScopeGlobal   = "global"
	pluginUIScopeProject  = "project"
	pluginUIScopeWorkItem = "workItem"
	pluginUIScopeRun      = "run"
	pluginUIScopeGate     = "gate"
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

const pluginProfilePrefix = "plugin:"

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
	input := struct {
		PluginID   string            `json:"pluginId"`
		TemplateID string            `json:"templateId"`
		ProjectID  string            `json:"projectId"`
		Values     map[string]string `json:"values,omitempty"`
	}{PluginID: req.PluginID, TemplateID: req.TemplateID, ProjectID: req.ProjectID, Values: req.Values}
	result, err := runPluginCommand(ctx, PluginCommandRequest{
		PluginID: req.PluginID,
		Dir:      dir,
		Command:  action.Command,
		Input:    input,
	})
	if err != nil {
		return app.AddProjectAttachmentRequest{}, err
	}
	var out app.PluginProjectAttachmentOutput
	if err := json.Unmarshal(result.Stdout, &out); err != nil {
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

func (m *Manager) ListAgentProfiles(context.Context) ([]agents.ProfileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := []agents.ProfileInfo{}
	for _, status := range m.statuses {
		if !status.Valid {
			continue
		}
		manifest, ok := m.manifests[status.ID]
		if !ok {
			continue
		}
		for _, profile := range manifest.AgentProfiles {
			info := pluginProfileInfo(status.ID, profile, m.trusted[status.ID])
			out = append(out, info)
		}
	}
	return out, nil
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

func NamespacedAgentProfileID(pluginID, localProfileID string) string {
	return pluginProfilePrefix + pluginID + "/" + localProfileID
}

func pluginProfileInfo(pluginID string, profile agents.Profile, trusted bool) agents.ProfileInfo {
	info := agents.ProfileInfo{
		ID:                  NamespacedAgentProfileID(pluginID, profile.ID),
		Provider:            profile.Provider,
		Label:               profile.Label,
		Description:         profile.Description,
		Source:              agents.ProfileSourcePlugin,
		PluginID:            pluginID,
		Launchable:          false,
		DetectCmd:           profile.DetectCmd,
		DetectAliases:       append([]string(nil), profile.DetectAliases...),
		ExpectedProcess:     profile.ExpectedProcess,
		PromptInjectionMode: profile.PromptInjectionMode,
		DraftPromptFlag:     profile.DraftPromptFlag,
		DraftPromptEnvVar:   profile.DraftPromptEnvVar,
		PreflightTrust:      profile.PreflightTrust,
		ReadySignal:         profile.ReadySignal,
		HookProvider:        profile.HookProvider,
	}
	if trusted {
		info.LaunchBlockedReason = "plugin agent profile launch is not implemented yet"
	} else {
		info.LaunchBlockedReason = fmt.Sprintf("plugin %s is not trusted", pluginID)
	}
	return info
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
	usageResolvers := make([]app.PluginUsageResolver, 0, len(manifest.UsageResolvers))
	for _, resolver := range manifest.UsageResolvers {
		if strings.TrimSpace(resolver.ID) == "" || strings.TrimSpace(resolver.Provider) == "" {
			continue
		}
		usageResolvers = append(usageResolvers, app.PluginUsageResolver{
			ID:             resolver.ID,
			Provider:       resolver.Provider,
			Label:          resolver.Label,
			Profiles:       resolver.Profiles,
			TimeoutMs:      resolver.TimeoutMs,
			OutputCapBytes: resolver.OutputCapBytes,
			MinRefreshMs:   resolver.MinRefreshMs,
			StaleAfterMs:   resolver.StaleAfterMs,
		})
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
	panels := make([]app.PluginUIPanel, 0, len(manifest.UI.Panels))
	for _, panel := range manifest.UI.Panels {
		if strings.TrimSpace(panel.ID) == "" {
			continue
		}
		summary := app.PluginUIPanel{
			ID:      panel.ID,
			Title:   panel.Title,
			Scope:   app.PluginUIScope(panel.Scope),
			Kind:    panel.Kind,
			Actions: make([]app.PluginUICommandRef, 0, len(panel.Actions)),
		}
		if strings.TrimSpace(panel.Read.Command) != "" {
			summary.Read = &app.PluginUICommandRef{
				TimeoutMs:      panel.Read.TimeoutMs,
				OutputCapBytes: panel.Read.OutputCapBytes,
			}
		}
		if panel.Entry.Path != "" || panel.Entry.Forward != "" {
			summary.Entry = &app.PluginUIPanelEntry{Path: panel.Entry.Path, Forward: panel.Entry.Forward}
		}
		for _, action := range panel.Actions {
			if strings.TrimSpace(action.ID) == "" {
				continue
			}
			summary.Actions = append(summary.Actions, app.PluginUICommandRef{
				ID:             action.ID,
				Label:          action.Label,
				TimeoutMs:      action.TimeoutMs,
				OutputCapBytes: action.OutputCapBytes,
			})
		}
		panels = append(panels, summary)
	}
	commands := make([]app.PluginUICommand, 0, len(manifest.UI.Commands))
	for _, command := range manifest.UI.Commands {
		if strings.TrimSpace(command.ID) == "" {
			continue
		}
		commands = append(commands, app.PluginUICommand{
			ID:             command.ID,
			Label:          command.Label,
			Scope:          app.PluginUIScope(command.Scope),
			TimeoutMs:      command.TimeoutMs,
			OutputCapBytes: command.OutputCapBytes,
		})
	}
	reviewActions := make([]app.PluginReviewAction, 0, len(manifest.UI.ReviewActions))
	for _, action := range manifest.UI.ReviewActions {
		if strings.TrimSpace(action.ID) == "" {
			continue
		}
		scope := action.Scope
		if strings.TrimSpace(scope) == "" {
			scope = pluginUIScopeWorkItem
		}
		reviewActions = append(reviewActions, app.PluginReviewAction{
			ID:             action.ID,
			Label:          action.Label,
			Scope:          app.PluginUIScope(scope),
			URLTemplate:    action.URLTemplate,
			HasSubmit:      strings.TrimSpace(action.SubmitCommand) != "",
			Blocking:       action.Blocking,
			TimeoutMs:      action.TimeoutMs,
			OutputCapBytes: action.OutputCapBytes,
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
		UsageResolvers:             usageResolvers,
		ProjectAttachmentTemplates: templates,
		UIPanels:                   panels,
		UICommands:                 commands,
		ReviewActions:              reviewActions,
		Permissions:                pluginPermissionsSummary(manifest.Permissions),
	}
}

func pluginPermissionsSummary(permissions Permissions) *app.PluginPermissions {
	if !permissions.PTYOutput && len(permissions.EnvPrefixes) == 0 && len(permissions.Network) == 0 {
		return nil
	}
	return &app.PluginPermissions{
		PTYOutput:   permissions.PTYOutput,
		EnvPrefixes: append([]string(nil), permissions.EnvPrefixes...),
		Network:     append([]string(nil), permissions.Network...),
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
		"usageResolvers":  true,
		"agentProfiles":   true,
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
		manifest.AgentProfiles = nil
		manifest.UsageResolvers = nil
		return nil
	}
	return validateManifestV2(manifest)
}

func validateManifestV2(manifest *Manifest) error {
	seenContributionIDs := map[string]string{}
	for i := range manifest.AgentProfiles {
		profile := &manifest.AgentProfiles[i]
		id := strings.TrimSpace(profile.ID)
		if id == "" {
			return fmt.Errorf("agentProfiles[%d].id required", i)
		}
		if strings.Contains(id, "/") {
			return fmt.Errorf("agentProfiles[%s].id must not contain /", id)
		}
		if agents.IsBuiltinProfileID(id) {
			return fmt.Errorf("agentProfiles[%s].id shadows builtin agent profile", id)
		}
		if err := recordManifestContributionID(seenContributionIDs, id, "agentProfiles"); err != nil {
			return err
		}
		profile.ID = id
		profile.Label = strings.TrimSpace(profile.Label)
		if profile.Label == "" {
			return fmt.Errorf("agentProfiles[%s].label required", id)
		}
		profile.Provider = agents.Provider(strings.TrimSpace(string(profile.Provider)))
		if profile.Provider == "" {
			return fmt.Errorf("agentProfiles[%s].provider required", id)
		}
		profile.Command = strings.TrimSpace(profile.Command)
		if profile.Command == "" {
			return fmt.Errorf("agentProfiles[%s].command required", id)
		}
		if profile.PromptInjectionMode == "" {
			profile.PromptInjectionMode = agents.PromptInjectionArgv
		}
		if !validPromptInjectionMode(profile.PromptInjectionMode) {
			return fmt.Errorf("agentProfiles[%s].promptInjectionMode %q is unsupported", id, profile.PromptInjectionMode)
		}
		for _, alias := range profile.DetectAliases {
			if strings.TrimSpace(alias) == "" {
				return fmt.Errorf("agentProfiles[%s].detectAliases contains empty value", id)
			}
		}
		for key := range profile.Env {
			if strings.TrimSpace(key) == "" {
				return fmt.Errorf("agentProfiles[%s].env contains empty key", id)
			}
		}
	}
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
	for i := range manifest.UsageResolvers {
		resolver := &manifest.UsageResolvers[i]
		id := strings.TrimSpace(resolver.ID)
		if id == "" {
			return fmt.Errorf("usageResolvers[%d].id required", i)
		}
		if err := recordManifestContributionID(seenContributionIDs, id, "usageResolvers"); err != nil {
			return err
		}
		resolver.ID = id
		resolver.Provider = strings.TrimSpace(resolver.Provider)
		if resolver.Provider == "" {
			return fmt.Errorf("usageResolvers[%s].provider required", id)
		}
		resolver.Label = strings.TrimSpace(resolver.Label)
		if resolver.Label == "" {
			return fmt.Errorf("usageResolvers[%s].label required", id)
		}
		resolver.Command = strings.TrimSpace(resolver.Command)
		if resolver.Command == "" {
			return fmt.Errorf("usageResolvers[%s].command required", id)
		}
		seenProfiles := map[string]bool{}
		for j, profile := range resolver.Profiles {
			profile = strings.TrimSpace(profile)
			if profile == "" {
				return fmt.Errorf("usageResolvers[%s].profiles[%d] required", id, j)
			}
			if seenProfiles[profile] {
				return fmt.Errorf("usageResolvers[%s].profiles contains duplicate value %q", id, profile)
			}
			seenProfiles[profile] = true
			resolver.Profiles[j] = profile
		}
		if resolver.MinRefreshMs < 0 {
			return fmt.Errorf("usageResolvers[%s].minRefreshMs must be non-negative", id)
		}
		if resolver.StaleAfterMs < 0 {
			return fmt.Errorf("usageResolvers[%s].staleAfterMs must be non-negative", id)
		}
		limits, err := normalizeManifestCommandLimits("usageResolver", resolver.TimeoutMs, resolver.OutputCapBytes)
		if err != nil {
			return fmt.Errorf("usageResolvers[%s]: %w", id, err)
		}
		resolver.TimeoutMs = limits.TimeoutMs
		resolver.OutputCapBytes = limits.OutputCapBytes
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
	if err := validateManifestUI(&manifest.UI, seenContributionIDs); err != nil {
		return err
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

func validPromptInjectionMode(mode agents.PromptInjectionMode) bool {
	switch mode {
	case agents.PromptInjectionArgv,
		agents.PromptInjectionFlagPrompt,
		agents.PromptInjectionFlagPromptInteractive,
		agents.PromptInjectionFlagInteractive,
		agents.PromptInjectionStdinAfterStart:
		return true
	default:
		return false
	}
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

func validateManifestUI(ui *UI, seenContributionIDs map[string]string) error {
	for i := range ui.ProjectAttachments {
		action := &ui.ProjectAttachments[i]
		id := strings.TrimSpace(action.ID)
		if id == "" {
			return fmt.Errorf("ui.projectAttachments[%d].id required", i)
		}
		if err := recordManifestContributionID(seenContributionIDs, id, "ui.projectAttachments"); err != nil {
			return err
		}
		action.ID = id
		action.Label = strings.TrimSpace(action.Label)
		if action.Label == "" {
			return fmt.Errorf("ui.projectAttachments[%s].label required", id)
		}
		action.Provider = strings.TrimSpace(action.Provider)
		if action.Provider == "" {
			return fmt.Errorf("ui.projectAttachments[%s].provider required", id)
		}
		action.Kind = strings.TrimSpace(action.Kind)
		if action.Kind == "" {
			return fmt.Errorf("ui.projectAttachments[%s].kind required", id)
		}
		action.Command = strings.TrimSpace(action.Command)
		if action.Command == "" {
			return fmt.Errorf("ui.projectAttachments[%s].command required", id)
		}
		if err := validateManifestTemplateFields("ui.projectAttachments", id, action.Fields); err != nil {
			return err
		}
	}
	for i := range ui.ReviewActions {
		action := &ui.ReviewActions[i]
		id := strings.TrimSpace(action.ID)
		if id == "" {
			return fmt.Errorf("ui.reviewActions[%d].id required", i)
		}
		if err := recordManifestContributionID(seenContributionIDs, id, "ui.reviewActions"); err != nil {
			return err
		}
		action.ID = id
		action.Label = strings.TrimSpace(action.Label)
		if action.Label == "" {
			return fmt.Errorf("ui.reviewActions[%s].label required", id)
		}
		action.Scope = normalizeManifestUIScope(action.Scope, pluginUIScopeWorkItem)
		if !validManifestUIScope(action.Scope) {
			return fmt.Errorf("ui.reviewActions[%s].scope %q is unsupported", id, action.Scope)
		}
		action.URLTemplate = strings.TrimSpace(action.URLTemplate)
		action.SubmitCommand = strings.TrimSpace(action.SubmitCommand)
		if action.URLTemplate == "" && action.SubmitCommand == "" {
			return fmt.Errorf("ui.reviewActions[%s].urlTemplate or submitCommand required", id)
		}
		if action.SubmitCommand != "" {
			limits, err := normalizeManifestCommandLimits("uiCommand", action.TimeoutMs, action.OutputCapBytes)
			if err != nil {
				return fmt.Errorf("ui.reviewActions[%s]: %w", id, err)
			}
			action.TimeoutMs = limits.TimeoutMs
			action.OutputCapBytes = limits.OutputCapBytes
		}
	}
	for i := range ui.Panels {
		panel := &ui.Panels[i]
		id := strings.TrimSpace(panel.ID)
		if id == "" {
			return fmt.Errorf("ui.panels[%d].id required", i)
		}
		if err := recordManifestContributionID(seenContributionIDs, id, "ui.panels"); err != nil {
			return err
		}
		panel.ID = id
		panel.Title = strings.TrimSpace(panel.Title)
		if panel.Title == "" {
			return fmt.Errorf("ui.panels[%s].title required", id)
		}
		panel.Scope = normalizeManifestUIScope(panel.Scope, "")
		if panel.Scope == "" {
			return fmt.Errorf("ui.panels[%s].scope required", id)
		}
		if !validManifestUIScope(panel.Scope) {
			return fmt.Errorf("ui.panels[%s].scope %q is unsupported", id, panel.Scope)
		}
		panel.Kind = strings.TrimSpace(panel.Kind)
		if panel.Kind == "" {
			panel.Kind = pluginUIPanelKindView
		}
		switch panel.Kind {
		case pluginUIPanelKindView:
			panel.Read.Command = strings.TrimSpace(panel.Read.Command)
			if panel.Read.Command == "" {
				return fmt.Errorf("ui.panels[%s].read.command required", id)
			}
			limits, err := normalizeManifestCommandLimits("uiCommand", panel.Read.TimeoutMs, panel.Read.OutputCapBytes)
			if err != nil {
				return fmt.Errorf("ui.panels[%s].read: %w", id, err)
			}
			panel.Read.TimeoutMs = limits.TimeoutMs
			panel.Read.OutputCapBytes = limits.OutputCapBytes
		case pluginUIPanelKindHTML:
			if err := validateManifestUIPanelEntry(id, &panel.Entry); err != nil {
				return err
			}
		default:
			return fmt.Errorf("ui.panels[%s].kind %q is unsupported", id, panel.Kind)
		}
		if err := validateManifestUIPanelActions(id, panel.Actions); err != nil {
			return err
		}
	}
	for i := range ui.Commands {
		command := &ui.Commands[i]
		id := strings.TrimSpace(command.ID)
		if id == "" {
			return fmt.Errorf("ui.commands[%d].id required", i)
		}
		if err := recordManifestContributionID(seenContributionIDs, id, "ui.commands"); err != nil {
			return err
		}
		command.ID = id
		command.Label = strings.TrimSpace(command.Label)
		if command.Label == "" {
			return fmt.Errorf("ui.commands[%s].label required", id)
		}
		command.Scope = normalizeManifestUIScope(command.Scope, "")
		if command.Scope == "" {
			return fmt.Errorf("ui.commands[%s].scope required", id)
		}
		if !validManifestUIScope(command.Scope) {
			return fmt.Errorf("ui.commands[%s].scope %q is unsupported", id, command.Scope)
		}
		command.Command = strings.TrimSpace(command.Command)
		if command.Command == "" {
			return fmt.Errorf("ui.commands[%s].command required", id)
		}
		limits, err := normalizeManifestCommandLimits("uiCommand", command.TimeoutMs, command.OutputCapBytes)
		if err != nil {
			return fmt.Errorf("ui.commands[%s]: %w", id, err)
		}
		command.TimeoutMs = limits.TimeoutMs
		command.OutputCapBytes = limits.OutputCapBytes
	}
	return nil
}

func validateManifestTemplateFields(section, id string, fields []app.PluginTemplateField) error {
	seenFields := map[string]bool{}
	for i := range fields {
		field := &fields[i]
		fieldID := strings.TrimSpace(field.ID)
		if fieldID == "" {
			return fmt.Errorf("%s[%s].fields[%d].id required", section, id, i)
		}
		if seenFields[fieldID] {
			return fmt.Errorf("%s[%s].fields contains duplicate id %q", section, id, fieldID)
		}
		seenFields[fieldID] = true
		field.ID = fieldID
		field.Label = strings.TrimSpace(field.Label)
		if field.Label == "" {
			return fmt.Errorf("%s[%s].fields[%s].label required", section, id, fieldID)
		}
		field.Type = strings.TrimSpace(field.Type)
		if field.Type == "" {
			return fmt.Errorf("%s[%s].fields[%s].type required", section, id, fieldID)
		}
		seenOptions := map[string]bool{}
		for j, option := range field.Options {
			option = strings.TrimSpace(option)
			if option == "" {
				return fmt.Errorf("%s[%s].fields[%s].options[%d] required", section, id, fieldID, j)
			}
			if seenOptions[option] {
				return fmt.Errorf("%s[%s].fields[%s].options contains duplicate value %q", section, id, fieldID, option)
			}
			seenOptions[option] = true
			field.Options[j] = option
		}
	}
	return nil
}

func validateManifestUIPanelEntry(panelID string, entry *UIPanelEntry) error {
	entry.Path = strings.TrimSpace(entry.Path)
	entry.Forward = strings.TrimSpace(entry.Forward)
	if entry.Path == "" && entry.Forward == "" {
		return fmt.Errorf("ui.panels[%s].entry.path or entry.forward required", panelID)
	}
	if entry.Path != "" && entry.Forward != "" {
		return fmt.Errorf("ui.panels[%s].entry must not set both path and forward", panelID)
	}
	return nil
}

func validateManifestUIPanelActions(panelID string, actions []UIPanelAction) error {
	seenActions := map[string]bool{}
	for i := range actions {
		action := &actions[i]
		id := strings.TrimSpace(action.ID)
		if id == "" {
			return fmt.Errorf("ui.panels[%s].actions[%d].id required", panelID, i)
		}
		if seenActions[id] {
			return fmt.Errorf("ui.panels[%s].actions contains duplicate id %q", panelID, id)
		}
		seenActions[id] = true
		action.ID = id
		action.Label = strings.TrimSpace(action.Label)
		if action.Label == "" {
			return fmt.Errorf("ui.panels[%s].actions[%s].label required", panelID, id)
		}
		action.Command = strings.TrimSpace(action.Command)
		if action.Command == "" {
			return fmt.Errorf("ui.panels[%s].actions[%s].command required", panelID, id)
		}
		limits, err := normalizeManifestCommandLimits("uiCommand", action.TimeoutMs, action.OutputCapBytes)
		if err != nil {
			return fmt.Errorf("ui.panels[%s].actions[%s]: %w", panelID, id, err)
		}
		action.TimeoutMs = limits.TimeoutMs
		action.OutputCapBytes = limits.OutputCapBytes
	}
	return nil
}

func normalizeManifestUIScope(scope, fallback string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return fallback
	}
	return scope
}

func validManifestUIScope(scope string) bool {
	switch scope {
	case pluginUIScopeGlobal,
		pluginUIScopeProject,
		pluginUIScopeWorkItem,
		pluginUIScopeRun,
		pluginUIScopeGate:
		return true
	default:
		return false
	}
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
	case "usageResolver":
		return manifestUsageResolverDefaultTimeoutMs, manifestUsageResolverMaxTimeoutMs, true
	case "uiCommand":
		return manifestUICommandDefaultTimeoutMs, manifestUICommandMaxTimeoutMs, true
	default:
		return 0, 0, false
	}
}

func (r CommandResolver) ResolveProjectAttachment(ctx context.Context, req app.ResolveProjectAttachmentRequest) (app.ResolvedProjectAttachment, error) {
	result, err := runPluginCommand(ctx, PluginCommandRequest{
		PluginID: r.PluginID,
		Dir:      r.Dir,
		Command:  r.Command,
		Input:    req,
	})
	if err != nil {
		return app.ResolvedProjectAttachment{}, err
	}
	var out app.ResolvedProjectAttachment
	if err := json.Unmarshal(result.Stdout, &out); err != nil {
		return app.ResolvedProjectAttachment{}, err
	}
	return out, nil
}
