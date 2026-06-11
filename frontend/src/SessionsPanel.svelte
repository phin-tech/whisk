<script lang="ts">
  import Plus from "@lucide/svelte/icons/plus";
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import SidebarPanelHeader from "./SidebarPanelHeader.svelte";

  export let sessions: Session[] = [];
  export let activeSessionId = "";
  export let loading = false;
  export let onclose: () => void;
  export let onNewSession: () => void;
  export let onSelectSession: (session: Session) => void;
</script>

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
      <div class="flex h-full items-center justify-center px-4 text-center text-sm text-text-muted">
        No sessions yet.
      </div>
    {:else}
      <div class="space-y-1">
        {#each sessions as session (session.id)}
          <button
            type="button"
            class="w-full rounded-lg border px-2.5 py-2 text-left transition-colors {session.id ===
            activeSessionId
              ? 'border-accent-dim/50 bg-bg-active text-text-primary'
              : 'border-border-subtle/60 bg-bg-surface/25 text-text-secondary hover:border-border hover:bg-bg-surface/50'}"
            on:click={() => onSelectSession(session)}
          >
            <div class="truncate text-[13px] font-medium">{session.name}</div>
            <div class="mt-0.5 truncate font-mono text-[10px] text-text-muted">
              {session.workingDir || "."}
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </div>
</div>
