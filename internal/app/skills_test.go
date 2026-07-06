package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
	domainskills "github.com/phin-tech/whisk/internal/domain/skills"
)

func TestRuntimeListSkillsScansConfiguredAndProjectRoots(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	bundled := filepath.Join(root, "bundled")
	projectRoot := filepath.Join(root, "repo")
	writeSkillFile(t, filepath.Join(home, ".codex", "skills", "review", "SKILL.md"), "---\nname: code-review\ndescription: Review code.\n---\n")
	writeSkillFile(t, filepath.Join(bundled, "whisk", "SKILL.md"), "---\nname: whisk\ndescription: Use Whisk.\n---\n")
	writeSkillFile(t, filepath.Join(projectRoot, "skills", "docs", "SKILL.md"), "# Docs\n\nWrite docs.")

	runtime := app.NewRuntime(app.RuntimeConfig{
		SkillHomeDir:    home,
		BundledSkillDir: bundled,
		SkillNow:        func() time.Time { return time.Unix(123, 0).UTC() },
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	project, err := runtime.CreateProject(context.Background(), app.CreateProjectRequest{Name: "Repo", RootDir: projectRoot})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	catalog, err := runtime.ListSkills(context.Background(), app.ListSkillsRequest{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if !catalog.ScannedAt.Equal(time.Unix(123, 0).UTC()) {
		t.Fatalf("scanned at = %s", catalog.ScannedAt)
	}
	names := map[string]bool{}
	for _, skill := range catalog.Skills {
		names[skill.Name] = true
	}
	for _, name := range []string{"code-review", "whisk", "Docs"} {
		if !names[name] {
			t.Fatalf("missing skill %q from %#v", name, catalog.Skills)
		}
	}
}

func TestRuntimeListSkillsRejectsUnknownScope(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{SkillHomeDir: t.TempDir()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	if _, err := runtime.ListSkills(context.Background(), app.ListSkillsRequest{ProjectID: "missing"}); err == nil {
		t.Fatalf("expected missing project error")
	}
	if _, err := runtime.ListSkills(context.Background(), app.ListSkillsRequest{SessionID: "missing"}); err == nil {
		t.Fatalf("expected missing session error")
	}
}

func TestRuntimeRescanSkillsUsesConfiguredSourcesAndProjectRoots(t *testing.T) {
	root := t.TempDir()
	customRoot := filepath.Join(root, "custom")
	projectRoot := filepath.Join(root, "repo")
	writeSkillFile(t, filepath.Join(customRoot, "ops", "SKILL.md"), "---\nname: ops\ndescription: Run ops.\n---\n")
	writeSkillFile(t, filepath.Join(projectRoot, ".claude", "skills", "docs", "SKILL.md"), "# Docs\n\nWrite docs.")

	runtime := app.NewRuntime(app.RuntimeConfig{
		SkillSources: []domainskills.Source{{
			ID:        "custom",
			Label:     "Custom",
			Path:      customRoot,
			Kind:      domainskills.SourceKindPlugin,
			Providers: []domainskills.Provider{domainskills.ProviderCodex},
		}},
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	project, err := runtime.CreateProject(context.Background(), app.CreateProjectRequest{Name: "Repo", RootDir: projectRoot})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	catalog, err := runtime.RescanSkills(context.Background(), app.ListSkillsRequest{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("rescan skills: %v", err)
	}
	byName := map[string]domainskills.Skill{}
	for _, skill := range catalog.Skills {
		byName[skill.Name] = skill
	}
	if byName["ops"].SourceKind != domainskills.SourceKindPlugin ||
		byName["Docs"].SourceKind != domainskills.SourceKindProject {
		t.Fatalf("skills = %#v", catalog.Skills)
	}
}

func TestRuntimeListSkillsUsesDefaultHomeDir(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	writeSkillFile(t, filepath.Join(home, ".agents", "skills", "triage", "SKILL.md"), "---\nname: triage\ndescription: Triage issues.\n---\n")

	runtime := app.NewRuntime(app.RuntimeConfig{
		BundledSkillDir: filepath.Join(root, "missing-bundled"),
		SkillNow:        func() time.Time { return time.Unix(456, 0).UTC() },
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	catalog, err := runtime.ListSkills(context.Background(), app.ListSkillsRequest{})
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	found := false
	for _, skill := range catalog.Skills {
		if skill.Name == "triage" && skill.SourceKind == domainskills.SourceKindHome {
			found = true
		}
	}
	if !found {
		t.Fatalf("default home skill missing from %#v", catalog.Skills)
	}
	if !catalog.ScannedAt.Equal(time.Unix(456, 0).UTC()) {
		t.Fatalf("scanned at = %s", catalog.ScannedAt)
	}
}

func TestRuntimeListSkillsIncludesSessionAndLinkedProjectRoots(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	sessionRoot := filepath.Join(root, "session")
	projectRoot := filepath.Join(root, "repo")
	writeSkillFile(t, filepath.Join(sessionRoot, "skills", "session", "SKILL.md"), "---\nname: session-skill\ndescription: Session skill.\n---\n")
	writeSkillFile(t, filepath.Join(projectRoot, ".claude", "skills", "project", "SKILL.md"), "---\nname: project-skill\ndescription: Project skill.\n---\n")

	runtime := app.NewRuntime(app.RuntimeConfig{
		SkillHomeDir:    home,
		BundledSkillDir: filepath.Join(root, "missing-bundled"),
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	project, err := runtime.CreateProject(context.Background(), app.CreateProjectRequest{Name: "Repo", RootDir: projectRoot})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	created, err := runtime.CreateSession(context.Background(), app.CreateSessionRequest{
		Name:      "Session",
		RootDir:   sessionRoot,
		ProjectID: project.ID,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	catalog, err := runtime.ListSkills(context.Background(), app.ListSkillsRequest{SessionID: created.Session.ID})
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	names := map[string]bool{}
	for _, skill := range catalog.Skills {
		names[skill.Name] = true
	}
	for _, name := range []string{"session-skill", "project-skill"} {
		if !names[name] {
			t.Fatalf("missing skill %q from %#v", name, catalog.Skills)
		}
	}
}

func writeSkillFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}
