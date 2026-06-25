<script lang="ts">
  import ListFilter from "@lucide/svelte/icons/list-filter";
  import Plus from "@lucide/svelte/icons/plus";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Search from "@lucide/svelte/icons/search";
  import type { WorkflowStage } from "../bindings/github.com/phin-tech/whisk/internal/domain/workitem/models";
  import type { Project, WorkItem } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import SidebarPanelHeader from "./SidebarPanelHeader.svelte";

  export let projects: Project[] = [];
  export let workItems: WorkItem[] = [];
  export let activeProjectId = "";
  export let filterQuery = "";
  export let filterStageId = "";
  export let filterRunState = "";
  export let loading = false;
  export let onclose: () => void;
  export let onRefresh: () => void;
  export let onSelectProject: (projectId: string) => void;
  export let onCreateWorkItem: (request: {
    projectId: string;
    title: string;
    bodyMarkdown: string;
  }) => void;

  let newItemTitle = "";
  let newItemBody = "";

  $: activeProject = projects.find((project) => project.id === activeProjectId) ?? null;
  $: stages = activeProject?.workflow?.stages ?? [];
  $: runStates = Array.from(new Set(workItems.map((item) => item.runState || "idle"))).sort();
  $: filteredCount = workItems.filter((item) => {
    const query = filterQuery.trim().toLowerCase();
    if (filterStageId && item.stageId !== filterStageId) return false;
    if (filterRunState && (item.runState || "idle") !== filterRunState) return false;
    if (!query) return true;
    return `#${item.number} ${item.title} ${item.bodyMarkdown} ${item.stageId} ${item.runState}`
      .toLowerCase()
      .includes(query);
  }).length;

  function stageCount(stage: WorkflowStage) {
    return workItems.filter((item) => item.stageId === stage.id).length;
  }

  function runStateCount(runState: string) {
    return workItems.filter((item) => (item.runState || "idle") === runState).length;
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

  function clearFilters() {
    filterQuery = "";
    filterStageId = "";
    filterRunState = "";
  }
</script>

<div class="flex h-full min-h-0 w-full flex-col bg-bg-deep">
  <SidebarPanelHeader title="Board" {onclose}>
    <div slot="actions" class="flex items-center gap-1">
      <button
        type="button"
        class="inline-flex h-6 w-6 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-wait disabled:opacity-60"
        disabled={loading}
        aria-label="Refresh board"
        title="Refresh board"
        on:click={onRefresh}
      >
        <RefreshCw size={13} class={loading ? "animate-spin" : ""} />
      </button>
    </div>
  </SidebarPanelHeader>

  <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
    <section class="space-y-2 border-b border-hairline pb-3">
      <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">Scope</div>
      {#if projects.length > 0}
        <select
          class="h-8 w-full rounded border border-border-subtle bg-bg-surface/50 px-2 text-[12px] text-text-primary outline-none focus:border-accent-dim"
          value={activeProjectId}
          disabled={loading}
          aria-label="Board project"
          on:change={(event) => onSelectProject(event.currentTarget.value)}
        >
          {#each projects as project (project.id)}
            <option value={project.id}>{project.name}</option>
          {/each}
        </select>
        {#if activeProject}
          <div class="truncate font-mono text-[10px] text-text-muted">{activeProject.rootDir}</div>
        {/if}
      {:else}
        <div class="text-[12px] text-text-muted">No projects.</div>
      {/if}
    </section>

    {#if activeProject}
      <section class="space-y-2 border-b border-hairline py-3">
        <div class="flex items-center justify-between gap-2">
          <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">Filters</div>
          <button
            type="button"
            class="text-[11px] text-text-muted transition-colors hover:text-text-primary"
            on:click={clearFilters}
          >
            Clear
          </button>
        </div>

        <label class="grid h-8 grid-cols-[14px_minmax(0,1fr)] items-center gap-2 rounded border border-border-subtle bg-bg-surface/35 px-2 text-text-muted focus-within:border-accent-dim">
          <Search size={14} />
          <input
            class="min-w-0 bg-transparent text-[12px] text-text-primary outline-none placeholder:text-text-muted"
            bind:value={filterQuery}
            placeholder="Search cards"
            aria-label="Search cards"
          />
        </label>

        <select
          class="h-8 w-full rounded border border-border-subtle bg-bg-surface/50 px-2 text-[12px] text-text-primary outline-none focus:border-accent-dim"
          bind:value={filterStageId}
          aria-label="Stage filter"
        >
          <option value="">All stages</option>
          {#each stages as stage (stage.id)}
            <option value={stage.id}>{stage.name} ({stageCount(stage)})</option>
          {/each}
        </select>

        <select
          class="h-8 w-full rounded border border-border-subtle bg-bg-surface/50 px-2 text-[12px] text-text-primary outline-none focus:border-accent-dim"
          bind:value={filterRunState}
          aria-label="Run state filter"
        >
          <option value="">All run states</option>
          {#each runStates as runState (runState)}
            <option value={runState}>{runState} ({runStateCount(runState)})</option>
          {/each}
        </select>

        <div class="flex items-center gap-2 rounded border border-border-subtle bg-bg-surface/25 px-2 py-1.5 text-[11px] text-text-muted">
          <ListFilter size={13} />
          <span>{filteredCount} of {workItems.length} cards visible</span>
        </div>
      </section>

      <section class="space-y-2 pt-3">
        <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">Quick add</div>
        <input
          class="h-8 w-full rounded border border-border bg-bg-deep px-2 text-[12px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
          type="text"
          bind:value={newItemTitle}
          placeholder="Card title"
          disabled={loading}
          on:keydown={(event) => {
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault();
              createWorkItem();
            }
          }}
        />
        <textarea
          class="min-h-16 w-full resize-none rounded border border-border bg-bg-deep px-2 py-1.5 text-[12px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
          bind:value={newItemBody}
          placeholder="Notes"
          disabled={loading}
        ></textarea>
        <button
          type="button"
          class="inline-flex h-8 w-full items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[12px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
          disabled={loading || !newItemTitle.trim()}
          on:click={createWorkItem}
        >
          <Plus size={13} />
          <span>Create card</span>
        </button>
      </section>
    {:else}
      <div class="flex min-h-[180px] items-center justify-center px-4 text-center text-[13px] text-text-muted">
        Select a project.
      </div>
    {/if}
  </div>
</div>
