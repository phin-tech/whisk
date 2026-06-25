<script lang="ts">
  import CircleStop from "@lucide/svelte/icons/circle-stop";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import type { PTYHistory, PTYHistorySummary, PTYInfo } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { ptyHistoryRows, ptyRowsFromInventory } from "./sessionView";
  import SidebarPanelHeader from "./SidebarPanelHeader.svelte";

  export let ptys: PTYInfo[] = [];
  export let ptyHistory: PTYHistorySummary[] = [];
  export let selectedPTYHistory: PTYHistory | null = null;
  export let loading = false;
  export let loadingHistory = false;
  export let onclose: () => void;
  export let onRefresh: () => void;
  export let onKill: (ptyId: string) => void;
  export let onDelete: (ptyId: string) => void;
  export let onSelectHistory: (ptyId: string) => void;

  $: rows = ptyRowsFromInventory(ptys);
  $: historyRows = ptyHistoryRows(ptyHistory);
</script>

<div class="flex h-full min-h-0 w-full flex-col bg-bg-deep">
  <SidebarPanelHeader title="PTYs" {onclose}>
    <button
      slot="actions"
      type="button"
      class="inline-flex h-6 w-6 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-wait disabled:opacity-60"
      disabled={loading || loadingHistory}
      aria-label="Refresh PTYs"
      title="Refresh PTYs"
      on:click={onRefresh}
    >
      <RefreshCw size={13} class={loading || loadingHistory ? "animate-spin" : ""} />
    </button>
  </SidebarPanelHeader>

  <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
    {#if (loading || loadingHistory) && rows.length === 0 && historyRows.length === 0}
      <div class="flex h-full items-center justify-center text-[13px] text-text-muted">
        Loading PTYs...
      </div>
    {:else if rows.length === 0 && historyRows.length === 0}
      <div class="flex h-full items-center justify-center text-[13px] text-text-muted">
        No daemon PTYs or history
      </div>
    {:else}
      <div class="space-y-3">
        {#if rows.length > 0}
          <section class="space-y-1">
            <div class="px-1 text-[10px] font-semibold uppercase text-text-muted">
              Live
            </div>
            {#each rows as row (row.id)}
              <div
                class="rounded-lg border border-border-subtle/60 bg-bg-surface/35 px-2 py-1.5 text-[12px]"
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
                    {row.status}
                  </span>
                  {#if row.canDelete}
                    <button
                      type="button"
                      class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-red/40 hover:bg-red/10 hover:text-red focus:outline-none focus:ring-1 focus:ring-red"
                      aria-label={`Delete PTY ${row.id}`}
                      title={`Delete PTY ${row.id}`}
                      on:click={() => onDelete(row.id)}
                    >
                      <Trash2 size={13} />
                    </button>
                  {:else}
                    <button
                      type="button"
                      class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-red/40 hover:bg-red/10 hover:text-red focus:outline-none focus:ring-1 focus:ring-red"
                      aria-label={`Kill PTY ${row.id}`}
                      title={`Kill PTY ${row.id}`}
                      on:click={() => onKill(row.id)}
                    >
                      <CircleStop size={13} />
                    </button>
                  {/if}
                </div>
                <div class="mt-1 grid gap-0.5 text-[10px] text-text-muted">
                  <div class="truncate">{row.subtitle}</div>
                  <div class="truncate" title={row.detail}>{row.detail}</div>
                </div>
              </div>
            {/each}
          </section>
        {/if}

        {#if historyRows.length > 0}
          <section class="space-y-1">
            <div class="px-1 text-[10px] font-semibold uppercase text-text-muted">
              History
            </div>
            {#each historyRows as row (row.id)}
              <button
                type="button"
                class="w-full rounded-lg border border-border-subtle/60 bg-bg-surface/25 px-2 py-1.5 text-left text-[12px] transition-colors hover:border-accent/40 hover:bg-bg-hover focus:outline-none focus:ring-1 focus:ring-accent-dim/50"
                on:click={() => onSelectHistory(row.id)}
              >
                <div class="flex min-w-0 items-center gap-2">
                  <span class="h-2 w-2 shrink-0 rounded-full bg-text-muted"></span>
                  <span class="min-w-0 flex-1 truncate font-medium text-text-primary">
                    {row.title}
                  </span>
                  <span class="shrink-0 text-[10px] uppercase text-text-muted">
                    {row.exitCode === null ? "saved" : `exit ${row.exitCode}`}
                  </span>
                </div>
                <div class="mt-1 grid gap-0.5 text-[10px] text-text-muted">
                  <div class="truncate">{row.subtitle}</div>
                  <div class="truncate" title={row.detail}>{row.detail}</div>
                </div>
              </button>
              {#if selectedPTYHistory?.ptyId === row.id}
                <pre class="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border-subtle/60 bg-bg-base/70 p-2 font-mono text-[10px] leading-relaxed text-text-secondary">{selectedPTYHistory.output || "(no output)"}</pre>
              {/if}
            {/each}
          </section>
        {/if}
      </div>
    {/if}
  </div>
</div>
