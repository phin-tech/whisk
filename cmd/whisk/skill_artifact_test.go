package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWhiskSkillArtifactsCoverCLI(t *testing.T) {
	root := filepath.Join("..", "..", "skills", "whisk")
	skill, err := os.ReadFile(filepath.Join(root, "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	skillText := string(skill)
	if !strings.Contains(skillText, "name: whisk") || !strings.Contains(skillText, "description:") {
		t.Fatalf("SKILL.md frontmatter is incomplete")
	}

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readmeText := string(readme)
	for _, command := range []string{
		"whisk daemon",
		"whisk forward",
		"whisk session",
		"whisk session pty",
		"whisk project",
		"whisk work-item",
		"whisk run",
		"whisk workflow",
		"whisk question",
		"whisk gate",
		"whisk status",
		"whisk agent-bridge",
		"whisk plugin",
	} {
		if !strings.Contains(readmeText, command) {
			t.Fatalf("README.md does not cover %q", command)
		}
	}
}
