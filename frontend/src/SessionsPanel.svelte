<script lang="ts">
  import Check from "@lucide/svelte/icons/check";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import Plus from "@lucide/svelte/icons/plus";
  import Search from "@lucide/svelte/icons/search";
  import X from "@lucide/svelte/icons/x";
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type { Project } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { sessionGroups, type SessionGroupMode } from "./sessionView";
  import SidebarPanelHeader from "./SidebarPanelHeader.svelte";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import ModalShell from "./ui/ModalShell.svelte";

  export let sessions: Session[] = [];
  export let projects: Project[] = [];
  export let activeSessionId = "";
  export let loading = false;
  export let onclose: () => void;
  export let onNewSession: () => void;
  export let onSelectSession: (session: Session) => void;
  export let onCloseSession: (session: Session) => void;
  export let onSetSessionProject: (sessionId: string, projectId: string) => void;

  let confirmingSessionId = "";
  let groupMode: SessionGroupMode = "project";
  let query = "";
  let collapsedGroupIds = new Set<string>();
  let contextSessionId = "";
  let contextX = 0;
  let contextY = 0;
  let projectPickerSessionId = "";

  $: groups = sessionGroups(sessions, projects, groupMode, query);
  $: contextSession = sessions.find((session) => session.id === contextSessionId) ?? null;
  $: projectPickerSession =
    sessions.find((session) => session.id === projectPickerSessionId) ?? null;

  function requestClose(session: Session) {
    if (confirmingSessionId === session.id) {
      confirmingSessionId = "";
      onCloseSession(session);
      return;
    }
    confirmingSessionId = session.id;
  }

  function toggleGroup(groupId: string) {
    const next = new Set(collapsedGroupIds);
    if (next.has(groupId)) {
      next.delete(groupId);
    } else {
      next.add(groupId);
    }
    collapsedGroupIds = next;
  }

  function setGroupMode(mode: SessionGroupMode) {
    groupMode = mode;
    collapsedGroupIds = new Set<string>();
  }

  function openContextMenu(event: MouseEvent, session: Session) {
    event.preventDefault();
    contextSessionId = session.id;
    contextX = event.clientX;
    contextY = event.clientY;
  }

  function closeContextMenu() {
    contextSessionId = "";
  }

  function openProjectPicker(session: Session) {
    projectPickerSessionId = session.id;
    closeContextMenu();
  }

  function assignProject(projectId: string) {
    if (!projectPickerSession) return;
    onSetSessionProject(projectPickerSession.id, projectId);
    projectPickerSessionId = "";
  }

  function handleKey(event: KeyboardEvent) {
    if (event.key !== "Escape") return;
    closeContextMenu();
    projectPickerSessionId = "";
  }

  function closeProjectPicker() {
    projectPickerSessionId = "";
  }

  function handleProjectPickerOpenChange(open: boolean) {
    if (!open && projectPickerSession) closeProjectPicker();
  }
</script>

<svelte:window on:click={closeContextMenu} on:keydown={handleKey} />

<div class="flex h-full min-h-0 w-full flex-col bg-bg-deep">
  <SidebarPanelHeader title="Sessions" {onclose}>
    <button
      slot="actions"
      type="button"
      aria-label="New session"
      title="New session"
      class="inline-flex h-6 w-6 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
      disabled={loading}
      on:click={onNewSession}
    >
      <Plus size={13} />
    </button>
  </SidebarPanelHeader>

  <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
    {#if sessions.length === 0}
      <div class="flex h-full items-center justify-center px-4 text-center text-[13px] text-text-muted">
        No sessions yet.
      </div>
    {:else}
      <div class="grid gap-2">
        <div class="grid grid-cols-3 rounded border border-border-subtle bg-bg-surface/35 p-0.5">
          {#each [{ id: "recent", label: "Recent" }, { id: "project", label: "Project" }, { id: "folder", label: "Folder" }] as mode (mode.id)}
            <button
              type="button"
              class="h-7 rounded text-[11px] transition-colors {groupMode === mode.id
                ? 'bg-bg-active text-text-primary'
                : 'text-text-muted hover:bg-bg-hover hover:text-text-primary'}"
              on:click={() => setGroupMode(mode.id as SessionGroupMode)}
            >
              {mode.label}
            </button>
          {/each}
        </div>

        <label class="grid h-8 grid-cols-[14px_minmax(0,1fr)] items-center gap-2 rounded border border-border-subtle bg-bg-surface/35 px-2 text-text-muted focus-within:border-accent-dim">
          <Search size={14} />
          <input
            class="min-w-0 bg-transparent text-[12px] text-text-primary outline-none placeholder:text-text-muted"
            bind:value={query}
            placeholder="Search sessions"
            aria-label="Search sessions"
          />
        </label>

        {#if groups.length === 0}
          <div class="px-2 py-3 text-[12px] text-text-muted">No matching sessions.</div>
        {:else}
          {#each groups as group (group.id)}
            {@const collapsed = collapsedGroupIds.has(group.id)}
            <section class="grid gap-1">
              <button
                type="button"
                class="flex h-7 min-w-0 items-center gap-1 rounded px-1 text-left text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary"
                on:click={() => toggleGroup(group.id)}
              >
                <ChevronRight
                  size={13}
                  class="shrink-0 transition-transform {collapsed ? '' : 'rotate-90'}"
                />
                <span class="min-w-0 flex-1 truncate text-[11px] font-semibold uppercase tracking-widest">
                  {group.title}
                </span>
                <span class="font-mono text-[10px]">{group.sessions.length}</span>
              </button>

              {#if !collapsed}
                <div class="space-y-1">
                  {#each group.sessions as session (session.id)}
                    <div
                      class="w-full rounded-lg border px-2.5 py-2 text-left transition-colors {session.id ===
                      activeSessionId
                        ? 'border-accent-dim/50 bg-bg-active text-text-primary'
                        : 'border-border-subtle/60 bg-bg-surface/25 text-text-secondary hover:border-border hover:bg-bg-surface/50'}"
                    >
                      <div class="flex min-w-0 items-start gap-2">
                        <button
                          type="button"
                          class="min-w-0 flex-1 text-left"
                          on:click={() => onSelectSession(session)}
                          on:contextmenu={(event) => openContextMenu(event, session)}
                        >
                          <div class="truncate text-[13px] font-medium">{session.name}</div>
                          <div class="mt-0.5 truncate font-mono text-[10px] text-text-muted">
                            {session.rootDir || "."}
                          </div>
                        </button>
                        <button
                          type="button"
                          aria-label={confirmingSessionId === session.id
                            ? "Confirm close session"
                            : "Close session"}
                          title={confirmingSessionId === session.id ? "Confirm close session" : "Close session"}
                          class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded border transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 {confirmingSessionId ===
                          session.id
                            ? 'border-red/40 bg-red/10 text-red hover:bg-red/15'
                            : 'border-transparent text-text-muted hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary'}"
                          disabled={loading}
                          on:click|stopPropagation={() => requestClose(session)}
                        >
                          {#if confirmingSessionId === session.id}
                            <Check size={13} />
                          {:else}
                            <X size={13} />
                          {/if}
                        </button>
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}
            </section>
          {/each}
        {/if}
      </div>
    {/if}
  </div>
</div>

{#if contextSession}
  <div
    class="fixed z-50 min-w-44 rounded-md border border-border bg-bg-base py-1 shadow-[0_18px_50px_rgba(0,0,0,0.45)]"
    style="left: {contextX}px; top: {contextY}px"
    role="menu"
    tabindex="-1"
  >
    <button
      type="button"
      class="block h-8 w-full px-3 text-left text-[12px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
      on:click={() => {
        onSelectSession(contextSession);
        closeContextMenu();
      }}
    >
      Open
    </button>
    <button
      type="button"
      class="block h-8 w-full px-3 text-left text-[12px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-50"
      disabled={projects.length === 0}
      on:click={() => openProjectPicker(contextSession)}
    >
      Move to project...
    </button>
    {#if contextSession.projectId}
      <button
        type="button"
        class="block h-8 w-full px-3 text-left text-[12px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
        on:click={() => {
          onSetSessionProject(contextSession.id, "");
          closeContextMenu();
        }}
      >
        Remove from project
      </button>
    {/if}
    <div class="my-1 border-t border-hairline"></div>
    <button
      type="button"
      class="block h-8 w-full px-3 text-left text-[12px] text-red transition-colors hover:bg-red/10"
      on:click={() => {
        requestClose(contextSession);
        closeContextMenu();
      }}
    >
      Close
    </button>
  </div>
{/if}

{#if projectPickerSession}
  <ModalShell
    open={true}
    titleId="move-session-project-title"
    titleClass="sr-only"
    class="max-w-sm overflow-hidden bg-bg-base shadow-[0_24px_80px_rgba(0,0,0,0.45)]"
    onOpenChange={handleProjectPickerOpenChange}
    onEscapeKeydown={(event) => {
      event.preventDefault();
      closeProjectPicker();
    }}
  >
    {#snippet heading()}
      Move session to project
    {/snippet}

    <div class="flex h-11 items-center justify-between border-b border-hairline px-4">
      <div class="min-w-0">
        <div class="truncate text-[13px] font-semibold text-text-primary">Move to project</div>
        <div class="truncate text-[11px] text-text-muted">{projectPickerSession.name}</div>
      </div>
      <IconButton label="Close" onclick={closeProjectPicker}>
        <X size={14} />
      </IconButton>
    </div>

    <div class="app-scrollbar max-h-80 overflow-y-auto p-2">
      {#if projects.length === 0}
        <div class="px-3 py-4 text-center text-[12px] text-text-muted">No projects.</div>
      {:else}
        <div class="grid gap-1">
          {#each projects as project (project.id)}
            <Button
              type="button"
              variant="outline"
              size="lg"
              align="start"
              class="h-auto w-full flex-col items-start gap-0 px-3 py-2 {projectPickerSession.projectId ===
                project.id
                  ? 'border-accent-dim bg-bg-active text-text-primary hover:text-text-primary'
                  : 'border-border-subtle bg-bg-surface/25 text-text-secondary hover:border-border hover:bg-bg-surface/50 hover:text-text-primary'}"
              onclick={() => assignProject(project.id)}
            >
              <span class="w-full truncate text-[13px] font-medium">{project.name}</span>
              <span class="w-full truncate font-mono text-[10px] text-text-muted">{project.rootDir}</span>
            </Button>
          {/each}
        </div>
      {/if}
    </div>
  </ModalShell>
{/if}
