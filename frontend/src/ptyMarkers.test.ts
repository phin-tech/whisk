import { describe, expect, it } from "vitest";
import { bookmarkMarkerPoints } from "./ptyMarkers";

describe("bookmarkMarkerPoints", () => {
  it("returns sorted marker points for unmarked bookmarks inside a chunk byte range", () => {
    const points = bookmarkMarkerPoints(
      [
        { id: "later", offset: 14 },
        { id: "before", offset: 9 },
        { id: "at-start", offset: 10 },
        { id: "already", offset: 11 },
        { id: "same-offset-b", offset: 12 },
        { id: "same-offset-a", offset: 12 },
        { id: "at-end", offset: 15 },
        { id: "after", offset: 16 },
      ],
      new Set(["already"]),
      10,
      5,
    );

    expect(points).toEqual([
      { bookmarkId: "at-start", offset: 10, byteIndex: 0 },
      { bookmarkId: "same-offset-a", offset: 12, byteIndex: 2 },
      { bookmarkId: "same-offset-b", offset: 12, byteIndex: 2 },
      { bookmarkId: "later", offset: 14, byteIndex: 4 },
      { bookmarkId: "at-end", offset: 15, byteIndex: 5 },
    ]);
  });

  it("ignores invalid chunk ranges", () => {
    expect(bookmarkMarkerPoints([{ id: "bm", offset: 0 }], new Set(), 0, 0)).toEqual([]);
    expect(bookmarkMarkerPoints([{ id: "bm", offset: 0 }], new Set(), 0, -1)).toEqual([]);
    expect(bookmarkMarkerPoints([{ id: "bm", offset: 0 }], new Set(), -1, 5)).toEqual([]);
  });
});
