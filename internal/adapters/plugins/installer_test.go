package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phin-tech/whisk/internal/domain/pluginregistry"
)

// fakeTransport serves a canned catalog and per-id file bundles.
type fakeTransport struct {
	registry []byte
	bundles  map[string]map[string][]byte
	fetched  []pluginregistry.Source
}

func (f *fakeTransport) Registry(context.Context) ([]byte, error) { return f.registry, nil }

func (f *fakeTransport) Fetch(_ context.Context, source pluginregistry.Source) (map[string][]byte, error) {
	f.fetched = append(f.fetched, source)
	// Index bundles by the path or repo for simplicity.
	key := source.Path
	if source.Type == pluginregistry.SourceGit {
		key = source.Repo
	}
	bundle, ok := f.bundles[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return bundle, nil
}

func newFixtureInstaller(t *testing.T, transport Transport) (*Installer, string, string) {
	t.Helper()
	target := t.TempDir()
	lockPath := filepath.Join(t.TempDir(), "plugins.lock.json")
	registries := []namedRegistry{{Name: "test", Transport: transport}}
	return NewInstaller(registries, target, lockPath), target, lockPath
}

func TestInstallerInstallsPathPluginUntrusted(t *testing.T) {
	transport := &fakeTransport{
		registry: []byte(`{"version":1,"plugins":[{"id":"github-issues","name":"GitHub Issues","source":{"type":"path","path":"plugins/github-issues"}}]}`),
		bundles: map[string]map[string][]byte{
			"plugins/github-issues": {
				"plugin.json": []byte(`{"id":"github-issues","name":"GitHub Issues","version":"0.1.0"}`),
				"resolve.mjs": []byte("export default {}"),
			},
		},
	}
	installer, target, lockPath := newFixtureInstaller(t, transport)

	result, err := installer.Install(context.Background(), "test", "github-issues")
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if result.Version != "0.1.0" || result.Fingerprint == "" {
		t.Fatalf("result = %#v", result)
	}

	// Files landed on disk.
	manifest := filepath.Join(target, "test", "github-issues", "plugin.json")
	if _, err := os.Stat(manifest); err != nil {
		t.Fatalf("plugin.json not installed: %v", err)
	}
	script := filepath.Join(target, "test", "github-issues", "resolve.mjs")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("resolve.mjs not installed: %v", err)
	}

	// Lockfile recorded the install with a fingerprint.
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	lock, err := pluginregistry.ParseLock(data)
	if err != nil {
		t.Fatalf("parse lock: %v", err)
	}
	entry, ok := lock.Get("test", "github-issues")
	if !ok || entry.Fingerprint != result.Fingerprint {
		t.Fatalf("lock entry = %#v ok = %v", entry, ok)
	}
}

func TestInstallerGitSource(t *testing.T) {
	transport := &fakeTransport{
		registry: []byte(`{"version":1,"plugins":[{"id":"linear","source":{"type":"git","repo":"phin-tech/whisk-plugin-linear","ref":"v1"}}]}`),
		bundles: map[string]map[string][]byte{
			"phin-tech/whisk-plugin-linear": {
				"plugin.json": []byte(`{"id":"linear","version":"1.0.0"}`),
			},
		},
	}
	installer, _, _ := newFixtureInstaller(t, transport)
	if _, err := installer.Install(context.Background(), "test", "linear"); err != nil {
		t.Fatalf("Install git source: %v", err)
	}
	if len(transport.fetched) != 1 || transport.fetched[0].Type != pluginregistry.SourceGit {
		t.Fatalf("fetched = %#v", transport.fetched)
	}
}

func TestInstallerRejectsUnknownID(t *testing.T) {
	transport := &fakeTransport{registry: []byte(`{"version":1,"plugins":[]}`)}
	installer, _, _ := newFixtureInstaller(t, transport)
	if _, err := installer.Install(context.Background(), "test", "missing"); err == nil {
		t.Fatal("Install(missing) = nil error, want error")
	}
}

func TestInstallerRejectsManifestIDMismatch(t *testing.T) {
	transport := &fakeTransport{
		registry: []byte(`{"version":1,"plugins":[{"id":"github-issues","source":{"type":"path","path":"p"}}]}`),
		bundles: map[string]map[string][]byte{
			"p": {"plugin.json": []byte(`{"id":"something-else"}`)},
		},
	}
	installer, _, _ := newFixtureInstaller(t, transport)
	if _, err := installer.Install(context.Background(), "test", "github-issues"); err == nil {
		t.Fatal("Install with mismatched manifest id = nil error, want error")
	}
}

func TestInstallerRejectsBundleWithoutManifest(t *testing.T) {
	transport := &fakeTransport{
		registry: []byte(`{"version":1,"plugins":[{"id":"x","source":{"type":"path","path":"p"}}]}`),
		bundles:  map[string]map[string][]byte{"p": {"resolve.mjs": []byte("x")}},
	}
	installer, _, _ := newFixtureInstaller(t, transport)
	if _, err := installer.Install(context.Background(), "test", "x"); err == nil {
		t.Fatal("Install without plugin.json = nil error, want error")
	}
}

func TestInstallerReinstallReplacesFiles(t *testing.T) {
	transport := &fakeTransport{
		registry: []byte(`{"version":1,"plugins":[{"id":"x","source":{"type":"path","path":"p"}}]}`),
		bundles: map[string]map[string][]byte{
			"p": {"plugin.json": []byte(`{"id":"x"}`), "stale.mjs": []byte("old")},
		},
	}
	installer, target, _ := newFixtureInstaller(t, transport)
	if _, err := installer.Install(context.Background(), "test", "x"); err != nil {
		t.Fatalf("first install: %v", err)
	}
	// Second install drops stale.mjs.
	transport.bundles["p"] = map[string][]byte{"plugin.json": []byte(`{"id":"x"}`)}
	if _, err := installer.Install(context.Background(), "test", "x"); err != nil {
		t.Fatalf("reinstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "test", "x", "stale.mjs")); !os.IsNotExist(err) {
		t.Fatalf("stale file survived reinstall: err = %v", err)
	}
}
