import { describe, expect, it } from "vitest";
import type { WorkBoardCardView } from "./work-board-state";
import { deriveWorkBoardCardWindow } from "./work-board-virtualization";

function card(id: string): WorkBoardCardView {
  return {
    key: `work-item:${id}`,
    item: { id, title: id },
    targets: { previous: null, next: null, blockedNext: null },
    latestRun: null,
    terminalRun: null,
    attention: { severity: "none", signals: [] },
    indicators: [],
    canExecute: false,
  } as unknown as WorkBoardCardView;
}

function cards(count: number) {
  return Array.from({ length: count }, (_, index) => card(`item-${index}`));
}

describe("work-board-virtualization", () => {
  it("returns an empty window for empty columns", () => {
    expect(
      deriveWorkBoardCardWindow({
        cards: [],
        rowHeight: 80,
        viewportHeight: 320,
        scrollOffset: 0,
        overscan: 4,
      }),
    ).toEqual({
      totalHeight: 0,
      beforeHeight: 0,
      afterHeight: 0,
      scrollOffset: 0,
      visibleStartIndex: 0,
      visibleEndIndex: 0,
      startIndex: 0,
      endIndex: 0,
      cards: [],
    });
  });

  it("keeps small columns fully mounted with stable card keys", () => {
    const inputCards = [card("a"), card("b"), card("c")];
    const window = deriveWorkBoardCardWindow({
      cards: inputCards,
      rowHeight: 76,
      viewportHeight: 400,
      scrollOffset: 0,
      overscan: 4,
    });

    expect(window.startIndex).toBe(0);
    expect(window.endIndex).toBe(3);
    expect(window.beforeHeight).toBe(0);
    expect(window.afterHeight).toBe(0);
    expect(window.totalHeight).toBe(228);
    expect(window.cards.map((entry) => entry.key)).toEqual([
      "work-item:a",
      "work-item:b",
      "work-item:c",
    ]);
    expect(window.cards.map((entry) => entry.card)).toEqual(inputCards);
  });

  it("calculates visible and overscanned ranges for large columns", () => {
    const window = deriveWorkBoardCardWindow({
      cards: cards(100),
      rowHeight: 32,
      viewportHeight: 96,
      scrollOffset: 320,
      overscan: 2,
    });

    expect(window.visibleStartIndex).toBe(10);
    expect(window.visibleEndIndex).toBe(13);
    expect(window.startIndex).toBe(8);
    expect(window.endIndex).toBe(15);
    expect(window.beforeHeight).toBe(256);
    expect(window.afterHeight).toBe(2720);
    expect(window.totalHeight).toBe(3200);
    expect(window.cards.map((entry) => entry.index)).toEqual([8, 9, 10, 11, 12, 13, 14]);
    expect(window.cards.map((entry) => entry.offsetTop)).toEqual([256, 288, 320, 352, 384, 416, 448]);
  });

  it("clamps overscan and overscrolled offsets at list boundaries", () => {
    const window = deriveWorkBoardCardWindow({
      cards: cards(10),
      rowHeight: 20,
      viewportHeight: 60,
      scrollOffset: 9999,
      overscan: 1,
    });

    expect(window.scrollOffset).toBe(140);
    expect(window.visibleStartIndex).toBe(7);
    expect(window.visibleEndIndex).toBe(10);
    expect(window.startIndex).toBe(6);
    expect(window.endIndex).toBe(10);
    expect(window.afterHeight).toBe(0);
    expect(window.cards.map((entry) => entry.key)).toEqual([
      "work-item:item-6",
      "work-item:item-7",
      "work-item:item-8",
      "work-item:item-9",
    ]);
  });

  it("preserves deterministic input ordering instead of sorting card keys", () => {
    const orderedCards = [card("z"), card("a"), card("m")];
    const window = deriveWorkBoardCardWindow({
      cards: orderedCards,
      rowHeight: 50,
      viewportHeight: 150,
      scrollOffset: 0,
      overscan: 0,
    });

    expect(window.cards.map((entry) => entry.key)).toEqual([
      "work-item:z",
      "work-item:a",
      "work-item:m",
    ]);
    expect(window.cards.map((entry) => entry.card.item.id)).toEqual(["z", "a", "m"]);
  });

  it("rejects invalid fixed row heights before deriving a window", () => {
    expect(() =>
      deriveWorkBoardCardWindow({
        cards: cards(1),
        rowHeight: 0,
        viewportHeight: 100,
        scrollOffset: 0,
      }),
    ).toThrow("rowHeight must be a positive finite number");
  });
});
