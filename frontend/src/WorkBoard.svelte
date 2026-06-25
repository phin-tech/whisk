<script lang="ts">
  import ArrowLeft from "@lucide/svelte/icons/arrow-left";
  import ArrowRight from "@lucide/svelte/icons/arrow-right";
  import Clock3 from "@lucide/svelte/icons/clock-3";
  import GitBranch from "@lucide/svelte/icons/git-branch";
  import MoreHorizontal from "@lucide/svelte/icons/ellipsis";
  import Play from "@lucide/svelte/icons/play";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
  import WorkItemDetail from "./WorkItemDetail.svelte";
  import EmptyState from "./ui/EmptyState.svelte";
  import StatusDot from "./ui/StatusDot.svelte";
  import type { WorkflowStage } from "../bindings/github.com/phin-tech/whisk/internal/domain/workitem/models";
  import type {
    AgentProfile,
    Artifact,
    GateReport,
    Project,
    Question,
    WorkItem,
    WorkItemLink,
    WorkItemRun,
    WorkflowEvent,
    ReadyWorkExplanation,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import {
    adjacentStageTargets,
    collapsedStageStorageKey,
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
    return value.includes("execution") || value.includes("review") || value.includes("done");
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
          <input
            class="h-9 min-w-0 flex-1 bg-transparent px-3 text-[14px] text-text-primary outline-none placeholder:text-text-muted"
            type="text"
            bind:value={newItemTitle}
            placeholder="Create work item"
            disabled={loading}
          />
          <button
            type="button"
            class="inline-flex h-8 w-8 shrink-0 items-center justify-center text-text-muted transition-colors hover:text-text-primary"
            aria-label={createBodyOpen ? "Hide work item body" : "Add work item body"}
            title={createBodyOpen ? "Hide body" : "Add body"}
            on:click={() => (createBodyOpen = !createBodyOpen)}
          >
            <MoreHorizontal size={16} />
          </button>
          <button
            type="button"
            class="mr-1 inline-flex h-7 shrink-0 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 px-2.5 text-[12px] font-semibold text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
            disabled={loading || !newItemTitle.trim()}
            on:click={createWorkItem}
          >
            Create
          </button>
        </div>
      {/if}
      <button
        type="button"
        class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-surface hover:text-text-primary disabled:cursor-wait disabled:opacity-60"
        disabled={loading}
        aria-label="Refresh work board"
        title="Refresh work board"
        on:click={onRefresh}
      >
        <RefreshCw size={15} class={loading ? "animate-spin" : ""} />
      </button>
      <button
        type="button"
        class="hidden h-9 items-center justify-center rounded-md border border-border-subtle bg-bg-surface/60 px-2.5 text-[12px] font-medium text-text-secondary transition-colors hover:bg-bg-surface hover:text-text-primary sm:inline-flex"
        disabled={loading || boardStages.length === 0}
        on:click={() => setAllColumnsCollapsed(false)}
      >
        Expand
      </button>
      <button
        type="button"
        class="hidden h-9 items-center justify-center rounded-md border border-border-subtle bg-bg-surface/60 px-2.5 text-[12px] font-medium text-text-secondary transition-colors hover:bg-bg-surface hover:text-text-primary sm:inline-flex"
        disabled={loading || boardStages.length === 0}
        on:click={() => setAllColumnsCollapsed(true)}
      >
        Collapse
      </button>
    </div>
  </div>

  {#if activeProject}
    {#if createBodyOpen}
      <div class="shrink-0 border-b border-hairline bg-bg-base px-3 py-2">
        <textarea
          class="h-20 w-full resize-none rounded-md border border-border bg-bg-deep px-3 py-2 text-[13px] leading-5 text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
          bind:value={newItemBody}
          placeholder="Markdown body"
          disabled={loading}
        ></textarea>
      </div>
    {/if}

    <div class="app-scrollbar min-h-0 flex-1 overflow-auto p-3">
      <div class="flex min-h-full min-w-max items-stretch gap-3">
        {#each boardStages as stage (stage.id)}
          {@const stageItems = itemsByStage[stage.id] ?? []}
          {@const collapsed = collapsedStageIds.has(stage.id)}
          {@const stageHasAttention = hasStageAttention(stage)}
          {#if collapsed}
            <button
              type="button"
              class="flex min-h-[420px] w-12 shrink-0 flex-col items-center justify-between rounded-md border border-border-subtle bg-bg-base px-2 py-3 text-text-secondary transition-colors hover:border-border hover:bg-bg-surface hover:text-text-primary"
              aria-label={`Expand ${stage.name}`}
              title={`Expand ${stage.name}`}
              on:click={() => toggleStageCollapsed(stage.id)}
            >
              <span class="writing-vertical truncate text-[13px] font-semibold">{stage.name}</span>
              <span class="flex flex-col items-center gap-2">
                {#if stageHasAttention}
                  <span class="h-2 w-2 rounded-full {stageAttentionClass(stage)}"></span>
                {/if}
                <span class="rounded border border-border-subtle bg-bg-deep/80 px-1.5 py-1 font-mono text-[12px]">
                  {stageItems.length}
                </span>
              </span>
            </button>
          {:else}
            <section class="flex min-h-[420px] w-[320px] shrink-0 flex-col overflow-hidden rounded-md border border-border-subtle bg-bg-base">
              <div class="flex h-12 min-w-0 shrink-0 items-center justify-between gap-2 border-b border-hairline px-3">
                <div class="flex min-w-0 items-center gap-2">
                  {#if stageHasAttention}
                    <span class="h-2 w-2 shrink-0 rounded-full {stageAttentionClass(stage)}"></span>
                  {/if}
                  <button
                    type="button"
                    class="min-w-0 truncate text-left text-[13px] font-semibold text-text-primary outline-none focus-visible:text-accent"
                    aria-label={`Collapse ${stage.name}`}
                    title="Double-click to collapse"
                    on:dblclick={() => toggleStageCollapsed(stage.id)}
                  >
                    {stage.name}
                  </button>
                  <div class="rounded border border-hairline bg-bg-deep/70 px-1.5 py-0.5 font-mono text-[12px] text-text-secondary">
                    {stageItems.length}
                  </div>
                </div>
                <button
                  type="button"
                  class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-surface/60 hover:text-text-primary"
                  aria-label={`Collapse ${stage.name}`}
                  title={`Collapse ${stage.name}`}
                  on:click={() => toggleStageCollapsed(stage.id)}
                >
                  <ArrowLeft size={13} />
                </button>
              </div>

              <div class="min-w-0 flex-1 divide-y divide-hairline">
                {#if stageItems.length === 0}
                  <EmptyState
                    message="Empty"
                    class="flex min-h-24 items-center justify-center py-7 text-center text-[13px]"
                  />
                {:else}
                  {#each stageItems as item (item.id)}
                    {@const targets = adjacentStageTargets(item, stages)}
                    {@const latestRun = (runsByItem[item.id] ?? [])[0] ?? null}
                    {@const canExecute = canQueueOrLaunchExecution(item, latestRun)}
                    {@const attention = attentionFor(item, stage)}
                    {@const terminalRun = attention.terminalRunId
                      ? (runsByItem[item.id] ?? []).find((run) => run.id === attention.terminalRunId)
                      : null}
                    <article class="group relative min-h-[76px] overflow-hidden bg-bg-base transition-colors hover:bg-bg-surface/60 focus-within:bg-bg-surface/60">
                      <div class="absolute inset-y-3 left-0 w-1 rounded-r {cardRailClass(attention.severity)}"></div>
                      <div class="grid gap-2 px-3 py-3 pl-4">
                        <div class="flex min-w-0 items-start gap-2">
                          <button
                            type="button"
                            class="work-card-title min-w-0 flex-1 text-left text-[14px] font-semibold leading-5 text-text-primary outline-none transition-colors hover:text-accent focus-visible:text-accent"
                            on:click={() => openDetail(item)}
                          >
                            <span class="font-mono text-[12px] font-medium text-text-muted">#{item.number}</span>
                            {item.title}
                          </button>
                          {#if terminalRun}
                            <button
                              type="button"
                              aria-label="Open running terminal"
                              title="Open running terminal"
                              class="inline-flex h-7 w-7 shrink-0 animate-pulse items-center justify-center rounded border border-green/45 bg-green/12 text-green transition-colors hover:border-green disabled:cursor-not-allowed"
                              disabled={loading}
                              on:click={() => openRunTerminal(terminalRun)}
                            >
                              <SquareTerminal size={14} />
                            </button>
                          {/if}
                          <div class="flex shrink-0 gap-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
                            {#if latestRun?.status === "queued"}
                              <button
                                type="button"
                                aria-label="Launch queued run"
                                title="Launch queued run"
                                class="inline-flex h-7 w-7 items-center justify-center rounded border border-blue/35 bg-blue/10 text-blue transition-colors hover:border-blue disabled:cursor-not-allowed"
                                disabled={loading}
                                on:click={() => onLaunchRun(latestRun.id)}
                              >
                                <Play size={13} />
                              </button>
                            {/if}
                            {#if canExecute}
                              <button
                                type="button"
                                aria-label="Queue execution"
                                title="Queue execution"
                                class="inline-flex h-7 w-7 items-center justify-center rounded border border-blue/35 bg-blue/10 text-blue transition-colors hover:border-blue disabled:cursor-not-allowed"
                                disabled={loading}
                                on:click={() => onQueueExecution(item.id)}
                              >
                                <Clock3 size={13} />
                              </button>
                              <button
                                type="button"
                                aria-label="Launch execution"
                                title="Launch execution"
                                class="inline-flex h-7 w-7 items-center justify-center rounded border border-green/35 bg-green/10 text-green transition-colors hover:border-green disabled:cursor-not-allowed"
                                disabled={loading}
                                on:click={() => onLaunchExecution(item.id)}
                              >
                                <Play size={13} />
                              </button>
                            {/if}
                            {#if targets.blockedNext && !item.worktree}
                              <button
                                type="button"
                                aria-label="Generate worktree"
                                title="Generate worktree"
                                class="inline-flex h-7 w-7 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
                                disabled={loading}
                                on:click={() => generateWorktree(item)}
                              >
                                <GitBranch size={13} />
                              </button>
                            {/if}
                            <button
                              type="button"
                              aria-label="Move previous"
                              title="Move previous"
                              class="inline-flex h-7 w-7 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-surface hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-40"
                              disabled={loading || !targets.previous}
                              on:click={() => movePrevious(item)}
                            >
                              <ArrowLeft size={13} />
                            </button>
                            <button
                              type="button"
                              aria-label={targets.blockedNext ? "Generate worktree before moving" : "Move next"}
                              title={targets.blockedNext ? "Generate worktree before moving" : "Move next"}
                              class="inline-flex h-7 w-7 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-surface hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-40"
                              disabled={loading || !targets.next}
                              on:click={() => moveNext(item)}
                            >
                              <ArrowRight size={13} />
                            </button>
                          </div>
                        </div>

                        <div class="flex min-w-0 flex-wrap items-center gap-1.5">
                          {#if attention.signals.length > 0}
                            {#each attention.signals as signal (signal.id)}
                              <span class="inline-flex min-w-0 items-center gap-1 text-[12px]">
                                <span class={attentionDotClass(signal.tone)}>●</span>
                                <span class="truncate text-text-muted">{signal.label}</span>
                              </span>
                            {/each}
                          {:else}
                            <StatusDot
                              status={item.runState || "idle"}
                              label={item.runState || "Idle"}
                              showLabel
                              class="text-[12px]"
                            />
                          {/if}
                        </div>
                      </div>
                    </article>
                  {/each}
                {/if}
              </div>
            </section>
          {/if}
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
