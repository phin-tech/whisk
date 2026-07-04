package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	domainskills "github.com/phin-tech/whisk/internal/domain/skills"
)

func TestDiscoverFindsSkillsWithProviderMetadata(t *testing.T) {
	root := t.TempDir()
	codexRoot := filepath.Join(root, "home", ".codex", "skills")
	projectRoot := filepath.Join(root, "repo", "skills")
	writeFile(t, filepath.Join(codexRoot, "review", SkillFileName), "---\nname: code-review\ndescription: Review code.\n---\n")
	writeFile(t, filepath.Join(projectRoot, "docs", SkillFileName), "# Docs\n\nWrite project docs.")

	result := Discover([]Source{
		{ID: "home-codex", Label: "Codex home", Path: codexRoot, Kind: SourceKindHome, Providers: []Provider{domainskills.ProviderCodex}},
		{ID: "project-skills", Label: "Project skills", Path: projectRoot, Kind: SourceKindProject, Providers: []Provider{domainskills.ProviderCodex, domainskills.ProviderAgentSkills}},
		{ID: "missing", Label: "Missing", Path: filepath.Join(root, "missing"), Kind: SourceKindProject},
	}, time.Unix(123, 0))

	names := skillNames(result.Skills)
	if !reflect.DeepEqual(names, []string{"code-review", "Docs"}) {
		t.Fatalf("skill names = %#v", names)
	}
	review := findSkill(result.Skills, "code-review")
	if review.SourceKind != SourceKindHome || review.SourceLabel != "Codex home" || !reflect.DeepEqual(review.Providers, []Provider{domainskills.ProviderCodex}) {
		t.Fatalf("review skill = %#v", review)
	}
	docs := findSkill(result.Skills, "Docs")
	if docs.SourceKind != SourceKindProject || !reflect.DeepEqual(docs.Providers, []Provider{domainskills.ProviderCodex, domainskills.ProviderAgentSkills}) {
		t.Fatalf("docs skill = %#v", docs)
	}
	if got := sourceByID(result.Sources, "missing"); got.Exists || got.SkippedReason == "" {
		t.Fatalf("missing source status = %#v", got)
	}
	if !result.ScannedAt.Equal(time.Unix(123, 0)) {
		t.Fatalf("scannedAt = %v", result.ScannedAt)
	}
}

func TestDiscoverEnforcesDepthLimitWithoutRejectingDotDotCacheNames(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "skills")
	writeFile(t, filepath.Join(skillRoot, "..cache", "a", "b", "c", SkillFileName), "# Cache\n\nValid child directory.")
	writeFile(t, filepath.Join(skillRoot, "deep", "a", "b", "c", "d", SkillFileName), "# Too Deep\n\nShould not be discovered.")

	result := Discover([]Source{{
		ID:       "project-skills",
		Label:    "Project skills",
		Path:     skillRoot,
		Kind:     SourceKindProject,
		MaxDepth: 4,
	}}, time.Time{})

	names := skillNames(result.Skills)
	if !reflect.DeepEqual(names, []string{"Cache"}) {
		t.Fatalf("skill names = %#v, want Cache only", names)
	}
}

func TestDiscoverSkipsSymlinkRealpathEscapes(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "skills")
	outside := filepath.Join(root, "outside")
	writeFile(t, filepath.Join(skillRoot, "local", SkillFileName), "# Local\n\nAllowed.")
	writeFile(t, filepath.Join(outside, "escape", SkillFileName), "# Escape\n\nShould not be discovered.")
	if err := os.Symlink(filepath.Join(outside, "escape"), filepath.Join(skillRoot, "linked-escape")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	result := Discover([]Source{{
		ID:    "project-skills",
		Label: "Project skills",
		Path:  skillRoot,
		Kind:  SourceKindProject,
	}}, time.Time{})

	names := skillNames(result.Skills)
	if !reflect.DeepEqual(names, []string{"Local"}) {
		t.Fatalf("skill names = %#v, want Local only", names)
	}
}

func TestDiscoverSkipsSymlinkedSkillFilesOutsideRoot(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "skills")
	outsideSkill := filepath.Join(root, "outside", SkillFileName)
	writeFile(t, outsideSkill, "# Escape\n\nShould not be discovered.")
	if err := os.MkdirAll(filepath.Join(skillRoot, "linked-file"), 0o755); err != nil {
		t.Fatalf("mkdir linked-file: %v", err)
	}
	if err := os.Symlink(outsideSkill, filepath.Join(skillRoot, "linked-file", SkillFileName)); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	result := Discover([]Source{{
		ID:    "project-skills",
		Label: "Project skills",
		Path:  skillRoot,
		Kind:  SourceKindProject,
	}}, time.Time{})

	if len(result.Skills) != 0 {
		t.Fatalf("skills = %#v, want none", result.Skills)
	}
}

func TestDiscoverDoesNotLoopThroughRecursiveSymlinkedDirectories(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "skills")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatalf("mkdir skill root: %v", err)
	}
	if err := os.Symlink(skillRoot, filepath.Join(skillRoot, "loop")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	result := Discover([]Source{{
		ID:    "project-skills",
		Label: "Project skills",
		Path:  skillRoot,
		Kind:  SourceKindProject,
	}}, time.Time{})

	if len(result.Skills) != 0 {
		t.Fatalf("skills = %#v, want none", result.Skills)
	}
}

func TestDiscoverCapsDiscoveredSkillFiles(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "skills")
	for i := 0; i < MaxDiscoveredSkills+5; i++ {
		writeFile(t, filepath.Join(skillRoot, formatNumber(i), SkillFileName), "# Skill\n\nDescription.")
	}

	result := Discover([]Source{{
		ID:    "project-skills",
		Label: "Project skills",
		Path:  skillRoot,
		Kind:  SourceKindProject,
	}}, time.Time{})

	if len(result.Skills) != MaxDiscoveredSkills {
		t.Fatalf("skills = %d, want cap %d", len(result.Skills), MaxDiscoveredSkills)
	}
}

func TestDiscoverCapsSkillMarkdownRead(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "skills")
	lateHeading := strings.Repeat("a", MaxSkillMarkdownBytes+1024) + "\n# Too Late\n"
	writeFile(t, filepath.Join(skillRoot, "large", SkillFileName), lateHeading)

	result := Discover([]Source{{
		ID:    "project-skills",
		Label: "Project skills",
		Path:  skillRoot,
		Kind:  SourceKindProject,
	}}, time.Time{})

	if len(result.Skills) != 1 {
		t.Fatalf("skills = %#v, want one skill", result.Skills)
	}
	if result.Skills[0].Name != "large" {
		t.Fatalf("name = %q, want fallback directory name", result.Skills[0].Name)
	}
	if len(result.Skills[0].Description) > domainskills.MaxFallbackDescriptionBytes {
		t.Fatalf("description length = %d", len(result.Skills[0].Description))
	}
}

func TestDiscoverCountFilesHonorsPackageDepth(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "skills")
	skillDir := filepath.Join(skillRoot, "package")
	writeFile(t, filepath.Join(skillDir, SkillFileName), "# Package\n\nDescription.")
	deep := skillDir
	for i := 0; i < MaxPackageDepth+1; i++ {
		deep = filepath.Join(deep, formatNumber(i))
	}
	writeFile(t, filepath.Join(deep, "ignored.md"), "too deep")

	result := Discover([]Source{{
		ID:    "project-skills",
		Label: "Project skills",
		Path:  skillRoot,
		Kind:  SourceKindProject,
	}}, time.Time{})

	if len(result.Skills) != 1 {
		t.Fatalf("skills = %#v, want one skill", result.Skills)
	}
	if result.Skills[0].FileCount != 1 {
		t.Fatalf("file count = %d, want only root SKILL.md counted", result.Skills[0].FileCount)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func skillNames(skills []Skill) []string {
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		names = append(names, skill.Name)
	}
	slices.SortFunc(names, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	return names
}

func findSkill(skills []Skill, name string) Skill {
	for _, skill := range skills {
		if skill.Name == name {
			return skill
		}
	}
	return Skill{}
}

func sourceByID(sources []DiscoverySource, id string) DiscoverySource {
	for _, source := range sources {
		if source.ID == id {
			return source
		}
	}
	return DiscoverySource{}
}

func formatNumber(n int) string {
	return fmt.Sprintf("%03d", n)
}
