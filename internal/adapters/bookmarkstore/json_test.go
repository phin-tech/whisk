package bookmarkstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/adapters/bookmarkstore"
	"github.com/phin-tech/whisk/internal/domain/ptybookmark"
)

func TestDefaultPathUsesXDGConfigHomeOrHomeDotConfig(t *testing.T) {
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path, err := bookmarkstore.DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	if path != filepath.Join(xdg, "whisk", "bookmarks.json") {
		t.Fatalf("path = %q", path)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err = bookmarkstore.DefaultPath()
	if err != nil {
		t.Fatalf("default path with home: %v", err)
	}
	if path != filepath.Join(home, ".config", "whisk", "bookmarks.json") {
		t.Fatalf("path = %q", path)
	}
}

func TestJSONStoreLoadMissingFileReturnsEmpty(t *testing.T) {
	store, err := bookmarkstore.NewJSONStore(filepath.Join(t.TempDir(), "missing", "bookmarks.json"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	bookmarks, err := store.LoadBookmarks(context.Background())
	if err != nil {
		t.Fatalf("load bookmarks: %v", err)
	}
	if len(bookmarks) != 0 {
		t.Fatalf("bookmarks = %#v", bookmarks)
	}
}

func TestJSONStoreRoundTripsBookmarks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "whisk", "bookmarks.json")
	store, err := bookmarkstore.NewJSONStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	input := []ptybookmark.Bookmark{
		{ID: "bm_01", PTYID: "pty_01", Offset: 10, Kind: "prompt", Label: "Prompt", CreatedAt: createdAt},
	}
	if err := store.SaveBookmarks(context.Background(), input); err != nil {
		t.Fatalf("save bookmarks: %v", err)
	}
	loaded, err := store.LoadBookmarks(context.Background())
	if err != nil {
		t.Fatalf("load bookmarks: %v", err)
	}
	if len(loaded) != 1 || loaded[0].ID != "bm_01" || loaded[0].CreatedAt.IsZero() {
		t.Fatalf("loaded = %#v", loaded)
	}
}
