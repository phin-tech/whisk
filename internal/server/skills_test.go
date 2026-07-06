package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func TestHTTPServerSkillRoutes(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	projectRoot := filepath.Join(root, "repo")
	writeSkillFile(t, filepath.Join(home, ".codex", "skills", "review", "SKILL.md"), "---\nname: code-review\ndescription: Review code.\n---\n")
	writeSkillFile(t, filepath.Join(projectRoot, ".claude", "skills", "docs", "SKILL.md"), "# Docs\n\nWrite docs.")

	runtime := app.NewRuntime(app.RuntimeConfig{
		SkillHomeDir: home,
		SkillNow:     func() time.Time { return time.Unix(789, 0).UTC() },
	})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	project, err := runtime.CreateProject(context.Background(), app.CreateProjectRequest{Name: "Repo", RootDir: projectRoot})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	handler := server.NewHTTP(runtime)

	catalog := getJSON[protocol.SkillCatalog](t, handler, "/v1/skills?projectId="+project.ID, http.StatusOK)
	if !catalog.ScannedAt.Equal(time.Unix(789, 0).UTC()) || len(catalog.Skills) != 2 {
		t.Fatalf("catalog = %#v", catalog)
	}
	byName := map[string]protocol.Skill{}
	for _, skill := range catalog.Skills {
		byName[skill.Name] = skill
	}
	if byName["code-review"].SourceKind != "home" || byName["Docs"].SourceKind != "project" {
		t.Fatalf("skills = %#v", catalog.Skills)
	}

	rescanned := postJSON[protocol.SkillCatalog](t, handler, "/v1/skills/rescan", protocol.ListSkillsRequest{ProjectID: project.ID}, http.StatusOK)
	if len(rescanned.Skills) != len(catalog.Skills) {
		t.Fatalf("rescanned = %#v, catalog = %#v", rescanned, catalog)
	}
}

func TestHTTPServerSkillRoutesRejectBadInputs(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{SkillHomeDir: t.TempDir()})
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	handler := server.NewHTTP(runtime)

	getJSON[map[string]any](t, handler, "/v1/skills?projectId=missing", http.StatusBadRequest)

	req, err := http.NewRequest(http.MethodPost, "/v1/skills/rescan", strings.NewReader("{"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("malformed rescan status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	postJSON[map[string]any](t, handler, "/v1/skills/rescan", protocol.ListSkillsRequest{SessionID: "missing"}, http.StatusBadRequest)
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
