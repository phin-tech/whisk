// Package pluginregistry holds the pure model for the Whisk plugin registry:
// the catalog of installable plugins, the per-install lockfile, and the
// deterministic bundle fingerprint used to record what was installed.
//
// It performs no I/O. Fetching catalog/plugin bytes and writing files to disk
// lives in the imperative shell (internal/adapters/plugins).
package pluginregistry

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SourceType identifies where a registry entry's files come from.
type SourceType string

const (
	// SourcePath points at a directory inside the registry repo itself
	// (monorepo style). Path is relative to the registry repo root.
	SourcePath SourceType = "path"
	// SourceGit points at a directory inside a separate git repository.
	SourceGit SourceType = "git"
)

// Source describes where a plugin's files live.
type Source struct {
	Type SourceType `json:"type"`
	// Path is the directory within the registry repo, for SourcePath.
	Path string `json:"path,omitempty"`
	// Repo is the external repository ("owner/repo" or a full URL), for SourceGit.
	Repo string `json:"repo,omitempty"`
	// Subdir is the directory within Repo, for SourceGit (empty = repo root).
	Subdir string `json:"subdir,omitempty"`
	// Ref is the branch, tag, or commit to fetch, for SourceGit (empty = default branch).
	Ref string `json:"ref,omitempty"`
}

// Entry is one installable plugin advertised by the registry.
type Entry struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Source      Source `json:"source"`
}

// Registry is the parsed catalog (registry.json).
type Registry struct {
	Version int     `json:"version"`
	Plugins []Entry `json:"plugins"`
}

// ParseRegistry decodes and validates a registry.json document.
func ParseRegistry(data []byte) (Registry, error) {
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return Registry{}, fmt.Errorf("parse registry: %w", err)
	}
	if err := registry.Validate(); err != nil {
		return Registry{}, err
	}
	return registry, nil
}

// Validate checks that every entry is well-formed and ids are unique.
func (r Registry) Validate() error {
	seen := map[string]bool{}
	for _, entry := range r.Plugins {
		if err := entry.Validate(); err != nil {
			return err
		}
		if seen[entry.ID] {
			return fmt.Errorf("duplicate plugin id %q", entry.ID)
		}
		seen[entry.ID] = true
	}
	return nil
}

// Find returns the entry with the given id.
func (r Registry) Find(id string) (Entry, bool) {
	id = strings.TrimSpace(id)
	for _, entry := range r.Plugins {
		if entry.ID == id {
			return entry, true
		}
	}
	return Entry{}, false
}

// Validate checks that an entry has an id and a coherent source.
func (e Entry) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return fmt.Errorf("registry entry is missing an id")
	}
	switch e.Source.Type {
	case SourcePath:
		if strings.TrimSpace(e.Source.Path) == "" {
			return fmt.Errorf("plugin %q: path source requires a path", e.ID)
		}
		if strings.Contains(e.Source.Path, "..") {
			return fmt.Errorf("plugin %q: path must not escape the registry root", e.ID)
		}
	case SourceGit:
		if strings.TrimSpace(e.Source.Repo) == "" {
			return fmt.Errorf("plugin %q: git source requires a repo", e.ID)
		}
		if strings.Contains(e.Source.Subdir, "..") {
			return fmt.Errorf("plugin %q: subdir must not escape the repo root", e.ID)
		}
	case "":
		return fmt.Errorf("plugin %q: source type is required", e.ID)
	default:
		return fmt.Errorf("plugin %q: unknown source type %q", e.ID, e.Source.Type)
	}
	return nil
}

// SortedPlugins returns the entries ordered by id, for stable presentation.
func (r Registry) SortedPlugins() []Entry {
	out := append([]Entry(nil), r.Plugins...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
