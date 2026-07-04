export type VirtualWindowInput = {
  count: number;
  heights: readonly number[];
  viewportHeight: number;
  scrollOffset: number;
  overscan?: number;
};

export type VirtualIndexWindow = {
  totalHeight: number;
  startIndex: number;
  endIndex: number;
  beforeHeight: number;
  afterHeight: number;
  visibleStartIndex: number;
  visibleEndIndex: number;
};

export function deriveVirtualIndexWindow(input: VirtualWindowInput): VirtualIndexWindow {
  const count = input.count;
  const heights = input.heights.slice(0, count);
  const viewportHeight = nonNegativeFinite(input.viewportHeight);
  const overscan = nonNegativeInteger(input.overscan ?? 2);

  const totalHeight = heights.reduce((sum, h) => sum + h, 0);
  const maxScrollOffset = Math.max(0, totalHeight - viewportHeight);
  const scrollOffset = Math.min(nonNegativeFinite(input.scrollOffset), maxScrollOffset);

  let cumBefore = 0;
  let visibleStartIndex = 0;
  for (let i = 0; i < count; i++) {
    if (cumBefore + heights[i] > scrollOffset) {
      visibleStartIndex = i;
      break;
    }
    cumBefore += heights[i];
  }
  if (count > 0 && visibleStartIndex === 0 && cumBefore + heights[0] <= scrollOffset) {
    visibleStartIndex = count;
  }

  let cum = 0;
  let visibleEndIndex = count;
  for (let i = 0; i < count; i++) {
    if (cum >= scrollOffset + viewportHeight) {
      visibleEndIndex = i;
      break;
    }
    cum += heights[i];
  }
  if (visibleEndIndex <= visibleStartIndex) {
    visibleEndIndex = Math.min(count, visibleStartIndex + 1);
  }

  const startIndex = Math.max(0, visibleStartIndex - overscan);
  const endIndex = Math.min(count, visibleEndIndex + overscan);
  const beforeHeight = computeBeforeHeight(heights, startIndex);
  const afterHeight = computeBeforeHeight(heights, count) - computeBeforeHeight(heights, endIndex);

  return {
    totalHeight,
    startIndex,
    endIndex,
    beforeHeight,
    afterHeight,
    visibleStartIndex,
    visibleEndIndex,
  };
}

export type VirtualRow<T> = {
  key: string;
  row: T;
  index: number;
  offsetTop: number;
  height: number;
};

export function deriveVirtualRows<T extends { key: string }>(
  rows: readonly T[],
  heights: readonly number[],
  window: VirtualIndexWindow,
): VirtualRow<T>[] {
  const result: VirtualRow<T>[] = [];
  let offset = window.beforeHeight;
  for (let i = window.startIndex; i < window.endIndex && i < rows.length; i++) {
    const height = heights[i] ?? 0;
    result.push({
      key: rows[i].key,
      row: rows[i],
      index: i,
      offsetTop: window.totalHeight > 0 ? offset : 0,
      height,
    });
    offset += height;
  }
  return result;
}

function computeBeforeHeight(heights: readonly number[], endIndex: number): number {
  let sum = 0;
  for (let i = 0; i < endIndex && i < heights.length; i++) {
    sum += heights[i];
  }
  return sum;
}

function nonNegativeFinite(value: number): number {
  return Number.isFinite(value) ? Math.max(0, value) : 0;
}

function nonNegativeInteger(value: number): number {
  return Number.isFinite(value) ? Math.max(0, Math.floor(value)) : 2;
}
