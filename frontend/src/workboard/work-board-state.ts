import type { WorkflowStage } from "../../bindings/github.com/phin-tech/whisk/internal/domain/workitem/models";
import type {
  Artifact,
  GateReport,
  Project,
  Question,
  WorkItem,
  WorkItemRun,
  WorkflowDefinitionRecord,
} from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
import {
  adjacentStageTargets,
  deriveWorkItemAttention,
  deriveWorkItemCardIndicators,
  groupWorkItemsByStage,
  type WorkItemAttention,
  type WorkItemCardIndicator,
} from "../workView";

export type WorkBoardFilters = {
  query?: string;
  stageId?: string;
  runState?: string;
};

export type WorkBoardCardTargets = {
  previous: WorkflowStage | null;
  next: WorkflowStage | null;
  blockedNext: WorkflowStage | null;
};

export type WorkBoardCardView = {
  key: string;
  item: WorkItem;
  targets: WorkBoardCardTargets;
  latestRun: WorkItemRun | null;
  terminalRun: WorkItemRun | null;
  attention: WorkItemAttention;
  indicators: WorkItemCardIndicator[];
  canExecute: boolean;
};

export type WorkBoardStageView = {
  key: string;
  stage: WorkflowStage;
  collapsed: boolean;
  count: number;
  hasAttention: boolean;
  attentionClass: string;
  cards: WorkBoardCardView[];
};

export type WorkBoardView = {
  activeProject: Project | null;
  workflowLabel: string;
  stages: WorkflowStage[];
  stageViews: WorkBoardStageView[];
  detailItem: WorkItem | null;
};

export type WorkBoardStateInput = {
  projects: Project[];
  activeProjectId: string;
  workItems: WorkItem[];
  workItemRuns: WorkItemRun[];
  artifacts: Artifact[];
  questions: Question[];
  gateReports: GateReport[];
  workflowDefinitions: WorkflowDefinitionRecord[];
  filters?: WorkBoardFilters;
  collapsedStageIds?: Set<string>;
  detailItemId?: string;
};

const UNASSIGNED_STAGE = {
  id: "__unassigned__",
  name: "Unassigned",
  kind: "unassigned",
} as WorkflowStage;

export function deriveWorkBoardView(input: WorkBoardStateInput): WorkBoardView {
  const activeProject = input.projects.find((project) => project.id === input.activeProjectId) ?? null;
  const stages = activeProject?.workflow?.stages ?? [];
  const filters = input.filters ?? {};
  const filteredWorkItems = filterWorkItems(input.workItems, filters);
  const boardStages = filters.stageId ? stages.filter((stage) => stage.id === filters.stageId) : stages;
  const itemsByStage = groupWorkItemsByStage(filteredWorkItems, stages);
  const knownStageIds = new Set(stages.map((stage) => stage.id));
  const runsByItem = groupRunsByItem(input.workItemRuns);
  const artifactsByItem = groupRecordsByItem(input.artifacts);
  const questionsByItem = groupRecordsByItem(input.questions);
  const gatesByItem = groupRecordsByItem(input.gateReports);
  const collapsedStageIds = input.collapsedStageIds ?? new Set<string>();

  const stageViews = boardStages.map((stage) => {
    const cards = (itemsByStage[stage.id] ?? []).map((item) =>
      deriveWorkBoardCardView(item, {
        stage,
        stages,
        runs: runsByItem[item.id] ?? [],
        artifacts: artifactsByItem[item.id] ?? [],
        questions: questionsByItem[item.id] ?? [],
        gates: gatesByItem[item.id] ?? [],
      }),
    );
    return {
      key: `stage:${stage.id}`,
      stage,
      collapsed: collapsedStageIds.has(stage.id),
      count: cards.length,
      hasAttention: hasStageAttention(cards),
      attentionClass: stageAttentionClass(cards),
      cards,
    };
  });
  const orphanedItems = activeProject
    ? Object.entries(itemsByStage)
        .filter(([stageId]) => !knownStageIds.has(stageId))
        .flatMap(([, items]) => items)
    : [];
  if (orphanedItems.length > 0) {
    const cards = orphanedItems.map((item) =>
      deriveWorkBoardCardView(item, {
        stage: UNASSIGNED_STAGE,
        stages,
        runs: runsByItem[item.id] ?? [],
        artifacts: artifactsByItem[item.id] ?? [],
        questions: questionsByItem[item.id] ?? [],
        gates: gatesByItem[item.id] ?? [],
      }),
    );
    stageViews.push({
      key: `stage:${UNASSIGNED_STAGE.id}`,
      stage: UNASSIGNED_STAGE,
      collapsed: collapsedStageIds.has(UNASSIGNED_STAGE.id),
      count: cards.length,
      hasAttention: hasStageAttention(cards),
      attentionClass: stageAttentionClass(cards),
      cards,
    });
  }

  return {
    activeProject,
    workflowLabel: workflowDefinitionLabel(activeProject, input.workflowDefinitions),
    stages,
    stageViews,
    detailItem: input.workItems.find((item) => item.id === input.detailItemId) ?? null,
  };
}

export function deriveWorkBoardCardView(
  item: WorkItem,
  context: {
    stage: WorkflowStage;
    stages: WorkflowStage[];
    runs: WorkItemRun[];
    artifacts: Artifact[];
    questions: Question[];
    gates: GateReport[];
  },
): WorkBoardCardView {
  const latestRun = context.runs[0] ?? null;
  const attention = deriveWorkItemAttention(item, {
    runs: context.runs,
    questions: context.questions,
    gates: context.gates,
    artifacts: context.artifacts,
    stageRequiresWorktree: Boolean(context.stage.provisionWorktree),
    stageRequiresPlan: stageRequiresPlan(context.stage),
  });
  return {
    key: `work-item:${item.id}`,
    item,
    targets: adjacentStageTargets(item, context.stages),
    latestRun,
    terminalRun: attention.terminalRunId
      ? context.runs.find((run) => run.id === attention.terminalRunId) ?? null
      : null,
    attention,
    indicators: deriveWorkItemCardIndicators(item, {
      runs: context.runs,
      artifacts: context.artifacts,
      gates: context.gates,
    }),
    canExecute: canQueueOrLaunchExecution(item, latestRun, context.artifacts),
  };
}

export function workflowDefinitionLabel(
  activeProject: Project | null,
  workflowDefinitions: WorkflowDefinitionRecord[],
) {
  if (!activeProject) return "";
  const workflow = activeProject.workflow;
  const id = workflow.definitionId || workflow.templateId || workflow.id;
  const version = workflow.definitionVersion ?? 0;
  const definition = workflowDefinitions.find(
    (candidate) =>
      candidate.id === workflow.definitionId && candidate.version === workflow.definitionVersion,
  );
  if (!id) return "workflow";
  const identityId = definition?.id || id;
  return version > 0 ? `${identityId}@${version}` : identityId;
}

export function defaultWorktreeBranch(item: WorkItem, activeProject: Project | null) {
  const projectSlug = activeProject?.slug || "work";
  const itemSlug = slugify(item.title) || "item";
  return `whisk/${projectSlug}-${item.number}-${itemSlug}`;
}

export function groupRunsByItem(runs: WorkItemRun[]) {
  const result: Record<string, WorkItemRun[]> = {};
  for (const run of runs) {
    if (!result[run.workItemId]) result[run.workItemId] = [];
    result[run.workItemId].push(run);
  }
  for (const itemRuns of Object.values(result)) {
    itemRuns.sort((a, b) => timestamp(b.createdAt) - timestamp(a.createdAt));
  }
  return result;
}

export function filterWorkItems(items: WorkItem[], filters: WorkBoardFilters = {}) {
  const query = (filters.query ?? "").trim().toLowerCase();
  const stageId = filters.stageId ?? "";
  const runState = filters.runState ?? "";
  return items.filter((item) => {
    if (stageId && item.stageId !== stageId) return false;
    if (runState && (item.runState || "idle") !== runState) return false;
    if (!query) return true;
    return `#${item.number} ${item.title} ${item.bodyMarkdown} ${item.stageId} ${item.runState}`
      .toLowerCase()
      .includes(query);
  });
}

export function canOpenRunTerminal(run: WorkItemRun | null) {
  return Boolean(run?.sessionId || run?.ptyId);
}

export function canQueueOrLaunchExecution(
  item: WorkItem,
  latestRun: WorkItemRun | null,
  artifacts: Artifact[],
) {
  return item.stageId === "ready" && hasApprovedPlan(item.id, artifacts) && !hasActiveRun(latestRun);
}

export function stageRequiresPlan(stage: WorkflowStage) {
  const value = `${stage.id} ${stage.name}`.toLowerCase();
  return value.includes("execution") || value.includes("review");
}

export function attentionDotClass(tone: string) {
  if (tone === "danger") return "text-red";
  if (tone === "warning") return "text-amber";
  if (tone === "success") return "text-green";
  return "text-blue";
}

export function cardRailClass(severity: string) {
  if (severity === "danger") return "bg-red";
  if (severity === "warning") return "bg-amber";
  if (severity === "info") return "bg-blue";
  return "bg-border";
}

export function hasStageAttention(cards: Pick<WorkBoardCardView, "attention">[]) {
  return cards.some((card) => card.attention.severity !== "none");
}

export function stageAttentionClass(cards: Pick<WorkBoardCardView, "attention">[]) {
  const severities = cards.map((card) => card.attention.severity);
  if (severities.includes("danger")) return "bg-red";
  if (severities.includes("warning")) return "bg-amber";
  if (severities.includes("info")) return "bg-blue";
  return "bg-border";
}

function slugify(value: string) {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function timestamp(value: unknown) {
  if (!value) return 0;
  if (value instanceof Date) return value.getTime();
  return new Date(String(value)).getTime() || 0;
}

function hasApprovedPlan(itemId: string, artifacts: Artifact[]) {
  return artifacts.some(
    (artifact) =>
      artifact.workItemId === itemId && artifact.kind === "plan" && artifact.status === "approved",
  );
}

function hasActiveRun(run: WorkItemRun | null) {
  return run?.status === "queued" || run?.status === "running" || run?.status === "awaiting_input";
}

function groupRecordsByItem<T extends { workItemId: string }>(records: T[]) {
  const result: Record<string, T[]> = {};
  for (const record of records) {
    if (!result[record.workItemId]) result[record.workItemId] = [];
    result[record.workItemId].push(record);
  }
  return result;
}
