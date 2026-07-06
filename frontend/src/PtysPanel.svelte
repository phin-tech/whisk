<script lang="ts">
  import { onMount } from "svelte";
  import CircleStop from "@lucide/svelte/icons/circle-stop";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import type { PTYHistory, PTYHistorySummary, PTYInfo } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import {
    derivePtysPanelView,
    derivePtysPanelVirtualState,
    PTYS_PANEL_ITEM_ROW_HEIGHT,
  } from "./ptys-panel-state";
  import SidebarPanelHeader from "./SidebarPanelHeader.svelte";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";

  const PTYS_PANEL_OVERSCAN = 4;

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

  let viewport: HTMLDivElement;
  let viewportHeight = PTYS_PANEL_ITEM_ROW_HEIGHT * 8;
  let scrollOffset = 0;

  $: panelView = derivePtysPanelView({
    ptys,
    ptyHistory,
    selectedPTYHistory,
    loading,
    loadingHistory,
  });
  $: virtualState = derivePtysPanelVirtualState({
    rows: panelView.rows,
    viewportHeight,
    scrollOffset,
    overscan: PTYS_PANEL_OVERSCAN,
  });

  function measureViewport() {
    if (!viewport) return;
    viewportHeight = viewport.clientHeight;
    scrollOffset = viewport.scrollTop;
  }

  onMount(() => {
    measureViewport();
    const resizeObserver = new ResizeObserver(measureViewport);
    resizeObserver.observe(viewport);
    return () => resizeObserver.disconnect();
  });
</script>

<div class="flex h-full min-h-0 w-full flex-col bg-bg-deep">
  <SidebarPanelHeader title="PTYs" {onclose}>
    <IconButton
      slot="actions"
      disabled={loading || loadingHistory}
      label="Refresh PTYs"
      size="sm"
      onclick={onRefresh}
    >
      <RefreshCw size={13} class={loading || loadingHistory ? "animate-spin" : ""} />
    </IconButton>
  </SidebarPanelHeader>

  <div
    bind:this={viewport}
    class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2"
    aria-label="PTYs"
    data-ptys-virtual-list
    onscroll={measureViewport}
  >
    {#if panelView.showLoadingEmpty}
      <div class="flex h-full items-center justify-center text-[13px] text-text-muted">
        Loading PTYs...
      </div>
    {:else if panelView.showEmpty}
      <div class="flex h-full items-center justify-center text-[13px] text-text-muted">
        No daemon PTYs or history
      </div>
    {:else}
      <div class="relative min-w-0" style={`height: ${virtualState.window.totalHeight}px;`}>
        {#each virtualState.virtualRows as virtualRow (virtualRow.key)}
          {@const row = virtualRow.row}
          <div
            class="absolute left-0 right-0 overflow-hidden bg-bg-deep"
            style={`transform: translateY(${virtualRow.offsetTop}px); height: ${virtualRow.height}px;`}
            data-pty-virtual-row
            data-pty-row-key={virtualRow.key}
            data-pty-row-index={virtualRow.index}
          >
            {#if row.kind === "section"}
              <div class="flex h-full items-end px-1 pb-1 text-[10px] font-semibold uppercase text-text-muted">
                {row.title}
              </div>
            {:else if row.kind === "live"}
              <div class="h-full px-0.5 py-1">
                <div class="h-full rounded-lg border border-border-subtle/60 bg-bg-surface/35 px-2 py-1.5 text-[12px]">
                  <div class="flex min-w-0 items-center gap-2">
                    <span
                      class="h-2 w-2 shrink-0 rounded-full {row.running
                        ? row.statusTone === 'waiting'
                          ? 'bg-amber'
                          : row.statusTone === 'working'
                            ? 'bg-blue'
                            : 'bg-green'
                        : 'bg-text-muted'}"
                      title={row.agentLabel ? `${row.agentLabel}: ${row.status}` : row.status}
                    ></span>
                    <span class="min-w-0 flex-1 truncate font-medium text-text-primary">
                      {row.title}
                    </span>
                    <span class="shrink-0 text-[10px] uppercase text-text-muted">
                      {row.status}
                    </span>
                    {#if row.canDelete}
                      <IconButton
                        tone="danger"
                        size="sm"
                        label={`Delete PTY ${row.id}`}
                        title={`Delete PTY ${row.id}`}
                        onclick={() => onDelete(row.id)}
                      >
                        <Trash2 size={13} />
                      </IconButton>
                    {:else}
                      <IconButton
                        tone="danger"
                        size="sm"
                        label={`Kill PTY ${row.id}`}
                        title={`Kill PTY ${row.id}`}
                        onclick={() => onKill(row.id)}
                      >
                        <CircleStop size={13} />
                      </IconButton>
                    {/if}
                  </div>
                  <div class="mt-1 grid gap-0.5 text-[10px] text-text-muted">
                    <div class="truncate">{row.subtitle}</div>
                    <div class="truncate" title={row.detail}>{row.detail}</div>
                  </div>
                </div>
              </div>
            {:else if row.kind === "history"}
              <div class="h-full px-0.5 py-1">
                <Button
                  variant="ghost"
                  align="start"
                  class="grid !h-full w-full gap-1 rounded-lg border-border-subtle/60 bg-bg-surface/25 px-2 py-1.5 text-left text-[12px] hover:border-accent/40 hover:bg-bg-hover"
                  onclick={() => onSelectHistory(row.id)}
                >
                  <div class="flex min-w-0 items-center gap-2">
                    <span class="h-2 w-2 shrink-0 rounded-full bg-text-muted"></span>
                    <span class="min-w-0 flex-1 truncate font-medium text-text-primary">
                      {row.title}
                    </span>
                    <span class="shrink-0 text-[10px] uppercase text-text-muted">
                      {row.statusLabel}
                    </span>
                  </div>
                  <div class="mt-1 grid gap-0.5 text-[10px] text-text-muted">
                    <div class="truncate">{row.subtitle}</div>
                    <div class="truncate" title={row.detail}>{row.detail}</div>
                  </div>
                </Button>
              </div>
            {:else if row.kind === "history-output"}
              <div class="h-full px-0.5 py-1">
                <pre class="h-full max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border-subtle/60 bg-bg-base/70 p-2 font-mono text-[10px] leading-relaxed text-text-secondary">{row.output}</pre>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
