export function updateJumpRecents(
  currentIds: readonly string[],
  activatedId: string,
  maxSize: number,
): string[] {
  if (maxSize <= 0) return [];
  const targetId = activatedId.trim();
  const current = sanitizeRecentIds(currentIds);
  if (!targetId) return current.slice(0, maxSize);

  return [targetId, ...current.filter((id) => id !== targetId)].slice(0, maxSize);
}

export function reconcileJumpRecents(
  recentIds: readonly string[],
  availableIds: readonly string[],
): string[] {
  const available = new Set(sanitizeRecentIds(availableIds));
  return sanitizeRecentIds(recentIds).filter((id) => available.has(id));
}

export function applyRecentJumpTargets<T extends { id: string; current?: boolean }>(
  targets: readonly T[],
  recentIds: readonly string[],
): T[] {
  const current: T[] = [];
  const recent: T[] = [];
  const usedIndexes = new Set<number>();

  for (const [index, target] of targets.entries()) {
    if (target.current) {
      current.push(target);
      usedIndexes.add(index);
    }
  }

  for (const recentId of sanitizeRecentIds(recentIds)) {
    const index = targets.findIndex(
      (target, candidateIndex) =>
        !usedIndexes.has(candidateIndex) && target.id === recentId && !target.current,
    );
    if (index < 0) continue;
    recent.push(targets[index]);
    usedIndexes.add(index);
  }

  const other = targets.filter((_, index) => !usedIndexes.has(index));

  return [...current, ...recent, ...other];
}

function sanitizeRecentIds(ids: readonly string[]): string[] {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const rawId of ids) {
    const id = rawId.trim();
    if (!id || seen.has(id)) continue;
    seen.add(id);
    result.push(id);
  }
  return result;
}
