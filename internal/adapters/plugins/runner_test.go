package plugins

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/appsettings"
)

func TestRunPluginCommandReceivesJSONStdinAndUsesPluginCwd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test command uses POSIX shell quoting")
	}
	dir := t.TempDir()
	result, err := runPluginCommand(context.Background(), PluginCommandRequest{
		PluginID: "github",
		Dir:      dir,
		Command:  `printf '{"cwd":"%s","stdin":' "$PWD"; cat; printf '}'`,
		Input:    map[string]string{"value": "ok"},
	})
	if err != nil {
		t.Fatalf("runPluginCommand: %v", err)
	}
	var got struct {
		CWD   string            `json:"cwd"`
		Stdin map[string]string `json:"stdin"`
	}
	if err := json.Unmarshal(result.Stdout, &got); err != nil {
		t.Fatalf("unmarshal stdout %q: %v", result.Stdout, err)
	}
	wantCWD, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("abs dir: %v", err)
	}
	if got.CWD != wantCWD {
		t.Fatalf("cwd = %q, want %q", got.CWD, wantCWD)
	}
	if got.Stdin["value"] != "ok" {
		t.Fatalf("stdin = %#v", got.Stdin)
	}
}

func TestRunPluginCommandTimesOut(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test command uses POSIX shell command")
	}
	start := time.Now()
	_, err := runPluginCommand(context.Background(), PluginCommandRequest{
		PluginID: "slow",
		Dir:      t.TempDir(),
		Command:  "sleep 5",
		Input:    map[string]string{"value": "ok"},
		Timeout:  20 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("runPluginCommand error = nil, want timeout")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("timeout took %s, want under 1s", elapsed)
	}
	if !strings.Contains(err.Error(), "timed out after") {
		t.Fatalf("error = %v, want timeout", err)
	}
}

func TestRunPluginCommandReportsNonZeroExitAndStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test command uses POSIX shell quoting")
	}
	_, err := runPluginCommand(context.Background(), PluginCommandRequest{
		PluginID: "bad",
		Dir:      t.TempDir(),
		Command:  `printf 'bad stderr' >&2; exit 7`,
		Input:    map[string]string{"value": "ok"},
	})
	if err == nil {
		t.Fatal("runPluginCommand error = nil, want exit error")
	}
	if !strings.Contains(err.Error(), "exit status 7") || !strings.Contains(err.Error(), "bad stderr") {
		t.Fatalf("error = %v, want exit status and stderr", err)
	}
}

func TestRunPluginCommandCapsStdout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test command uses POSIX shell command")
	}
	_, err := runPluginCommand(context.Background(), PluginCommandRequest{
		PluginID:       "loud",
		Dir:            t.TempDir(),
		Command:        `printf 'abcdef'`,
		Input:          map[string]string{"value": "ok"},
		StdoutCapBytes: 3,
	})
	if err == nil {
		t.Fatal("runPluginCommand error = nil, want stdout cap error")
	}
	if !strings.Contains(err.Error(), "stdout exceeded 3 bytes") {
		t.Fatalf("error = %v, want stdout cap", err)
	}
}

func TestRunPluginCommandCapsStderrInErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test command uses POSIX shell quoting")
	}
	_, err := runPluginCommand(context.Background(), PluginCommandRequest{
		PluginID:       "bad",
		Dir:            t.TempDir(),
		Command:        `printf 'abcdef' >&2; exit 1`,
		Input:          map[string]string{"value": "ok"},
		StderrCapBytes: 3,
	})
	if err == nil {
		t.Fatal("runPluginCommand error = nil, want exit error")
	}
	if !strings.Contains(err.Error(), "abc") || strings.Contains(err.Error(), "def") || !strings.Contains(err.Error(), "stderr truncated to 3 bytes") {
		t.Fatalf("error = %v, want capped stderr", err)
	}
}

func TestRunPluginCommandUsesPlatformShell(t *testing.T) {
	command := `printf '%s' shell && printf '%s' -ok`
	if runtime.GOOS == "windows" {
		command = `echo shell-ok`
	}
	result, err := runPluginCommand(context.Background(), PluginCommandRequest{
		PluginID: "shell",
		Dir:      t.TempDir(),
		Command:  command,
		Input:    map[string]string{"value": "ok"},
	})
	if err != nil {
		t.Fatalf("runPluginCommand: %v", err)
	}
	if got := strings.TrimSpace(string(result.Stdout)); got != "shell-ok" {
		t.Fatalf("stdout = %q, want shell-ok", got)
	}
}

func TestExistingCommandCallersPropagateMalformedPluginJSON(t *testing.T) {
	t.Run("resolver", func(t *testing.T) {
		resolver := CommandResolver{
			PluginID: "github",
			Dir:      t.TempDir(),
			Command:  pluginTestPrintCommand("not-json"),
		}
		_, err := resolver.ResolveProjectAttachment(context.Background(), app.ResolveProjectAttachmentRequest{Provider: "github", Target: "owner/repo#1"})
		if err == nil || !strings.Contains(err.Error(), "invalid character") {
			t.Fatalf("ResolveProjectAttachment error = %v, want JSON parse error", err)
		}
	})

	t.Run("template", func(t *testing.T) {
		configHome := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", configHome)
		dir := filepath.Join(configHome, "whisk", "plugins", "github")
		writeManifestOnly(t, dir, `{
			"id": "github",
			"name": "GitHub",
			"version": "0.1.0",
			"ui": {"projectAttachments": [{
				"id": "github.issue",
				"label": "GitHub Issue",
				"provider":"github",
				"kind":"external",
				"command": `+strconv.Quote(pluginTestPrintCommand("not-json"))+`
			}]}
		}`)
		manager, err := NewManager(nil, appsettings.NewStore(filepath.Join(t.TempDir(), "whisk.json")))
		if err != nil {
			t.Fatalf("new manager: %v", err)
		}
		if _, err := manager.TrustPlugin(context.Background(), "github"); err != nil {
			t.Fatalf("trust: %v", err)
		}
		_, err = manager.RunProjectAttachmentTemplate(context.Background(), app.RunPluginProjectAttachmentTemplateRequest{
			PluginID:   "github",
			TemplateID: "github.issue",
			ProjectID:  "proj_01",
		})
		if err == nil || !strings.Contains(err.Error(), "invalid character") {
			t.Fatalf("RunProjectAttachmentTemplate error = %v, want JSON parse error", err)
		}
	})
}

func pluginTestPrintCommand(value string) string {
	if runtime.GOOS == "windows" {
		return "echo " + value
	}
	return "printf " + strconv.Quote(value)
}
