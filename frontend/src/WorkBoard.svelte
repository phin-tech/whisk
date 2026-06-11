<script lang="ts">
  import { Dialogs } from "@wailsio/runtime";
  import ArrowLeft from "@lucide/svelte/icons/arrow-left";
  import ArrowRight from "@lucide/svelte/icons/arrow-right";
  import ClipboardCheck from "@lucide/svelte/icons/clipboard-check";
  import GitBranch from "@lucide/svelte/icons/git-branch";
  import Paperclip from "@lucide/svelte/icons/paperclip";
  import Play from "@lucide/svelte/icons/play";
  import Plus from "@lucide/svelte/icons/plus";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Search from "@lucide/svelte/icons/search";
  import Square from "@lucide/svelte/icons/square";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import X from "@lucide/svelte/icons/x";
  import type { WorkflowStage } from "../bindings/github.com/phin-tech/whisk/internal/domain/workitem/models";
  import type { Project, WorkItem, WorkItemRun } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { adjacentStageTargets, canMoveToStage, groupWorkItemsByStage } from "./workView";

  export let projects: Project[] = [];
  export let workItems: WorkItem[] = [];
  export let workItemRuns: WorkItemRun[] = [];
  export let activeProjectId = "";
  export let loading = false;
  export let onRefresh: () => void;
  export let onNewProject: () => void;
  export let onSelectProject: (projectId: string) => void;
  export let onCreateWorkItem: (request: {
    projectId: string;
    title: string;
    bodyMarkdown: string;
  }) => void;
  export let onMoveWorkItem: (workItemId: string, stageId: string) => void;
  export let onGenerateWorktree: (request: { workItemId: string; branch: string }) => void;
  export let onAttachFile: (workItemId: string, path: string) => void;
  export let onDeleteWorkItem: (workItemId: string) => void;
  export let onStartRun: (request: {
    workItemId: string;
    preset: string;
    promptTemplateId: string;
    agentProfileId: string;
  }) => void;
  export let onCancelRun: (runId: string) => void;

  let newItemTitle = "";
  let newItemBody = "";
  let worktreeBranches: Record<string, string> = {};
  let detailItemId = "";
  let agentProfileId = "codex";

  $: activeProject = projects.find((project) => project.id === activeProjectId) ?? null;
  $: stages = activeProject?.workflow?.stages ?? [];
  $: itemsByStage = groupWorkItemsByStage(workItems, stages);
  $: detailItem = workItems.find((item) => item.id === detailItemId) ?? null;
  $: runsByItem = groupRunsByItem(workItemRuns);
  $: detailRuns = detailItem ? (runsByItem[detailItem.id] ?? []) : [];

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

  function timestamp(value: unknown) {
    if (!value) return 0;
    if (value instanceof Date) return value.getTime();
    return new Date(String(value)).getTime() || 0;
  }

  function formattedTime(value: unknown) {
    if (!value) return "";
    if (value instanceof Date) return value.toLocaleString();
    const parsed = new Date(String(value));
    return Number.isNaN(parsed.getTime()) ? String(value) : parsed.toLocaleString();
  }

  function canCancelRun(run: WorkItemRun) {
    return run.status === "queued" || run.status === "running" || run.status === "awaiting_input";
  }

  function startRun(item: WorkItem, preset: string, promptTemplateId: string) {
    if (loading) return;
    onStartRun({ workItemId: item.id, preset, promptTemplateId, agentProfileId });
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

  async function attachFile(item: WorkItem) {
    const selected = await Dialogs.OpenFile({
      Title: "Attach file",
      ButtonText: "Attach",
      Directory: activeProject?.rootDir || undefined,
      CanChooseDirectories: false,
      CanChooseFiles: true,
      AllowsMultipleSelection: false,
    });
    if (typeof selected === "string" && selected.length > 0) {
      onAttachFile(item.id, selected);
    }
  }

  function deleteWorkItem(item: WorkItem) {
    if (loading) return;
    if (window.confirm(`Delete #${item.number} ${item.title}?`)) {
      onDeleteWorkItem(item.id);
      if (detailItemId === item.id) detailItemId = "";
    }
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
  }

  function handleKey(event: KeyboardEvent) {
    if (event.key === "Escape" && detailItemId) {
      event.preventDefault();
      closeDetail();
    }
  }
</script>

<svelte:window on:keydown={handleKey} />

<div class="flex min-h-0 flex-1 flex-col bg-bg-deep">
  <div class="flex h-12 shrink-0 items-center justify-between gap-3 border-b border-hairline bg-bg-base/80 px-3">
    <div class="flex min-w-0 items-center gap-2">
      <select
        class="h-8 max-w-[280px] rounded border border-border bg-bg-surface px-2 text-[12px] text-text-secondary outline-none focus:border-accent-dim"
        value={activeProjectId}
        disabled={loading || projects.length === 0}
        on:change={(event) => onSelectProject(event.currentTarget.value)}
      >
        {#if projects.length === 0}
          <option value="">No projects</option>
        {/if}
        {#each projects as project (project.id)}
          <option value={project.id}>{project.name}</option>
        {/each}
      </select>
      {#if activeProject}
        <div class="hidden min-w-0 sm:block">
          <div class="truncate text-[12px] font-semibold text-text-primary">
            {activeProject.name}
          </div>
          <div class="truncate font-mono text-[10px] text-text-muted">
            {activeProject.rootDir}
          </div>
        </div>
      {/if}
    </div>
    <div class="flex shrink-0 items-center gap-1">
      <button
        type="button"
        class="inline-flex h-8 w-8 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-wait disabled:opacity-60"
        disabled={loading}
        aria-label="Refresh work board"
        title="Refresh work board"
        on:click={onRefresh}
      >
        <RefreshCw size={14} class={loading ? "animate-spin" : ""} />
      </button>
      <button
        type="button"
        class="inline-flex h-8 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[12px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
        disabled={loading}
        on:click={onNewProject}
      >
        <Plus size={14} />
        <span>Project</span>
      </button>
    </div>
  </div>

  {#if activeProject}
    <div class="grid shrink-0 gap-2 border-b border-hairline p-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.6fr)_auto]">
      <input
        class="h-9 rounded border border-border bg-bg-deep px-2.5 text-[13px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
        type="text"
        bind:value={newItemTitle}
        placeholder="Work item title"
        disabled={loading}
      />
      <input
        class="h-9 rounded border border-border bg-bg-deep px-2.5 text-[13px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
        type="text"
        bind:value={newItemBody}
        placeholder="Markdown body"
        disabled={loading}
      />
      <button
        type="button"
        class="inline-flex h-9 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-3 text-[13px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
        disabled={loading || !newItemTitle.trim()}
        on:click={createWorkItem}
      >
        <Plus size={14} />
        <span>Create</span>
      </button>
    </div>

    <div class="app-scrollbar min-h-0 flex-1 overflow-auto p-3">
      <div
        class="grid min-h-full gap-3"
        style="grid-template-columns: repeat({Math.max(stages.length, 1)}, minmax(260px, 1fr));"
      >
        {#each stages as stage (stage.id)}
          {@const stageItems = itemsByStage[stage.id] ?? []}
          <section class="flex min-h-[360px] flex-col rounded border border-border-subtle/70 bg-bg-surface/20">
            <div class="flex h-10 shrink-0 items-center justify-between border-b border-border-subtle/60 px-3">
              <div class="truncate text-[12px] font-semibold text-text-primary">
                {stage.name}
              </div>
              <div class="rounded bg-bg-deep px-1.5 py-0.5 font-mono text-[10px] text-text-muted">
                {stageItems.length}
              </div>
            </div>
            <div class="grid gap-2 p-2">
              {#if stageItems.length === 0}
                <div class="rounded border border-dashed border-border-subtle/70 px-3 py-6 text-center text-[12px] text-text-muted">
                  Empty
                </div>
              {:else}
                {#each stageItems as item (item.id)}
                  {@const targets = adjacentStageTargets(item, stages)}
                  {@const itemRuns = runsByItem[item.id] ?? []}
                  {@const latestRun = itemRuns[0]}
                  <article class="rounded border border-border-subtle bg-bg-deep/90 p-2">
                    <div class="flex min-w-0 items-start justify-between gap-2">
                      <button
                        type="button"
                        class="min-w-0 flex-1 truncate text-left text-[13px] font-semibold text-text-primary transition-colors hover:text-accent"
                        on:click={() => (detailItemId = item.id)}
                      >
                        #{item.number} {item.title}
                      </button>
                      <div class="flex shrink-0 gap-1">
                        <button
                          type="button"
                          aria-label="Move previous"
                          title="Move previous"
                          class="inline-flex h-6 w-6 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-40"
                          disabled={loading || !targets.previous}
                          on:click={() => movePrevious(item)}
                        >
                          <ArrowLeft size={12} />
                        </button>
                        <button
                          type="button"
                          aria-label={targets.blockedNext ? "Generate worktree before moving" : "Move next"}
                          title={targets.blockedNext ? "Generate worktree before moving" : "Move next"}
                          class="inline-flex h-6 w-6 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-40"
                          disabled={loading || !targets.next}
                          on:click={() => moveNext(item)}
                        >
                          <ArrowRight size={12} />
                        </button>
                      </div>
                    </div>
                    <div class="mt-1 flex min-w-0 gap-1 text-[10px] text-text-muted">
                      <span class="shrink-0">{latestRun?.status || item.runState || "idle"}</span>
                      {#if item.worktree}
                        <span class="min-w-0 truncate font-mono">{item.worktree.branch}</span>
                      {/if}
                    </div>
                    {#if item.worktree}
                      <div class="mt-2 truncate font-mono text-[10px] text-text-muted">
                        {item.worktree.worktreePath}
                      </div>
                    {:else}
                      <div class="mt-2 grid grid-cols-[minmax(0,1fr)_28px] gap-1">
                        <input
                          class="h-7 min-w-0 rounded border border-border bg-bg-surface px-1.5 font-mono text-[10px] text-text-secondary outline-none placeholder:text-text-muted focus:border-accent-dim"
                          type="text"
                          value={worktreeBranches[item.id] || defaultWorktreeBranch(item)}
                          disabled={loading}
                          aria-label="Worktree branch"
                          on:input={(event) =>
                            (worktreeBranches = {
                              ...worktreeBranches,
                              [item.id]: event.currentTarget.value,
                            })}
                        />
                        <button
                          type="button"
                          aria-label="Generate worktree"
                          title="Generate worktree"
                          class="inline-flex h-7 w-7 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-accent disabled:cursor-not-allowed"
                          disabled={loading}
                          on:click={() => generateWorktree(item)}
                        >
                          <GitBranch size={12} />
                        </button>
                      </div>
                    {/if}
                  </article>
                {/each}
              {/if}
            </div>
          </section>
        {/each}
      </div>
    </div>
  {:else}
    <div class="flex min-h-0 flex-1 items-center justify-center p-6">
      <div class="grid max-w-sm gap-3 text-center">
        <div class="text-base font-semibold text-text-primary">No project selected</div>
        <button
          type="button"
          class="inline-flex h-9 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-3 text-[13px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent"
          on:click={onNewProject}
        >
          <Plus size={14} />
          <span>New project</span>
        </button>
      </div>
    </div>
  {/if}
</div>

{#if detailItem && activeProject}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/45 px-4"
    role="dialog"
    aria-modal="true"
    aria-label="Work item detail"
  >
    <div class="flex max-h-[88vh] w-full max-w-[760px] flex-col rounded-lg border border-border bg-bg-base shadow-[0_24px_80px_rgba(0,0,0,0.45)]">
      <div class="flex h-11 shrink-0 items-center justify-between border-b border-hairline px-4">
        <div class="min-w-0">
          <div class="truncate text-sm font-semibold text-text-primary">
            #{detailItem.number} {detailItem.title}
          </div>
          <div class="truncate font-mono text-[10px] text-text-muted">
            {activeProject.name}
          </div>
        </div>
        <button
          type="button"
          aria-label="Close"
          class="inline-flex h-7 w-7 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
          on:click={closeDetail}
        >
          <X size={14} />
        </button>
      </div>

      <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto px-4 py-4">
        <div class="grid gap-4">
          <section class="grid gap-2">
            <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">
              State
            </div>
            <div class="grid gap-2 sm:grid-cols-[180px_minmax(0,1fr)]">
              <select
                class="h-8 rounded border border-border bg-bg-surface px-2 text-[12px] text-text-secondary outline-none focus:border-accent-dim"
                value={detailItem.stageId}
                disabled={loading}
                on:change={(event) => onMoveWorkItem(detailItem.id, event.currentTarget.value)}
              >
                {#each stages as targetStage (targetStage.id)}
                  <option
                    value={targetStage.id}
                    disabled={!canMoveToStage(detailItem, targetStage)}
                  >
                    {targetStage.name}{!canMoveToStage(detailItem, targetStage)
                      ? " - generate worktree first"
                      : ""}
                  </option>
                {/each}
              </select>
              <div class="min-w-0 rounded border border-border-subtle bg-bg-surface/25 px-2 py-1.5">
                {#if detailItem.worktree}
                  <div class="truncate font-mono text-[11px] text-text-secondary">
                    {detailItem.worktree.branch}
                  </div>
                  <div class="truncate font-mono text-[10px] text-text-muted">
                    {detailItem.worktree.worktreePath}
                  </div>
                {:else}
                  <div class="grid grid-cols-[minmax(0,1fr)_32px] gap-1.5">
                    <input
                      class="h-8 min-w-0 rounded border border-border bg-bg-deep px-2 font-mono text-[11px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                      type="text"
                      value={worktreeBranches[detailItem.id] || defaultWorktreeBranch(detailItem)}
                      disabled={loading}
                      aria-label="Worktree branch"
                      on:input={(event) =>
                        (worktreeBranches = {
                          ...worktreeBranches,
                          [detailItem.id]: event.currentTarget.value,
                        })}
                    />
                    <button
                      type="button"
                      aria-label="Generate worktree"
                      title="Generate worktree"
                      class="inline-flex h-8 w-8 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-accent disabled:cursor-not-allowed"
                      disabled={loading}
                      on:click={() => generateWorktree(detailItem)}
                    >
                      <GitBranch size={13} />
                    </button>
                  </div>
                {/if}
              </div>
            </div>
          </section>

          <section class="grid gap-2">
            <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">
              Run
            </div>
            <div class="grid gap-1">
              <label class="text-[11px] font-medium text-text-muted" for="agent-profile">
                Agent
              </label>
              <select
                id="agent-profile"
                class="h-8 rounded border border-border bg-bg-surface px-2 text-[12px] text-text-secondary outline-none focus:border-accent-dim"
                bind:value={agentProfileId}
                disabled={loading}
              >
                <option value="codex">codex</option>
                <option value="claude">claude</option>
                <option value="claude-plan">claude-plan</option>
                <option value="prompt-capture">prompt-capture</option>
              </select>
            </div>
            <div class="grid gap-2 sm:grid-cols-3">
              <button
                type="button"
                class="inline-flex h-8 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[12px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-60"
                disabled={loading}
                on:click={() => startRun(detailItem, "manager", "plan")}
              >
                <ClipboardCheck size={13} />
                <span>Plan</span>
              </button>
              <button
                type="button"
                class="inline-flex h-8 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[12px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-60"
                disabled={loading}
                on:click={() => startRun(detailItem, "writer", "implement")}
              >
                <Play size={13} />
                <span>Implement</span>
              </button>
              <button
                type="button"
                class="inline-flex h-8 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[12px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-60"
                disabled={loading}
                on:click={() => startRun(detailItem, "reviewer", "review")}
              >
                <Search size={13} />
                <span>Review</span>
              </button>
            </div>
          </section>

          <section class="grid gap-2">
            <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">
              Body
            </div>
            <div class="min-h-28 whitespace-pre-wrap rounded border border-border-subtle bg-bg-deep px-3 py-2 text-[12px] leading-5 text-text-secondary">
              {detailItem.bodyMarkdown || "No body."}
            </div>
          </section>

          <section class="grid gap-2">
            <div class="flex items-center justify-between">
              <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">
                Attachments
              </div>
              <button
                type="button"
                class="inline-flex h-7 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[11px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-not-allowed"
                disabled={loading}
                on:click={() => attachFile(detailItem)}
              >
                <Paperclip size={12} />
                <span>Attach</span>
              </button>
            </div>
            {#if detailItem.attachments.length > 0}
              <div class="grid gap-1">
                {#each detailItem.attachments as attachment (attachment.id)}
                  <div class="truncate rounded border border-border-subtle bg-bg-surface/25 px-2 py-1.5 font-mono text-[11px] text-text-secondary">
                    {attachment.path || attachment.url || attachment.note}
                  </div>
                {/each}
              </div>
            {:else}
              <div class="rounded border border-border-subtle bg-bg-surface/25 px-2 py-2 text-[12px] text-text-muted">
                No attachments.
              </div>
            {/if}
          </section>

          <section class="grid gap-2">
            <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">
              Runs
            </div>
            {#if detailRuns.length > 0}
              <div class="grid gap-2">
                {#each detailRuns as run (run.id)}
                  <div class="rounded border border-border-subtle bg-bg-surface/25">
                    <div class="flex items-center justify-between gap-2 border-b border-border-subtle/60 px-2 py-1.5">
                      <div class="min-w-0">
                        <div class="flex min-w-0 items-center gap-2 text-[11px] text-text-secondary">
                          <span class="shrink-0 font-semibold">{run.status}</span>
                          <span class="shrink-0 font-mono text-text-muted">{run.preset}</span>
                          <span class="min-w-0 truncate font-mono text-text-muted">{run.promptTemplateId}</span>
                        </div>
                        <div class="truncate font-mono text-[10px] text-text-muted">
                          {formattedTime(run.createdAt)}
                        </div>
                      </div>
                      {#if canCancelRun(run)}
                        <button
                          type="button"
                          aria-label="Cancel run"
                          title="Cancel run"
                          class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded border border-border-subtle bg-bg-deep/80 text-text-secondary transition-colors hover:border-red hover:text-red disabled:cursor-not-allowed disabled:opacity-60"
                          disabled={loading}
                          on:click={() => onCancelRun(run.id)}
                        >
                          <Square size={12} />
                        </button>
                      {/if}
                    </div>
                    {#if run.promptSnapshot}
                      <pre class="app-scrollbar max-h-40 overflow-auto whitespace-pre-wrap px-2 py-2 font-mono text-[10px] leading-4 text-text-muted">{run.promptSnapshot}</pre>
                    {/if}
                  </div>
                {/each}
              </div>
            {:else}
              <div class="rounded border border-border-subtle bg-bg-surface/25 px-2 py-2 text-[12px] text-text-muted">
                No runs.
              </div>
            {/if}
          </section>

          <section class="grid gap-2">
            <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">
              History
            </div>
            <div class="grid gap-1">
              {#each detailItem.history as event (event.id)}
                <div class="rounded border border-border-subtle bg-bg-surface/25 px-2 py-1.5">
                  <div class="text-[11px] text-text-secondary">
                    {event.type}
                    {#if event.stageId}
                      <span class="font-mono text-text-muted"> {event.stageId}</span>
                    {/if}
                  </div>
                  <div class="font-mono text-[10px] text-text-muted">
                    {event.at?.toLocaleString?.() ?? event.at}
                  </div>
                </div>
              {/each}
            </div>
          </section>
        </div>
      </div>

      <div class="flex shrink-0 justify-between gap-2 border-t border-hairline px-4 py-3">
        <button
          type="button"
          class="inline-flex h-8 items-center justify-center gap-1 rounded border border-red/30 bg-red/10 px-3 text-[12px] text-red transition-colors hover:border-red disabled:cursor-not-allowed"
          disabled={loading}
          on:click={() => deleteWorkItem(detailItem)}
        >
          <Trash2 size={13} />
          <span>Delete</span>
        </button>
        <button
          type="button"
          class="rounded border border-border-subtle bg-bg-surface/40 px-3 py-1.5 text-[12px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
          on:click={closeDetail}
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}
