import { describe, expect, it } from "vitest";
import { deriveVirtualIndexWindow, deriveVirtualRows } from "./virtual-list";

describe("deriveVirtualIndexWindow", () => {
  it("returns zero-height window for empty rows", () => {
    const win = deriveVirtualIndexWindow({
      count: 0,
      heights: [],
      viewportHeight: 400,
      scrollOffset: 0,
      overscan: 2,
    });
    expect(win.totalHeight).toBe(0);
    expect(win.startIndex).toBe(0);
    expect(win.endIndex).toBe(0);
    expect(win.beforeHeight).toBe(0);
    expect(win.afterHeight).toBe(0);
  });

  it("keeps small lists fully visible with correct before/after", () => {
    const win = deriveVirtualIndexWindow({
      count: 3,
      heights: [40, 40, 40],
      viewportHeight: 400,
      scrollOffset: 0,
      overscan: 2,
    });
    expect(win.totalHeight).toBe(120);
    expect(win.startIndex).toBe(0);
    expect(win.endIndex).toBe(3);
    expect(win.beforeHeight).toBe(0);
    expect(win.afterHeight).toBe(0);
  });

  it("calculates visible and overscanned ranges for scrolled positions", () => {
    const win = deriveVirtualIndexWindow({
      count: 100,
      heights: Array(100).fill(40),
      viewportHeight: 120,
      scrollOffset: 400,
      overscan: 2,
    });

    expect(win.visibleStartIndex).toBe(10);
    expect(win.visibleEndIndex).toBe(13);
    expect(win.startIndex).toBe(8);
    expect(win.endIndex).toBe(15);
    expect(win.beforeHeight).toBe(320);
    expect(win.afterHeight).toBe(3400);
    expect(win.totalHeight).toBe(4000);
  });

  it("clamps overscroll at list boundaries", () => {
    const win = deriveVirtualIndexWindow({
      count: 10,
      heights: Array(10).fill(20),
      viewportHeight: 60,
      scrollOffset: 9999,
      overscan: 1,
    });

    expect(win.visibleStartIndex).toBe(7);
    expect(win.visibleEndIndex).toBe(10);
    expect(win.startIndex).toBe(6);
    expect(win.endIndex).toBe(10);
    expect(win.afterHeight).toBe(0);
  });

  it("handles variable row heights for scrolled views", () => {
    const heights = [28, 52, 28, 52, 52, 28, 52, 52, 52, 28];
    const win = deriveVirtualIndexWindow({
      count: heights.length,
      heights,
      viewportHeight: 100,
      scrollOffset: 80,
      overscan: 1,
    });

    expect(win.totalHeight).toBe(heights.reduce((a, b) => a + b, 0));
    expect(win.visibleStartIndex).toBe(2);
    expect(win.visibleEndIndex).toBe(5);
    expect(win.startIndex).toBe(1);
    expect(win.endIndex).toBe(6);
  });

  it("rejects invalid viewport height gracefully", () => {
    const win = deriveVirtualIndexWindow({
      count: 5,
      heights: [40, 40, 40, 40, 40],
      viewportHeight: -1,
      scrollOffset: 0,
    });
    expect(win.totalHeight).toBe(200);
    expect(win.startIndex).toBe(0);
  });
});

describe("deriveVirtualRows", () => {
  it("returns mapped rows with offset positions", () => {
    const rows = [
      { key: "a", value: 1 },
      { key: "b", value: 2 },
      { key: "c", value: 3 },
    ];
    const heights = [28, 52, 28];
    const win = deriveVirtualIndexWindow({
      count: 3,
      heights,
      viewportHeight: 200,
      scrollOffset: 0,
    });
    const virtualRows = deriveVirtualRows(rows, heights, win);

    expect(virtualRows).toHaveLength(3);
    expect(virtualRows[0].key).toBe("a");
    expect(virtualRows[0].index).toBe(0);
    expect(virtualRows[0].offsetTop).toBe(0);
    expect(virtualRows[0].height).toBe(28);
    expect(virtualRows[1].key).toBe("b");
    expect(virtualRows[1].offsetTop).toBe(28);
    expect(virtualRows[1].height).toBe(52);
    expect(virtualRows[2].offsetTop).toBe(80);
    expect(virtualRows[2].height).toBe(28);
  });

  it("preserves determininstic input ordering", () => {
    const rows = [
      { key: "z", value: 1 },
      { key: "a", value: 2 },
      { key: "m", value: 3 },
    ];
    const heights = [40, 40, 40];
    const win = deriveVirtualIndexWindow({
      count: 3,
      heights,
      viewportHeight: 200,
      scrollOffset: 0,
    });
    const virtualRows = deriveVirtualRows(rows, heights, win);

    expect(virtualRows.map((r) => r.key)).toEqual(["z", "a", "m"]);
  });
});
