package skills

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	SkillFileName         = "SKILL.md"
	MaxSkillMarkdownBytes = 256 * 1024
	MaxSkillFiles         = 200
	DefaultMaxDepth       = 4
	PluginMaxDepth        = 9
)

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

// Discover scans all sources and returns a stable, read-only skill catalog.
func Discover(sources []Source, scannedAt time.Time) DiscoveryResult {
	result := DiscoveryResult{ScannedAt: scannedAt}
	seen := map[string]Skill{}
	for _, source := range sources {
		if strings.TrimSpace(source.Path) == "" {
			result.Sources = append(result.Sources, DiscoverySource{Source: source, SkippedReason: "missing"})
			continue
		}
		source.Path = filepath.Clean(source.Path)
		discoverySource := DiscoverySource{Source: source}
		rootInfo, err := os.Stat(source.Path)
		if os.IsNotExist(err) {
			discoverySource.SkippedReason = "missing"
			result.Sources = append(result.Sources, discoverySource)
			continue
		}
		if err != nil || !rootInfo.IsDir() {
			discoverySource.SkippedReason = "unreadable"
			result.Sources = append(result.Sources, discoverySource)
			continue
		}
		discoverySource.Exists = true
		result.Sources = append(result.Sources, discoverySource)
		for _, skill := range scanSource(source) {
			if _, ok := seen[skill.SkillFilePath]; ok {
				continue
			}
			seen[skill.SkillFilePath] = skill
		}
	}
	for _, skill := range seen {
		result.Skills = append(result.Skills, skill)
	}
	sortSkills(result.Skills)
	sort.Slice(result.Sources, func(i, j int) bool {
		return strings.ToLower(result.Sources[i].Label) < strings.ToLower(result.Sources[j].Label)
	})
	return result
}

func scanSource(source Source) []Skill {
	rootReal, err := filepath.EvalSymlinks(source.Path)
	if err != nil {
		return nil
	}
	paths := findSkillFiles(source.Path, rootReal, maxDepth(source))
	skills := make([]Skill, 0, len(paths))
	for _, skillFilePath := range paths {
		directoryPath := filepath.Dir(skillFilePath)
		metadata, updatedAt := readSkillMetadata(skillFilePath)
		if metadata.Name == "" {
			metadata.Name = filepath.Base(directoryPath)
		}
		sourceKind := sourceKindForSkill(source, skillFilePath)
		skills = append(skills, Skill{
			ID:            StablePathID(skillFilePath),
			Name:          metadata.Name,
			Description:   metadata.Description,
			Providers:     append([]Provider(nil), source.Providers...),
			SourceKind:    sourceKind,
			SourceLabel:   sourceLabelForSkill(source, sourceKind),
			RootPath:      source.Path,
			DirectoryPath: directoryPath,
			SkillFilePath: skillFilePath,
			FileCount:     countFiles(directoryPath, rootReal),
			UpdatedAt:     updatedAt,
		})
	}
	return skills
}

func maxDepth(source Source) int {
	if source.MaxDepth > 0 {
		return source.MaxDepth
	}
	if source.Kind == SourceKindPlugin {
		return PluginMaxDepth
	}
	return DefaultMaxDepth
}

func findSkillFiles(rootPath, rootReal string, maxDepth int) []string {
	var out []string
	visitedDirs := map[string]struct{}{}
	var visit func(string)
	visit = func(dirPath string) {
		if !isWithinDepth(rootPath, dirPath, maxDepth) {
			return
		}
		dirReal, err := filepath.EvalSymlinks(dirPath)
		if err != nil || !isWithinRoot(rootReal, dirReal) {
			return
		}
		if _, ok := visitedDirs[dirReal]; ok {
			return
		}
		visitedDirs[dirReal] = struct{}{}

		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return
		}
		for _, entry := range entries {
			entryPath := filepath.Join(dirPath, entry.Name())
			if entry.Name() == SkillFileName {
				if isRegularFileWithinRoot(entryPath, rootReal) {
					out = append(out, entryPath)
				}
				continue
			}
			if isDirectory(entryPath, entry) {
				visit(entryPath)
			}
		}
	}
	visit(rootPath)
	sort.Strings(out)
	return out
}

func countFiles(dirPath, rootReal string) int {
	count := 0
	visitedDirs := map[string]struct{}{}
	var visit func(string)
	visit = func(currentPath string) {
		if count >= MaxSkillFiles {
			return
		}
		currentReal, err := filepath.EvalSymlinks(currentPath)
		if err != nil || !isWithinRoot(rootReal, currentReal) {
			return
		}
		if _, ok := visitedDirs[currentReal]; ok {
			return
		}
		visitedDirs[currentReal] = struct{}{}

		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if count >= MaxSkillFiles {
				return
			}
			entryPath := filepath.Join(currentPath, entry.Name())
			switch {
			case isRegularFileWithinRoot(entryPath, rootReal):
				count++
			case isDirectory(entryPath, entry):
				visit(entryPath)
			}
		}
	}
	visit(dirPath)
	return count
}

func readSkillMetadata(path string) (Metadata, time.Time) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return Metadata{}, time.Time{}
	}
	file, err := os.Open(path)
	if err != nil {
		return Metadata{}, info.ModTime()
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxSkillMarkdownBytes))
	if err != nil {
		return Metadata{}, info.ModTime()
	}
	return SummarizeMarkdown(string(data)), info.ModTime()
}

func isDirectory(path string, entry os.DirEntry) bool {
	if entry.IsDir() {
		return true
	}
	if entry.Type()&os.ModeSymlink == 0 {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isRegularFileWithinRoot(path, rootReal string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	return isWithinRoot(rootReal, realPath)
}

func isWithinRoot(root, child string) bool {
	rel, err := filepath.Rel(root, child)
	if err != nil {
		return false
	}
	return !pathEscapes(rel)
}

func isWithinDepth(rootPath, childPath string, maxDepth int) bool {
	rel, err := filepath.Rel(rootPath, childPath)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if pathEscapes(rel) {
		return false
	}
	return len(splitPath(rel)) <= maxDepth
}

func pathEscapes(rel string) bool {
	if rel == ".." || filepath.IsAbs(rel) {
		return true
	}
	return strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func splitPath(rel string) []string {
	parts := strings.Split(rel, string(os.PathSeparator))
	out := parts[:0]
	for _, part := range parts {
		if part != "" && part != "." {
			out = append(out, part)
		}
	}
	return out
}

func sourceKindForSkill(source Source, skillFilePath string) SourceKind {
	if source.Kind != SourceKindHome {
		return source.Kind
	}
	rel, err := filepath.Rel(source.Path, skillFilePath)
	if err != nil {
		return source.Kind
	}
	parts := splitPath(rel)
	if len(parts) > 0 && parts[0] == ".system" {
		return SourceKindBundled
	}
	return source.Kind
}

func sourceLabelForSkill(source Source, kind SourceKind) string {
	if kind == SourceKindBundled && source.Kind != SourceKindBundled {
		return source.Label + " bundled"
	}
	return source.Label
}

func sortSkills(skills []Skill) {
	sort.Slice(skills, func(i, j int) bool {
		left := strings.ToLower(skills[i].Name)
		right := strings.ToLower(skills[j].Name)
		if left != right {
			return left < right
		}
		left = strings.ToLower(skills[i].SourceLabel)
		right = strings.ToLower(skills[j].SourceLabel)
		if left != right {
			return left < right
		}
		return skills[i].SkillFilePath < skills[j].SkillFilePath
	})
}
