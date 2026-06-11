package ptybookmark_test

import (
	"testing"
	"time"

	"github.com/phin-tech/whisk/internal/domain/ptybookmark"
)

func TestStateAddsListsAndRemovesBookmarks(t *testing.T) {
	state := ptybookmark.NewState()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	if _, err := state.Add(ptybookmark.AddBookmark{ID: "bm_02", PTYID: "pty_01", Offset: 20, CreatedAt: now}); err != nil {
		t.Fatalf("add bookmark: %v", err)
	}
	if _, err := state.Add(ptybookmark.AddBookmark{ID: "bm_01", PTYID: "pty_01", Offset: 10, Kind: "prompt", Label: "Prompt", CreatedAt: now}); err != nil {
		t.Fatalf("add bookmark: %v", err)
	}
	if _, err := state.Add(ptybookmark.AddBookmark{ID: "bm_other", PTYID: "pty_02", Offset: 5, CreatedAt: now}); err != nil {
		t.Fatalf("add bookmark: %v", err)
	}

	bookmarks := state.List("pty_01")
	if len(bookmarks) != 2 || bookmarks[0].ID != "bm_01" || bookmarks[1].ID != "bm_02" {
		t.Fatalf("bookmarks = %#v", bookmarks)
	}
	if bookmarks[1].Kind != "manual" {
		t.Fatalf("default kind = %q", bookmarks[1].Kind)
	}

	removed, err := state.Remove("bm_01")
	if err != nil {
		t.Fatalf("remove bookmark: %v", err)
	}
	if removed.ID != "bm_01" || removed.PTYID != "pty_01" {
		t.Fatalf("removed = %#v", removed)
	}
	bookmarks = state.List("pty_01")
	if len(bookmarks) != 1 || bookmarks[0].ID != "bm_02" {
		t.Fatalf("bookmarks after remove = %#v", bookmarks)
	}
}

func TestStateRejectsInvalidBookmarks(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		req  ptybookmark.AddBookmark
	}{
		{name: "missing id", req: ptybookmark.AddBookmark{PTYID: "pty_01", CreatedAt: now}},
		{name: "missing pty", req: ptybookmark.AddBookmark{ID: "bm_01", CreatedAt: now}},
		{name: "missing created at", req: ptybookmark.AddBookmark{ID: "bm_01", PTYID: "pty_01"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ptybookmark.NewState().Add(test.req); err == nil {
				t.Fatalf("expected error")
			}
		})
	}

	state := ptybookmark.NewState()
	if _, err := state.Add(ptybookmark.AddBookmark{ID: "bm_01", PTYID: "pty_01", CreatedAt: now}); err != nil {
		t.Fatalf("add bookmark: %v", err)
	}
	if _, err := state.Add(ptybookmark.AddBookmark{ID: "bm_01", PTYID: "pty_01", CreatedAt: now}); err == nil {
		t.Fatalf("expected duplicate error")
	}
	if _, err := state.Remove("missing"); err == nil {
		t.Fatalf("expected missing remove error")
	}
}

func TestNewStateFromBookmarksValidatesPersistedBookmarks(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	state, err := ptybookmark.NewStateFromBookmarks([]ptybookmark.Bookmark{
		{ID: "bm_01", PTYID: "pty_01", Offset: 5, CreatedAt: now},
	})
	if err != nil {
		t.Fatalf("restore bookmarks: %v", err)
	}
	if got := state.List(""); len(got) != 1 || got[0].Kind != "manual" {
		t.Fatalf("restored = %#v", got)
	}

	if _, err := ptybookmark.NewStateFromBookmarks([]ptybookmark.Bookmark{{ID: "bad"}}); err == nil {
		t.Fatalf("expected invalid restore error")
	}
}
