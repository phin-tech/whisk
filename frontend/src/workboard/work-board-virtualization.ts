import type { WorkBoardCardView } from "./work-board-state";

export type WorkBoardCardWindowInput = {
  cards: readonly WorkBoardCardView[];
  rowHeight: number;
  viewportHeight: number;
  scrollOffset: number;
  overscan?: number;
};

export type WorkBoardVirtualCard = {
  key: string;
  card: WorkBoardCardView;
  index: number;
  offsetTop: number;
  height: number;
};

export type WorkBoardCardWindow = {
  totalHeight: number;
  beforeHeight: number;
  afterHeight: number;
  scrollOffset: number;
  visibleStartIndex: number;
  visibleEndIndex: number;
  startIndex: number;
  endIndex: number;
  cards: WorkBoardVirtualCard[];
};

const DEFAULT_OVERSCAN = 2;

export function deriveWorkBoardCardWindow(input: WorkBoardCardWindowInput): WorkBoardCardWindow {
  const rowHeight = positiveNumber(input.rowHeight, "rowHeight");
  const count = input.cards.length;
  const totalHeight = count * rowHeight;
  const viewportHeight = nonNegativeFinite(input.viewportHeight);
  const maxScrollOffset = Math.max(0, totalHeight - viewportHeight);
  const scrollOffset = Math.min(nonNegativeFinite(input.scrollOffset), maxScrollOffset);
  const overscan = nonNegativeInteger(input.overscan ?? DEFAULT_OVERSCAN);
  const visibleCount = viewportHeight > 0 ? Math.ceil(viewportHeight / rowHeight) : 0;
  const visibleStartIndex = count === 0 ? 0 : Math.min(count, Math.floor(scrollOffset / rowHeight));
  const visibleEndIndex = Math.min(count, visibleStartIndex + visibleCount);
  const startIndex = Math.max(0, visibleStartIndex - overscan);
  const endIndex = Math.min(count, visibleEndIndex + overscan);
  const cards = input.cards.slice(startIndex, endIndex).map((card, sliceIndex) => {
    const index = startIndex + sliceIndex;
    return {
      key: card.key,
      card,
      index,
      offsetTop: index * rowHeight,
      height: rowHeight,
    };
  });

  return {
    totalHeight,
    beforeHeight: startIndex * rowHeight,
    afterHeight: Math.max(0, (count - endIndex) * rowHeight),
    scrollOffset,
    visibleStartIndex,
    visibleEndIndex,
    startIndex,
    endIndex,
    cards,
  };
}

function positiveNumber(value: number, name: string) {
  if (!Number.isFinite(value) || value <= 0) {
    throw new Error(`${name} must be a positive finite number`);
  }
  return value;
}

function nonNegativeFinite(value: number) {
  return Number.isFinite(value) ? Math.max(0, value) : 0;
}

function nonNegativeInteger(value: number) {
  return Number.isFinite(value) ? Math.max(0, Math.floor(value)) : DEFAULT_OVERSCAN;
}
