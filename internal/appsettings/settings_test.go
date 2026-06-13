package appsettings_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phin-tech/whisk/internal/appsettings"
)

func TestDefaultSettingsOpenToSessions(t *testing.T) {
	settings := appsettings.Default()

	if settings.StartupView != appsettings.StartupViewSessions {
		t.Fatalf("startup view = %q, want %q", settings.StartupView, appsettings.StartupViewSessions)
	}
}

func TestNormalizeSettingsAllowsOnlyKnownStartupViews(t *testing.T) {
	tests := []struct {
		name    string
		input   appsettings.Settings
		want    string
		wantErr bool
	}{
		{name: "blank defaults to sessions", input: appsettings.Settings{}, want: appsettings.StartupViewSessions},
		{name: "sessions", input: appsettings.Settings{StartupView: appsettings.StartupViewSessions}, want: appsettings.StartupViewSessions},
		{name: "kanban", input: appsettings.Settings{StartupView: appsettings.StartupViewKanban}, want: appsettings.StartupViewKanban},
		{name: "invalid", input: appsettings.Settings{StartupView: "board"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := appsettings.Normalize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			if got.StartupView != tt.want {
				t.Fatalf("startup view = %q, want %q", got.StartupView, tt.want)
			}
		})
	}
}

func TestDefaultPathUsesXDGConfigHome(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	path, err := appsettings.DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	if path != filepath.Join(configDir, "whisk.json") {
		t.Fatalf("path = %q", path)
	}
}

func TestDefaultPathFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	path, err := appsettings.DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	if path != filepath.Join(home, ".config", "whisk.json") {
		t.Fatalf("path = %q", path)
	}
}

func TestNewDefaultStoreUsesDefaultPath(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	store, err := appsettings.NewDefaultStore()
	if err != nil {
		t.Fatalf("new default store: %v", err)
	}
	if _, err := store.Save(context.Background(), appsettings.Settings{StartupView: appsettings.StartupViewKanban}); err != nil {
		t.Fatalf("save: %v", err)
	}
	bytes, err := os.ReadFile(filepath.Join(configDir, "whisk.json"))
	if err != nil {
		t.Fatalf("read default file: %v", err)
	}
	if string(bytes) != "{\n  \"startupView\": \"kanban\"\n}\n" {
		t.Fatalf("file = %q", string(bytes))
	}
}

func TestStoreLoadMissingFileReturnsDefaults(t *testing.T) {
	store := appsettings.NewStore(filepath.Join(t.TempDir(), "whisk.json"))

	got, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.StartupView != appsettings.StartupViewSessions {
		t.Fatalf("startup view = %q", got.StartupView)
	}
}

func TestStoreLoadRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "whisk.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	store := appsettings.NewStore(path)

	if _, err := store.Load(context.Background()); err == nil {
		t.Fatalf("expected load error")
	}
}

func TestStoreLoadRejectsInvalidStartupView(t *testing.T) {
	path := filepath.Join(t.TempDir(), "whisk.json")
	if err := os.WriteFile(path, []byte(`{"startupView":"board"}`), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	store := appsettings.NewStore(path)

	if _, err := store.Load(context.Background()); err == nil {
		t.Fatalf("expected load error")
	}
}

func TestStoreLoadReturnsReadError(t *testing.T) {
	store := appsettings.NewStore(t.TempDir())

	if _, err := store.Load(context.Background()); err == nil {
		t.Fatalf("expected load error")
	}
}

func TestStoreSavesAndLoadsSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "whisk.json")
	store := appsettings.NewStore(path)

	saved, err := store.Save(context.Background(), appsettings.Settings{StartupView: appsettings.StartupViewKanban})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.StartupView != appsettings.StartupViewKanban {
		t.Fatalf("saved startup view = %q", saved.StartupView)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.StartupView != appsettings.StartupViewKanban {
		t.Fatalf("loaded startup view = %q", loaded.StartupView)
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(bytes) != "{\n  \"startupView\": \"kanban\"\n}\n" {
		t.Fatalf("file = %q", string(bytes))
	}
}

func TestStoreRejectsInvalidStartupView(t *testing.T) {
	store := appsettings.NewStore(filepath.Join(t.TempDir(), "whisk.json"))

	if _, err := store.Save(context.Background(), appsettings.Settings{StartupView: "board"}); err == nil {
		t.Fatalf("expected save error")
	}
}

func TestStoreSaveReturnsDirectoryError(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	store := appsettings.NewStore(filepath.Join(blocker, "whisk.json"))

	if _, err := store.Save(context.Background(), appsettings.Settings{StartupView: appsettings.StartupViewKanban}); err == nil {
		t.Fatalf("expected save error")
	}
}

func TestStoreSaveReturnsCreateTempError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	defer func() {
		_ = os.Chmod(dir, 0o700)
	}()
	store := appsettings.NewStore(filepath.Join(dir, "whisk.json"))

	if _, err := store.Save(context.Background(), appsettings.Settings{StartupView: appsettings.StartupViewKanban}); err == nil {
		t.Skip("filesystem permits writes to a non-writable test directory")
	}
}

func TestStoreSaveReturnsRenameError(t *testing.T) {
	store := appsettings.NewStore(t.TempDir())

	if _, err := store.Save(context.Background(), appsettings.Settings{StartupView: appsettings.StartupViewKanban}); err == nil {
		t.Fatalf("expected save error")
	}
}
