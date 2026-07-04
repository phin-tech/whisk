package appmenu

import (
	"testing"

	"github.com/phin-tech/whisk/internal/appsettings"
)

func TestCommandsIncludeAllSessionSlots(t *testing.T) {
	commands := Commands()
	byID := make(map[string]Command, len(commands))
	for _, cmd := range commands {
		byID[cmd.ID] = cmd
	}

	if _, ok := byID[CommandOpenPreferences]; !ok {
		t.Fatalf("registry missing %q", CommandOpenPreferences)
	}
	for i := 0; i < SessionSlots; i++ {
		id := SelectSessionID(i)
		cmd, ok := byID[id]
		if !ok {
			t.Fatalf("registry missing session command %q", id)
		}
		if !cmd.Editable {
			t.Fatalf("session command %q should be editable", id)
		}
	}
}

func TestCommandsIncludeSessionActionShortcuts(t *testing.T) {
	commands := Commands()
	byID := make(map[string]Command, len(commands))
	for _, cmd := range commands {
		byID[cmd.ID] = cmd
	}

	cases := map[string]struct {
		label    string
		shortcut string
	}{
		CommandSplitPaneVertical:   {"Split Pane Vertically", "CmdOrCtrl+D"},
		CommandSplitPaneHorizontal: {"Split Pane Horizontally", "CmdOrCtrl+Shift+D"},
		CommandClosePane:           {"Close Pane", "CmdOrCtrl+W"},
		CommandCloseSession:        {"Close Session", "CmdOrCtrl+Shift+W"},
	}
	for id, want := range cases {
		cmd, ok := byID[id]
		if !ok {
			t.Fatalf("registry missing session command %q", id)
		}
		if cmd.Label != want.label {
			t.Fatalf("%s label = %q, want %q", id, cmd.Label, want.label)
		}
		if cmd.Category != CategorySessions {
			t.Fatalf("%s category = %q, want %q", id, cmd.Category, CategorySessions)
		}
		if cmd.Default != want.shortcut {
			t.Fatalf("%s shortcut = %q, want %q", id, cmd.Default, want.shortcut)
		}
		if !cmd.Editable {
			t.Fatalf("%s should be editable", id)
		}
	}
}

func TestCommandsExcludeBookmarkShortcuts(t *testing.T) {
	commands := Commands()
	byID := make(map[string]Command, len(commands))
	for _, cmd := range commands {
		byID[cmd.ID] = cmd
	}

	removed := []string{
		"bookmark.add",
		"bookmark.previous",
		"bookmark.next",
		"bookmark.lastPrompt",
	}
	for _, id := range removed {
		if _, ok := byID[id]; ok {
			t.Fatalf("registry still includes bookmark command %q", id)
		}
	}

	cmd, ok := byID[CommandJumpToBottom]
	if !ok {
		t.Fatalf("registry missing jump to bottom command")
	}
	if cmd.Label != "Jump to Bottom" {
		t.Fatalf("jump to bottom label = %q", cmd.Label)
	}
	if cmd.Category != CategoryTerminal {
		t.Fatalf("jump to bottom category = %q, want %q", cmd.Category, CategoryTerminal)
	}
	if cmd.Default != "CmdOrCtrl+Alt+Down" {
		t.Fatalf("jump to bottom shortcut = %q, want CmdOrCtrl+Alt+Down", cmd.Default)
	}
	if !cmd.Editable {
		t.Fatalf("jump to bottom should be editable")
	}
}

func TestCommandsIncludeCommandPaletteShortcut(t *testing.T) {
	commands := Commands()
	byID := make(map[string]Command, len(commands))
	for _, cmd := range commands {
		byID[cmd.ID] = cmd
	}

	cmd, ok := byID[CommandOpenPalette]
	if !ok {
		t.Fatalf("registry missing command palette command")
	}
	if cmd.Label != "Open Command Palette" {
		t.Fatalf("label = %q", cmd.Label)
	}
	if cmd.Category != CategoryApplication {
		t.Fatalf("category = %q, want %q", cmd.Category, CategoryApplication)
	}
	if cmd.Default != "CmdOrCtrl+K" {
		t.Fatalf("shortcut = %q, want CmdOrCtrl+K", cmd.Default)
	}
	if !cmd.Editable {
		t.Fatalf("command palette should be editable")
	}

	jumpCmd, ok := byID[CommandOpenJumpPalette]
	if !ok {
		t.Fatalf("registry missing jump palette command")
	}
	if jumpCmd.Label != "Open Jump Palette" {
		t.Fatalf("jump palette label = %q", jumpCmd.Label)
	}
	if jumpCmd.Category != CategoryApplication {
		t.Fatalf("jump palette category = %q, want %q", jumpCmd.Category, CategoryApplication)
	}
	if jumpCmd.Default != "CmdOrCtrl+J" {
		t.Fatalf("jump palette shortcut = %q, want CmdOrCtrl+J", jumpCmd.Default)
	}
	if !jumpCmd.Editable {
		t.Fatalf("jump palette should be editable")
	}
}

func TestCommandsIncludeToggleSidebarShortcut(t *testing.T) {
	commands := Commands()
	byID := make(map[string]Command, len(commands))
	for _, cmd := range commands {
		byID[cmd.ID] = cmd
	}

	cmd, ok := byID[CommandToggleSidebar]
	if !ok {
		t.Fatalf("registry missing toggle sidebar command")
	}
	if cmd.Label != "Show/Hide Sidebar" {
		t.Fatalf("label = %q", cmd.Label)
	}
	if cmd.Category != CategoryApplication {
		t.Fatalf("category = %q, want %q", cmd.Category, CategoryApplication)
	}
	if cmd.Default != "CmdOrCtrl+\\" {
		t.Fatalf("shortcut = %q, want CmdOrCtrl+\\", cmd.Default)
	}
	if !cmd.Editable {
		t.Fatalf("toggle sidebar should be editable")
	}
}

func TestDefaultSessionAcceleratorWrapsTenthToZero(t *testing.T) {
	cases := map[int]string{
		0: "CmdOrCtrl+1",
		8: "CmdOrCtrl+9",
		9: "CmdOrCtrl+0",
	}
	for slot, want := range cases {
		if got := defaultSessionAccelerator(slot); got != want {
			t.Fatalf("slot %d accelerator = %q, want %q", slot, got, want)
		}
	}
}

func TestResolvePrefersOverrideForEditableCommand(t *testing.T) {
	settings := appsettings.Settings{Keybindings: map[string]string{
		CommandOpenPreferences: "Cmd+Shift+P",
		"std-quit":             "Cmd+Escape", // override on a non-editable command must be ignored
	}}

	effective := Resolve(settings)
	if effective[CommandOpenPreferences] != "Cmd+Shift+P" {
		t.Fatalf("open-preferences = %q, want override", effective[CommandOpenPreferences])
	}
	if effective["std-quit"] != "CmdOrCtrl+Q" {
		t.Fatalf("std-quit = %q, want default (overrides ignored)", effective["std-quit"])
	}
	if effective[SelectSessionID(0)] != "CmdOrCtrl+1" {
		t.Fatalf("session 1 = %q, want default", effective[SelectSessionID(0)])
	}
}

func TestViewReportsEffectiveAccelerators(t *testing.T) {
	settings := appsettings.Settings{Keybindings: map[string]string{CommandOpenPreferences: "Cmd+Shift+P"}}
	view := View(settings)

	var found bool
	for _, cmd := range view.Commands {
		if cmd.ID == CommandOpenPreferences {
			found = true
			if cmd.Accelerator != "Cmd+Shift+P" {
				t.Fatalf("accelerator = %q, want override", cmd.Accelerator)
			}
			if cmd.Default != "CmdOrCtrl+," {
				t.Fatalf("default = %q", cmd.Default)
			}
		}
	}
	if !found {
		t.Fatalf("view missing open-preferences")
	}
}

func TestSanitizeOverrides(t *testing.T) {
	t.Run("keeps valid non-default override", func(t *testing.T) {
		got, err := SanitizeOverrides(map[string]string{CommandOpenPreferences: " Cmd+Shift+P "})
		if err != nil {
			t.Fatalf("sanitize: %v", err)
		}
		if got[CommandOpenPreferences] != "Cmd+Shift+P" {
			t.Fatalf("got = %#v", got)
		}
	})

	t.Run("drops override equal to default", func(t *testing.T) {
		got, err := SanitizeOverrides(map[string]string{CommandOpenPreferences: "CmdOrCtrl+,"})
		if err != nil {
			t.Fatalf("sanitize: %v", err)
		}
		if got != nil {
			t.Fatalf("got = %#v, want nil", got)
		}
	})

	t.Run("rejects unknown command", func(t *testing.T) {
		if _, err := SanitizeOverrides(map[string]string{"does-not-exist": "Cmd+J"}); err == nil {
			t.Fatalf("expected error for unknown command")
		}
	})

	t.Run("rejects non-editable command", func(t *testing.T) {
		if _, err := SanitizeOverrides(map[string]string{"std-quit": "Cmd+Escape"}); err == nil {
			t.Fatalf("expected error for non-editable command")
		}
	})

	t.Run("rejects invalid accelerator", func(t *testing.T) {
		if _, err := SanitizeOverrides(map[string]string{CommandOpenPreferences: "Cmd+"}); err == nil {
			t.Fatalf("expected error for invalid accelerator")
		}
	})
}

func TestValidateAccelerator(t *testing.T) {
	valid := []string{"CmdOrCtrl+,", "Cmd+Shift+P", "Ctrl+1", "Cmd+0", "F11", "Cmd+plus", "Cmd+Up"}
	for _, accel := range valid {
		if err := validateAccelerator(accel); err != nil {
			t.Fatalf("validateAccelerator(%q) = %v, want nil", accel, err)
		}
	}
	invalid := []string{"", "Cmd+", "Bogus+A", "Cmd+ab"}
	for _, accel := range invalid {
		if err := validateAccelerator(accel); err == nil {
			t.Fatalf("validateAccelerator(%q) = nil, want error", accel)
		}
	}
}
