package sessionstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phin-tech/whisk/internal/domain/session"
)

const stateFileName = "runtime-state.json"

type JSONStore struct {
	path string
}

type stateFile struct {
	Version  int               `json:"version"`
	Sessions []session.Session `json:"sessions"`
}

func NewJSONStore(path string) (*JSONStore, error) {
	if path == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return nil, err
		}
		path = defaultPath
	}
	return &JSONStore{path: filepath.Clean(path)}, nil
}

func DefaultPath() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "whisk", stateFileName), nil
}

func (s *JSONStore) LoadSessions(context.Context) ([]session.Session, error) {
	bytes, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var file stateFile
	if err := json.Unmarshal(bytes, &file); err != nil {
		return nil, err
	}
	if file.Version != 1 {
		return nil, fmt.Errorf("unsupported session state version %d", file.Version)
	}
	return file.Sessions, nil
}

func (s *JSONStore) SaveSessions(_ context.Context, sessions []session.Session) error {
	file := stateFile{Version: 1, Sessions: sessions}
	bytes, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	bytes = append(bytes, '\n')
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(s.path), ".runtime-state-*.json")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := temp.Write(bytes); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
