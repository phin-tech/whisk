package skills

import (
	"crypto/sha1"
	"encoding/hex"
	"path/filepath"
	"strings"
	"time"
)

const SkillFileName = "SKILL.md"

type Provider string

const (
	ProviderCodex       Provider = "codex"
	ProviderClaude      Provider = "claude"
	ProviderAgentSkills Provider = "agent-skills"
)

type SourceKind string

const (
	SourceKindBundled SourceKind = "bundled"
	SourceKindHome    SourceKind = "home"
	SourceKindProject SourceKind = "project"
	SourceKindPlugin  SourceKind = "plugin"
)

// Source is a filesystem root that may contain one or more SKILL.md packages.
type Source struct {
	ID        string
	Label     string
	Path      string
	Kind      SourceKind
	Providers []Provider
	MaxDepth  int
}

// DiscoverySource records whether a source root was present during a scan.
type DiscoverySource struct {
	Source
	Exists        bool
	SkippedReason string
}

// Skill is the daemon read-model entry for one discovered SKILL.md package.
type Skill struct {
	ID            string
	Name          string
	Description   string
	Providers     []Provider
	SourceKind    SourceKind
	SourceLabel   string
	RootPath      string
	DirectoryPath string
	SkillFilePath string
	FileCount     int
	UpdatedAt     time.Time
}

// DiscoveryResult is the deterministic skill catalog produced from source roots.
type DiscoveryResult struct {
	Skills    []Skill
	Sources   []DiscoverySource
	ScannedAt time.Time
}

// SourceConfig builds the known daemon-side roots for later runtime wiring.
type SourceConfig struct {
	HomeDir      string
	ProjectPaths []string
	BundledPath  string
}

// DefaultSources returns the source roots Whisk expects the daemon to scan.
func DefaultSources(config SourceConfig) []Source {
	var out []Source
	if strings.TrimSpace(config.BundledPath) != "" {
		out = append(out, Source{
			ID:        "bundled",
			Label:     "Whisk bundled",
			Path:      filepath.Clean(config.BundledPath),
			Kind:      SourceKindBundled,
			Providers: []Provider{ProviderCodex, ProviderClaude, ProviderAgentSkills},
		})
	}
	home := strings.TrimSpace(config.HomeDir)
	if home != "" {
		out = append(out,
			Source{ID: "home-codex", Label: "Codex home", Path: filepath.Join(home, ".codex", "skills"), Kind: SourceKindHome, Providers: []Provider{ProviderCodex}},
			Source{ID: "codex-plugin-cache", Label: "Codex plugin cache", Path: filepath.Join(home, ".codex", "plugins", "cache"), Kind: SourceKindPlugin, Providers: []Provider{ProviderCodex, ProviderAgentSkills}},
			Source{ID: "home-claude", Label: "Claude home", Path: filepath.Join(home, ".claude", "skills"), Kind: SourceKindHome, Providers: []Provider{ProviderClaude}},
			Source{ID: "home-agents", Label: "Agent skills home", Path: filepath.Join(home, ".agents", "skills"), Kind: SourceKindHome, Providers: []Provider{ProviderAgentSkills}},
		)
	}

	seenProjects := map[string]struct{}{}
	for _, projectPath := range config.ProjectPaths {
		projectPath = strings.TrimSpace(projectPath)
		if projectPath == "" {
			continue
		}
		clean := filepath.Clean(projectPath)
		if _, ok := seenProjects[clean]; ok {
			continue
		}
		seenProjects[clean] = struct{}{}
		base := filepath.Base(clean)
		id := StablePathID(clean)
		out = append(out,
			Source{ID: "project-skills-" + id, Label: "Project " + base + " skills", Path: filepath.Join(clean, "skills"), Kind: SourceKindProject, Providers: []Provider{ProviderCodex, ProviderAgentSkills}},
			Source{ID: "project-claude-" + id, Label: "Project " + base + " .claude", Path: filepath.Join(clean, ".claude", "skills"), Kind: SourceKindProject, Providers: []Provider{ProviderClaude}},
		)
	}
	return out
}

// StablePathID returns a short deterministic ID for source paths.
func StablePathID(path string) string {
	sum := sha1.Sum([]byte(path))
	return hex.EncodeToString(sum[:])[:16]
}
