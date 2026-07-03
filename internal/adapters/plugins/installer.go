package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phin-tech/whisk/internal/domain/pluginregistry"
)

// namedRegistry pairs a registry namespace with the transport that fetches it.
type namedRegistry struct {
	Name      string
	Transport Transport
}

// Installer materializes registry plugins onto disk across one or more
// registries. It downloads a plugin's file bundle through the owning registry's
// Transport, verifies the bundle carries a matching plugin.json, writes it
// atomically into the plugin directory (namespaced by registry), and records the
// install in a lockfile. Installation never grants trust: the plugin lands
// untrusted and the user must trust it before its commands run.
type Installer struct {
	registries []namedRegistry
	targetDir  string
	lockPath   string
}

// AvailablePlugin is a registry catalog entry tagged with its registry.
type AvailablePlugin struct {
	Registry string
	Entry    pluginregistry.Entry
}

// InstalledPlugin reports the outcome of an install.
type InstalledPlugin struct {
	Registry    string
	ID          string
	Name        string
	Dir         string
	Version     string
	Fingerprint string
}

func NewInstaller(registries []namedRegistry, targetDir, lockPath string) *Installer {
	return &Installer{registries: registries, targetDir: targetDir, lockPath: lockPath}
}

// RegistryNames returns the configured registry namespaces in order.
func (i *Installer) RegistryNames() []string {
	names := make([]string, 0, len(i.registries))
	for _, registry := range i.registries {
		names = append(names, registry.Name)
	}
	return names
}

// Available returns the merged catalog across all registries, sorted by
// (registry, id). A registry that fails to fetch is skipped and its error
// returned alongside the entries that did resolve, so one unreachable registry
// does not blank the whole list.
func (i *Installer) Available(ctx context.Context) ([]AvailablePlugin, error) {
	var out []AvailablePlugin
	var firstErr error
	for _, registry := range i.registries {
		catalog, err := i.catalog(ctx, registry)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("registry %q: %w", registry.Name, err)
			}
			continue
		}
		for _, entry := range catalog.SortedPlugins() {
			out = append(out, AvailablePlugin{Registry: registry.Name, Entry: entry})
		}
	}
	return out, firstErr
}

func (i *Installer) catalog(ctx context.Context, registry namedRegistry) (pluginregistry.Registry, error) {
	data, err := registry.Transport.Registry(ctx)
	if err != nil {
		return pluginregistry.Registry{}, fmt.Errorf("fetch registry: %w", err)
	}
	return pluginregistry.ParseRegistry(data)
}

func (i *Installer) find(registryName string) (namedRegistry, bool) {
	for _, registry := range i.registries {
		if registry.Name == registryName {
			return registry, true
		}
	}
	return namedRegistry{}, false
}

// Install fetches and installs the plugin id from the named registry. When
// registryName is empty and exactly one registry is configured, that registry
// is used; otherwise the registry must be named to disambiguate.
func (i *Installer) Install(ctx context.Context, registryName, id string) (InstalledPlugin, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return InstalledPlugin{}, fmt.Errorf("plugin id is required")
	}
	registryName = strings.TrimSpace(registryName)
	if registryName == "" {
		if len(i.registries) != 1 {
			return InstalledPlugin{}, fmt.Errorf("registry must be specified (configured: %s)", strings.Join(i.RegistryNames(), ", "))
		}
		registryName = i.registries[0].Name
	}
	registry, ok := i.find(registryName)
	if !ok {
		return InstalledPlugin{}, fmt.Errorf("registry %q is not configured", registryName)
	}

	catalog, err := i.catalog(ctx, registry)
	if err != nil {
		return InstalledPlugin{}, err
	}
	entry, ok := catalog.Find(id)
	if !ok {
		return InstalledPlugin{}, fmt.Errorf("plugin %q not found in registry %q", id, registryName)
	}

	files, err := registry.Transport.Fetch(ctx, entry.Source)
	if err != nil {
		return InstalledPlugin{}, fmt.Errorf("fetch plugin %q: %w", id, err)
	}

	manifest, err := manifestFromBundle(files)
	if err != nil {
		return InstalledPlugin{}, fmt.Errorf("plugin %q: %w", id, err)
	}
	if manifest.ID != id {
		return InstalledPlugin{}, fmt.Errorf("plugin %q: manifest id %q does not match registry id", id, manifest.ID)
	}

	fingerprint := pluginregistry.Fingerprint(files)
	dir := filepath.Join(i.targetDir, registryName, id)
	if err := writeBundle(dir, files); err != nil {
		return InstalledPlugin{}, fmt.Errorf("install plugin %q: %w", id, err)
	}

	if err := i.recordLock(registryName, entry, manifest.Version, fingerprint); err != nil {
		return InstalledPlugin{}, fmt.Errorf("record lock for %q: %w", id, err)
	}

	return InstalledPlugin{
		Registry:    registryName,
		ID:          id,
		Name:        manifest.Name,
		Dir:         dir,
		Version:     manifest.Version,
		Fingerprint: fingerprint,
	}, nil
}

func manifestFromBundle(files map[string][]byte) (Manifest, error) {
	data, ok := files["plugin.json"]
	if !ok {
		return Manifest{}, fmt.Errorf("bundle is missing plugin.json")
	}
	manifest, err := parseManifest(data)
	if err != nil {
		return Manifest{}, fmt.Errorf("parse plugin.json: %w", err)
	}
	return manifest, nil
}

// writeBundle materializes files into dir atomically: it writes to a sibling
// temp directory, removes any prior install, then renames into place.
func writeBundle(dir string, files map[string][]byte) error {
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, "."+filepath.Base(dir)+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	for rel, data := range files {
		// Defend against path traversal in fetched bundles.
		clean := filepath.Clean(filepath.FromSlash(rel))
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
			return fmt.Errorf("unsafe path in bundle: %q", rel)
		}
		dest := filepath.Join(staging, clean)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return err
		}
	}

	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.Rename(staging, dir)
}

func (i *Installer) recordLock(registry string, entry pluginregistry.Entry, version, fingerprint string) error {
	if i.lockPath == "" {
		return nil
	}
	data, err := os.ReadFile(i.lockPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	lock, err := pluginregistry.ParseLock(data)
	if err != nil {
		return err
	}
	lock = lock.Set(pluginregistry.LockEntry{
		Registry:    registry,
		ID:          entry.ID,
		Source:      entry.Source,
		Version:     version,
		Fingerprint: fingerprint,
	})
	out, err := lock.Marshal()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(i.lockPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(i.lockPath, out, 0o644)
}
