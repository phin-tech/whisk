// Helpers for the Keyboard Shortcuts panel: turning a recorded KeyboardEvent into an accelerator
// string that matches the Go accelerator format (internal/appmenu), formatting accelerators for
// display, and detecting when two commands share the same accelerator.

// Maps a few KeyboardEvent.key values to the named keys the Wails accelerator parser expects.
const NAMED_KEYS: Record<string, string> = {
  ArrowUp: "Up",
  ArrowDown: "Down",
  ArrowLeft: "Left",
  ArrowRight: "Right",
  " ": "Space",
  Spacebar: "Space",
  Escape: "Escape",
  Enter: "Enter",
  Tab: "Tab",
  Backspace: "Backspace",
  Delete: "Delete",
  Home: "Home",
  End: "End",
  PageUp: "Page Up",
  PageDown: "Page Down",
  "+": "plus",
};

const MODIFIER_KEYS = new Set(["Meta", "Control", "Shift", "Alt", "AltGraph", "CapsLock"]);

// formatAccelerator converts a keydown event into an accelerator string (e.g. "Cmd+Shift+P",
// "Cmd+,"). It returns "" when only modifier keys are held, since that is not yet a complete
// binding. Modifier order is fixed (Cmd, Ctrl, Alt, Shift) so equal bindings compare equal.
export function formatAccelerator(event: KeyboardEvent): string {
  if (MODIFIER_KEYS.has(event.key)) {
    return "";
  }
  const parts: string[] = [];
  if (event.metaKey) parts.push("Cmd");
  if (event.ctrlKey) parts.push("Ctrl");
  if (event.altKey) parts.push("Alt");
  if (event.shiftKey) parts.push("Shift");

  const key = keyToken(event.key);
  if (key === "") {
    return "";
  }
  parts.push(key);
  return parts.join("+");
}

function keyToken(key: string): string {
  if (key in NAMED_KEYS) {
    return NAMED_KEYS[key];
  }
  // Function keys (F1..F35) arrive verbatim.
  if (/^F\d{1,2}$/.test(key)) {
    return key;
  }
  if (key.length === 1) {
    // Letters are upper-cased for display; the Go parser is case-insensitive.
    return key.toUpperCase();
  }
  return "";
}

// Symbols rendered for modifiers when showing an accelerator on macOS.
const DISPLAY_SYMBOLS: Record<string, string> = {
  cmd: "⌘",
  cmdorctrl: "⌘",
  command: "⌘",
  ctrl: "⌃",
  control: "⌃",
  alt: "⌥",
  option: "⌥",
  shift: "⇧",
};

// displayAccelerator renders an accelerator string with macOS modifier symbols, e.g.
// "Cmd+Shift+P" -> "⌘⇧P". Unknown tokens are passed through unchanged.
export function displayAccelerator(accelerator: string): string {
  if (!accelerator) return "";
  const parts = accelerator.split("+");
  return parts
    .map((part, index) => {
      const symbol = DISPLAY_SYMBOLS[part.toLowerCase()];
      if (symbol) return symbol;
      // Last component is the key; show comma/plus literally, upper-case single letters.
      if (index === parts.length - 1 && part.length === 1) return part.toUpperCase();
      return part;
    })
    .join("");
}

// findConflicts returns, for each accelerator used by more than one command, the list of command
// ids that share it. Empty/blank accelerators are ignored.
export function findConflicts(bindings: Record<string, string>): Record<string, string[]> {
  const byAccelerator: Record<string, string[]> = {};
  for (const [id, accelerator] of Object.entries(bindings)) {
    const key = accelerator.trim().toLowerCase();
    if (key === "") continue;
    (byAccelerator[key] ??= []).push(id);
  }
  const conflicts: Record<string, string[]> = {};
  for (const [accelerator, ids] of Object.entries(byAccelerator)) {
    if (ids.length > 1) {
      conflicts[accelerator] = ids;
    }
  }
  return conflicts;
}
