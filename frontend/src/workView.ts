type StageLike = {
  id: string;
  provisionWorktree?: boolean;
};

type WorkItemLike = {
  id: string;
  stageId: string;
  worktree?: unknown;
};

export function groupWorkItemsByStage<T extends WorkItemLike, S extends StageLike>(
  items: T[],
  stages: S[],
) {
  const result: Record<string, T[]> = {};
  for (const stage of stages) result[stage.id] = [];
  for (const item of items) {
    if (!result[item.stageId]) result[item.stageId] = [];
    result[item.stageId].push(item);
  }
  return result;
}

export function adjacentStageTargets<T extends WorkItemLike, S extends StageLike>(
  item: T,
  stages: S[],
) {
  const index = stages.findIndex((stage) => stage.id === item.stageId);
  const previous = index > 0 ? stages[index - 1] : null;
  const next = index >= 0 && index < stages.length - 1 ? stages[index + 1] : null;
  return {
    previous: previous && canMoveToStage(item, previous) ? previous : null,
    next: next && canMoveToStage(item, next) ? next : null,
    blockedNext: next && !canMoveToStage(item, next) ? next : null,
  };
}

export function canMoveToStage<T extends WorkItemLike, S extends StageLike>(item: T, stage: S) {
  return !stage.provisionWorktree || Boolean(item.worktree);
}
