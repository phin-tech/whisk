package client_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPClientListsSkills(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	projectRoot := filepath.Join(root, "repo")
	writeSkillFile(t, filepath.Join(home, ".codex", "skills", "review", "SKILL.md"), "---\nname: code-review\ndescription: Review code.\n---\n")
	writeSkillFile(t, filepath.Join(projectRoot, "skills", "docs", "SKILL.md"), "# Docs\n\nWrite docs.")

	runtime := app.NewRuntime(app.RuntimeConfig{
		SkillHomeDir: home,
		SkillNow:     func() time.Time { return time.Unix(456, 0).UTC() },
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	project, err := runtime.CreateProject(context.Background(), app.CreateProjectRequest{Name: "Repo", RootDir: projectRoot})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	httpServer := httptest.NewServer(server.NewHTTP(runtime))
	t.Cleanup(httpServer.Close)

	daemon := client.NewHTTP(httpServer.URL, httpServer.Client())
	catalog, err := daemon.ListSkills(context.Background(), protocol.ListSkillsRequest{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if !catalog.ScannedAt.Equal(time.Unix(456, 0).UTC()) || len(catalog.Skills) != 2 {
		t.Fatalf("catalog = %#v", catalog)
	}

	rescanned, err := daemon.RescanSkills(context.Background(), protocol.ListSkillsRequest{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("rescan skills: %v", err)
	}
	if len(rescanned.Skills) != len(catalog.Skills) {
		t.Fatalf("rescanned = %#v, catalog = %#v", rescanned, catalog)
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
