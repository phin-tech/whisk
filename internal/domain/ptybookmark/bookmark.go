package ptybookmark

import (
	"fmt"
	"sort"
	"time"
)

type Bookmark struct {
	ID        string    `json:"id"`
	PTYID     string    `json:"ptyId"`
	SessionID string    `json:"sessionId"`
	WindowID  string    `json:"windowId"`
	PaneID    string    `json:"paneId"`
	Offset    uint64    `json:"offset"`
	Kind      string    `json:"kind"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"createdAt"`
}

type AddBookmark struct {
	ID        string
	PTYID     string
	SessionID string
	WindowID  string
	PaneID    string
	Offset    uint64
	Kind      string
	Label     string
	CreatedAt time.Time
}

type State struct {
	bookmarks map[string]Bookmark
}

func NewState() *State {
	return &State{bookmarks: map[string]Bookmark{}}
}

func NewStateFromBookmarks(bookmarks []Bookmark) (*State, error) {
	state := NewState()
	if err := state.ReplaceBookmarks(bookmarks); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *State) ReplaceBookmarks(bookmarks []Bookmark) error {
	next := make(map[string]Bookmark, len(bookmarks))
	for _, bookmark := range bookmarks {
		normalized, err := validateBookmark(bookmark)
		if err != nil {
			return err
		}
		if _, exists := next[normalized.ID]; exists {
			return fmt.Errorf("bookmark %s already exists", normalized.ID)
		}
		next[normalized.ID] = normalized
	}
	s.bookmarks = next
	return nil
}

func (s *State) Add(req AddBookmark) (Bookmark, error) {
	bookmark, err := validateBookmark(Bookmark{
		ID:        req.ID,
		PTYID:     req.PTYID,
		SessionID: req.SessionID,
		WindowID:  req.WindowID,
		PaneID:    req.PaneID,
		Offset:    req.Offset,
		Kind:      req.Kind,
		Label:     req.Label,
		CreatedAt: req.CreatedAt,
	})
	if err != nil {
		return Bookmark{}, err
	}
	if _, exists := s.bookmarks[bookmark.ID]; exists {
		return Bookmark{}, fmt.Errorf("bookmark %s already exists", bookmark.ID)
	}
	s.bookmarks[bookmark.ID] = bookmark
	return bookmark, nil
}

func (s *State) Remove(id string) (Bookmark, error) {
	bookmark, ok := s.bookmarks[id]
	if !ok {
		return Bookmark{}, fmt.Errorf("bookmark %s not found", id)
	}
	delete(s.bookmarks, id)
	return bookmark, nil
}

func (s *State) List(ptyID string) []Bookmark {
	out := make([]Bookmark, 0, len(s.bookmarks))
	for _, bookmark := range s.bookmarks {
		if ptyID != "" && bookmark.PTYID != ptyID {
			continue
		}
		out = append(out, bookmark)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PTYID == out[j].PTYID {
			if out[i].Offset == out[j].Offset {
				return out[i].ID < out[j].ID
			}
			return out[i].Offset < out[j].Offset
		}
		return out[i].PTYID < out[j].PTYID
	})
	return out
}

func validateBookmark(bookmark Bookmark) (Bookmark, error) {
	if bookmark.ID == "" {
		return Bookmark{}, fmt.Errorf("bookmark id required")
	}
	if bookmark.PTYID == "" {
		return Bookmark{}, fmt.Errorf("pty id required")
	}
	if bookmark.CreatedAt.IsZero() {
		return Bookmark{}, fmt.Errorf("created at required")
	}
	if bookmark.Kind == "" {
		bookmark.Kind = "manual"
	}
	return bookmark, nil
}
