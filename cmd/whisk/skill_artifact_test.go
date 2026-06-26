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

func TestMakeWhiskWorkflowSkillArtifactGuidesWorkflowCreation(t *testing.T) {
	root := filepath.Join("..", "..", "skills", "make-whisk-workflow")
	skill, err := os.ReadFile(filepath.Join(root, "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	skillText := string(skill)
	for _, want := range []string{
		"name: make-whisk-workflow",
		"description:",
		"version:",
		"questions",
		"workflow definition",
		"gates",
	} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("SKILL.md missing %q", want)
		}
	}

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readmeText := string(readme)
	for _, want := range []string{
		"WorkflowDefinition",
		"plan-execute-review",
		"promptTemplateId",
		"requiresHuman",
		"questions",
	} {
		if !strings.Contains(readmeText, want) {
			t.Fatalf("README.md missing %q", want)
		}
	}
}

func TestDarwinBundleCopiesEveryBundledSkill(t *testing.T) {
	taskfile, err := os.ReadFile(filepath.Join("..", "..", "build", "darwin", "Taskfile.yml"))
	if err != nil {
		t.Fatalf("read Taskfile: %v", err)
	}
	text := string(taskfile)
	for _, skill := range []string{"whisk", "make-whisk-workflow"} {
		for _, file := range []string{"SKILL.md", "README.md"} {
			want := filepath.ToSlash(filepath.Join("skills", skill, file))
			if !strings.Contains(text, want) {
				t.Fatalf("bundle task does not copy %s", want)
			}
		}
	}
}
