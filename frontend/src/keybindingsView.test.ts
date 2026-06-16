import { describe, expect, it } from "vitest";
import { displayAccelerator, findConflicts, formatAccelerator } from "./keybindingsView";

function keyEvent(init: Partial<KeyboardEvent> & { key: string }): KeyboardEvent {
  return {
    key: init.key,
    metaKey: init.metaKey ?? false,
    ctrlKey: init.ctrlKey ?? false,
    altKey: init.altKey ?? false,
    shiftKey: init.shiftKey ?? false,
  } as KeyboardEvent;
}

describe("formatAccelerator", () => {
  it("formats Cmd+, for preferences", () => {
    expect(formatAccelerator(keyEvent({ key: ",", metaKey: true }))).toBe("Cmd+,");
  });

  it("orders modifiers Cmd, Ctrl, Alt, Shift and upper-cases letters", () => {
    expect(formatAccelerator(keyEvent({ key: "p", metaKey: true, shiftKey: true }))).toBe(
      "Cmd+Shift+P",
    );
  });

  it("maps arrow keys to named keys", () => {
    expect(formatAccelerator(keyEvent({ key: "ArrowUp", metaKey: true }))).toBe("Cmd+Up");
  });

  it("keeps function keys verbatim", () => {
    expect(formatAccelerator(keyEvent({ key: "F11" }))).toBe("F11");
  });

  it("returns empty while only modifiers are held", () => {
    expect(formatAccelerator(keyEvent({ key: "Meta", metaKey: true }))).toBe("");
  });

  it("maps digits for session shortcuts", () => {
    expect(formatAccelerator(keyEvent({ key: "1", metaKey: true }))).toBe("Cmd+1");
  });
});

describe("displayAccelerator", () => {
  it("renders macOS symbols", () => {
    expect(displayAccelerator("Cmd+Shift+P")).toBe("⌘⇧P");
    expect(displayAccelerator("CmdOrCtrl+,")).toBe("⌘,");
    expect(displayAccelerator("")).toBe("");
  });
});

describe("findConflicts", () => {
  it("reports accelerators shared by more than one command", () => {
    const conflicts = findConflicts({
      "open-preferences": "Cmd+1",
      "select-session-1": "Cmd+1",
      "select-session-2": "Cmd+2",
    });
    expect(conflicts["cmd+1"]).toEqual(["open-preferences", "select-session-1"]);
    expect(conflicts["cmd+2"]).toBeUndefined();
  });

  it("ignores blank accelerators", () => {
    expect(findConflicts({ a: "", b: "" })).toEqual({});
  });
});
