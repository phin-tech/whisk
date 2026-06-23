import { describe, expect, it } from "vitest";
import { nextSidebarAfterToggle } from "./sidebarCommands";

describe("sidebar commands", () => {
  it("toggles the current sidebar closed or reopens the sidebar for the active main view", () => {
    expect(nextSidebarAfterToggle("sessions", "session")).toBeNull();
    expect(nextSidebarAfterToggle(null, "session")).toBe("sessions");
    expect(nextSidebarAfterToggle(null, "work")).toBe("work");
    expect(nextSidebarAfterToggle(null, "projects")).toBe("projects");
  });
});
