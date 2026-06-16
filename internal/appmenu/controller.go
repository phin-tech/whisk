package appmenu

import (
	"fmt"
	"sync"

	"github.com/phin-tech/whisk/internal/appsettings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// EventCommandRun is the frontend command-dispatch event. A native menu item / accelerator emits it
// with a command id, and App.svelte's Events.On("command:run") handler runs the matching entry from
// its command registry — the same path the command palette uses.
const EventCommandRun = "command:run"

// Frontend command ids, kept in sync with the entries registered in App.svelte's `commands` array.
const FrontendCommandOpenPreferences = "preferences.open"

// FrontendSelectSessionCommand returns the frontend command id for the session in slot i (0-based),
// e.g. "session.select.1" for the first session.
func FrontendSelectSessionCommand(i int) string {
	return fmt.Sprintf("session.select.%d", i+1)
}

// SessionRef is the minimal session description the frontend pushes to keep the Sessions menu in
// sync. Order matches the session bar; Index 0 is the first session (Cmd+1).
type SessionRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SessionMenuEntry is the resolved row for one session in the Sessions menu: its label, the
// accelerator for its slot (empty past slot 10), and its 0-based index used as the click payload.
type SessionMenuEntry struct {
	Label       string
	Accelerator string
	Index       int
}

// sessionMenuEntries computes the Sessions submenu rows from the current session list and the
// effective accelerators. Only the first SessionSlots sessions get an accelerator; the rest are
// still listed (clickable) but without a shortcut.
func sessionMenuEntries(sessions []SessionRef, effective map[string]string) []SessionMenuEntry {
	entries := make([]SessionMenuEntry, 0, len(sessions))
	for i, ref := range sessions {
		label := ref.Name
		if label == "" {
			label = "Untitled session"
		}
		accelerator := ""
		if i < SessionSlots {
			accelerator = effective[SelectSessionID(i)]
		}
		entries = append(entries, SessionMenuEntry{Label: label, Accelerator: accelerator, Index: i})
	}
	return entries
}

// Controller owns the live application menu and rebuilds it whenever the settings (accelerators) or
// the session list change. All state mutation goes through the mutex because Set/SetSessions are
// invoked both at startup and from frontend RPC goroutines.
type Controller struct {
	app *application.App

	mu       sync.Mutex
	settings appsettings.Settings
	sessions []SessionRef
}

// NewController creates a controller bound to the given app and initial settings. Call Rebuild once
// after construction to install the menu.
func NewController(app *application.App, settings appsettings.Settings) *Controller {
	return &Controller{app: app, settings: settings}
}

// SetKeybindings replaces the accelerator settings and rebuilds the menu.
func (c *Controller) SetKeybindings(settings appsettings.Settings) {
	c.mu.Lock()
	c.settings = settings
	c.mu.Unlock()
	c.rebuildOnMainThread()
}

// SetSessions replaces the session list and rebuilds the Sessions menu.
func (c *Controller) SetSessions(sessions []SessionRef) {
	c.mu.Lock()
	c.sessions = append(c.sessions[:0:0], sessions...)
	c.mu.Unlock()
	c.rebuildOnMainThread()
}

// rebuildOnMainThread marshals the rebuild onto the UI thread. SetSessions/SetKeybindings are
// called from Wails binding goroutines, but macOS requires all menu (AppKit) mutation to happen on
// the main thread — doing it off-thread crashes the process. InvokeAsync queues Rebuild on the main
// thread without blocking the RPC response.
func (c *Controller) rebuildOnMainThread() {
	if c.app == nil {
		return
	}
	application.InvokeAsync(c.Rebuild)
}

// Rebuild constructs a fresh native menu from current state and installs it. It must run on the
// main thread: call it directly only during startup (before Run()); at runtime go through
// rebuildOnMainThread. It is a no-op when no app is attached (e.g. in unit tests).
func (c *Controller) Rebuild() {
	if c.app == nil {
		return
	}
	c.mu.Lock()
	settings := c.settings
	sessions := append([]SessionRef(nil), c.sessions...)
	c.mu.Unlock()

	menu := c.build(settings, sessions)
	c.app.Menu.Set(menu)
	menu.Update()
}

// build assembles the menu bar: the standard macOS roles plus our custom Preferences item and the
// dynamic Sessions menu.
func (c *Controller) build(settings appsettings.Settings, sessions []SessionRef) *application.Menu {
	effective := Resolve(settings)
	menu := c.app.NewMenu()

	// Application menu (About / Services / Hide / Quit). Insert our Preferences item, which the
	// AppMenu role does not provide in this Wails version.
	menu.AddRole(application.AppMenu)
	if appSubmenu := submenuForRole(menu, application.AppMenu); appSubmenu != nil {
		pref := appSubmenu.Add("Preferences…").SetAccelerator(effective[CommandOpenPreferences])
		pref.OnClick(func(*application.Context) {
			c.app.Event.Emit(EventCommandRun, FrontendCommandOpenPreferences)
		})
	}

	// Edit menu — gives Undo/Cut/Copy/Paste/Select All to the app's text fields for free.
	menu.AddRole(application.EditMenu)

	// Sessions menu — rebuilt from the current session list with Cmd+1..Cmd+0 accelerators.
	sessionsMenu := menu.AddSubmenu("Sessions")
	for _, entry := range sessionMenuEntries(sessions, effective) {
		index := entry.Index
		item := sessionsMenu.Add(entry.Label)
		if entry.Accelerator != "" {
			item.SetAccelerator(entry.Accelerator)
		}
		item.OnClick(func(*application.Context) {
			c.app.Event.Emit(EventCommandRun, FrontendSelectSessionCommand(index))
		})
	}

	// Window menu + an explicit Close Window (the darwin WindowMenu role omits it).
	menu.AddRole(application.WindowMenu)
	if windowSubmenu := submenuForRole(menu, application.WindowMenu); windowSubmenu != nil {
		windowSubmenu.AddRole(application.CloseWindow)
	}

	menu.AddRole(application.HelpMenu)
	return menu
}

// submenuForRole returns the submenu attached to a role's top-level item, or nil if the role is not
// present (e.g. AppMenu off macOS).
func submenuForRole(menu *application.Menu, role application.Role) *application.Menu {
	item := menu.FindByRole(role)
	if item == nil {
		return nil
	}
	return item.GetSubmenu()
}
