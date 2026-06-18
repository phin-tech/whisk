package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/appsettings"
	"github.com/phin-tech/whisk/internal/domain/pluginregistry"
)

func TestInstallerNamespacesAndDisambiguates(t *testing.T) {
	phin := &fakeTransport{
		registry: []byte(`{"version":1,"plugins":[{"id":"github","source":{"type":"path","path":"p"}}]}`),
		bundles:  map[string]map[string][]byte{"p": {"plugin.json": []byte(`{"id":"github","name":"phin github"}`)}},
	}
	acme := &fakeTransport{
		registry: []byte(`{"version":1,"plugins":[{"id":"github","source":{"type":"path","path":"p"}}]}`),
		bundles:  map[string]map[string][]byte{"p": {"plugin.json": []byte(`{"id":"github","name":"acme github"}`)}},
	}
	target := t.TempDir()
	lockPath := filepath.Join(t.TempDir(), "plugins.lock.json")
	installer := NewInstaller([]namedRegistry{{Name: "phin-tech", Transport: phin}, {Name: "acme", Transport: acme}}, target, lockPath)

	// Ambiguous id requires a registry.
	if _, err := installer.Install(context.Background(), "", "github"); err == nil {
		t.Fatal("ambiguous install = nil error, want error")
	}

	// Same id from both registries installs side by side under distinct namespaces.
	if _, err := installer.Install(context.Background(), "phin-tech", "github"); err != nil {
		t.Fatalf("install phin-tech/github: %v", err)
	}
	if _, err := installer.Install(context.Background(), "acme", "github"); err != nil {
		t.Fatalf("install acme/github: %v", err)
	}
	for _, ns := range []string{"phin-tech", "acme"} {
		if _, err := os.Stat(filepath.Join(target, ns, "github", "plugin.json")); err != nil {
			t.Fatalf("%s/github not installed: %v", ns, err)
		}
	}
	data, _ := os.ReadFile(lockPath)
	lock, _ := pluginregistry.ParseLock(data)
	if len(lock.Plugins) != 2 {
		t.Fatalf("lock has %d entries, want 2", len(lock.Plugins))
	}
}

func TestInstallerAvailableMergesAndToleratesFailure(t *testing.T) {
	ok := &fakeTransport{registry: []byte(`{"version":1,"plugins":[{"id":"a","source":{"type":"path","path":"p"}}]}`)}
	broken := &fakeTransport{registry: []byte(`not json`)}
	installer := NewInstaller([]namedRegistry{{Name: "ok", Transport: ok}, {Name: "broken", Transport: broken}}, t.TempDir(), "")

	available, err := installer.Available(context.Background())
	if err == nil {
		t.Fatal("expected a surfaced error for the broken registry")
	}
	if len(available) != 1 || available[0].Registry != "ok" || available[0].Entry.ID != "a" {
		t.Fatalf("available = %#v", available)
	}
}

func TestResolveRegistriesSelectsTransport(t *testing.T) {
	t.Setenv("ACME_TOKEN", "secret")
	registries, err := resolveRegistries([]appsettings.PluginRegistryConfig{
		{Name: "phin-tech", Source: "phin-tech/whisk-plugins"},
		{Name: "acme", Source: "acme/whisk-plugins", TokenEnv: "ACME_TOKEN"},
		{Name: "ssh", Source: "git@github.com:corp/plugins.git"},
		{Name: "local", Source: t.TempDir()},
	}, "", t.TempDir())
	if err != nil {
		t.Fatalf("resolveRegistries: %v", err)
	}
	if _, ok := registries[0].Transport.(*GitHubTransport); !ok {
		t.Fatalf("phin-tech = %T, want *GitHubTransport", registries[0].Transport)
	}
	acme, ok := registries[1].Transport.(*GitHubTransport)
	if !ok || acme.Token != "secret" {
		t.Fatalf("acme = %#v", registries[1].Transport)
	}
	if _, ok := registries[2].Transport.(*GitTransport); !ok {
		t.Fatalf("ssh = %T, want *GitTransport", registries[2].Transport)
	}
	if _, ok := registries[3].Transport.(*LocalTransport); !ok {
		t.Fatalf("local = %T, want *LocalTransport", registries[3].Transport)
	}
}

func TestResolveRegistriesDefaultsToBuiltIn(t *testing.T) {
	registries, err := resolveRegistries(nil, "", t.TempDir())
	if err != nil {
		t.Fatalf("resolveRegistries: %v", err)
	}
	if len(registries) != 1 || registries[0].Name != "phin-tech" {
		t.Fatalf("default = %#v", registries)
	}
}

func TestGitHubTransportUsesAPIWhenTokenSet(t *testing.T) {
	var sawAuth, hitContents, hitTarball bool
	tarball := makeTarGz(t, "repo-main", map[string]string{"plugin/plugin.json": `{"id":"x"}`})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer tok" {
			sawAuth = true
		}
		switch {
		case strings.Contains(r.URL.Path, "/contents/registry.json"):
			hitContents = true
			_, _ = w.Write([]byte(`{"version":1,"plugins":[]}`))
		case strings.Contains(r.URL.Path, "/tarball/"):
			hitTarball = true
			_, _ = w.Write(tarball)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	transport := &GitHubTransport{Owner: "o", Repo: "repo", Ref: "main", Token: "tok", Client: server.Client(), APIBase: server.URL}
	if _, err := transport.Registry(context.Background()); err != nil {
		t.Fatalf("Registry: %v", err)
	}
	if _, err := transport.Fetch(context.Background(), pluginregistry.Source{Type: pluginregistry.SourcePath, Path: "plugin"}); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !sawAuth || !hitContents || !hitTarball {
		t.Fatalf("auth=%v contents=%v tarball=%v", sawAuth, hitContents, hitTarball)
	}
}

func TestGitTransportClonesLocalRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	// Build a real local git repo to act as the registry.
	origin := t.TempDir()
	writeFile(t, filepath.Join(origin, "registry.json"), `{"version":1,"plugins":[{"id":"x","source":{"type":"path","path":"plugins/x"}}]}`)
	writeFile(t, filepath.Join(origin, "plugins", "x", "plugin.json"), `{"id":"x","name":"X"}`)
	gitInit(t, origin)

	transport := &GitTransport{RepoURL: origin, Ref: "main", CacheDir: filepath.Join(t.TempDir(), "clone")}
	data, err := transport.Registry(context.Background())
	if err != nil {
		t.Fatalf("Registry: %v", err)
	}
	if _, err := pluginregistry.ParseRegistry(data); err != nil {
		t.Fatalf("catalog: %v", err)
	}
	files, err := transport.Fetch(context.Background(), pluginregistry.Source{Type: pluginregistry.SourcePath, Path: "plugins/x"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if _, ok := files["plugin.json"]; !ok {
		t.Fatalf("plugin.json missing: %v", keys(files))
	}
	// Second call updates the existing clone rather than failing.
	if _, err := transport.Registry(context.Background()); err != nil {
		t.Fatalf("second Registry (update path): %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("add", "-A")
	run("commit", "-m", "seed")
}

func TestGitCloneURL(t *testing.T) {
	cases := map[string]string{
		"owner/repo":                    "git@github.com:owner/repo.git",
		"owner/repo.git":                "git@github.com:owner/repo.git",
		"git@github.com:owner/repo.git": "git@github.com:owner/repo.git",
		"https://gitlab.com/a/b.git":    "https://gitlab.com/a/b.git",
		"/tmp/local/registry":           "/tmp/local/registry",
		"./relative/repo":               "./relative/repo",
	}
	for in, want := range cases {
		if got := gitCloneURL(in); got != want {
			t.Fatalf("gitCloneURL(%q) = %q, want %q", in, got, want)
		}
	}
}
