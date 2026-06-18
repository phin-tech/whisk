<script lang="ts">
  import Download from "@lucide/svelte/icons/download";
  import X from "@lucide/svelte/icons/x";
  import type { OnboardingStatus } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";

  export let visible = false;
  export let status: OnboardingStatus | null = null;
  export let busy = false;
  export let onclose: () => void;
  export let onapply: (ids: string[]) => void;

  let selected: Record<string, boolean> = {};
  let statusKey = "";

  $: nextKey = (status?.items ?? [])
    .map((item) => `${item.id}:${item.status}:${item.selectedByDefault}`)
    .join("|");
  $: if (nextKey !== statusKey) {
    statusKey = nextKey;
    selected = Object.fromEntries((status?.items ?? []).map((item) => [item.id, item.selectedByDefault]));
  }
  $: selectedIDs = Object.entries(selected)
    .filter(([, value]) => value)
    .map(([id]) => id);

  function statusClass(value: string) {
    if (value === "current") return "border-green/35 bg-green/10 text-green";
    if (value === "missing" || value === "outdated" || value === "untrusted") return "border-amber/35 bg-amber/10 text-amber";
    if (value === "modified" || value === "unavailable") return "border-red/30 bg-red/10 text-red";
    return "border-border bg-bg-deep text-text-muted";
  }
</script>

{#if visible && status}
  <div class="absolute inset-0 z-30 flex items-center justify-center bg-black/45 p-4">
    <section class="flex max-h-[82vh] w-full max-w-3xl flex-col overflow-hidden rounded-lg border border-border bg-bg-deep shadow-2xl">
      <header class="flex h-11 shrink-0 items-center justify-between border-b border-hairline px-4">
        <div class="text-sm font-semibold">Onboarding</div>
        <button
          type="button"
          aria-label="Close onboarding"
          class="rounded border border-transparent bg-transparent p-1 text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary"
          on:click={onclose}
        >
          <X size={15} />
        </button>
      </header>

      <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto">
        <div class="divide-y divide-hairline border-b border-hairline">
          {#each status.items as item (item.id)}
            <label class="grid cursor-pointer gap-3 px-4 py-3 md:grid-cols-[auto_minmax(160px,220px)_1fr] md:items-start">
              <input
                class="mt-1 h-4 w-4 accent-accent disabled:opacity-50"
                type="checkbox"
                disabled={busy || item.status === "current" || item.status === "unavailable" || !status.localDaemon}
                checked={selected[item.id] ?? false}
                on:change={(event) => (selected = { ...selected, [item.id]: event.currentTarget.checked })}
              />
              <div class="min-w-0">
                <div class="truncate text-[13px] font-medium text-text-primary">{item.label}</div>
                <span class="mt-1 inline-flex rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider {statusClass(item.status)}">
                  {item.status}
                </span>
              </div>
              <div class="min-w-0 space-y-1 text-[11px] text-text-muted">
                {#if item.description}
                  <div>{item.description}</div>
                {/if}
                {#if item.detail}
                  <div class="text-text-secondary">{item.detail}</div>
                {/if}
                {#if item.latestVersion || item.installedVersion}
                  <div>
                    Version <span class="font-mono text-text-secondary">{item.installedVersion || "none"} / {item.latestVersion || "unknown"}</span>
                  </div>
                {/if}
                {#if item.path}
                  <div class="truncate">
                    Path <span class="font-mono text-text-secondary">{item.path}</span>
                  </div>
                {/if}
              </div>
            </label>
          {/each}
        </div>
      </div>

      <footer class="flex shrink-0 items-center justify-between gap-3 border-t border-hairline px-4 py-3">
        <div class="min-w-0 truncate text-[11px] text-text-muted">
          {status.localDaemon ? status.statePath : "Remote daemon"}
        </div>
        <button
          type="button"
          class="inline-flex h-8 items-center justify-center gap-1 rounded border border-accent/50 bg-accent-dim px-3 text-[12px] font-medium text-text-primary transition-colors hover:border-accent disabled:cursor-not-allowed disabled:opacity-50"
          disabled={busy || selectedIDs.length === 0 || !status.localDaemon}
          on:click={() => onapply(selectedIDs)}
        >
          <Download size={14} />
          <span>{busy ? "Applying" : "Apply Selected"}</span>
        </button>
      </footer>
    </section>
  </div>
{/if}
