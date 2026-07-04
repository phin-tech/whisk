import { describe, expect, it } from "vitest";
import source from "./sidebarState.ts?raw";
import {
  SIDEBAR_MAX_WIDTH_PX,
  SIDEBAR_MIN_WIDTH_PX,
  clampSidebarWidthPx,
  sidebarWidthFromDrag,
  toggleCollapsedId,
} from "./sidebarState";

describe("sidebar width state", () => {
  it("clamps dock width to the supported sidebar bounds", () => {
    expect(clampSidebarWidthPx(SIDEBAR_MIN_WIDTH_PX - 1)).toBe(SIDEBAR_MIN_WIDTH_PX);
    expect(clampSidebarWidthPx(320)).toBe(320);
    expect(clampSidebarWidthPx(SIDEBAR_MAX_WIDTH_PX + 1)).toBe(SIDEBAR_MAX_WIDTH_PX);
  });

  it("derives drag width from the rail side without mutating browser state", () => {
    expect(
      sidebarWidthFromDrag({
        startWidthPx: 320,
        startClientX: 100,
        currentClientX: 160,
        railSide: "left",
      }),
    ).toBe(380);

    expect(
      sidebarWidthFromDrag({
        startWidthPx: 320,
        startClientX: 100,
        currentClientX: 160,
        railSide: "right",
      }),
    ).toBe(260);
  });
});

describe("sidebar collapse state", () => {
  it("toggles collapsed IDs with immutable set updates", () => {
    const collapsed = new Set(["project"]);

    const expanded = toggleCollapsedId(collapsed, "project");
    expect([...expanded]).toEqual([]);
    expect([...collapsed]).toEqual(["project"]);

    const nextCollapsed = toggleCollapsedId(collapsed, "folder");
    expect([...nextCollapsed]).toEqual(["project", "folder"]);
    expect(nextCollapsed).not.toBe(collapsed);
  });
});

describe("sidebar state ownership", () => {
  it("stays frontend-local and independent from daemon/runtime ownership", () => {
    expect(source).not.toContain("@wailsio/runtime");
    expect(source).not.toMatch(/bindings\/github\.com\/phin-tech\/whisk\/internal/);
    expect(source).not.toMatch(/\b(localStorage|sessionStorage)\b/);
  });
});
