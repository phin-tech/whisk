<script lang="ts">
  import MoreHorizontal from "@lucide/svelte/icons/ellipsis";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import WorkBoardColumn from "./workboard/WorkBoardColumn.svelte";
  import WorkItemCard from "./workboard/WorkItemCard.svelte";
  import WorkItemDetail from "./WorkItemDetail.svelte";
  import Button from "./ui/Button.svelte";
  import EmptyState from "./ui/EmptyState.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import TextArea from "./ui/TextArea.svelte";
  import TextField from "./ui/TextField.svelte";
  import type { WorkflowStage } from "../bindings/github.com/phin-tech/whisk/internal/domain/workitem/models";
  import type {
    AgentProfile,
    Artifact,
    GateReport,
    Project,
    Question,
    ReadyWorkExplanation,
    WorkItem,
    WorkItemLink,
    WorkItemRun,
    WorkflowEvent,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import {
    adjacentStageTargets,
    collapsedStageStorageKey,
    deriveWorkItemCardIndicators,
    deriveWorkItemAttention,
    groupWorkItemsByStage,
    parseCollapsedStages,
    serializeCollapsedStages,
  } from "./workView";

  export let projects: Project[] = [];
  export let workItems: WorkItem[] = [];
  export let workItemLinks: WorkItemLink[] = [];
  export let readyWork: ReadyWorkExplanation = { ready: [], blocked: [], summary: { totalReady: 0, totalBlocked: 0, cycleCount: 0 } };
  export let workItemRuns: WorkItemRun[] = [];
  export let artifacts: Artifact[] = [];
  export let questions: Question[] = [];
  export let gateReports: GateReport[] = [];
  export let workflowEvents: WorkflowEvent[] = [];
  export let agentProfiles: AgentProfile[] = [];
  export let activeProjectId = "";
  export let filterQuery = "";
  export let filterStageId = "";
  export let filterRunState = "";
  export let loading = false;
  export let onRefresh: () => void;
  export let onCreateWorkItem: (request: {
    projectId: string;
    title: string;
    bodyMarkdown: string;
  }) => void;
  export let onUpdateWorkItem: (request: {
    id: string;
    title: string;
    bodyMarkdown: string;
  }) => void;
  export let onMoveWorkItem: (workItemId: string, stageId: string) => void;
  export let onAddWorkItemLink: (request: {
    sourceWorkItemId: string;
    targetWorkItemId: string;
    type: string;
  }) => void;
  export let onGenerateWorktree: (request: { workItemId: string; branch: string }) => void;
  export let onAttachFile: (workItemId: string, path: string) => void;
  export let onDeleteWorkItem: (workItemId: string) => void;
  export let onCancelRun: (runId: string) => void;
  export let onLaunchRun: (runId: string, agentProfileId?: string) => void;
  export let onOpenRunTerminal: (run: WorkItemRun) => void;
  export let onStartPlanning: (workItemId: string) => void;
  export let onSubmitPlan: (request: {
    workItemId: string;
    runId: string;
    title: string;
    body: string;
  }) => void;
  export let onApprovePlan: (workItemId: string, artifactId: string) => void;
  export let onQueueExecution: (workItemId: string) => void;
  export let onLaunchExecution: (workItemId: string, agentProfileId?: string) => void;
  export let onSetPhaseAgent: (projectId: string, preset: string, agentProfileId: string) => void;
  export let onSetInteractiveAgentShell: (projectId: string, enabled: boolean) => void;
  export let onCompleteExecution: (request: {
    workItemId: string;
    runId: string;
    message: string;
  }) => void;
  export let onSubmitReviewFeedback: (request: {
    workItemId: string;
    runId: string;
    body: string;
  }) => void;
  export let onAskQuestion: (request: { workItemId: string; runId: string; prompt: string }) => void;
  export let onAnswerQuestion: (questionId: string, answer: string) => void;
  export let onCompleteGate: (request: { id: string; status: string; overrideReason: string }) => void;
  export let onApproveDone: (workItemId: string, reason: string) => void;

  export let openItemId = "";
  export let onDetailClose: (() => void) | null = null;

  let newItemTitle = "";
  let newItemBody = "";
  let worktreeBranches: Record<string, string> = {};
  let detailItemId = "";
  let appliedOpenItemId = "";

  $: if (openItemId && openItemId !== appliedOpenItemId) {
    appliedOpenItemId = openItemId;
    detailItemId = openItemId;
  }
  let createBodyOpen = false;
  let collapsedStageIds = new Set<string>();
  let collapsedProjectId = "";

  $: activeProject = projects.find((project) => project.id === activeProjectId) ?? null;
  $: stages = activeProject?.workflow?.stages ?? [];
  $: filteredWorkItems = filterWorkItems(workItems);
  $: boardStages = filterStageId ? stages.filter((stage) => stage.id === filterStageId) : stages;
  $: itemsByStage = groupWorkItemsByStage(filteredWorkItems, stages);
  $: detailItem = workItems.find((item) => item.id === detailItemId) ?? null;
  $: runsByItem = groupRunsByItem(workItemRuns);
  $: if (activeProjectId !== collapsedProjectId) {
    collapsedProjectId = activeProjectId;
    collapsedStageIds =
      typeof localStorage === "undefined"
        ? new Set<string>()
        : parseCollapsedStages(localStorage.getItem(collapsedStageStorageKey(activeProjectId)));
  }

  function slugify(value: string) {
    return value
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "");
  }

  function defaultWorktreeBranch(item: WorkItem) {
    const projectSlug = activeProject?.slug || "work";
    const itemSlug = slugify(item.title) || "item";
    return `whisk/${projectSlug}-${item.number}-${itemSlug}`;
  }

  function groupRunsByItem(runs: WorkItemRun[]) {
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

  function filterWorkItems(items: WorkItem[]) {
    const query = filterQuery.trim().toLowerCase();
    return items.filter((item) => {
      if (filterStageId && item.stageId !== filterStageId) return false;
      if (filterRunState && (item.runState || "idle") !== filterRunState) return false;
      if (!query) return true;
      return `#${item.number} ${item.title} ${item.bodyMarkdown} ${item.stageId} ${item.runState}`
        .toLowerCase()
        .includes(query);
    });
  }

  function timestamp(value: unknown) {
    if (!value) return 0;
    if (value instanceof Date) return value.getTime();
    return new Date(String(value)).getTime() || 0;
  }

  function canOpenRunTerminal(run: WorkItemRun | null) {
    return Boolean(run?.sessionId || run?.ptyId);
  }

  function hasApprovedPlan(itemId: string) {
    return artifacts.some(
      (artifact) =>
        artifact.workItemId === itemId && artifact.kind === "plan" && artifact.status === "approved",
    );
  }

  function hasActiveRun(run: WorkItemRun | null) {
    return run?.status === "queued" || run?.status === "running" || run?.status === "awaiting_input";
  }

  function canQueueOrLaunchExecution(item: WorkItem, latestRun: WorkItemRun | null) {
    return item.stageId === "ready" && hasApprovedPlan(item.id) && !hasActiveRun(latestRun);
  }

  function openRunTerminal(run: WorkItemRun) {
    if (!canOpenRunTerminal(run)) return;
    onOpenRunTerminal(run);
  }

  function stageRequiresPlan(stage: WorkflowStage) {
    const value = `${stage.id} ${stage.name}`.toLowerCase();
    return value.includes("execution") || value.includes("review");
  }

  function attentionFor(item: WorkItem, stage: WorkflowStage) {
    return deriveWorkItemAttention(item, {
      runs: runsByItem[item.id] ?? [],
      questions: detailRecordsForItem(questions, item.id),
      gates: detailRecordsForItem(gateReports, item.id),
      artifacts: detailRecordsForItem(artifacts, item.id),
      stageRequiresWorktree: Boolean(stage.provisionWorktree),
      stageRequiresPlan: stageRequiresPlan(stage),
    });
  }

  function detailRecordsForItem<T extends { workItemId: string }>(records: T[], itemId: string) {
    return records.filter((record) => record.workItemId === itemId);
  }

  function attentionDotClass(tone: string) {
    if (tone === "danger") return "text-red";
    if (tone === "warning") return "text-amber";
    if (tone === "success") return "text-green";
    return "text-blue";
  }

  function cardRailClass(severity: string) {
    if (severity === "danger") return "bg-red";
    if (severity === "warning") return "bg-amber";
    if (severity === "info") return "bg-blue";
    return "bg-border";
  }

  function hasStageAttention(stage: WorkflowStage) {
    return (itemsByStage[stage.id] ?? []).some(
      (item) => attentionFor(item, stage).severity !== "none",
    );
  }

  function stageAttentionClass(stage: WorkflowStage) {
    const severities = (itemsByStage[stage.id] ?? []).map((item) => attentionFor(item, stage).severity);
    if (severities.includes("danger")) return "bg-red";
    if (severities.includes("warning")) return "bg-amber";
    if (severities.includes("info")) return "bg-blue";
    return "bg-border";
  }

  function toggleStageCollapsed(stageId: string) {
    const next = new Set(collapsedStageIds);
    if (next.has(stageId)) {
      next.delete(stageId);
    } else {
      next.add(stageId);
    }
    collapsedStageIds = next;
    if (typeof localStorage !== "undefined" && activeProjectId) {
      localStorage.setItem(collapsedStageStorageKey(activeProjectId), serializeCollapsedStages(next));
    }
  }

  function setAllColumnsCollapsed(collapsed: boolean) {
    const next = collapsed ? new Set(boardStages.map((stage) => stage.id)) : new Set<string>();
    collapsedStageIds = next;
    if (typeof localStorage !== "undefined" && activeProjectId) {
      localStorage.setItem(collapsedStageStorageKey(activeProjectId), serializeCollapsedStages(next));
    }
  }

  function createWorkItem() {
    if (!activeProject || !newItemTitle.trim() || loading) return;
    onCreateWorkItem({
      projectId: activeProject.id,
      title: newItemTitle.trim(),
      bodyMarkdown: newItemBody.trim(),
    });
    newItemTitle = "";
    newItemBody = "";
  }

  function generateWorktree(item: WorkItem) {
    const branch = (worktreeBranches[item.id] || defaultWorktreeBranch(item)).trim();
    if (!branch || loading) return;
    onGenerateWorktree({ workItemId: item.id, branch });
  }

  function openDetail(item: WorkItem) {
    detailItemId = item.id;
  }

  function movePrevious(item: WorkItem) {
    const { previous } = adjacentStageTargets(item, stages);
    if (previous) onMoveWorkItem(item.id, previous.id);
  }

  function moveNext(item: WorkItem) {
    const { next } = adjacentStageTargets(item, stages);
    if (next) onMoveWorkItem(item.id, next.id);
  }

  function closeDetail() {
    detailItemId = "";
    onDetailClose?.();
  }
</script>

<div class="flex min-h-0 flex-1 flex-col bg-bg-deep">
  <div class="flex min-h-14 shrink-0 flex-wrap items-center justify-between gap-2 border-b border-hairline bg-bg-base px-3 py-2">
    <div class="flex min-w-0 flex-1 items-center gap-2">
      {#if activeProject}
        <div class="min-w-0">
          <div class="truncate text-[13px] font-semibold text-text-primary">{activeProject.name}</div>
          <div class="truncate text-[12px] text-text-secondary">{activeProject.rootDir}</div>
        </div>
      {:else}
        <div class="truncate text-[13px] text-text-muted">No project selected</div>
      {/if}
    </div>

    <div class="flex min-w-0 flex-1 items-center justify-end gap-2">
      {#if activeProject}
        <div class="flex min-w-[260px] max-w-[520px] flex-1 items-center overflow-hidden rounded-md border border-border-subtle bg-bg-surface/60 focus-within:border-accent-dim">
          <TextField
            bind:value={newItemTitle}
            variant="seamless"
            placeholder="Create work item"
            disabled={loading}
            class="h-9 min-w-0 flex-1 border-transparent bg-transparent px-3 text-[14px]"
          />
          <IconButton
            label={createBodyOpen ? "Hide work item body" : "Add work item body"}
            title={createBodyOpen ? "Hide body" : "Add body"}
            class="border-transparent hover:border-transparent"
            onclick={() => (createBodyOpen = !createBodyOpen)}
          >
            <MoreHorizontal size={16} />
          </IconButton>
          <Button
            size="sm"
            class="mr-1 shrink-0"
            disabled={loading || !newItemTitle.trim()}
            onclick={createWorkItem}
          >
            Create
          </Button>
        </div>
      {/if}
      <IconButton label="Refresh work board" disabled={loading} class="h-9 w-9" onclick={onRefresh}>
        <RefreshCw size={15} class={loading ? "animate-spin" : ""} />
      </IconButton>
      <Button size="lg" class="hidden sm:inline-flex" disabled={loading || boardStages.length === 0} onclick={() => setAllColumnsCollapsed(false)}>
        Expand
      </Button>
      <Button size="lg" class="hidden sm:inline-flex" disabled={loading || boardStages.length === 0} onclick={() => setAllColumnsCollapsed(true)}>
        Collapse
      </Button>
    </div>
  </div>

  {#if activeProject}
    {#if createBodyOpen}
      <div class="shrink-0 border-b border-hairline bg-bg-base px-3 py-2">
        <TextArea
          bind:value={newItemBody}
          placeholder="Markdown body"
          disabled={loading}
          class="h-20 resize-none text-[13px] leading-5"
        />
      </div>
    {/if}

    <div class="app-scrollbar min-h-0 flex-1 overflow-auto p-3">
      <div class="flex min-h-full min-w-max items-stretch gap-3">
        {#each boardStages as stage (stage.id)}
          {@const stageItems = itemsByStage[stage.id] ?? []}
          {@const collapsed = collapsedStageIds.has(stage.id)}
          {@const stageHasAttention = hasStageAttention(stage)}
          <WorkBoardColumn
            {stage}
            count={stageItems.length}
            {collapsed}
            hasAttention={stageHasAttention}
            attentionClass={stageAttentionClass(stage)}
            onToggle={toggleStageCollapsed}
          >
            {#each stageItems as item (item.id)}
              {@const targets = adjacentStageTargets(item, stages)}
              {@const latestRun = (runsByItem[item.id] ?? [])[0] ?? null}
              {@const canExecute = canQueueOrLaunchExecution(item, latestRun)}
              {@const attention = attentionFor(item, stage)}
              {@const indicators = deriveWorkItemCardIndicators(item, {
                runs: runsByItem[item.id] ?? [],
                artifacts: detailRecordsForItem(artifacts, item.id),
                gates: detailRecordsForItem(gateReports, item.id),
              })}
              {@const terminalRun = attention.terminalRunId
                ? (runsByItem[item.id] ?? []).find((run) => run.id === attention.terminalRunId) ?? null
                : null}
              <WorkItemCard
                {item}
                {targets}
                {latestRun}
                {terminalRun}
                {attention}
                {indicators}
                {canExecute}
                {loading}
                {cardRailClass}
                {attentionDotClass}
                onOpenDetail={openDetail}
                onOpenRunTerminal={openRunTerminal}
                {onLaunchRun}
                {onQueueExecution}
                onLaunchExecution={(workItemId) => onLaunchExecution(workItemId)}
                onGenerateWorktree={generateWorktree}
                onMovePrevious={movePrevious}
                onMoveNext={moveNext}
              />
            {/each}
          </WorkBoardColumn>
        {/each}
      </div>
    </div>
  {:else}
    <div class="flex min-h-0 flex-1 items-center justify-center p-6">
      <EmptyState
        title="No project selected"
        message="Select or create a project from the sidebar."
        class="grid max-w-sm gap-3 text-center text-[13px]"
      />
    </div>
  {/if}
</div>

{#if detailItem && activeProject}
  <WorkItemDetail
    item={detailItem}
    project={activeProject}
    {stages}
    {agentProfiles}
    {workItems}
    workItemLinks={workItemLinks}
    readyWork={readyWork}
    {workItemRuns}
    {artifacts}
    {questions}
    {gateReports}
    {workflowEvents}
    {loading}
    onClose={closeDetail}
    {onUpdateWorkItem}
    {onMoveWorkItem}
    onAddWorkItemLink={onAddWorkItemLink}
    {onGenerateWorktree}
    {onAttachFile}
    {onDeleteWorkItem}
    {onCancelRun}
    {onLaunchRun}
    {onOpenRunTerminal}
    {onStartPlanning}
    {onSubmitPlan}
    {onApprovePlan}
    {onQueueExecution}
    {onLaunchExecution}
    {onSetPhaseAgent}
    {onSetInteractiveAgentShell}
    {onCompleteExecution}
    {onSubmitReviewFeedback}
    {onAskQuestion}
    {onAnswerQuestion}
    {onCompleteGate}
    {onApproveDone}
  />
{/if}
