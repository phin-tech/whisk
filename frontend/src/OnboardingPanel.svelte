<script lang="ts">
  import Download from "@lucide/svelte/icons/download";
  import X from "@lucide/svelte/icons/x";
  import type { OnboardingStatus } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import Button from "./ui/Button.svelte";
  import Checkbox from "./ui/Checkbox.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import ModalShell from "./ui/ModalShell.svelte";

  type Props = {
    visible?: boolean;
    status?: OnboardingStatus | null;
    busy?: boolean;
    onclose: () => void;
    onapply: (ids: string[]) => void;
  };

  let { visible = false, status = null, busy = false, onclose, onapply }: Props = $props();

  let selected = $state<Record<string, boolean>>({});
  let statusKey = $state("");

  const nextKey = $derived(
    (status?.items ?? [])
      .map((item) => `${item.id}:${item.status}:${item.selectedByDefault}`)
      .join("|"),
  );
  const selectedIDs = $derived(
    Object.entries(selected)
      .filter(([, value]) => value)
      .map(([id]) => id),
  );

  function statusClass(value: string) {
    if (value === "current") return "border-green/35 bg-green/10 text-green";
    if (value === "missing" || value === "outdated" || value === "untrusted") {
      return "border-amber/35 bg-amber/10 text-amber";
    }
    if (value === "modified" || value === "unavailable") return "border-red/30 bg-red/10 text-red";
    return "border-border bg-bg-deep text-text-muted";
  }

  function itemDisabled(item: OnboardingStatus["items"][number]) {
    return busy || item.status === "current" || item.status === "unavailable" || !status?.localDaemon;
  }

  function setSelected(id: string, checked: boolean) {
    selected = { ...selected, [id]: checked };
  }

  function handleOpenChange(open: boolean) {
    if (!open && visible) onclose();
  }

  $effect(() => {
    if (nextKey !== statusKey) {
      statusKey = nextKey;
      selected = Object.fromEntries((status?.items ?? []).map((item) => [item.id, item.selectedByDefault]));
    }
  });
</script>

{#if status}
  <ModalShell
    open={visible}
    titleId="onboarding-title"
    titleClass="sr-only"
    class="flex max-h-[82vh] max-w-3xl flex-col overflow-hidden bg-bg-deep shadow-2xl"
    onOpenChange={handleOpenChange}
  >
    {#snippet heading()}
      Onboarding
    {/snippet}

    <header class="flex h-11 shrink-0 items-center justify-between border-b border-hairline px-4">
      <div class="text-[13px] font-semibold">Onboarding</div>
      <IconButton label="Close onboarding" onclick={onclose}>
        <X size={15} />
      </IconButton>
    </header>

    <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto">
      <div class="divide-y divide-hairline border-b border-hairline">
        {#each status.items as item (item.id)}
          <div class="grid gap-3 px-4 py-3 md:grid-cols-[auto_minmax(160px,220px)_1fr] md:items-start">
            <Checkbox
              class="mt-1"
              checked={selected[item.id] ?? false}
              disabled={itemDisabled(item)}
              onCheckedChange={(checked) => setSelected(item.id, checked)}
            >
              <span class="sr-only">Select {item.label}</span>
            </Checkbox>
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
          </div>
        {/each}
      </div>
    </div>

    <footer class="flex shrink-0 items-center justify-between gap-3 border-t border-hairline px-4 py-3">
      <div class="min-w-0 truncate text-[11px] text-text-muted">
        {status.localDaemon ? status.statePath : "Remote daemon"}
      </div>
      <Button
        type="button"
        variant="primary"
        disabled={busy || selectedIDs.length === 0 || !status.localDaemon}
        onclick={() => onapply(selectedIDs)}
      >
        <Download size={14} />
        <span>{busy ? "Applying" : "Apply Selected"}</span>
      </Button>
    </footer>
  </ModalShell>
{/if}
