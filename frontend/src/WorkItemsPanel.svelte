<script lang="ts">
  import ListFilter from "@lucide/svelte/icons/list-filter";
  import Plus from "@lucide/svelte/icons/plus";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Search from "@lucide/svelte/icons/search";
  import type { WorkflowStage } from "../bindings/github.com/phin-tech/whisk/internal/domain/workitem/models";
  import type { Project, WorkItem } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import SidebarPanelHeader from "./SidebarPanelHeader.svelte";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import SelectField from "./ui/SelectField.svelte";
  import TextArea from "./ui/TextArea.svelte";
  import TextField from "./ui/TextField.svelte";

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
  $: projectOptions = projects.map((project) => ({ value: project.id, label: project.name }));
  $: stageOptions = [
    { value: "", label: "All stages" },
    ...stages.map((stage) => ({ value: stage.id, label: `${stage.name} (${stageCount(stage)})` })),
  ];
  $: runStateOptions = [
    { value: "", label: "All run states" },
    ...runStates.map((runState) => ({ value: runState, label: `${runState} (${runStateCount(runState)})` })),
  ];
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

  function submitOnEnter(event: KeyboardEvent) {
    if (event.key !== "Enter" || event.shiftKey) return;
    event.preventDefault();
    createWorkItem();
  }
</script>

<div class="flex h-full min-h-0 w-full flex-col bg-bg-deep">
  <SidebarPanelHeader title="Board" {onclose}>
    <div slot="actions" class="flex items-center gap-1">
      <IconButton
        disabled={loading}
        label="Refresh board"
        size="sm"
        onclick={onRefresh}
      >
        <RefreshCw size={13} class={loading ? "animate-spin" : ""} />
      </IconButton>
    </div>
  </SidebarPanelHeader>

  <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
    <section class="space-y-2 border-b border-hairline pb-3">
      <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">Scope</div>
      {#if projects.length > 0}
        <SelectField
          value={activeProjectId}
          label="Board project"
          options={projectOptions}
          disabled={loading}
          class="w-full border-border-subtle bg-bg-surface/50"
          onValueChange={onSelectProject}
        />
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
          <Button variant="ghost" size="sm" class="h-auto border-transparent bg-transparent px-0 text-[11px] text-text-muted hover:bg-transparent hover:text-text-primary" onclick={clearFilters}>
            Clear
          </Button>
        </div>

        <div class="grid h-8 grid-cols-[14px_minmax(0,1fr)] items-center gap-2 rounded border border-border-subtle bg-bg-surface/35 px-2 text-text-muted focus-within:border-accent-dim">
          <Search size={14} />
          <TextField
            variant="seamless"
            bind:value={filterQuery}
            placeholder="Search cards"
            aria-label="Search cards"
            class="min-w-0 border-transparent bg-transparent p-0 text-[12px]"
          />
        </div>

        <SelectField
          bind:value={filterStageId}
          label="Stage filter"
          options={stageOptions}
          class="w-full border-border-subtle bg-bg-surface/50"
        />

        <SelectField
          bind:value={filterRunState}
          label="Run state filter"
          options={runStateOptions}
          class="w-full border-border-subtle bg-bg-surface/50"
        />

        <div class="flex items-center gap-2 rounded border border-border-subtle bg-bg-surface/25 px-2 py-1.5 text-[11px] text-text-muted">
          <ListFilter size={13} />
          <span>{filteredCount} of {workItems.length} cards visible</span>
        </div>
      </section>

      <section class="space-y-2 pt-3">
        <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">Quick add</div>
        <TextField
          bind:value={newItemTitle}
          placeholder="Card title"
          disabled={loading}
          onkeydown={submitOnEnter}
        />
        <TextArea
          bind:value={newItemBody}
          placeholder="Notes"
          disabled={loading}
          class="min-h-16 resize-none py-1.5"
        />
        <Button
          class="w-full"
          disabled={loading || !newItemTitle.trim()}
          onclick={createWorkItem}
        >
          <Plus size={13} />
          <span>Create card</span>
        </Button>
      </section>
    {:else}
      <div class="flex min-h-[180px] items-center justify-center px-4 text-center text-[13px] text-text-muted">
        Select a project.
      </div>
    {/if}
  </div>
</div>
