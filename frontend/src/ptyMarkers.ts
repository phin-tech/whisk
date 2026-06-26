export type BookmarkMarkerInput = {
  id: string;
  offset: number;
};

export type BookmarkMarkerPoint = {
  bookmarkId: string;
  offset: number;
  byteIndex: number;
};

export type BookmarkJumpRequest = {
  bookmarkId: string;
  offset: number;
  revision: number;
};

export function bookmarkMarkerPoints(
  bookmarks: BookmarkMarkerInput[],
  markedBookmarkIds: ReadonlySet<string>,
  chunkStartOffset: number,
  byteLength: number,
): BookmarkMarkerPoint[] {
  if (!Number.isFinite(chunkStartOffset) || chunkStartOffset < 0) return [];
  if (!Number.isFinite(byteLength) || byteLength <= 0) return [];

  const start = Math.floor(chunkStartOffset);
  const end = start + Math.floor(byteLength);
  return bookmarks
    .filter((bookmark) => {
      if (!bookmark.id || markedBookmarkIds.has(bookmark.id)) return false;
      if (!Number.isFinite(bookmark.offset)) return false;
      return bookmark.offset >= start && bookmark.offset <= end;
    })
    .map((bookmark) => ({
      bookmarkId: bookmark.id,
      offset: Math.floor(bookmark.offset),
      byteIndex: Math.floor(bookmark.offset) - start,
    }))
    .sort((a, b) => a.byteIndex - b.byteIndex || a.bookmarkId.localeCompare(b.bookmarkId));
}
