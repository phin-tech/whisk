package pluginregistry_test

import (
	"testing"

	"github.com/phin-tech/whisk/internal/domain/pluginregistry"
)

func TestParseRegistryValidPathAndGit(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"plugins": [
			{"id": "github-issues", "name": "GitHub Issues", "source": {"type": "path", "path": "plugins/github-issues"}},
			{"id": "linear", "source": {"type": "git", "repo": "phin-tech/whisk-plugin-linear", "subdir": "plugin", "ref": "v0.2.0"}}
		]
	}`)
	registry, err := pluginregistry.ParseRegistry(data)
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	if len(registry.Plugins) != 2 {
		t.Fatalf("plugins = %d, want 2", len(registry.Plugins))
	}
	entry, ok := registry.Find("linear")
	if !ok {
		t.Fatal("Find(linear) not found")
	}
	if entry.Source.Type != pluginregistry.SourceGit || entry.Source.Ref != "v0.2.0" {
		t.Fatalf("linear source = %#v", entry.Source)
	}
}

func TestParseRegistryRejectsInvalid(t *testing.T) {
	cases := map[string]string{
		"missing id":        `{"plugins": [{"source": {"type": "path", "path": "x"}}]}`,
		"path without path": `{"plugins": [{"id": "a", "source": {"type": "path"}}]}`,
		"git without repo":  `{"plugins": [{"id": "a", "source": {"type": "git"}}]}`,
		"unknown type":      `{"plugins": [{"id": "a", "source": {"type": "svn", "path": "x"}}]}`,
		"missing type":      `{"plugins": [{"id": "a", "source": {"path": "x"}}]}`,
		"path traversal":    `{"plugins": [{"id": "a", "source": {"type": "path", "path": "../etc"}}]}`,
		"duplicate id":      `{"plugins": [{"id": "a", "source": {"type": "path", "path": "x"}}, {"id": "a", "source": {"type": "path", "path": "y"}}]}`,
		"malformed json":    `{`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := pluginregistry.ParseRegistry([]byte(doc)); err == nil {
				t.Fatalf("ParseRegistry(%s) = nil error, want error", name)
			}
		})
	}
}

func TestFindMissing(t *testing.T) {
	registry := pluginregistry.Registry{Plugins: []pluginregistry.Entry{{ID: "a", Source: pluginregistry.Source{Type: pluginregistry.SourcePath, Path: "x"}}}}
	if _, ok := registry.Find("nope"); ok {
		t.Fatal("Find(nope) = ok, want not found")
	}
}

func TestSortedPlugins(t *testing.T) {
	registry := pluginregistry.Registry{Plugins: []pluginregistry.Entry{
		{ID: "zeta", Source: pluginregistry.Source{Type: pluginregistry.SourcePath, Path: "z"}},
		{ID: "alpha", Source: pluginregistry.Source{Type: pluginregistry.SourcePath, Path: "a"}},
	}}
	sorted := registry.SortedPlugins()
	if sorted[0].ID != "alpha" || sorted[1].ID != "zeta" {
		t.Fatalf("sorted = %#v", sorted)
	}
}
