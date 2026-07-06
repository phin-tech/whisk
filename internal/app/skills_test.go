package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
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

func writeSkillFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}
