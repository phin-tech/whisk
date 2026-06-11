<script lang="ts">
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import type { PTYInfo } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { ptyRowsFromInventory } from "./sessionView";
  import SidebarPanelHeader from "./SidebarPanelHeader.svelte";

  export let ptys: PTYInfo[] = [];
  export let loading = false;
  export let onclose: () => void;
  export let onRefresh: () => void;

  $: rows = ptyRowsFromInventory(ptys);
</script>

<div class="flex h-full min-h-0 w-full flex-col bg-bg-deep">
  <SidebarPanelHeader title="PTYs" {onclose}>
    <button
      slot="actions"
      type="button"
      class="inline-flex h-6 w-6 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-wait disabled:opacity-60"
      disabled={loading}
      aria-label="Refresh PTYs"
      title="Refresh PTYs"
      on:click={onRefresh}
    >
      <RefreshCw size={13} class={loading ? "animate-spin" : ""} />
    </button>
  </SidebarPanelHeader>

  <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
    {#if loading && rows.length === 0}
      <div class="flex h-full items-center justify-center text-sm text-text-muted">
        Loading PTYs...
      </div>
    {:else if rows.length === 0}
      <div class="flex h-full items-center justify-center text-sm text-text-muted">
        No daemon PTYs
      </div>
    {:else}
      <div class="space-y-1">
        {#each rows as row (row.id)}
          <div
            class="rounded-lg border border-border-subtle/60 bg-bg-surface/35 px-2 py-1.5 text-xs"
          >
            <div class="flex min-w-0 items-center gap-2">
              <span
                class="h-2 w-2 shrink-0 rounded-full {row.running
                  ? 'bg-green'
                  : 'bg-text-muted'}"
              ></span>
              <span class="min-w-0 flex-1 truncate font-medium text-text-primary">
                {row.title}
              </span>
              <span class="shrink-0 text-[10px] uppercase text-text-muted">
                {row.running ? "running" : "exited"}
              </span>
            </div>
            <div class="mt-1 grid gap-0.5 text-[10px] text-text-muted">
              <div class="truncate">{row.subtitle}</div>
              <div class="truncate" title={row.detail}>{row.detail}</div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
