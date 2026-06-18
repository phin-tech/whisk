package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phin-tech/whisk/internal/domain/pluginregistry"
)

// Installer materializes registry plugins onto disk. It downloads a plugin's
// file bundle through a Transport, verifies the bundle carries a matching
// plugin.json, writes it atomically into the plugin directory, and records the
// install in a lockfile. Installation never grants trust: the plugin lands
// untrusted and the user must trust it before its commands run.
type Installer struct {
	transport Transport
	targetDir string
	lockPath  string
}

// InstalledPlugin reports the outcome of an install.
type InstalledPlugin struct {
	ID          string
	Name        string
	Dir         string
	Version     string
	Fingerprint string
}

func NewInstaller(transport Transport, targetDir, lockPath string) *Installer {
	return &Installer{transport: transport, targetDir: targetDir, lockPath: lockPath}
}

// Available returns the registry catalog entries, sorted by id.
func (i *Installer) Available(ctx context.Context) ([]pluginregistry.Entry, error) {
	registry, err := i.registry(ctx)
	if err != nil {
		return nil, err
	}
	return registry.SortedPlugins(), nil
}

func (i *Installer) registry(ctx context.Context) (pluginregistry.Registry, error) {
	data, err := i.transport.Registry(ctx)
	if err != nil {
		return pluginregistry.Registry{}, fmt.Errorf("fetch registry: %w", err)
	}
	return pluginregistry.ParseRegistry(data)
}

// Install fetches and installs the plugin with the given id. It returns an error
// if the id is unknown, the bundle is missing a valid plugin.json, or the
// manifest id does not match the registry id.
func (i *Installer) Install(ctx context.Context, id string) (InstalledPlugin, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return InstalledPlugin{}, fmt.Errorf("plugin id is required")
	}
	registry, err := i.registry(ctx)
	if err != nil {
		return InstalledPlugin{}, err
	}
	entry, ok := registry.Find(id)
	if !ok {
		return InstalledPlugin{}, fmt.Errorf("plugin %q not found in registry", id)
	}

	files, err := i.transport.Fetch(ctx, entry.Source)
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
	dir := filepath.Join(i.targetDir, id)
	if err := writeBundle(dir, files); err != nil {
		return InstalledPlugin{}, fmt.Errorf("install plugin %q: %w", id, err)
	}

	if err := i.recordLock(entry, manifest.Version, fingerprint); err != nil {
		return InstalledPlugin{}, fmt.Errorf("record lock for %q: %w", id, err)
	}

	return InstalledPlugin{
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
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse plugin.json: %w", err)
	}
	if strings.TrimSpace(manifest.ID) == "" {
		return Manifest{}, fmt.Errorf("plugin.json is missing an id")
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

func (i *Installer) recordLock(entry pluginregistry.Entry, version, fingerprint string) error {
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
