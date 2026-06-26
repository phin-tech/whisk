package appmenu

import (
	"testing"

	"github.com/phin-tech/whisk/internal/appsettings"
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
	settings := appsettings.Settings{Keybindings: map[string]string{CommandOpenPalette: "Cmd+Shift+K"}}
	menu := NewController(nil, settings).build(settings, nil)

	cases := map[string]string{
		"Open Command Palette":    "Cmd+Shift+K",
		"Show/Hide Sidebar":       "Cmd+\\",
		"Split Pane Vertically":   "Cmd+D",
		"Split Pane Horizontally": "Cmd+Shift+D",
		"Close Pane":              "Cmd+W",
		"Close Session":           "Cmd+Shift+W",
		"Add Bookmark":            "Cmd+B",
		"Previous Bookmark":       "Cmd+Option+LEFT",
		"Next Bookmark":           "Cmd+Option+RIGHT",
		"Jump to Last Prompt":     "Cmd+Option+P",
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
