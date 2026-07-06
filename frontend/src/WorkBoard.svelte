<script lang="ts">
  import MoreHorizontal from "@lucide/svelte/icons/ellipsis";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import VirtualWorkItemList from "./workboard/VirtualWorkItemList.svelte";
  import WorkBoardColumn from "./workboard/WorkBoardColumn.svelte";
  import WorkItemDetail from "./WorkItemDetail.svelte";
  import Button from "./ui/Button.svelte";
  import EmptyState from "./ui/EmptyState.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import TextArea from "./ui/TextArea.svelte";
  import TextField from "./ui/TextField.svelte";
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
    WorkflowActionAvailability,
    WorkflowDefinitionRecord,
    WorkflowEvent,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import {
    attentionDotClass,
    canOpenRunTerminal,
    cardRailClass,
    defaultWorktreeBranch,
    deriveWorkBoardView,
  } from "./workboard/work-board-state";
  import {
    adjacentStageTargets,
    collapsedStageStorageKey,
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
  export let workflowDefinitions: WorkflowDefinitionRecord[] = [];
  export let workflowActionsByItem: Record<string, WorkflowActionAvailability[]> = {};
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
  export let onRunWorkflowAction: (request: {
    workItemId: string;
    actionId: string;
    runId?: string;
    reason?: string;
  }) => void;
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

  $: workBoardView = deriveWorkBoardView({
    projects,
    activeProjectId,
    workItems,
    workItemRuns,
    artifacts,
    questions,
    gateReports,
    workflowDefinitions,
    filters: {
      query: filterQuery,
      stageId: filterStageId,
      runState: filterRunState,
    },
    collapsedStageIds,
    detailItemId,
  });
  $: activeProject = workBoardView.activeProject;
  $: workflowLabel = workBoardView.workflowLabel;
  $: stages = workBoardView.stages;
  $: stageViews = workBoardView.stageViews;
  $: detailItem = workBoardView.detailItem;
  $: if (activeProjectId !== collapsedProjectId) {
    collapsedProjectId = activeProjectId;
    collapsedStageIds =
      typeof localStorage === "undefined"
        ? new Set<string>()
        : parseCollapsedStages(localStorage.getItem(collapsedStageStorageKey(activeProjectId)));
  }

  function openRunTerminal(run: WorkItemRun) {
    if (!canOpenRunTerminal(run)) return;
    onOpenRunTerminal(run);
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
    const next = collapsed ? new Set(stageViews.map((stageView) => stageView.stage.id)) : new Set<string>();
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
    const branch = (worktreeBranches[item.id] || defaultWorktreeBranch(item, activeProject)).trim();
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
          <div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5 text-[12px] text-text-secondary">
            <span class="truncate">{activeProject.rootDir}</span>
            <span class="shrink-0 text-text-muted">{workflowLabel}</span>
          </div>
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
      <Button size="lg" class="hidden sm:inline-flex" disabled={loading || stageViews.length === 0} onclick={() => setAllColumnsCollapsed(false)}>
        Expand
      </Button>
      <Button size="lg" class="hidden sm:inline-flex" disabled={loading || stageViews.length === 0} onclick={() => setAllColumnsCollapsed(true)}>
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
      <div class="flex h-full min-w-max items-stretch gap-3">
        {#each stageViews as stageView (stageView.key)}
          <WorkBoardColumn
            stage={stageView.stage}
            count={stageView.count}
            collapsed={stageView.collapsed}
            hasAttention={stageView.hasAttention}
            attentionClass={stageView.attentionClass}
            onToggle={toggleStageCollapsed}
          >
            <VirtualWorkItemList
              stageName={stageView.stage.name}
              cards={stageView.cards}
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
    workflowActions={workflowActionsByItem[detailItem.id] ?? []}
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
    {onLaunchExecution}
    {onSetPhaseAgent}
    {onSetInteractiveAgentShell}
    {onCompleteExecution}
    {onSubmitReviewFeedback}
    {onAskQuestion}
    {onAnswerQuestion}
    {onCompleteGate}
    {onRunWorkflowAction}
    {onApproveDone}
  />
{/if}
