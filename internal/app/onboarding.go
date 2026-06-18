package app

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/phin-tech/whisk/internal/buildinfo"
	"github.com/phin-tech/whisk/internal/domain/onboarding"
)

type OnboardingStatus struct {
	Items       []onboarding.Item `json:"items"`
	ShouldShow  bool              `json:"shouldShow"`
	LocalDaemon bool              `json:"localDaemon"`
	StatePath   string            `json:"statePath"`
}

type OnboardingApplyRequest struct {
	ItemIDs []string `json:"itemIds"`
}

func (r *Runtime) OnboardingStatus(ctx context.Context) (OnboardingStatus, error) {
	state, _ := r.loadOnboardingState()
	items, err := r.onboardingItems(ctx, state)
	if err != nil {
		return OnboardingStatus{}, err
	}
	items = onboarding.SelectDefaults(items)
	return OnboardingStatus{
		Items:       items,
		ShouldShow:  onboarding.ShouldShow(items, state),
		LocalDaemon: isLocalDaemonURL(r.daemonURL),
		StatePath:   r.onboardingPath(),
	}, nil
}

func (r *Runtime) ApplyOnboarding(ctx context.Context, req OnboardingApplyRequest) (OnboardingStatus, error) {
	if !isLocalDaemonURL(r.daemonURL) {
		return OnboardingStatus{}, fmt.Errorf("onboarding apply requires a local daemon")
	}
	state, _ := r.loadOnboardingState()
	items, err := r.onboardingItems(ctx, state)
	if err != nil {
		return OnboardingStatus{}, err
	}
	selected := map[string]bool{}
	for _, id := range req.ItemIDs {
		selected[id] = true
	}
	for _, item := range items {
		if !selected[item.ID] {
			continue
		}
		if err := r.applyOnboardingItem(ctx, item); err != nil {
			return OnboardingStatus{}, err
		}
	}
	items, err = r.onboardingItems(ctx, state)
	if err != nil {
		return OnboardingStatus{}, err
	}
	next := onboarding.NextState(state, items, selected, r.daemonAPIVersion, buildinfo.GitSHA())
	if err := r.saveOnboardingState(next); err != nil {
		return OnboardingStatus{}, err
	}
	return r.OnboardingStatus(ctx)
}

func (r *Runtime) onboardingItems(ctx context.Context, state onboarding.State) ([]onboarding.Item, error) {
	items := []onboarding.Item{r.daemonOnboardingItem(state)}
	hooks, err := r.ListAgentHookIntegrations(ctx)
	if err != nil {
		return nil, err
	}
	for _, hook := range hooks {
		items = append(items, onboarding.Item{
			ID:               "hook:" + hook.Provider,
			Kind:             onboarding.KindHook,
			Label:            agentProviderLabel(hook.Provider) + " hooks",
			Description:      "Installs Whisk's helper into the agent's hook config so Whisk can record activity and handle approval requests.",
			Target:           hook.Provider,
			Status:           hook.Status,
			InstalledVersion: hook.InstalledVersion,
			LatestVersion:    hook.LatestVersion,
			Path:             hook.ConfigPath,
			Detail:           hook.Detail,
		})
	}
	items = append(items, r.skillOnboardingItems()...)
	plugins, err := r.ListPlugins(ctx)
	if err != nil {
		return nil, err
	}
	for _, plugin := range plugins {
		status := onboarding.StatusCurrent
		if !plugin.Valid {
			status = onboarding.StatusUnavailable
		} else if !plugin.Trusted {
			status = onboarding.StatusUntrusted
		}
		items = append(items, onboarding.Item{
			ID:               "plugin:" + plugin.ID,
			Kind:             onboarding.KindPlugin,
			Label:            pluginLabel(plugin),
			Description:      "Marks this discovered local plugin as trusted so Whisk can use its resolvers and project attachment actions.",
			Target:           plugin.ID,
			Status:           status,
			InstalledVersion: plugin.Version,
			LatestVersion:    plugin.Version,
			Path:             plugin.Dir,
			Detail:           plugin.Error,
		})
	}
	return items, nil
}

func (r *Runtime) applyOnboardingItem(ctx context.Context, item onboarding.Item) error {
	switch item.Kind {
	case onboarding.KindDaemon:
		return nil
	case onboarding.KindHook:
		_, err := r.InstallAgentHookIntegration(ctx, AgentHookIntegrationRequest{Provider: item.Target})
		return err
	case onboarding.KindSkill:
		return r.installSkillTarget(item.Target)
	case onboarding.KindPlugin:
		_, err := r.TrustPlugin(ctx, item.Target)
		return err
	default:
		return fmt.Errorf("unknown onboarding item kind %q", item.Kind)
	}
}

func (r *Runtime) daemonOnboardingItem(state onboarding.State) onboarding.Item {
	status := onboarding.StatusCurrent
	gitSHA := buildinfo.GitSHA()
	if (state.DaemonAPIVersion != 0 && state.DaemonAPIVersion != r.daemonAPIVersion) ||
		(state.DaemonGitSHA != "" && state.DaemonGitSHA != gitSHA) {
		status = onboarding.StatusOutdated
	}
	return onboarding.Item{
		ID:               "daemon:version",
		Kind:             onboarding.KindDaemon,
		Label:            "Daemon version",
		Description:      "Records the whiskd API and build version used for onboarding so Whisk can notice later drift.",
		Target:           "whiskd",
		Status:           status,
		InstalledVersion: versionLabel(state.DaemonAPIVersion, state.DaemonGitSHA),
		LatestVersion:    versionLabel(r.daemonAPIVersion, gitSHA),
	}
}

func (r *Runtime) skillOnboardingItems() []onboarding.Item {
	source := r.skillSourceDir()
	sourceHash, sourceVersion, sourceErr := skillHashAndVersion(source)
	targets := detectedSkillTargets()
	items := make([]onboarding.Item, 0, len(targets))
	for _, target := range targets {
		item := onboarding.Item{
			ID:            "skill:" + target.ID,
			Kind:          onboarding.KindSkill,
			Label:         target.Label + " skill",
			Description:   "Copies Whisk's agent instructions into this tool's skills folder so the agent knows how to use whiskd and the whisk CLI.",
			Target:        target.ID,
			LatestVersion: sourceVersion,
			Hash:          sourceHash,
			Path:          target.Path,
		}
		if sourceErr != nil {
			item.Status = onboarding.StatusUnavailable
			item.Detail = sourceErr.Error()
			items = append(items, item)
			continue
		}
		installedHash, installedVersion, err := skillHashAndVersion(target.Path)
		item.InstalledHash = installedHash
		item.InstalledVersion = installedVersion
		switch {
		case os.IsNotExist(err):
			item.Status = onboarding.StatusMissing
		case err != nil:
			item.Status = onboarding.StatusUnavailable
			item.Detail = err.Error()
		case installedHash != sourceHash:
			item.Status = onboarding.StatusOutdated
		default:
			item.Status = onboarding.StatusCurrent
		}
		items = append(items, item)
	}
	return items
}

func (r *Runtime) installSkillTarget(targetID string) error {
	var target skillTarget
	for _, candidate := range detectedSkillTargets() {
		if candidate.ID == targetID {
			target = candidate
			break
		}
	}
	if target.ID == "" {
		return fmt.Errorf("skill target %s not detected", targetID)
	}
	source := r.skillSourceDir()
	if _, _, err := skillHashAndVersion(source); err != nil {
		return err
	}
	for _, name := range []string{"SKILL.md", "README.md"} {
		bytes, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(target.Path, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(target.Path, name), bytes, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) skillSourceDir() string {
	if r.onboardingSkillDir != "" {
		return r.onboardingSkillDir
	}
	if value := os.Getenv("WHISK_SKILL_DIR"); value != "" {
		return value
	}
	if _, err := os.Stat(filepath.Join("skills", "whisk", "SKILL.md")); err == nil {
		return filepath.Join("skills", "whisk")
	}
	exe, err := os.Executable()
	if err == nil {
		return filepath.Join(filepath.Dir(exe), "skills", "whisk")
	}
	return filepath.Join("skills", "whisk")
}

func (r *Runtime) loadOnboardingState() (onboarding.State, error) {
	raw, err := os.ReadFile(r.onboardingPath())
	if os.IsNotExist(err) {
		return onboarding.State{}, nil
	}
	if err != nil {
		return onboarding.State{}, err
	}
	var state onboarding.State
	if err := json.Unmarshal(raw, &state); err != nil {
		return onboarding.State{}, err
	}
	return state, nil
}

func (r *Runtime) saveOnboardingState(state onboarding.State) error {
	path := r.onboardingPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

func (r *Runtime) onboardingPath() string {
	if r.onboardingStatePath != "" {
		return r.onboardingStatePath
	}
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return filepath.Join(configDir, "whisk", "onboarding.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "whisk", "onboarding.json")
	}
	return filepath.Join(home, ".config", "whisk", "onboarding.json")
}

type skillTarget struct {
	ID    string
	Label string
	Path  string
}

func detectedSkillTargets() []skillTarget {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	candidates := []skillTarget{
		{ID: "codex", Label: "Codex", Path: filepath.Join(home, ".codex", "skills", "whisk")},
		{ID: "claude", Label: "Claude", Path: filepath.Join(home, ".claude", "skills", "whisk")},
	}
	var out []skillTarget
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Dir(candidate.Path)); err == nil && info.IsDir() {
			out = append(out, candidate)
		}
	}
	return out
}

func skillHashAndVersion(dir string) (string, string, error) {
	var files []string
	for _, name := range []string{"SKILL.md", "README.md"} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			return "", "", err
		}
		if info.IsDir() {
			return "", "", fmt.Errorf("%s is a directory", path)
		}
		files = append(files, name)
	}
	sort.Strings(files)
	hash := sha256.New()
	for _, name := range files {
		bytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return "", "", err
		}
		_, _ = hash.Write([]byte(name))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(bytes)
	}
	version := skillVersion(filepath.Join(dir, "SKILL.md"))
	return hex.EncodeToString(hash.Sum(nil)), version, nil
}

func skillVersion(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" && scanner.Text() != line {
			continue
		}
		if value, ok := strings.CutPrefix(line, "version:"); ok {
			return strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return ""
}

func isLocalDaemonURL(raw string) bool {
	if raw == "" {
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func agentProviderLabel(provider string) string {
	switch provider {
	case "codex":
		return "Codex"
	case "claude":
		return "Claude"
	default:
		return provider
	}
}

func pluginLabel(plugin PluginStatus) string {
	if plugin.Name != "" {
		return plugin.Name
	}
	return plugin.ID
}

func versionLabel(apiVersion int, gitSHA string) string {
	if gitSHA == "" {
		return "api-" + strconv.Itoa(apiVersion)
	}
	return "api-" + strconv.Itoa(apiVersion) + " " + gitSHA
}
