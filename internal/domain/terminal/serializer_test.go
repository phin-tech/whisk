package terminal_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/phin-tech/whisk/internal/domain/terminal"
)

func TestSnapshotRehydrateOrderingForAltScreenModes(t *testing.T) {
	state := terminal.New(20, 4, terminal.Options{})
	input := "main\n" +
		"\x1b[?1049h" +
		"\x1b[?1003;1016;2004h" +
		"alt"
	if err := state.Write(0, []byte(input)); err != nil {
		t.Fatalf("write: %v", err)
	}

	snapshot := state.Snapshot()
	if snapshot.RehydrateBeforeViewport != ansi.SetModeAltScreenSaveCursor {
		t.Fatalf("rehydrate before viewport = %q", snapshot.RehydrateBeforeViewport)
	}
	wantTail := ansi.SetModeMouseAnyEvent +
		ansi.SetModeMouseExtSgrPixel +
		ansi.SetModeBracketedPaste +
		"\x1b[1;4H" +
		ansi.SetModeTextCursorEnable
	if snapshot.RehydrateSequences != wantTail {
		t.Fatalf("rehydrate sequences = %q, want %q", snapshot.RehydrateSequences, wantTail)
	}
	tail := snapshot.RehydrateSequences
	assertOrder(t, tail,
		ansi.SetModeMouseAnyEvent,
		ansi.SetModeMouseExtSgrPixel,
		ansi.SetModeBracketedPaste,
		"\x1b[",
		ansi.SetModeTextCursorEnable,
	)
	if strings.Contains(snapshot.ViewportAnsi, ansi.SetModeBracketedPaste) {
		t.Fatalf("viewport should not contain tail mode rehydrate sequence: %q", snapshot.ViewportAnsi)
	}
	if !snapshot.Modes.AltScreen || !snapshot.Modes.BracketedPaste ||
		snapshot.Modes.MouseTracking != terminal.MouseTrackingAny ||
		snapshot.Modes.MouseEncoding != terminal.MouseEncodingSGRPixel {
		t.Fatalf("modes = %#v", snapshot.Modes)
	}
}

func TestSnapshotResetClearsRehydrateModes(t *testing.T) {
	state := terminal.New(20, 4, terminal.Options{})
	input := "\x1b[?1;1002;1006;1049;2004h\x1b[?25l\x1bc"
	if err := state.Write(0, []byte(input)); err != nil {
		t.Fatalf("write: %v", err)
	}

	snapshot := state.Snapshot()
	if snapshot.RehydrateBeforeViewport != "" {
		t.Fatalf("rehydrate before viewport = %q", snapshot.RehydrateBeforeViewport)
	}
	for _, seq := range []string{
		ansi.SetModeCursorKeys,
		ansi.SetModeMouseButtonEvent,
		ansi.SetModeMouseExtSgr,
		ansi.SetModeAltScreenSaveCursor,
		ansi.SetModeBracketedPaste,
		ansi.ResetModeTextCursorEnable,
	} {
		if strings.Contains(snapshot.RehydrateSequences, seq) {
			t.Fatalf("rehydrate sequences %q unexpectedly contain %q", snapshot.RehydrateSequences, seq)
		}
	}
	if !strings.Contains(snapshot.RehydrateSequences, ansi.SetModeTextCursorEnable) {
		t.Fatalf("rehydrate sequences should restore default visible cursor: %q", snapshot.RehydrateSequences)
	}
}

func TestSnapshotRehydratesMouseModeStackForFutureResets(t *testing.T) {
	state := terminal.New(20, 4, terminal.Options{})
	input := "\x1b[?1000;1002;1006;1016hstacked modes"
	if err := state.Write(0, []byte(input)); err != nil {
		t.Fatalf("write: %v", err)
	}

	snapshot := state.Snapshot()
	wantTrackingModes := []terminal.MouseTrackingMode{
		terminal.MouseTrackingNormal,
		terminal.MouseTrackingButton,
	}
	if !reflect.DeepEqual(snapshot.MouseTrackingModes, wantTrackingModes) {
		t.Fatalf("mouse tracking modes = %#v, want %#v", snapshot.MouseTrackingModes, wantTrackingModes)
	}
	wantEncodingModes := []terminal.MouseEncodingMode{
		terminal.MouseEncodingSGR,
		terminal.MouseEncodingSGRPixel,
	}
	if !reflect.DeepEqual(snapshot.MouseEncodingModes, wantEncodingModes) {
		t.Fatalf("mouse encoding modes = %#v, want %#v", snapshot.MouseEncodingModes, wantEncodingModes)
	}
	assertOrder(t, snapshot.RehydrateSequences,
		ansi.SetModeMouseNormal,
		ansi.SetModeMouseButtonEvent,
		ansi.SetModeMouseExtSgr,
		ansi.SetModeMouseExtSgrPixel,
	)

	tracker := terminal.NewModeTracker()
	tracker.Feed([]byte(snapshot.RehydrateSequences))
	tracker.Feed([]byte("\x1b[?1002;1016l"))

	wantModes := terminal.Modes{
		CursorVisible: true,
		MouseTracking: terminal.MouseTrackingNormal,
		MouseEncoding: terminal.MouseEncodingSGR,
	}
	if got := tracker.Modes(); got != wantModes {
		t.Fatalf("rehydrated modes after live reset = %#v, want %#v", got, wantModes)
	}
}

func assertOrder(t *testing.T, value string, parts ...string) {
	t.Helper()
	at := 0
	for _, part := range parts {
		next := strings.Index(value[at:], part)
		if next < 0 {
			t.Fatalf("%q does not contain %q after byte %d", value, part, at)
		}
		at += next + len(part)
	}
}
