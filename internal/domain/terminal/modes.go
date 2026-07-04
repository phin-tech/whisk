package terminal

import (
	"github.com/charmbracelet/x/ansi"
)

type MouseTrackingMode string

const (
	MouseTrackingNone   MouseTrackingMode = ""
	MouseTrackingNormal MouseTrackingMode = "normal"
	MouseTrackingButton MouseTrackingMode = "button"
	MouseTrackingAny    MouseTrackingMode = "any"
)

type MouseEncodingMode string

const (
	MouseEncodingNone     MouseEncodingMode = ""
	MouseEncodingSGR      MouseEncodingMode = "sgr"
	MouseEncodingSGRPixel MouseEncodingMode = "sgrPixel"
)

type Modes struct {
	ApplicationCursor bool              `json:"applicationCursor"`
	CursorVisible     bool              `json:"cursorVisible"`
	AltScreen         bool              `json:"altScreen"`
	SaveCursor        bool              `json:"saveCursor,omitempty"`
	AltScreenSave     bool              `json:"altScreenSave,omitempty"`
	BracketedPaste    bool              `json:"bracketedPaste"`
	MouseTracking     MouseTrackingMode `json:"mouseTracking,omitempty"`
	MouseEncoding     MouseEncodingMode `json:"mouseEncoding,omitempty"`
}

type ModeTracker struct {
	parser                *ansi.Parser
	modes                 Modes
	mouseTrackingNormal   bool
	mouseTrackingButton   bool
	mouseTrackingAny      bool
	mouseEncodingSGR      bool
	mouseEncodingSGRPixel bool
}

func NewModeTracker() *ModeTracker {
	t := &ModeTracker{}
	t.Reset()
	t.parser = ansi.NewParser()
	t.parser.SetHandler(ansi.Handler{
		HandleCsi: t.handleCSI,
		HandleEsc: t.handleESC,
	})
	return t
}

func (t *ModeTracker) Feed(data []byte) {
	if len(data) == 0 {
		return
	}
	for _, b := range data {
		t.parser.Advance(b)
	}
}

func (t *ModeTracker) Modes() Modes {
	return t.modes
}

func (t *ModeTracker) Reset() {
	t.modes = Modes{CursorVisible: true}
	t.mouseTrackingNormal = false
	t.mouseTrackingButton = false
	t.mouseTrackingAny = false
	t.mouseEncodingSGR = false
	t.mouseEncodingSGRPixel = false
}

func (t *ModeTracker) handleESC(cmd ansi.Cmd) {
	if cmd.Prefix() == 0 && cmd.Intermediate() == 0 && cmd.Final() == 'c' {
		t.Reset()
	}
}

func (t *ModeTracker) handleCSI(cmd ansi.Cmd, params ansi.Params) {
	if cmd.Prefix() != '?' || cmd.Intermediate() != 0 {
		return
	}
	var set bool
	switch cmd.Final() {
	case 'h':
		set = true
	case 'l':
		set = false
	default:
		return
	}
	params.ForEach(0, func(_ int, mode int, hasMore bool) {
		if hasMore {
			return
		}
		t.applyDECPrivateMode(mode, set)
	})
}

func (t *ModeTracker) applyDECPrivateMode(mode int, set bool) {
	switch mode {
	case 1:
		t.modes.ApplicationCursor = set
	case 25:
		t.modes.CursorVisible = set
	case 1000:
		t.setMouseTracking(MouseTrackingNormal, set)
	case 1002:
		t.setMouseTracking(MouseTrackingButton, set)
	case 1003:
		t.setMouseTracking(MouseTrackingAny, set)
	case 1006:
		t.setMouseEncoding(MouseEncodingSGR, set)
	case 1016:
		t.setMouseEncoding(MouseEncodingSGRPixel, set)
	case 1047:
		t.modes.AltScreen = set
		if !set {
			t.modes.AltScreenSave = false
		}
	case 1048:
		t.modes.SaveCursor = set
	case 1049:
		t.modes.AltScreen = set
		t.modes.SaveCursor = set
		t.modes.AltScreenSave = set
	case 2004:
		t.modes.BracketedPaste = set
	}
}

func (t *ModeTracker) setMouseTracking(mode MouseTrackingMode, set bool) {
	switch mode {
	case MouseTrackingNormal:
		t.mouseTrackingNormal = set
	case MouseTrackingButton:
		t.mouseTrackingButton = set
	case MouseTrackingAny:
		t.mouseTrackingAny = set
	}
	t.refreshMouseTracking()
}

func (t *ModeTracker) refreshMouseTracking() {
	switch {
	case t.mouseTrackingAny:
		t.modes.MouseTracking = MouseTrackingAny
	case t.mouseTrackingButton:
		t.modes.MouseTracking = MouseTrackingButton
	case t.mouseTrackingNormal:
		t.modes.MouseTracking = MouseTrackingNormal
	default:
		t.modes.MouseTracking = MouseTrackingNone
	}
}

func (t *ModeTracker) setMouseEncoding(mode MouseEncodingMode, set bool) {
	switch mode {
	case MouseEncodingSGR:
		t.mouseEncodingSGR = set
	case MouseEncodingSGRPixel:
		t.mouseEncodingSGRPixel = set
	}
	t.refreshMouseEncoding()
}

func (t *ModeTracker) refreshMouseEncoding() {
	switch {
	case t.mouseEncodingSGRPixel:
		t.modes.MouseEncoding = MouseEncodingSGRPixel
	case t.mouseEncodingSGR:
		t.modes.MouseEncoding = MouseEncodingSGR
	default:
		t.modes.MouseEncoding = MouseEncodingNone
	}
}
