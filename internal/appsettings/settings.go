package appsettings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	StartupViewSessions = "sessions"
	StartupViewKanban   = "kanban"
)

type Settings struct {
	StartupView string `json:"startupView"`
}

type Store struct {
	path string
}

func Default() Settings {
	return Settings{StartupView: StartupViewSessions}
}

func Normalize(settings Settings) (Settings, error) {
	if settings.StartupView == "" {
		settings.StartupView = StartupViewSessions
	}
	switch settings.StartupView {
	case StartupViewSessions, StartupViewKanban:
		return settings, nil
	default:
		return Settings{}, fmt.Errorf("invalid startup view %q", settings.StartupView)
	}
}

func DefaultPath() (string, error) {
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return filepath.Join(configDir, "whisk.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "whisk.json"), nil
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func NewDefaultStore() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return NewStore(path), nil
}

func (s *Store) Load(context.Context) (Settings, error) {
	bytes, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return Default(), nil
	}
	if err != nil {
		return Settings{}, err
	}

	settings := Default()
	if err := json.Unmarshal(bytes, &settings); err != nil {
		return Settings{}, err
	}
	return Normalize(settings)
}

func (s *Store) Save(_ context.Context, settings Settings) (Settings, error) {
	normalized, err := Normalize(settings)
	if err != nil {
		return Settings{}, err
	}
	bytes, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return Settings{}, err
	}
	bytes = append(bytes, '\n')

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return Settings{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".whisk-*.json")
	if err != nil {
		return Settings{}, err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(bytes); err != nil {
		_ = tmp.Close()
		return Settings{}, err
	}
	if err := tmp.Close(); err != nil {
		return Settings{}, err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return Settings{}, err
	}
	return normalized, nil
}
