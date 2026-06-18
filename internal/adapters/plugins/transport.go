package plugins

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/phin-tech/whisk/internal/domain/pluginregistry"
)

// Transport fetches the registry catalog and individual plugin file bundles. It
// is the imperative-shell boundary for plugin installation: implementations do
// network or filesystem I/O, while the Installer stays pure orchestration.
type Transport interface {
	// Registry returns the raw registry.json document.
	Registry(ctx context.Context) ([]byte, error)
	// Fetch returns the plugin's files as relpath -> content for the given source.
	Fetch(ctx context.Context, source pluginregistry.Source) (map[string][]byte, error)
}

// Size guards against decompression bombs in fetched archives. Plugins are
// small script bundles; anything larger is almost certainly hostile.
const (
	maxBundleBytes = 25 << 20 // 25 MiB total per plugin
	maxFileBytes   = 5 << 20  // 5 MiB per file
)

const defaultRegistryRef = "main"

// NewTransport resolves a registry base spec into a Transport.
//
//	""                          -> github phin-tech/whisk-plugins@main
//	"owner/repo" / "owner/repo@ref" / github URL -> GitHub tarball
//	a local directory / file:// path             -> on-disk registry (dev/tests)
func NewTransport(base string) (Transport, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return &GitHubTransport{Owner: "phin-tech", Repo: "whisk-plugins", Ref: defaultRegistryRef}, nil
	}
	if dir, ok := localBase(base); ok {
		return &LocalTransport{Dir: dir}, nil
	}
	owner, repo, ref, err := parseGitHubRepo(base)
	if err != nil {
		return nil, err
	}
	return &GitHubTransport{Owner: owner, Repo: repo, Ref: ref}, nil
}

func localBase(base string) (string, bool) {
	if rest, ok := strings.CutPrefix(base, "file://"); ok {
		return rest, true
	}
	if strings.HasPrefix(base, "/") || strings.HasPrefix(base, "./") || strings.HasPrefix(base, "../") || strings.HasPrefix(base, "~") {
		return base, true
	}
	if info, err := os.Stat(base); err == nil && info.IsDir() {
		return base, true
	}
	return "", false
}

// parseGitHubRepo accepts "owner/repo", "owner/repo@ref", "github:owner/repo",
// and "https://github.com/owner/repo(.git)" forms.
func parseGitHubRepo(spec string) (owner, repo, ref string, err error) {
	ref = defaultRegistryRef
	spec = strings.TrimPrefix(strings.TrimSpace(spec), "github:")
	if rest, found := strings.CutPrefix(spec, "https://github.com/"); found {
		spec = rest
	} else if rest, found := strings.CutPrefix(spec, "git@github.com:"); found {
		spec = rest
	}
	spec = strings.TrimSuffix(spec, ".git")
	if at := strings.LastIndex(spec, "@"); at >= 0 {
		ref = spec[at+1:]
		spec = spec[:at]
	}
	parts := strings.Split(strings.Trim(spec, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("invalid github repo %q (want owner/repo)", spec)
	}
	if ref == "" {
		ref = defaultRegistryRef
	}
	return parts[0], parts[1], ref, nil
}

// LocalTransport serves a registry from a directory on disk. Used for the
// in-repo seed registry during development and for tests. Git sources resolve
// against Repos[repo] when provided; otherwise they error.
type LocalTransport struct {
	Dir   string
	Repos map[string]string
}

func (t *LocalTransport) Registry(context.Context) ([]byte, error) {
	return os.ReadFile(filepath.Join(t.Dir, "registry.json"))
}

func (t *LocalTransport) Fetch(_ context.Context, source pluginregistry.Source) (map[string][]byte, error) {
	switch source.Type {
	case pluginregistry.SourcePath:
		return readDirFiles(filepath.Join(t.Dir, filepath.FromSlash(source.Path)))
	case pluginregistry.SourceGit:
		root, ok := t.Repos[source.Repo]
		if !ok {
			return nil, fmt.Errorf("local transport has no mapping for git repo %q", source.Repo)
		}
		return readDirFiles(filepath.Join(root, filepath.FromSlash(source.Subdir)))
	default:
		return nil, fmt.Errorf("unsupported source type %q", source.Type)
	}
}

func readDirFiles(root string) (map[string][]byte, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	files := map[string][]byte{}
	total := 0
	err = filepath.WalkDir(root, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if len(data) > maxFileBytes {
			return fmt.Errorf("file %s exceeds %d bytes", p, maxFileBytes)
		}
		total += len(data)
		if total > maxBundleBytes {
			return fmt.Errorf("plugin bundle exceeds %d bytes", maxBundleBytes)
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// GitTransport fetches a registry by shelling out to git, reusing whatever auth
// the user already has locally (SSH keys, credential helpers, gh). It works with
// any git host, not just GitHub. The registry repo is kept as a shallow clone in
// CacheDir and updated in place; plugin bundles are read from that clone.
type GitTransport struct {
	RepoURL  string
	Ref      string
	CacheDir string
	GitPath  string // defaults to "git"
}

func (t *GitTransport) git() string {
	if t.GitPath != "" {
		return t.GitPath
	}
	return "git"
}

func (t *GitTransport) ref() string {
	if t.Ref == "" {
		return ""
	}
	return t.Ref
}

func (t *GitTransport) Registry(ctx context.Context) ([]byte, error) {
	if err := t.ensureRepo(ctx, t.RepoURL, t.ref(), t.CacheDir); err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(t.CacheDir, "registry.json"))
}

func (t *GitTransport) Fetch(ctx context.Context, source pluginregistry.Source) (map[string][]byte, error) {
	switch source.Type {
	case pluginregistry.SourcePath:
		if err := t.ensureRepo(ctx, t.RepoURL, t.ref(), t.CacheDir); err != nil {
			return nil, err
		}
		return readDirFiles(filepath.Join(t.CacheDir, filepath.FromSlash(source.Path)))
	case pluginregistry.SourceGit:
		// An external git-source entry: clone it into a temp dir using the same
		// local auth and read its subdir.
		dir, err := os.MkdirTemp("", "whisk-plugin-git-*")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(dir)
		url := gitCloneURL(source.Repo)
		if err := t.ensureRepo(ctx, url, source.Ref, dir); err != nil {
			return nil, err
		}
		return readDirFiles(filepath.Join(dir, filepath.FromSlash(source.Subdir)))
	default:
		return nil, fmt.Errorf("unsupported source type %q", source.Type)
	}
}

// ensureRepo makes dir a shallow clone of url at ref, cloning on first use and
// fetching+resetting on subsequent calls.
func (t *GitTransport) ensureRepo(ctx context.Context, url, ref, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		// Existing clone: fetch and hard-reset to the ref.
		fetchRef := ref
		if fetchRef == "" {
			fetchRef = "HEAD"
		}
		if err := t.run(ctx, dir, "fetch", "--depth", "1", "origin", fetchRef); err != nil {
			return err
		}
		return t.run(ctx, dir, "reset", "--hard", "FETCH_HEAD")
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	args := []string{"clone", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, url, dir)
	return t.run(ctx, "", args...)
}

func (t *GitTransport) run(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, t.git(), args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("git %s: %w: %s", args[0], err, msg)
		}
		return fmt.Errorf("git %s: %w", args[0], err)
	}
	return nil
}

// gitCloneURL turns an "owner/repo" shorthand into an SSH URL (so local keys are
// used). Full URLs, SSH specs, and local filesystem paths are passed through
// unchanged so git can clone them directly.
func gitCloneURL(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return spec
	}
	if strings.Contains(spec, "://") || strings.HasPrefix(spec, "git@") {
		return spec
	}
	if filepath.IsAbs(spec) || strings.HasPrefix(spec, ".") || strings.HasPrefix(spec, "~") {
		return spec
	}
	parts := strings.Split(strings.Trim(spec, "/"), "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return "git@github.com:" + strings.TrimSuffix(spec, ".git") + ".git"
	}
	return spec
}

// GitHubTransport fetches the catalog and plugin bundles from GitHub. For public
// repos it uses raw.githubusercontent + codeload tarballs (no auth, no git). When
// Token is set (private repos) it uses the authenticated GitHub API: the contents
// endpoint for the catalog and the tarball endpoint for bundles.
type GitHubTransport struct {
	Owner  string
	Repo   string
	Ref    string
	Token  string
	Client *http.Client
	// BaseURLs override host roots for testing. Empty means the real hosts.
	RawBase      string
	CodeloadBase string
	APIBase      string
}

func (t *GitHubTransport) useAPI() bool { return t.Token != "" }

func (t *GitHubTransport) apiBase() string {
	if t.APIBase != "" {
		return t.APIBase
	}
	return "https://api.github.com"
}

func (t *GitHubTransport) httpClient() *http.Client {
	if t.Client != nil {
		return t.Client
	}
	return http.DefaultClient
}

func (t *GitHubTransport) rawBase() string {
	if t.RawBase != "" {
		return t.RawBase
	}
	return "https://raw.githubusercontent.com"
}

func (t *GitHubTransport) codeloadBase() string {
	if t.CodeloadBase != "" {
		return t.CodeloadBase
	}
	return "https://codeload.github.com"
}

func (t *GitHubTransport) ref() string {
	if t.Ref == "" {
		return defaultRegistryRef
	}
	return t.Ref
}

func (t *GitHubTransport) Registry(ctx context.Context) ([]byte, error) {
	if t.useAPI() {
		url := fmt.Sprintf("%s/repos/%s/%s/contents/registry.json?ref=%s", t.apiBase(), t.Owner, t.Repo, t.ref())
		return t.getWith(ctx, url, "application/vnd.github.raw")
	}
	url := fmt.Sprintf("%s/%s/%s/%s/registry.json", t.rawBase(), t.Owner, t.Repo, t.ref())
	return t.getWith(ctx, url, "")
}

func (t *GitHubTransport) Fetch(ctx context.Context, source pluginregistry.Source) (map[string][]byte, error) {
	switch source.Type {
	case pluginregistry.SourcePath:
		return t.tarball(ctx, t.Owner, t.Repo, t.ref(), source.Path)
	case pluginregistry.SourceGit:
		owner, repo, _, err := parseGitHubRepo(source.Repo)
		if err != nil {
			return nil, fmt.Errorf("plugin git source: %w", err)
		}
		ref := source.Ref
		if ref == "" {
			ref = "HEAD"
		}
		return t.tarball(ctx, owner, repo, ref, source.Subdir)
	default:
		return nil, fmt.Errorf("unsupported source type %q", source.Type)
	}
}

// tarball downloads a repo tarball and extracts the files under subdir. GitHub
// wraps everything in a top-level "<repo>-<ref>/" directory which is stripped.
func (t *GitHubTransport) tarball(ctx context.Context, owner, repo, ref, subdir string) (map[string][]byte, error) {
	url := fmt.Sprintf("%s/%s/%s/tar.gz/%s", t.codeloadBase(), owner, repo, ref)
	if t.useAPI() {
		url = fmt.Sprintf("%s/repos/%s/%s/tarball/%s", t.apiBase(), owner, repo, ref)
	}
	body, err := t.getReader(ctx, url, "")
	if err != nil {
		return nil, err
	}
	defer body.Close()
	gz, err := gzip.NewReader(body)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	want := path.Clean(strings.Trim(subdir, "/"))
	if want == "." {
		want = ""
	}
	files := map[string][]byte{}
	total := 0
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar: %w", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		rel := stripTopDir(header.Name)
		if rel == "" {
			continue
		}
		rel, ok := withinSubdir(rel, want)
		if !ok {
			continue
		}
		if header.Size > maxFileBytes {
			return nil, fmt.Errorf("file %s exceeds %d bytes", rel, maxFileBytes)
		}
		total += int(header.Size)
		if total > maxBundleBytes {
			return nil, fmt.Errorf("plugin bundle exceeds %d bytes", maxBundleBytes)
		}
		data, err := io.ReadAll(io.LimitReader(reader, maxFileBytes+1))
		if err != nil {
			return nil, err
		}
		files[rel] = data
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no files found under %q", subdir)
	}
	return files, nil
}

// stripTopDir removes GitHub's "<repo>-<ref>/" archive prefix.
func stripTopDir(name string) string {
	name = strings.TrimPrefix(name, "./")
	if i := strings.IndexByte(name, '/'); i >= 0 {
		return name[i+1:]
	}
	return ""
}

// withinSubdir keeps only entries under want and returns their path relative to want.
func withinSubdir(rel, want string) (string, bool) {
	if want == "" {
		return rel, true
	}
	prefix := want + "/"
	if !strings.HasPrefix(rel, prefix) {
		return "", false
	}
	return strings.TrimPrefix(rel, prefix), true
}

func (t *GitHubTransport) getWith(ctx context.Context, url, accept string) ([]byte, error) {
	body, err := t.getReader(ctx, url, accept)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(io.LimitReader(body, maxBundleBytes))
}

func (t *GitHubTransport) getReader(ctx context.Context, url, accept string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if t.Token != "" {
		req.Header.Set("Authorization", "Bearer "+t.Token)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := t.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return resp.Body, nil
}
