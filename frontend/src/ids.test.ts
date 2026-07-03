import { describe, expect, it } from "vitest";
import {
  currentPtyIdOf,
  idString,
  layoutPaneIdOf,
  optionalIdString,
  paneIdOf,
  paneKey,
  parsePaneId,
  parsePaneKey,
  parseProjectId,
  parsePtyId,
  parseRunId,
  parseSessionId,
  parseWindowId,
  parseWorkItemId,
  projectIdOf,
  ptyIdOf,
  requirePaneId,
  requirePaneKey,
  runIdOf,
  sessionIdOf,
  splitPaneKey,
  unsafePaneId,
  unsafeSessionId,
  workItemIdOf,
} from "./ids";

describe("ID parsing", () => {
  it("brands non-empty daemon ID strings without enforcing backend prefixes", () => {
    expect(parseSessionId("sess_01")).toBe("sess_01");
    expect(parseSessionId("provider-session-uuid")).toBe("provider-session-uuid");
    expect(parseWindowId("win_01")).toBe("win_01");
    expect(parsePaneId("pane_01")).toBe("pane_01");
    expect(parsePtyId("pty/a")).toBe("pty/a");
    expect(parseProjectId("proj_01")).toBe("proj_01");
    expect(parseWorkItemId("work-item-1")).toBe("work-item-1");
    expect(parseRunId("run_01")).toBe("run_01");
  });

  it("trims valid persisted strings and rejects missing or blank IDs", () => {
    expect(parseSessionId("  sess_01  ")).toBe("sess_01");
    expect(parseSessionId("")).toBeNull();
    expect(parseSessionId("   ")).toBeNull();
    expect(parseSessionId(null)).toBeNull();
    expect(parseSessionId(undefined)).toBeNull();
    expect(parseSessionId(42)).toBeNull();
  });

  it("throws with field context for programmer-error request paths", () => {
    expect(requirePaneId("pane_01")).toBe("pane_01");
    expect(() => requirePaneId("", "targetPaneId")).toThrow("targetPaneId must be a non-empty string");
  });
});

describe("read-model ID helpers", () => {
  it("wraps generated binding-shaped fields at frontend boundaries", () => {
    expect(sessionIdOf({ id: "sess_01" })).toBe("sess_01");
    expect(paneIdOf({ id: "pane_01" })).toBe("pane_01");
    expect(currentPtyIdOf({ currentPtyId: "pty_01" })).toBe("pty_01");
    expect(layoutPaneIdOf({ paneId: "pane_02" })).toBe("pane_02");
    expect(ptyIdOf({ id: "pty_03" })).toBe("pty_03");
    expect(ptyIdOf({ ptyId: "pty_04" })).toBe("pty_04");
    expect(projectIdOf({ id: "proj_01" })).toBe("proj_01");
    expect(workItemIdOf({ workItemId: "wi_01" })).toBe("wi_01");
    expect(runIdOf({ runId: "run_01" })).toBe("run_01");
  });

  it("returns null for absent generated binding fields", () => {
    expect(sessionIdOf(null)).toBeNull();
    expect(paneIdOf({})).toBeNull();
    expect(currentPtyIdOf({ currentPtyId: "" })).toBeNull();
    expect(layoutPaneIdOf({})).toBeNull();
    expect(ptyIdOf({ id: "", ptyId: "" })).toBeNull();
    expect(projectIdOf({ id: "", projectId: "" })).toBeNull();
    expect(workItemIdOf({ id: "", workItemId: "" })).toBeNull();
    expect(runIdOf({ id: "", runId: "" })).toBeNull();
  });

  it("unwraps branded IDs explicitly for string-only storage and DTO boundaries", () => {
    const sessionId = unsafeSessionId("sess_01");
    expect(idString(sessionId)).toBe("sess_01");
    expect(optionalIdString(sessionId)).toBe("sess_01");
    expect(optionalIdString(null)).toBe("");
    expect(optionalIdString(undefined)).toBe("");
  });
});

describe("PaneKey", () => {
  it("makes and splits stable pane keys", () => {
    const key = paneKey(unsafeSessionId("sess_01"), unsafePaneId("pane_01"));
    expect(key).toBe("sess_01:pane_01");
    expect(splitPaneKey(key)).toEqual({ sessionId: "sess_01", paneId: "pane_01" });
    expect(parsePaneKey(key)).toEqual({ sessionId: "sess_01", paneId: "pane_01" });
    expect(requirePaneKey(key)).toBe(key);
  });

  it("rejects malformed delimiter forms instead of normalizing them", () => {
    for (const value of [
      "",
      ":",
      "sess_01:",
      ":pane_01",
      "sess_01:pane_01:extra",
      "sess_01::pane_01",
      "sess_01:pane_01:",
      " sess_01:pane_01",
      "sess_01:pane_01 ",
      "sess_01 :pane_01",
      "sess_01: pane_01",
    ]) {
      expect(parsePaneKey(value), value).toBeNull();
      expect(() => requirePaneKey(value), value).toThrow("paneKey must be a sessionId:paneId string");
    }
  });

  it("rejects unsafe ID components that cannot round-trip through the delimiter", () => {
    expect(() => paneKey(unsafeSessionId("sess:01"), unsafePaneId("pane_01"))).toThrow(
      "sessionId must not contain ':'",
    );
    expect(() => paneKey(unsafeSessionId("sess_01"), unsafePaneId("pane:01"))).toThrow(
      "paneId must not contain ':'",
    );
    expect(() => paneKey(unsafeSessionId(""), unsafePaneId("pane_01"))).toThrow(
      "sessionId must be a non-empty string",
    );
    expect(() => paneKey(unsafeSessionId(" sess_01"), unsafePaneId("pane_01"))).toThrow(
      "sessionId must not contain surrounding whitespace",
    );
  });
});
