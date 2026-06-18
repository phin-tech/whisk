package plugins

import (
	"context"
	"fmt"

	"github.com/phin-tech/whisk/internal/app"
)

// ListRegistryPlugins returns the merged catalog across all configured
// registries, each entry annotated with local install and trust state. If some
// registries are unreachable, the reachable ones are still returned; the call
// only errors when nothing could be listed.
func (m *Manager) ListRegistryPlugins(ctx context.Context) ([]app.RegistryPlugin, error) {
	m.mu.RLock()
	installer := m.installer
	installed := make(map[string]bool, len(m.manifests))
	for id := range m.manifests {
		installed[id] = true
	}
	trusted := m.trusted
	m.mu.RUnlock()

	if installer == nil {
		return nil, fmt.Errorf("plugin registry is not configured")
	}
	available, err := installer.Available(ctx)
	if err != nil && len(available) == 0 {
		return nil, err
	}

	out := make([]app.RegistryPlugin, 0, len(available))
	for _, item := range available {
		key := qualifiedID(item.Registry, item.Entry.ID)
		out = append(out, app.RegistryPlugin{
			Registry:    item.Registry,
			ID:          item.Entry.ID,
			Name:        item.Entry.Name,
			Description: item.Entry.Description,
			SourceType:  string(item.Entry.Source.Type),
			Installed:   installed[key],
			Trusted:     trusted[key],
		})
	}
	return out, nil
}

// InstallPlugin fetches and installs a plugin from the named registry, then
// rescans so the newly installed plugin appears in the discovered set. The
// plugin is installed untrusted; the caller must trust it before its commands
// run.
func (m *Manager) InstallPlugin(ctx context.Context, registry, id string) (app.PluginStatus, error) {
	m.mu.RLock()
	installer := m.installer
	m.mu.RUnlock()
	if installer == nil {
		return app.PluginStatus{}, fmt.Errorf("plugin registry is not configured")
	}
	installed, err := installer.Install(ctx, registry, id)
	if err != nil {
		return app.PluginStatus{}, err
	}
	if _, err := m.RescanPlugins(ctx); err != nil {
		return app.PluginStatus{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.statusFor(qualifiedID(installed.Registry, installed.ID))
}
