<script lang="ts">
  import Plus from "@lucide/svelte/icons/plus";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Search from "@lucide/svelte/icons/search";
  import type { Project } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import SidebarPanelHeader from "./SidebarPanelHeader.svelte";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import List from "./ui/List.svelte";
  import ListRow from "./ui/ListRow.svelte";
  import TextField from "./ui/TextField.svelte";

  export let projects: Project[] = [];
  export let activeProjectId = "";
  export let loading = false;
  export let onclose: () => void;
  export let onRefresh: () => void;
  export let onNewProject: () => void;
  export let onSelectProject: (projectId: string) => void;

  let query = "";
  $: filteredProjects = projects.filter((project) => {
    const needle = query.trim().toLowerCase();
    if (!needle) return true;
    return `${project.name} ${project.slug} ${project.rootDir}`.toLowerCase().includes(needle);
  });

  function projectInitial(project: Project) {
    return (project.name || project.slug || project.id || "?").slice(0, 1).toUpperCase();
  }
</script>

<div class="flex h-full min-h-0 w-full flex-col bg-bg-deep">
  <SidebarPanelHeader title="Projects" {onclose}>
    <div slot="actions" class="flex items-center gap-1">
      <IconButton
        disabled={loading}
        label="New project"
        size="sm"
        onclick={onNewProject}
      >
        <Plus size={13} />
      </IconButton>
      <IconButton
        disabled={loading}
        label="Refresh projects"
        size="sm"
        onclick={onRefresh}
      >
        <RefreshCw size={13} class={loading ? "animate-spin" : ""} />
      </IconButton>
    </div>
  </SidebarPanelHeader>

  <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
    {#if projects.length === 0}
      <Button
        size="lg"
        class="w-full"
        disabled={loading}
        onclick={onNewProject}
      >
        <Plus size={13} />
        <span>New project</span>
      </Button>
    {:else}
      <div class="mb-2 grid h-8 grid-cols-[14px_minmax(0,1fr)] items-center gap-2 rounded border border-border-subtle bg-bg-surface/35 px-2 text-text-muted focus-within:border-accent-dim">
        <Search size={14} />
        <TextField
          variant="seamless"
          bind:value={query}
          placeholder="Search projects"
          aria-label="Search projects"
          class="min-w-0 border-transparent bg-transparent p-0 text-[12px]"
        />
      </div>
      <List class="grid gap-1 divide-y-0">
        {#each filteredProjects as project (project.id)}
          <ListRow
            as="button"
            class="flex min-h-12 items-center gap-2 rounded border px-2 py-1.5 {project.id ===
            activeProjectId
              ? 'border-accent-dim bg-accent-dim/20 text-text-primary'
              : 'border-transparent bg-transparent text-text-secondary hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary'}"
            onclick={() => onSelectProject(project.id)}
          >
            <span
              class="flex h-7 w-7 shrink-0 items-center justify-center rounded border border-border-subtle bg-bg-surface font-mono text-[11px] text-accent"
              aria-hidden="true"
            >
              {projectInitial(project)}
            </span>
            <span class="min-w-0 flex-1">
              <span class="block truncate text-[12px] font-semibold">{project.name}</span>
              <span class="block truncate font-mono text-[10px] text-text-muted">
                {project.rootDir}
              </span>
            </span>
          </ListRow>
        {/each}
        {#if filteredProjects.length === 0}
          <div class="px-2 py-3 text-[12px] text-text-muted">No matching projects.</div>
        {/if}
      </List>
    {/if}
  </div>
</div>
