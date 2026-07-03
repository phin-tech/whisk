package controlauth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureTokenCreatesPrivateStateFile(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	t.Setenv("HOME", t.TempDir())

	token, err := EnsureToken()
	if err != nil {
		t.Fatalf("ensure token: %v", err)
	}
	if len(token) < 32 {
		t.Fatalf("token too short: %q", token)
	}

	path, err := TokenPath()
	if err != nil {
		t.Fatalf("token path: %v", err)
	}
	if want := filepath.Join(stateHome, "whisk", "control-token"); path != want {
		t.Fatalf("token path = %q, want %q", path, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat token: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("token mode = %o, want 600", got)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat token dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("token dir mode = %o, want 700", got)
	}

	read, err := ReadToken()
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	if read != token {
		t.Fatalf("read token = %q, want %q", read, token)
	}

	again, err := EnsureToken()
	if err != nil {
		t.Fatalf("ensure token again: %v", err)
	}
	if again != token {
		t.Fatalf("ensure token regenerated unexpectedly: %q -> %q", token, again)
	}
}

func TestEnsureTokenRepairsModeAndRegeneratesWhenMissing(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	t.Setenv("HOME", t.TempDir())

	path, err := TokenPath()
	if err != nil {
		t.Fatalf("token path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("existing-token\n"), 0o644); err != nil {
		t.Fatalf("write existing token: %v", err)
	}

	token, err := EnsureToken()
	if err != nil {
		t.Fatalf("ensure token: %v", err)
	}
	if token != "existing-token" {
		t.Fatalf("existing token = %q", token)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat token: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("repaired token mode = %o, want 600", got)
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove token: %v", err)
	}
	regenerated, err := EnsureToken()
	if err != nil {
		t.Fatalf("regenerate token: %v", err)
	}
	if regenerated == "" || regenerated == token {
		t.Fatalf("regenerated token = %q, old = %q", regenerated, token)
	}
}
