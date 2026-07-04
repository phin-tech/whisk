export const SIDEBAR_MIN_WIDTH_PX = 240;
export const SIDEBAR_MAX_WIDTH_PX = 800;
export const SIDEBAR_DEFAULT_WIDTH_PX = 320;

export type SidebarRailSide = "left" | "right";

export type SidebarWidthBounds = {
  minWidthPx?: number;
  maxWidthPx?: number;
};

export type SidebarDragInput = SidebarWidthBounds & {
  startWidthPx: number;
  startClientX: number;
  currentClientX: number;
  railSide: SidebarRailSide;
};

export function clampSidebarWidthPx(
  widthPx: number,
  { minWidthPx = SIDEBAR_MIN_WIDTH_PX, maxWidthPx = SIDEBAR_MAX_WIDTH_PX }: SidebarWidthBounds = {},
) {
  return Math.max(minWidthPx, Math.min(maxWidthPx, widthPx));
}

export function sidebarWidthFromDrag({
  startWidthPx,
  startClientX,
  currentClientX,
  railSide,
  minWidthPx,
  maxWidthPx,
}: SidebarDragInput) {
  const direction = railSide === "left" ? 1 : -1;
  return clampSidebarWidthPx(startWidthPx + direction * (currentClientX - startClientX), {
    minWidthPx,
    maxWidthPx,
  });
}

export function toggleCollapsedId(collapsedIds: ReadonlySet<string>, id: string) {
  const next = new Set(collapsedIds);
  if (!id) return next;
  if (next.has(id)) {
    next.delete(id);
  } else {
    next.add(id);
  }
  return next;
}
