package terminal_test

import (
	"testing"

	"github.com/phin-tech/whisk/internal/domain/terminal"
)

func TestModeTrackerTracksDECPrivateModes(t *testing.T) {
	tests := []struct {
		name string
		seq  string
		want terminal.Modes
	}{
		{
			name: "application cursor",
			seq:  "\x1b[?1h",
			want: terminal.Modes{ApplicationCursor: true, CursorVisible: true},
		},
		{
			name: "hide cursor",
			seq:  "\x1b[?25l",
			want: terminal.Modes{CursorVisible: false},
		},
		{
			name: "normal mouse",
			seq:  "\x1b[?1000h",
			want: terminal.Modes{CursorVisible: true, MouseTracking: terminal.MouseTrackingNormal},
		},
		{
			name: "button mouse",
			seq:  "\x1b[?1002h",
			want: terminal.Modes{CursorVisible: true, MouseTracking: terminal.MouseTrackingButton},
		},
		{
			name: "any mouse",
			seq:  "\x1b[?1003h",
			want: terminal.Modes{CursorVisible: true, MouseTracking: terminal.MouseTrackingAny},
		},
		{
			name: "sgr mouse",
			seq:  "\x1b[?1006h",
			want: terminal.Modes{CursorVisible: true, MouseEncoding: terminal.MouseEncodingSGR},
		},
		{
			name: "sgr pixel mouse",
			seq:  "\x1b[?1016h",
			want: terminal.Modes{CursorVisible: true, MouseEncoding: terminal.MouseEncodingSGRPixel},
		},
		{
			name: "alt screen",
			seq:  "\x1b[?1047h",
			want: terminal.Modes{CursorVisible: true, AltScreen: true},
		},
		{
			name: "save cursor",
			seq:  "\x1b[?1048h",
			want: terminal.Modes{CursorVisible: true, SaveCursor: true},
		},
		{
			name: "alt screen save cursor",
			seq:  "\x1b[?1049h",
			want: terminal.Modes{CursorVisible: true, AltScreen: true, SaveCursor: true, AltScreenSave: true},
		},
		{
			name: "bracketed paste",
			seq:  "\x1b[?2004h",
			want: terminal.Modes{CursorVisible: true, BracketedPaste: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := terminal.NewModeTracker()
			tracker.Feed([]byte(tt.seq))

			if got := tracker.Modes(); got != tt.want {
				t.Fatalf("modes = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestModeTrackerResetsModes(t *testing.T) {
	tracker := terminal.NewModeTracker()

	tracker.Feed([]byte("\x1b[?1;1003;1016;1049;2004h\x1b[?25l"))
	tracker.Feed([]byte("\x1bc"))

	want := terminal.Modes{CursorVisible: true}
	if got := tracker.Modes(); got != want {
		t.Fatalf("modes = %#v, want %#v", got, want)
	}
}

func TestModeTrackerHandlesModeReset(t *testing.T) {
	tracker := terminal.NewModeTracker()

	tracker.Feed([]byte("\x1b[?1;25;1002;1006;1049;2004h"))
	tracker.Feed([]byte("\x1b[?1;25;1002;1006;1049;2004l"))

	want := terminal.Modes{CursorVisible: false}
	if got := tracker.Modes(); got != want {
		t.Fatalf("modes = %#v, want %#v", got, want)
	}
}
