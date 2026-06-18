package plugins

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/domain/pluginregistry"
)

// makeTarGz builds a GitHub-style tarball: every entry is nested under a
// "<repo>-<ref>/" top-level directory.
func makeTarGz(t *testing.T, prefix string, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		full := prefix + "/" + name
		if err := tw.WriteHeader(&tar.Header{Name: full, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestGitHubTransportFetchesCatalogAndPathBundle(t *testing.T) {
	registryJSON := `{"version":1,"plugins":[{"id":"github-issues","source":{"type":"path","path":"plugins/github-issues"}}]}`
	tarball := makeTarGz(t, "whisk-plugins-main", map[string]string{
		"registry.json":                     registryJSON,
		"plugins/github-issues/plugin.json": `{"id":"github-issues"}`,
		"plugins/github-issues/resolve.mjs": "export default {}",
		"plugins/other-plugin/plugin.json":  `{"id":"other"}`,
		"README.md":                         "ignore me",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/registry.json"):
			fmt.Fprint(w, registryJSON)
		case strings.Contains(r.URL.Path, "/tar.gz/"):
			w.Write(tarball)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	transport := &GitHubTransport{
		Owner: "phin-tech", Repo: "whisk-plugins", Ref: "main",
		Client: server.Client(), RawBase: server.URL, CodeloadBase: server.URL,
	}

	data, err := transport.Registry(context.Background())
	if err != nil {
		t.Fatalf("Registry: %v", err)
	}
	if _, err := pluginregistry.ParseRegistry(data); err != nil {
		t.Fatalf("catalog did not parse: %v", err)
	}

	files, err := transport.Fetch(context.Background(), pluginregistry.Source{Type: pluginregistry.SourcePath, Path: "plugins/github-issues"})
	if err != nil {
		t.Fatalf("Fetch path: %v", err)
	}
	if string(files["plugin.json"]) != `{"id":"github-issues"}` {
		t.Fatalf("plugin.json = %q", files["plugin.json"])
	}
	if _, ok := files["resolve.mjs"]; !ok {
		t.Fatal("resolve.mjs missing from bundle")
	}
	// Only the requested subdir is extracted, relative to that subdir.
	for name := range files {
		if strings.Contains(name, "other-plugin") || name == "README.md" {
			t.Fatalf("bundle leaked unrelated file %q", name)
		}
	}
}

func TestGitHubTransportFetchesGitSubdir(t *testing.T) {
	tarball := makeTarGz(t, "whisk-plugin-linear-v1", map[string]string{
		"plugin/plugin.json": `{"id":"linear"}`,
		"plugin/index.mjs":   "export {}",
		"docs/README.md":     "ignore",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(tarball)
	}))
	defer server.Close()

	transport := &GitHubTransport{Owner: "x", Repo: "y", Client: server.Client(), CodeloadBase: server.URL}
	files, err := transport.Fetch(context.Background(), pluginregistry.Source{
		Type: pluginregistry.SourceGit, Repo: "phin-tech/whisk-plugin-linear", Subdir: "plugin", Ref: "v1",
	})
	if err != nil {
		t.Fatalf("Fetch git: %v", err)
	}
	if _, ok := files["plugin.json"]; !ok {
		t.Fatalf("plugin.json missing, got %v", keys(files))
	}
	if _, ok := files["README.md"]; ok {
		t.Fatal("docs leaked into bundle")
	}
}

func TestParseGitHubRepoForms(t *testing.T) {
	cases := []struct{ in, owner, repo, ref string }{
		{"phin-tech/whisk-plugins", "phin-tech", "whisk-plugins", "main"},
		{"phin-tech/whisk-plugins@dev", "phin-tech", "whisk-plugins", "dev"},
		{"github:phin-tech/whisk-plugins", "phin-tech", "whisk-plugins", "main"},
		{"https://github.com/phin-tech/whisk-plugins.git", "phin-tech", "whisk-plugins", "main"},
	}
	for _, tc := range cases {
		owner, repo, ref, err := parseGitHubRepo(tc.in)
		if err != nil || owner != tc.owner || repo != tc.repo || ref != tc.ref {
			t.Fatalf("parseGitHubRepo(%q) = %q/%q@%q err=%v", tc.in, owner, repo, ref, err)
		}
	}
	if _, _, _, err := parseGitHubRepo("not-a-repo"); err == nil {
		t.Fatal("parseGitHubRepo(not-a-repo) = nil error")
	}
}

func TestNewTransportResolvesLocalDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "registry.json"), []byte(`{"version":1,"plugins":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	transport, err := NewTransport(dir)
	if err != nil {
		t.Fatalf("NewTransport: %v", err)
	}
	if _, ok := transport.(*LocalTransport); !ok {
		t.Fatalf("transport = %T, want *LocalTransport", transport)
	}
}

func TestLocalTransportReadsPathBundle(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugins", "github-issues")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{"id":"github-issues"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	transport := &LocalTransport{Dir: dir}
	files, err := transport.Fetch(context.Background(), pluginregistry.Source{Type: pluginregistry.SourcePath, Path: "plugins/github-issues"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if _, ok := files["plugin.json"]; !ok {
		t.Fatalf("plugin.json missing, got %v", keys(files))
	}
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
