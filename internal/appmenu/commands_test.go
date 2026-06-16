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
