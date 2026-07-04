package terminal

import (
	"fmt"

	"github.com/charmbracelet/x/vt"
)

type State struct {
	term   *vt.Emulator
	modes  *ModeTracker
	opts   Options
	offset uint64
	title  string
	cwd    string
}

func New(cols, rows int, opts Options) *State {
	cols, rows = normalizeSize(cols, rows)
	opts = normalizeOptions(opts)
	state := &State{
		term:  vt.NewEmulator(cols, rows),
		modes: NewModeTracker(),
		opts:  opts,
	}
	state.term.SetScrollbackSize(opts.MaxScrollbackLines)
	state.term.SetCallbacks(vt.Callbacks{
		Title: func(title string) {
			state.title = sanitizeText(title, state.opts.MaxTitleBytes)
		},
		WorkingDirectory: func(cwd string) {
			state.cwd = sanitizeText(cwd, state.opts.MaxWorkingDirectoryBytes)
		},
	})
	return state
}

func (s *State) Write(offset uint64, data []byte) error {
	if offset != s.offset {
		return fmt.Errorf("terminal write offset %d does not match next offset %d", offset, s.offset)
	}
	if len(data) == 0 {
		return nil
	}
	n, err := s.term.Write(data)
	if err != nil {
		return fmt.Errorf("write terminal emulator: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("write terminal emulator: wrote %d of %d bytes", n, len(data))
	}
	s.modes.Feed(data)
	s.offset += uint64(len(data))
	return nil
}

func (s *State) Resize(cols, rows int) {
	cols, rows = normalizeSize(cols, rows)
	s.term.Resize(cols, rows)
}

func (s *State) Offset() uint64 {
	return s.offset
}

func (s *State) Snapshot() Snapshot {
	return snapshotFromState(s)
}
