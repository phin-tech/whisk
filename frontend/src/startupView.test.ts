import { describe, expect, it } from "vitest";
import { normalizeStartupView, startupTarget } from "./startupView";

describe("startupView", () => {
  it("normalizes unknown settings to sessions", () => {
    expect(normalizeStartupView("kanban")).toBe("kanban");
    expect(normalizeStartupView("sessions")).toBe("sessions");
    expect(normalizeStartupView("board")).toBe("sessions");
    expect(normalizeStartupView(undefined)).toBe("sessions");
  });

  it("maps startup settings to the initial main view and sidebar", () => {
    expect(startupTarget("sessions")).toEqual({ main: "session", sidebar: "sessions" });
    expect(startupTarget("kanban")).toEqual({ main: "work", sidebar: "work" });
  });
});
