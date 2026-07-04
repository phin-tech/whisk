package skills

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSummarizeMarkdownParsesYAMLSubsetFrontmatter(t *testing.T) {
	markdown := strings.Join([]string{
		"\ufeff---",
		`name: "code-review"`,
		"description: >-",
		"  Review code changes",
		"  with runtime-boundary focus.",
		"providers:",
		"  - codex",
		"  - claude",
		"---",
		"# Ignored Heading",
		"",
		"Ignored paragraph.",
	}, "\r\n")

	summary := SummarizeMarkdown(markdown)
	if summary.Name != "code-review" {
		t.Fatalf("name = %q, want code-review", summary.Name)
	}
	if summary.Description != "Review code changes with runtime-boundary focus." {
		t.Fatalf("description = %q", summary.Description)
	}
}

func TestSummarizeMarkdownFallsBackToHeadingAndParagraph(t *testing.T) {
	summary := SummarizeMarkdown("# Docs\n\nWrite project docs.\nKeep them current.\n\nNext paragraph.")
	if summary.Name != "Docs" {
		t.Fatalf("name = %q, want Docs", summary.Name)
	}
	if summary.Description != "Write project docs. Keep them current." {
		t.Fatalf("description = %q", summary.Description)
	}
}

func TestSummarizeMarkdownCapsLongFallbackDescription(t *testing.T) {
	summary := SummarizeMarkdown("# Huge\n\n" + strings.Repeat("a", MaxFallbackDescriptionBytes+1024))
	if len(summary.Description) != MaxFallbackDescriptionBytes {
		t.Fatalf("description length = %d, want %d", len(summary.Description), MaxFallbackDescriptionBytes)
	}
}

func TestDefaultSourcesBuildsDaemonSkillRoots(t *testing.T) {
	home := filepath.Join("home")
	project := filepath.Join("repo")
	bundled := filepath.Join("bundle", "skills")

	sources := DefaultSources(SourceConfig{
		HomeDir:      home,
		ProjectPaths: []string{project, project},
		BundledPath:  bundled,
	})

	got := map[string]Source{}
	for _, source := range sources {
		got[source.ID] = source
	}

	want := map[string]Source{
		"bundled":            {ID: "bundled", Label: "Whisk bundled", Path: bundled, Kind: SourceKindBundled, Providers: []Provider{ProviderCodex, ProviderClaude, ProviderAgentSkills}},
		"home-codex":         {ID: "home-codex", Label: "Codex home", Path: filepath.Join(home, ".codex", "skills"), Kind: SourceKindHome, Providers: []Provider{ProviderCodex}},
		"codex-plugin-cache": {ID: "codex-plugin-cache", Label: "Codex plugin cache", Path: filepath.Join(home, ".codex", "plugins", "cache"), Kind: SourceKindPlugin, Providers: []Provider{ProviderCodex, ProviderAgentSkills}},
		"home-claude":        {ID: "home-claude", Label: "Claude home", Path: filepath.Join(home, ".claude", "skills"), Kind: SourceKindHome, Providers: []Provider{ProviderClaude}},
		"home-agents":        {ID: "home-agents", Label: "Agent skills home", Path: filepath.Join(home, ".agents", "skills"), Kind: SourceKindHome, Providers: []Provider{ProviderAgentSkills}},
		"project-skills-" + StablePathID(project): {ID: "project-skills-" + StablePathID(project), Label: "Project repo skills", Path: filepath.Join(project, "skills"), Kind: SourceKindProject, Providers: []Provider{ProviderCodex, ProviderAgentSkills}},
		"project-claude-" + StablePathID(project): {ID: "project-claude-" + StablePathID(project), Label: "Project repo .claude", Path: filepath.Join(project, ".claude", "skills"), Kind: SourceKindProject, Providers: []Provider{ProviderClaude}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sources = %#v, want %#v", got, want)
	}
}
