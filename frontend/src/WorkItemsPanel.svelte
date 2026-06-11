<script lang="ts">
  import { Dialogs } from "@wailsio/runtime";
  import GitBranch from "@lucide/svelte/icons/git-branch";
  import Paperclip from "@lucide/svelte/icons/paperclip";
  import Plus from "@lucide/svelte/icons/plus";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import X from "@lucide/svelte/icons/x";
  import type { WorkflowStage } from "../bindings/github.com/phin-tech/whisk/internal/domain/workitem/models";
  import type { Project, WorkItem } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import SidebarPanelHeader from "./SidebarPanelHeader.svelte";

  export let projects: Project[] = [];
  export let workItems: WorkItem[] = [];
  export let activeProjectId = "";
  export let loading = false;
  export let onclose: () => void;
  export let onRefresh: () => void;
  export let onNewProject: () => void;
  export let onSelectProject: (projectId: string) => void;
  export let onCreateWorkItem: (request: {
    projectId: string;
    title: string;
    bodyMarkdown: string;
  }) => void;
  export let onMoveWorkItem: (workItemId: string, stageId: string) => void;
  export let onGenerateWorktree: (request: {
    workItemId: string;
    branch: string;
  }) => void;
  export let onAttachFile: (workItemId: string, path: string) => void;
  export let onDeleteWorkItem: (workItemId: string) => void;

  let newItemTitle = "";
  let newItemBody = "";
  let worktreeBranches: Record<string, string> = {};
  let detailItemId = "";

  $: activeProject = projects.find((project) => project.id === activeProjectId) ?? null;
  $: stages = activeProject?.workflow?.stages ?? [];
  $: itemsByStage = groupByStage(workItems, stages);
  $: detailItem = workItems.find((item) => item.id === detailItemId) ?? null;

  function groupByStage(items: WorkItem[], workflowStages: WorkflowStage[]) {
    const result: Record<string, WorkItem[]> = {};
    for (const stage of workflowStages) result[stage.id] = [];
    for (const item of items) {
      if (!result[item.stageId]) result[item.stageId] = [];
      result[item.stageId].push(item);
    }
    return result;
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
    }
  }

  function stageDisabledForItem(item: WorkItem, stage: WorkflowStage) {
    return stage.provisionWorktree && !item.worktree;
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

<div class="flex h-full min-h-0 w-full flex-col bg-bg-deep">
  <SidebarPanelHeader title="Work" {onclose}>
    <div slot="actions" class="flex items-center gap-1">
      <button
        type="button"
        class="inline-flex h-6 w-6 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-not-allowed disabled:opacity-60"
        disabled={loading}
        aria-label="New project"
        title="New project"
        on:click={onNewProject}
      >
        <Plus size={13} />
      </button>
      <button
        type="button"
        class="inline-flex h-6 w-6 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-wait disabled:opacity-60"
        disabled={loading}
        aria-label="Refresh work items"
        title="Refresh work items"
        on:click={onRefresh}
      >
        <RefreshCw size={13} class={loading ? "animate-spin" : ""} />
      </button>
    </div>
  </SidebarPanelHeader>

  <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
    <section class="space-y-2 border-b border-hairline pb-3">
      <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">
        Projects
      </div>
      {#if projects.length > 0}
        <div class="flex flex-wrap gap-1">
          {#each projects as project (project.id)}
            <button
              type="button"
              class="max-w-full truncate rounded border px-2 py-1 text-[11px] transition-colors {project.id ===
              activeProjectId
                ? 'border-accent-dim bg-accent-dim text-text-primary'
                : 'border-border-subtle bg-bg-surface/35 text-text-secondary hover:bg-bg-hover hover:text-text-primary'}"
              on:click={() => onSelectProject(project.id)}
            >
              {project.name}
            </button>
          {/each}
        </div>
      {:else}
        <div class="grid gap-2">
          <div class="text-[12px] text-text-muted">No projects.</div>
          <button
            type="button"
            class="inline-flex h-8 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[12px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
            disabled={loading}
            on:click={onNewProject}
          >
            <Plus size={13} />
            <span>New project</span>
          </button>
        </div>
      {/if}
      {#if projects.length > 0}
        <button
          type="button"
          class="inline-flex h-8 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[12px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
          disabled={loading}
          on:click={onNewProject}
        >
          <Plus size={13} />
          <span>New project</span>
        </button>
      {/if}
    </section>

    {#if activeProject}
      <section class="space-y-2 border-b border-hairline py-3">
        <div class="min-w-0">
          <div class="truncate text-[13px] font-semibold text-text-primary">
            {activeProject.name}
          </div>
          <div class="truncate font-mono text-[10px] text-text-muted">
            {activeProject.rootDir}
          </div>
        </div>
        <div class="grid gap-1.5">
          <input
            class="h-8 rounded border border-border bg-bg-deep px-2 text-[12px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
            type="text"
            bind:value={newItemTitle}
            placeholder="Work item title"
            disabled={loading}
          />
          <textarea
            class="min-h-16 resize-none rounded border border-border bg-bg-deep px-2 py-1.5 text-[12px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
            bind:value={newItemBody}
            placeholder="Markdown body"
            disabled={loading}
          ></textarea>
          <button
            type="button"
            class="inline-flex h-8 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[12px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
            disabled={loading || !newItemTitle.trim()}
            on:click={createWorkItem}
          >
            <Plus size={13} />
            <span>Create item</span>
          </button>
        </div>
      </section>

      <section class="space-y-2 pt-3">
        {#each stages as stage (stage.id)}
          {@const stageItems = itemsByStage[stage.id] ?? []}
          <div class="rounded border border-border-subtle/70 bg-bg-surface/25">
            <div class="flex items-center justify-between border-b border-border-subtle/60 px-2 py-1.5">
              <div class="truncate text-[12px] font-semibold text-text-primary">
                {stage.name}
              </div>
              <div class="font-mono text-[10px] text-text-muted">{stageItems.length}</div>
            </div>
            <div class="grid gap-1 p-1.5">
              {#if stageItems.length === 0}
                <div class="px-1 py-2 text-center text-[11px] text-text-muted">
                  Empty
                </div>
              {:else}
                {#each stageItems as item (item.id)}
                  <div class="rounded border border-border-subtle bg-bg-deep/80 px-2 py-1.5">
                    <div class="flex min-w-0 items-start gap-2">
                      <div class="min-w-0 flex-1">
                        <button
                          type="button"
                          class="block max-w-full truncate text-left text-[12px] font-medium text-text-primary transition-colors hover:text-accent"
                          on:click={() => (detailItemId = item.id)}
                        >
                          #{item.number} {item.title}
                        </button>
                        <div class="mt-0.5 flex min-w-0 gap-1 text-[10px] text-text-muted">
                          <span class="shrink-0">{item.runState || "idle"}</span>
                          {#if item.worktree}
                            <span class="min-w-0 truncate font-mono">{item.worktree.branch}</span>
                          {/if}
                        </div>
                      </div>
                      <select
                        class="h-6 shrink-0 rounded border border-border bg-bg-surface px-1 text-[10px] text-text-secondary outline-none focus:border-accent-dim"
                        value={item.stageId}
                        disabled={loading}
                        on:change={(event) =>
                          onMoveWorkItem(item.id, event.currentTarget.value)}
                      >
                        {#each stages as targetStage (targetStage.id)}
                          <option
                            value={targetStage.id}
                            disabled={stageDisabledForItem(item, targetStage)}
                          >
                            {targetStage.name}{stageDisabledForItem(item, targetStage)
                              ? " - bind worktree first"
                              : ""}
                          </option>
                        {/each}
                      </select>
                    </div>
                    <div class="mt-1.5 flex items-center justify-between gap-1">
                      <div class="min-w-0 flex-1">
                        {#if item.worktree}
                          <div class="truncate font-mono text-[10px] text-text-muted">
                            {item.worktree.worktreePath}
                          </div>
                        {:else}
                          <div class="mb-1 flex items-center gap-1 text-[10px] text-text-muted">
                            <GitBranch size={11} />
                            <span>Bind a worktree before moving to Ready.</span>
                          </div>
                          <div class="grid grid-cols-[minmax(0,1fr)_28px] gap-1">
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
                      </div>
                      <div class="flex shrink-0 gap-1">
                        <button
                          type="button"
                          aria-label="Attach file"
                          title="Attach file"
                          class="inline-flex h-7 w-7 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-not-allowed"
                          disabled={loading}
                          on:click={() => attachFile(item)}
                        >
                          <Paperclip size={12} />
                        </button>
                        <button
                          type="button"
                          aria-label="Delete work item"
                          title="Delete work item"
                          class="inline-flex h-7 w-7 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:border-red/40 hover:bg-red/10 hover:text-red disabled:cursor-not-allowed"
                          disabled={loading}
                          on:click={() => deleteWorkItem(item)}
                        >
                          <Trash2 size={12} />
                        </button>
                      </div>
                    </div>
                  </div>
                {/each}
              {/if}
            </div>
          </div>
        {/each}
      </section>
    {:else}
      <div class="flex min-h-[180px] items-center justify-center px-4 text-center text-sm text-text-muted">
        Create or select a project.
      </div>
    {/if}
  </div>
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
                    disabled={stageDisabledForItem(detailItem, targetStage)}
                  >
                    {targetStage.name}{stageDisabledForItem(detailItem, targetStage)
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
          on:click={() => {
            deleteWorkItem(detailItem);
            closeDetail();
          }}
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
