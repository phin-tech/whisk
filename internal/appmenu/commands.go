// Package appmenu builds Whisk's native macOS menu bar and keeps its editable keyboard shortcuts
// in sync with persisted settings. The command registry and the pure resolution/validation helpers
// live here so they can be unit-tested without constructing a live Wails application menu; the
// Controller (controller.go) wires the resolved state into the real *application.Menu.
package appmenu

import (
	"fmt"
	"strings"

	"github.com/phin-tech/whisk/internal/appsettings"
)

// Command categories shown as section headings in the Keyboard Shortcuts panel.
const (
	CategoryApplication = "Application"
	CategorySessions    = "Sessions"
	CategoryTerminal    = "Terminal"
	CategoryStandard    = "Standard"
)

// Command ids. The session ids are positional: SelectSessionID(0) addresses the first session in
// the session bar, SelectSessionID(9) the tenth.
const (
	CommandOpenPreferences     = "open-preferences"
	CommandOpenPalette         = "open-palette"
	CommandToggleSidebar       = "toggle-sidebar"
	CommandSplitPaneVertical   = "split-pane-vertical"
	CommandSplitPaneHorizontal = "split-pane-horizontal"
	CommandClosePane           = "close-pane"
	CommandCloseSession        = "close-session"
	CommandJumpToBottom        = "terminal.bottom"
	selectSessionPrefix        = "select-session-"
)

// SessionSlots is the number of session-switch shortcuts (Cmd+1..Cmd+9, Cmd+0).
const SessionSlots = 10

// Command describes a single bindable action. Editable commands can be rebound from the Keyboard
// Shortcuts panel; non-editable commands are standard macOS items shown for reference only (their
// accelerators come from native menu roles, not from us).
type Command struct {
	ID       string
	Label    string
	Category string
	Default  string
	Editable bool
}

// CommandView is the JSON-friendly projection sent to the frontend panel, carrying the effective
// accelerator (override if set, otherwise the default).
type CommandView struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Category    string `json:"category"`
	Accelerator string `json:"accelerator"`
	Default     string `json:"default"`
	Editable    bool   `json:"editable"`
}

// KeybindingsView is the payload returned by the service for the panel: the full command registry
// with each command's effective accelerator resolved.
type KeybindingsView struct {
	Commands []CommandView `json:"commands"`
}

// SelectSessionID returns the command id for the session in slot i (0-based).
func SelectSessionID(i int) string {
	return fmt.Sprintf("%s%d", selectSessionPrefix, i+1)
}

// defaultSessionAccelerator returns the built-in accelerator for session slot i: Cmd+1..Cmd+9 for
// the first nine, Cmd+0 for the tenth.
func defaultSessionAccelerator(i int) string {
	digit := i + 1
	if digit == SessionSlots {
		digit = 0
	}
	return fmt.Sprintf("CmdOrCtrl+%d", digit)
}

// Commands returns the full registry in display order: the editable app commands first, then the
// read-only standard macOS items surfaced for reference in the panel.
func Commands() []Command {
	commands := make([]Command, 0, 6+SessionSlots+8)
	commands = append(commands, Command{
		ID:       CommandOpenPreferences,
		Label:    "Open Preferences",
		Category: CategoryApplication,
		Default:  "CmdOrCtrl+,",
		Editable: true,
	})
	commands = append(commands, Command{
		ID:       CommandOpenPalette,
		Label:    "Open Command Palette",
		Category: CategoryApplication,
		Default:  "CmdOrCtrl+K",
		Editable: true,
	})
	commands = append(commands, Command{
		ID:       CommandToggleSidebar,
		Label:    "Show/Hide Sidebar",
		Category: CategoryApplication,
		Default:  "CmdOrCtrl+\\",
		Editable: true,
	})
	commands = append(commands,
		Command{
			ID:       CommandSplitPaneVertical,
			Label:    "Split Pane Vertically",
			Category: CategorySessions,
			Default:  "CmdOrCtrl+D",
			Editable: true,
		},
		Command{
			ID:       CommandSplitPaneHorizontal,
			Label:    "Split Pane Horizontally",
			Category: CategorySessions,
			Default:  "CmdOrCtrl+Shift+D",
			Editable: true,
		},
		Command{
			ID:       CommandClosePane,
			Label:    "Close Pane",
			Category: CategorySessions,
			Default:  "CmdOrCtrl+W",
			Editable: true,
		},
		Command{
			ID:       CommandCloseSession,
			Label:    "Close Session",
			Category: CategorySessions,
			Default:  "CmdOrCtrl+Shift+W",
			Editable: true,
		},
	)
	commands = append(commands, Command{
		ID:       CommandJumpToBottom,
		Label:    "Jump to Bottom",
		Category: CategoryTerminal,
		Default:  "CmdOrCtrl+Alt+Down",
		Editable: true,
	})
	for i := 0; i < SessionSlots; i++ {
		commands = append(commands, Command{
			ID:       SelectSessionID(i),
			Label:    fmt.Sprintf("Switch to Session %d", i+1),
			Category: CategorySessions,
			Default:  defaultSessionAccelerator(i),
			Editable: true,
		})
	}
	// Reference-only rows. These are provided by native menu roles (AppMenu/EditMenu/WindowMenu)
	// and are not rebindable; they appear in the panel so users can see the full shortcut map.
	standard := []Command{
		{ID: "std-quit", Label: "Quit Whisk", Default: "CmdOrCtrl+Q"},
		{ID: "std-undo", Label: "Undo", Default: "CmdOrCtrl+Z"},
		{ID: "std-redo", Label: "Redo", Default: "CmdOrCtrl+Shift+Z"},
		{ID: "std-cut", Label: "Cut", Default: "CmdOrCtrl+X"},
		{ID: "std-copy", Label: "Copy", Default: "CmdOrCtrl+C"},
		{ID: "std-paste", Label: "Paste", Default: "CmdOrCtrl+V"},
		{ID: "std-select-all", Label: "Select All", Default: "CmdOrCtrl+A"},
		{ID: "std-minimize", Label: "Minimize", Default: "CmdOrCtrl+M"},
	}
	for _, cmd := range standard {
		cmd.Category = CategoryStandard
		cmd.Editable = false
		commands = append(commands, cmd)
	}
	return commands
}

// Resolve returns the effective accelerator for every command: the user override when one exists
// for an editable command, otherwise the built-in default.
func Resolve(settings appsettings.Settings) map[string]string {
	effective := make(map[string]string, len(Commands()))
	for _, cmd := range Commands() {
		accelerator := cmd.Default
		if cmd.Editable {
			if override, ok := settings.Keybindings[cmd.ID]; ok && strings.TrimSpace(override) != "" {
				accelerator = strings.TrimSpace(override)
			}
		}
		effective[cmd.ID] = accelerator
	}
	return effective
}

// View resolves the registry against settings into the panel payload.
func View(settings appsettings.Settings) KeybindingsView {
	effective := Resolve(settings)
	commands := Commands()
	views := make([]CommandView, 0, len(commands))
	for _, cmd := range commands {
		views = append(views, CommandView{
			ID:          cmd.ID,
			Label:       cmd.Label,
			Category:    cmd.Category,
			Accelerator: effective[cmd.ID],
			Default:     cmd.Default,
			Editable:    cmd.Editable,
		})
	}
	return KeybindingsView{Commands: views}
}

// editableIDs returns the set of command ids that accept user overrides.
func editableIDs() map[string]struct{} {
	ids := make(map[string]struct{})
	for _, cmd := range Commands() {
		if cmd.Editable {
			ids[cmd.ID] = struct{}{}
		}
	}
	return ids
}

// SanitizeOverrides validates a set of proposed overrides and returns the subset that should be
// persisted: entries for known editable commands whose accelerator is valid and differs from the
// command's default. It returns an error if any provided override targets an unknown/non-editable
// command or carries an invalid accelerator, so the panel surfaces mistakes instead of silently
// dropping them.
func SanitizeOverrides(overrides map[string]string) (map[string]string, error) {
	editable := editableIDs()
	defaults := make(map[string]string)
	for _, cmd := range Commands() {
		defaults[cmd.ID] = cmd.Default
	}
	cleaned := make(map[string]string)
	for id, accelerator := range overrides {
		id = strings.TrimSpace(id)
		accelerator = strings.TrimSpace(accelerator)
		if accelerator == "" {
			// Treat a blank as "reset to default": skip persisting it.
			continue
		}
		if _, ok := editable[id]; !ok {
			return nil, fmt.Errorf("unknown or non-editable command %q", id)
		}
		if err := validateAccelerator(accelerator); err != nil {
			return nil, fmt.Errorf("command %q: %w", id, err)
		}
		if accelerator == defaults[id] {
			// Equal to default: no override needed.
			continue
		}
		cleaned[id] = accelerator
	}
	if len(cleaned) == 0 {
		return nil, nil
	}
	return cleaned, nil
}

// namedKeys mirrors the named keys the Wails accelerator parser accepts (keys.go). The parser is
// unexported, so we keep a light copy to validate user input before it reaches SetAccelerator.
var namedKeys = map[string]struct{}{
	"backspace": {}, "tab": {}, "return": {}, "enter": {}, "escape": {},
	"left": {}, "right": {}, "up": {}, "down": {}, "space": {}, "delete": {},
	"home": {}, "end": {}, "page up": {}, "page down": {}, "numlock": {}, "plus": {},
}

var validModifiers = map[string]struct{}{
	"cmd": {}, "command": {}, "cmdorctrl": {}, "ctrl": {}, "control": {},
	"alt": {}, "option": {}, "optionoralt": {}, "shift": {}, "super": {},
}

// validateAccelerator performs the same shape check as the Wails parser: components are split on
// "+", every component but the last must be a modifier, and the last must be a single printable
// key or a known named key.
func validateAccelerator(accelerator string) error {
	components := strings.Split(accelerator, "+")
	if len(components) == 0 {
		return fmt.Errorf("empty accelerator")
	}
	for index, component := range components {
		lower := strings.ToLower(component)
		if index == len(components)-1 {
			if isValidKey(lower) {
				continue
			}
			return fmt.Errorf("%q is not a valid key", component)
		}
		if _, ok := validModifiers[lower]; !ok {
			return fmt.Errorf("%q is not a valid modifier", component)
		}
	}
	return nil
}

func isValidKey(key string) bool {
	if _, ok := namedKeys[key]; ok {
		return true
	}
	// Function keys f1..f35.
	if strings.HasPrefix(key, "f") && len(key) > 1 {
		if _, err := fmt.Sscanf(key, "f%d", new(int)); err == nil {
			return true
		}
	}
	return len([]rune(key)) == 1
}
