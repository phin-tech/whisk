package terminal

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

func snapshotFromState(s *State) Snapshot {
	modes := s.modes.Modes()
	mouseTrackingModes := s.modes.MouseTrackingModes()
	mouseEncodingModes := s.modes.MouseEncodingModes()
	cursor := s.term.CursorPosition()
	scrollback, scrollbackTruncated := renderScrollback(s)
	viewport, viewportTruncated := renderViewport(s)
	snapshot := Snapshot{
		Offset:                  s.offset,
		Cols:                    s.term.Width(),
		Rows:                    s.term.Height(),
		Cursor:                  Cursor{X: cursor.X, Y: cursor.Y},
		Title:                   s.title,
		WorkingDirectory:        s.cwd,
		ScrollbackAnsi:          scrollback,
		RehydrateBeforeViewport: rehydrateBeforeViewport(modes),
		ViewportAnsi:            viewport,
		RehydrateSequences:      rehydrateSequences(modes, mouseTrackingModes, mouseEncodingModes, Cursor{X: cursor.X, Y: cursor.Y}),
		Modes:                   modes,
		MouseTrackingModes:      mouseTrackingModes,
		MouseEncodingModes:      mouseEncodingModes,
		Truncated:               scrollbackTruncated || viewportTruncated,
	}
	return snapshot
}

func renderScrollback(s *State) (string, bool) {
	scrollback := s.term.Scrollback()
	if scrollback == nil || scrollback.Len() == 0 {
		return "", false
	}
	lines := make([]string, 0, scrollback.Len())
	for _, line := range scrollback.Lines() {
		lines = append(lines, line.Render())
	}
	return joinTailBounded(lines, "\r\n", s.opts.MaxSnapshotFieldBytes)
}

func renderViewport(s *State) (string, bool) {
	lines := make([]string, 0, s.term.Height())
	for y := 0; y < s.term.Height(); y++ {
		line := make(uv.Line, s.term.Width())
		for x := 0; x < s.term.Width(); x++ {
			cell := s.term.CellAt(x, y)
			if cell == nil {
				line[x] = uv.EmptyCell
				continue
			}
			line[x] = *cell
		}
		lines = append(lines, line.Render())
	}
	const prefix = "\x1b[H\x1b[2J"
	if s.opts.MaxSnapshotFieldBytes < len(prefix) {
		return "", true
	}
	viewport, truncated := joinHeadBounded(lines, "\r\n", s.opts.MaxSnapshotFieldBytes-len(prefix))
	return prefix + viewport, truncated
}

func joinTailBounded(lines []string, sep string, maxBytes int) (string, bool) {
	if len(lines) == 0 || maxBytes <= 0 {
		return "", len(lines) > 0
	}
	var kept []string
	size := 0
	for i := len(lines) - 1; i >= 0; i-- {
		next := len(lines[i])
		if len(kept) > 0 {
			next += len(sep)
		}
		if size+next > maxBytes {
			break
		}
		kept = append(kept, lines[i])
		size += next
	}
	if len(kept) == 0 {
		return "", true
	}
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}
	return strings.Join(kept, sep), len(kept) < len(lines)
}

func joinHeadBounded(lines []string, sep string, maxBytes int) (string, bool) {
	if len(lines) == 0 || maxBytes <= 0 {
		return "", len(lines) > 0
	}
	kept := make([]string, 0, len(lines))
	size := 0
	for _, line := range lines {
		next := len(line)
		if len(kept) > 0 {
			next += len(sep)
		}
		if size+next > maxBytes {
			break
		}
		kept = append(kept, line)
		size += next
	}
	return strings.Join(kept, sep), len(kept) < len(lines)
}

func rehydrateBeforeViewport(modes Modes) string {
	if !modes.AltScreen {
		return ""
	}
	return ansi.SetModeAltScreenSaveCursor
}

func rehydrateSequences(modes Modes, mouseTrackingModes []MouseTrackingMode, mouseEncodingModes []MouseEncodingMode, cursor Cursor) string {
	var b strings.Builder
	if modes.ApplicationCursor {
		b.WriteString(ansi.SetModeCursorKeys)
	}
	writeMouseTrackingModes(&b, modes, mouseTrackingModes)
	writeMouseEncodingModes(&b, modes, mouseEncodingModes)
	if modes.BracketedPaste {
		b.WriteString(ansi.SetModeBracketedPaste)
	}
	fmt.Fprintf(&b, "\x1b[%d;%dH", cursor.Y+1, cursor.X+1)
	if modes.CursorVisible {
		b.WriteString(ansi.SetModeTextCursorEnable)
	} else {
		b.WriteString(ansi.ResetModeTextCursorEnable)
	}
	return b.String()
}

func writeMouseTrackingModes(b *strings.Builder, modes Modes, mouseTrackingModes []MouseTrackingMode) {
	if len(mouseTrackingModes) == 0 {
		mouseTrackingModes = collapsedMouseTrackingModes(modes.MouseTracking)
	}
	for _, mode := range mouseTrackingModes {
		switch mode {
		case MouseTrackingNormal:
			b.WriteString(ansi.SetModeMouseNormal)
		case MouseTrackingButton:
			b.WriteString(ansi.SetModeMouseButtonEvent)
		case MouseTrackingAny:
			b.WriteString(ansi.SetModeMouseAnyEvent)
		}
	}
}

func collapsedMouseTrackingModes(mode MouseTrackingMode) []MouseTrackingMode {
	switch mode {
	case MouseTrackingNormal:
		return []MouseTrackingMode{MouseTrackingNormal}
	case MouseTrackingButton:
		return []MouseTrackingMode{MouseTrackingButton}
	case MouseTrackingAny:
		return []MouseTrackingMode{MouseTrackingAny}
	default:
		return nil
	}
}

func writeMouseEncodingModes(b *strings.Builder, modes Modes, mouseEncodingModes []MouseEncodingMode) {
	if len(mouseEncodingModes) == 0 {
		mouseEncodingModes = collapsedMouseEncodingModes(modes.MouseEncoding)
	}
	for _, mode := range mouseEncodingModes {
		switch mode {
		case MouseEncodingSGR:
			b.WriteString(ansi.SetModeMouseExtSgr)
		case MouseEncodingSGRPixel:
			b.WriteString(ansi.SetModeMouseExtSgrPixel)
		}
	}
}

func collapsedMouseEncodingModes(mode MouseEncodingMode) []MouseEncodingMode {
	switch mode {
	case MouseEncodingSGR:
		return []MouseEncodingMode{MouseEncodingSGR}
	case MouseEncodingSGRPixel:
		return []MouseEncodingMode{MouseEncodingSGRPixel}
	default:
		return nil
	}
}

func sanitizeText(value string, maxBytes int) string {
	value = strings.ToValidUTF8(value, "")
	var b strings.Builder
	for _, r := range value {
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return validPrefix(b.String(), maxBytes)
}

func validPrefix(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	value = value[:maxBytes]
	for !utf8.ValidString(value) {
		_, size := utf8.DecodeLastRuneInString(value)
		if size == 0 {
			return ""
		}
		value = value[:len(value)-size]
	}
	return value
}
