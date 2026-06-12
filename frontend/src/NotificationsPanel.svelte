<script lang="ts">
  import AlertTriangle from "@lucide/svelte/icons/alert-triangle";
  import CheckCircle2 from "@lucide/svelte/icons/check-circle-2";
  import CircleHelp from "@lucide/svelte/icons/circle-help";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import X from "@lucide/svelte/icons/x";
  import type { StatusEvent } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { notificationRows } from "./notificationsView";
  import SidebarPanelHeader from "./SidebarPanelHeader.svelte";

  export let statusEvents: StatusEvent[] = [];
  export let loading = false;
  export let onclose: () => void;
  export let onRefresh: () => void;
  export let onSelectStatusEvent: (event: StatusEvent) => void;

  $: rows = notificationRows(statusEvents);

  function iconForTone(tone: string) {
    if (tone === "done") return CheckCircle2;
    if (tone === "warning") return AlertTriangle;
    return CircleHelp;
  }

  function eventById(id: string) {
    return statusEvents.find((event) => event.id === id);
  }
</script>

<div class="flex h-full min-h-0 w-full flex-col bg-bg-deep">
  <SidebarPanelHeader title="Notifications" {onclose}>
    <div slot="actions" class="flex items-center gap-1">
      <button
        type="button"
        class="inline-flex h-6 w-6 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-wait disabled:opacity-60"
        disabled={loading}
        aria-label="Refresh notifications"
        title="Refresh notifications"
        on:click={onRefresh}
      >
        <RefreshCw size={13} class={loading ? "animate-spin" : ""} />
      </button>
    </div>
  </SidebarPanelHeader>

  <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
    {#if rows.length === 0}
      <div class="flex h-full items-center justify-center px-4 text-center text-sm text-text-muted">
        No unread notifications.
      </div>
    {:else}
      <div class="space-y-1">
        {#each rows as row (row.id)}
          {@const Icon = iconForTone(row.tone)}
          <button
            type="button"
            class="w-full rounded border px-2.5 py-2 text-left transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 {row.tone ===
            'done'
              ? 'border-green/25 bg-green/5 text-text-secondary hover:border-green/40 hover:bg-green/10'
              : row.tone === 'warning'
                ? 'border-amber/30 bg-amber/5 text-text-primary hover:border-amber/50 hover:bg-amber/10'
                : 'border-accent-dim/40 bg-accent-dim/10 text-text-primary hover:border-accent hover:bg-accent-dim/15'}"
            on:click={() => {
              const event = eventById(row.id);
              if (event) onSelectStatusEvent(event);
            }}
          >
            <div class="flex min-w-0 items-start gap-2">
              <Icon size={14} class="mt-0.5 shrink-0" />
              <div class="min-w-0 flex-1">
                <div class="flex min-w-0 items-center justify-between gap-2">
                  <div class="truncate text-[12px] font-semibold">{row.title}</div>
                  <X size={11} class="shrink-0 opacity-45" />
                </div>
                <div class="mt-1 line-clamp-3 text-[12px] leading-4 text-text-secondary">
                  {row.message}
                </div>
                <div class="mt-1 truncate font-mono text-[10px] text-text-muted">
                  {row.meta}
                </div>
              </div>
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </div>
</div>
