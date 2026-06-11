package bookmarkstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phin-tech/whisk/internal/domain/ptybookmark"
)

const bookmarkFileName = "bookmarks.json"

type JSONStore struct {
	path string
}

type bookmarkFile struct {
	Version   int                    `json:"version"`
	Bookmarks []ptybookmark.Bookmark `json:"bookmarks"`
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
	return filepath.Join(configDir, "whisk", bookmarkFileName), nil
}

func (s *JSONStore) LoadBookmarks(context.Context) ([]ptybookmark.Bookmark, error) {
	bytes, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var file bookmarkFile
	if err := json.Unmarshal(bytes, &file); err != nil {
		return nil, err
	}
	if file.Version != 1 {
		return nil, fmt.Errorf("unsupported bookmark state version %d", file.Version)
	}
	return file.Bookmarks, nil
}

func (s *JSONStore) SaveBookmarks(_ context.Context, bookmarks []ptybookmark.Bookmark) error {
	file := bookmarkFile{Version: 1, Bookmarks: bookmarks}
	bytes, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	bytes = append(bytes, '\n')
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(s.path), ".bookmarks-*.json")
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
