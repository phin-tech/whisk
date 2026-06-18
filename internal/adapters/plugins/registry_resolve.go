package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phin-tech/whisk/internal/appsettings"
)

const (
	defaultRegistryName   = "phin-tech"
	defaultRegistrySource = "phin-tech/whisk-plugins"
)

// resolveRegistries turns configured registries into named transports. Precedence:
// the settings list, else a single registry from WHISK_PLUGIN_REGISTRY, else a
// built-in default. cacheRoot is where git-transport registries keep their clones.
func resolveRegistries(configs []appsettings.PluginRegistryConfig, envSource, cacheRoot string) ([]namedRegistry, error) {
	effective := configs
	if len(effective) == 0 {
		source := strings.TrimSpace(envSource)
		if source == "" {
			source = defaultRegistrySource
		}
		effective = []appsettings.PluginRegistryConfig{{Name: deriveRegistryName(source), Source: source}}
	}
	out := make([]namedRegistry, 0, len(effective))
	for _, cfg := range effective {
		transport, err := registryTransport(cfg, cacheRoot)
		if err != nil {
			return nil, fmt.Errorf("registry %q: %w", cfg.Name, err)
		}
		out = append(out, namedRegistry{Name: cfg.Name, Transport: transport})
	}
	return out, nil
}

// registryTransport builds the transport for one registry, auto-detecting the
// mechanism from the source unless cfg.Transport forces it.
func registryTransport(cfg appsettings.PluginRegistryConfig, cacheRoot string) (Transport, error) {
	source := strings.TrimSpace(cfg.Source)
	if source == "" {
		return nil, fmt.Errorf("source is required")
	}

	switch {
	case cfg.Transport == "local" || (cfg.Transport == "" && isLocalSource(source)):
		dir, _ := localBase(source)
		if dir == "" {
			dir = source
		}
		return &LocalTransport{Dir: expandUser(dir)}, nil

	case cfg.Transport == "git" || (cfg.Transport == "" && isSSHURL(source)):
		base, ref := splitSourceRef(source)
		return &GitTransport{
			RepoURL:  gitCloneURL(base),
			Ref:      ref,
			CacheDir: filepath.Join(cacheRoot, cfg.Name),
		}, nil

	default:
		owner, repo, ref, err := parseGitHubRepo(source)
		if err != nil {
			return nil, err
		}
		token := ""
		if cfg.TokenEnv != "" {
			token = strings.TrimSpace(os.Getenv(cfg.TokenEnv))
		}
		return &GitHubTransport{Owner: owner, Repo: repo, Ref: ref, Token: token}, nil
	}
}

func isLocalSource(source string) bool {
	_, ok := localBase(source)
	return ok
}

func isSSHURL(source string) bool {
	return strings.HasPrefix(source, "git@") || strings.HasPrefix(source, "ssh://")
}

// splitSourceRef splits a trailing "@ref" off an owner/repo or https source. SSH
// sources (which themselves contain a "@") are returned unchanged.
func splitSourceRef(source string) (base, ref string) {
	if isSSHURL(source) {
		return source, ""
	}
	if at := strings.LastIndex(source, "@"); at > 0 {
		return source[:at], source[at+1:]
	}
	return source, ""
}

func deriveRegistryName(source string) string {
	if dir, ok := localBase(source); ok {
		return filepath.Base(filepath.Clean(expandUser(dir)))
	}
	if owner, _, _, err := parseGitHubRepo(source); err == nil {
		return owner
	}
	return "registry"
}

func expandUser(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/"))
		}
	}
	return path
}
