package pluginregistry

import (
	"encoding/json"
	"fmt"
	"sort"
)

// LockEntry records exactly what was materialized for one installed plugin so
// installs are reproducible and auditable.
type LockEntry struct {
	ID          string `json:"id"`
	Source      Source `json:"source"`
	Version     string `json:"version,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

// Lock is the install lockfile (plugins.lock.json).
type Lock struct {
	Version int         `json:"version"`
	Plugins []LockEntry `json:"plugins"`
}

const lockVersion = 1

// ParseLock decodes a lockfile. An empty input yields an empty lock so callers
// can treat "no lockfile yet" the same as "lockfile with no entries".
func ParseLock(data []byte) (Lock, error) {
	if len(data) == 0 {
		return Lock{Version: lockVersion}, nil
	}
	var lock Lock
	if err := json.Unmarshal(data, &lock); err != nil {
		return Lock{}, fmt.Errorf("parse lock: %w", err)
	}
	if lock.Version == 0 {
		lock.Version = lockVersion
	}
	return lock, nil
}

// Get returns the lock entry for the given id.
func (l Lock) Get(id string) (LockEntry, bool) {
	for _, entry := range l.Plugins {
		if entry.ID == id {
			return entry, true
		}
	}
	return LockEntry{}, false
}

// Set inserts or replaces the entry for entry.ID and returns the updated lock.
// Entries are kept sorted by id so the serialized lockfile is stable.
func (l Lock) Set(entry LockEntry) Lock {
	next := make([]LockEntry, 0, len(l.Plugins)+1)
	replaced := false
	for _, existing := range l.Plugins {
		if existing.ID == entry.ID {
			next = append(next, entry)
			replaced = true
			continue
		}
		next = append(next, existing)
	}
	if !replaced {
		next = append(next, entry)
	}
	sort.Slice(next, func(i, j int) bool { return next[i].ID < next[j].ID })
	version := l.Version
	if version == 0 {
		version = lockVersion
	}
	return Lock{Version: version, Plugins: next}
}

// Marshal renders the lockfile as indented JSON with a trailing newline.
func (l Lock) Marshal() ([]byte, error) {
	if l.Version == 0 {
		l.Version = lockVersion
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
