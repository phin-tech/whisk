package skills

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	domainskills "github.com/phin-tech/whisk/internal/domain/skills"
)

const (
	SkillFileName         = domainskills.SkillFileName
	MaxSkillMarkdownBytes = 256 * 1024
	MaxDiscoveredSkills   = 200
	MaxSkillFiles         = 200
	MaxDirectoryEntries   = 256
	MaxVisitedDirectories = 1000
	MaxPackageDepth       = 8
	DefaultMaxDepth       = 4
	PluginMaxDepth        = 9
)

type Source = domainskills.Source
type DiscoverySource = domainskills.DiscoverySource
type Skill = domainskills.Skill
type DiscoveryResult = domainskills.DiscoveryResult
type Metadata = domainskills.Metadata
type SourceKind = domainskills.SourceKind
type Provider = domainskills.Provider

const (
	SourceKindBundled = domainskills.SourceKindBundled
	SourceKindHome    = domainskills.SourceKindHome
	SourceKindProject = domainskills.SourceKindProject
	SourceKindPlugin  = domainskills.SourceKindPlugin
)

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
			ID:            domainskills.StablePathID(skillFilePath),
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
		if len(out) >= MaxDiscoveredSkills || len(visitedDirs) >= MaxVisitedDirectories {
			return
		}
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

		entries, err := readDirLimited(dirPath)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if len(out) >= MaxDiscoveredSkills {
				return
			}
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
		if count >= MaxSkillFiles || len(visitedDirs) >= MaxVisitedDirectories || !isWithinDepth(dirPath, currentPath, MaxPackageDepth) {
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

		entries, err := readDirLimited(currentPath)
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
	return domainskills.SummarizeMarkdown(string(data)), info.ModTime()
}

func readDirLimited(dirPath string) ([]os.DirEntry, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(MaxDirectoryEntries)
	if err != nil && err != io.EOF {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	return entries, nil
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
