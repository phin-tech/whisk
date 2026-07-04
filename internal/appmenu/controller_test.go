package appmenu

import (
	"testing"

	"github.com/phin-tech/whisk/internal/appsettings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestSessionMenuEntriesAccelerateFirstTenOnly(t *testing.T) {
	sessions := make([]SessionRef, SessionSlots+2)
	for i := range sessions {
		sessions[i] = SessionRef{ID: string(rune('a' + i))}
	}
	effective := Resolve(appsettings.Settings{})

	entries := sessionMenuEntries(sessions, effective)
	if len(entries) != len(sessions) {
		t.Fatalf("entries = %d, want %d", len(entries), len(sessions))
	}
	if entries[0].Accelerator != "CmdOrCtrl+1" {
		t.Fatalf("first accelerator = %q", entries[0].Accelerator)
	}
	if entries[SessionSlots-1].Accelerator != "CmdOrCtrl+0" {
		t.Fatalf("tenth accelerator = %q", entries[SessionSlots-1].Accelerator)
	}
	if entries[SessionSlots].Accelerator != "" {
		t.Fatalf("eleventh accelerator = %q, want empty", entries[SessionSlots].Accelerator)
	}
	if entries[3].Index != 3 {
		t.Fatalf("entry index = %d, want 3", entries[3].Index)
	}
}

func TestSessionMenuEntriesUsesOverrideAndFallbackLabel(t *testing.T) {
	effective := Resolve(appsettings.Settings{Keybindings: map[string]string{SelectSessionID(0): "Cmd+Shift+1"}})
	entries := sessionMenuEntries([]SessionRef{{ID: "x", Name: ""}}, effective)
	if entries[0].Label != "Untitled session" {
		t.Fatalf("label = %q, want fallback", entries[0].Label)
	}
	if entries[0].Accelerator != "Cmd+Shift+1" {
		t.Fatalf("accelerator = %q, want override", entries[0].Accelerator)
	}
}

func TestControllerBuildIncludesNativeCommandAccelerators(t *testing.T) {
	settings := appsettings.Settings{Keybindings: map[string]string{
		CommandOpenPalette:     "Cmd+Shift+K",
		CommandOpenJumpPalette: "Cmd+Shift+J",
	}}
	menu := NewController(nil, settings).build(settings, nil)

	effective := Resolve(settings)
	cases := map[string]string{
		"Open Command Palette":    nativeAccelerator(effective[CommandOpenPalette]),
		"Open Jump Palette":       nativeAccelerator(effective[CommandOpenJumpPalette]),
		"Show/Hide Sidebar":       nativeAccelerator(effective[CommandToggleSidebar]),
		"Split Pane Vertically":   nativeAccelerator(effective[CommandSplitPaneVertical]),
		"Split Pane Horizontally": nativeAccelerator(effective[CommandSplitPaneHorizontal]),
		"Close Pane":              nativeAccelerator(effective[CommandClosePane]),
		"Close Session":           nativeAccelerator(effective[CommandCloseSession]),
		"Jump to Bottom":          nativeAccelerator(effective[CommandJumpToBottom]),
	}
	for label, want := range cases {
		item := menu.FindByLabel(label)
		if item == nil {
			t.Fatalf("menu missing %q", label)
		}
		if got := item.GetAccelerator(); got != want {
			t.Fatalf("%s accelerator = %q, want %q", label, got, want)
		}
	}

	removed := []string{"Add Bookmark", "Previous Bookmark", "Next Bookmark", "Jump to Last Prompt"}
	for _, label := range removed {
		if item := menu.FindByLabel(label); item != nil {
			t.Fatalf("menu still includes %q", label)
		}
	}
}

func nativeAccelerator(shortcut string) string {
	return application.NewMenu().Add("accelerator probe").SetAccelerator(shortcut).GetAccelerator()
}

func TestControllerStateMutatorsAreNoOpWithoutApp(t *testing.T) {
	// With no app attached, Rebuild must not panic and state setters should still record state.
	c := NewController(nil, appsettings.Settings{})
	c.SetSessions([]SessionRef{{ID: "a", Name: "alpha"}})
	c.SetKeybindings(appsettings.Settings{Keybindings: map[string]string{CommandOpenPreferences: "Cmd+J"}})

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.sessions) != 1 || c.sessions[0].Name != "alpha" {
		t.Fatalf("sessions = %#v", c.sessions)
	}
	if c.settings.Keybindings[CommandOpenPreferences] != "Cmd+J" {
		t.Fatalf("settings = %#v", c.settings)
	}
}
