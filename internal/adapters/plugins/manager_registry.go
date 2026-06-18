package plugins

import (
	"context"
	"fmt"

	"github.com/phin-tech/whisk/internal/app"
)

// ListRegistryPlugins returns the registry catalog annotated with local install
// and trust state.
func (m *Manager) ListRegistryPlugins(ctx context.Context) ([]app.RegistryPlugin, error) {
	if m.installer == nil {
		return nil, fmt.Errorf("plugin registry is not configured")
	}
	entries, err := m.installer.Available(ctx)
	if err != nil {
		return nil, err
	}
	m.mu.RLock()
	installed := make(map[string]bool, len(m.manifests))
	for id := range m.manifests {
		installed[id] = true
	}
	trusted := m.trusted
	m.mu.RUnlock()

	out := make([]app.RegistryPlugin, 0, len(entries))
	for _, entry := range entries {
		out = append(out, app.RegistryPlugin{
			ID:          entry.ID,
			Name:        entry.Name,
			Description: entry.Description,
			SourceType:  string(entry.Source.Type),
			Installed:   installed[entry.ID],
			Trusted:     trusted[entry.ID],
		})
	}
	return out, nil
}

// InstallPlugin fetches and installs a registry plugin, then rescans so the
// newly installed plugin appears in the discovered set. The plugin is installed
// untrusted; the caller must trust it before its commands run.
func (m *Manager) InstallPlugin(ctx context.Context, id string) (app.PluginStatus, error) {
	if m.installer == nil {
		return app.PluginStatus{}, fmt.Errorf("plugin registry is not configured")
	}
	installed, err := m.installer.Install(ctx, id)
	if err != nil {
		return app.PluginStatus{}, err
	}
	if _, err := m.RescanPlugins(ctx); err != nil {
		return app.PluginStatus{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.statusFor(installed.ID)
}
