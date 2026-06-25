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
  import TextField from "./ui/TextField.svelte";

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

  function requestCloseFromRow(event: MouseEvent, session: Session) {
    event.stopPropagation();
    requestClose(session);
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

<svelte:window onclick={closeContextMenu} onkeydown={handleKey} />

<div class="flex h-full min-h-0 w-full flex-col bg-bg-deep">
  <SidebarPanelHeader title="Sessions" {onclose}>
    <IconButton
      slot="actions"
      label="New session"
      size="sm"
      disabled={loading}
      onclick={onNewSession}
    >
      <Plus size={13} />
    </IconButton>
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
            <Button
              variant={groupMode === mode.id ? "primary" : "ghost"}
              size="sm"
              class="h-7 w-full border-transparent text-[11px] {groupMode === mode.id ? '' : 'bg-transparent'}"
              onclick={() => setGroupMode(mode.id as SessionGroupMode)}
            >
              {mode.label}
            </Button>
          {/each}
        </div>

        <label class="grid h-8 grid-cols-[14px_minmax(0,1fr)] items-center gap-2 rounded border border-border-subtle bg-bg-surface/35 px-2 text-text-muted focus-within:border-accent-dim">
          <Search size={14} />
          <TextField
            variant="seamless"
            class="min-w-0 border-transparent bg-transparent px-0 py-0 hover:border-transparent focus:border-transparent focus:bg-transparent"
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
              <Button
                variant="ghost"
                size="sm"
                align="start"
                class="h-7 min-w-0 border-transparent bg-transparent px-1 text-text-muted"
                onclick={() => toggleGroup(group.id)}
              >
                <ChevronRight
                  size={13}
                  class="shrink-0 transition-transform {collapsed ? '' : 'rotate-90'}"
                />
                <span class="min-w-0 flex-1 truncate text-[11px] font-semibold uppercase tracking-widest">
                  {group.title}
                </span>
                <span class="font-mono text-[10px]">{group.sessions.length}</span>
              </Button>

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
                        <Button
                          variant="ghost"
                          size="sm"
                          align="start"
                          class="!h-auto min-w-0 flex-1 flex-col !items-start gap-0 !border-transparent !bg-transparent !px-0 !py-0 text-left hover:!bg-transparent hover:!text-inherit"
                          onclick={() => onSelectSession(session)}
                          oncontextmenu={(event: MouseEvent) => openContextMenu(event, session)}
                        >
                          <div class="truncate text-[13px] font-medium">{session.name}</div>
                          <div class="mt-0.5 truncate font-mono text-[10px] text-text-muted">
                            {session.rootDir || "."}
                          </div>
                        </Button>
                        <IconButton
                          label={confirmingSessionId === session.id
                            ? "Confirm close session"
                            : "Close session"}
                          title={confirmingSessionId === session.id ? "Confirm close session" : "Close session"}
                          tone={confirmingSessionId === session.id ? "danger" : "default"}
                          size="sm"
                          class="shrink-0 {confirmingSessionId ===
                          session.id
                            ? '!border-red/40 !bg-red/10 !text-red hover:!bg-red/15'
                            : ''}"
                          disabled={loading}
                          onclick={(event) => requestCloseFromRow(event, session)}
                        >
                          {#if confirmingSessionId === session.id}
                            <Check size={13} />
                          {:else}
                            <X size={13} />
                          {/if}
                        </IconButton>
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
    <Button
      variant="ghost"
      align="start"
      class="h-8 w-full rounded-none border-transparent bg-transparent px-3 text-[12px]"
      onclick={() => {
        onSelectSession(contextSession);
        closeContextMenu();
      }}
    >
      Open
    </Button>
    <Button
      variant="ghost"
      align="start"
      class="h-8 w-full rounded-none border-transparent bg-transparent px-3 text-[12px]"
      disabled={projects.length === 0}
      onclick={() => openProjectPicker(contextSession)}
    >
      Move to project...
    </Button>
    {#if contextSession.projectId}
      <Button
        variant="ghost"
        align="start"
        class="h-8 w-full rounded-none border-transparent bg-transparent px-3 text-[12px]"
        onclick={() => {
          onSetSessionProject(contextSession.id, "");
          closeContextMenu();
        }}
      >
        Remove from project
      </Button>
    {/if}
    <div class="my-1 border-t border-hairline"></div>
    <Button
      variant="danger-ghost"
      align="start"
      class="h-8 w-full rounded-none border-transparent bg-transparent px-3 text-[12px]"
      onclick={() => {
        requestClose(contextSession);
        closeContextMenu();
      }}
    >
      Close
    </Button>
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
