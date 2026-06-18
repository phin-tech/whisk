package pluginregistry_test

import (
	"testing"

	"github.com/phin-tech/whisk/internal/domain/pluginregistry"
)

func TestLockRoundTrip(t *testing.T) {
	lock, err := pluginregistry.ParseLock(nil)
	if err != nil {
		t.Fatalf("ParseLock(nil): %v", err)
	}
	lock = lock.Set(pluginregistry.LockEntry{
		Registry:    "phin-tech",
		ID:          "github-issues",
		Source:      pluginregistry.Source{Type: pluginregistry.SourcePath, Path: "plugins/github-issues"},
		Version:     "0.1.0",
		Fingerprint: "sha256:abc",
	})
	data, err := lock.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	reparsed, err := pluginregistry.ParseLock(data)
	if err != nil {
		t.Fatalf("ParseLock: %v", err)
	}
	entry, ok := reparsed.Get("phin-tech", "github-issues")
	if !ok || entry.Fingerprint != "sha256:abc" || entry.Version != "0.1.0" {
		t.Fatalf("entry = %#v ok = %v", entry, ok)
	}
}

func TestLockNamespacesByRegistry(t *testing.T) {
	lock := pluginregistry.Lock{}
	lock = lock.Set(pluginregistry.LockEntry{Registry: "phin-tech", ID: "github", Fingerprint: "1"})
	lock = lock.Set(pluginregistry.LockEntry{Registry: "acme", ID: "github", Fingerprint: "2"})

	// Same id, different registries: both coexist.
	if len(lock.Plugins) != 2 {
		t.Fatalf("plugins = %d, want 2", len(lock.Plugins))
	}
	phin, _ := lock.Get("phin-tech", "github")
	acme, _ := lock.Get("acme", "github")
	if phin.Fingerprint != "1" || acme.Fingerprint != "2" {
		t.Fatalf("phin = %#v acme = %#v", phin, acme)
	}
}

func TestLockSetReplacesAndSorts(t *testing.T) {
	lock := pluginregistry.Lock{}
	lock = lock.Set(pluginregistry.LockEntry{Registry: "r", ID: "zeta", Fingerprint: "1"})
	lock = lock.Set(pluginregistry.LockEntry{Registry: "r", ID: "alpha", Fingerprint: "2"})
	lock = lock.Set(pluginregistry.LockEntry{Registry: "r", ID: "zeta", Fingerprint: "updated"})

	if len(lock.Plugins) != 2 {
		t.Fatalf("plugins = %d, want 2", len(lock.Plugins))
	}
	if lock.Plugins[0].ID != "alpha" {
		t.Fatalf("not sorted: %#v", lock.Plugins)
	}
	zeta, _ := lock.Get("r", "zeta")
	if zeta.Fingerprint != "updated" {
		t.Fatalf("zeta not replaced: %#v", zeta)
	}
}
