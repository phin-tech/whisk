<script lang="ts">
  import Search from "@lucide/svelte/icons/search";
  import type { JumpTarget } from "./jumpFilter";
  import { prepareJumpTargets, rankJumpTargets } from "./jumpFilter";
  import { applyRecentJumpTargets, reconcileJumpRecents } from "./jumpRecents";
  import Badge from "./ui/Badge.svelte";
  import Button from "./ui/Button.svelte";
  import ModalShell from "./ui/ModalShell.svelte";
  import TextField from "./ui/TextField.svelte";

  type Props = {
    visible?: boolean;
    targets?: JumpTarget[];
    recentTargetIds?: string[];
    onclose: () => void;
    onjump: (target: JumpTarget) => void;
  };

  let { visible = false, targets = [], recentTargetIds = [], onclose, onjump }: Props = $props();

  let query = $state("");
  let selected = $state(0);
  let input = $state<HTMLInputElement | null>(null);
  let previousVisible = $state(false);

  const validRecentTargetIds = $derived(
    reconcileJumpRecents(
      recentTargetIds,
      targets.map((target) => target.id),
    ),
  );
  const emptyQueryTargets = $derived(applyRecentJumpTargets(targets, validRecentTargetIds));
  const rankedTargets = $derived(query.trim() ? targets : emptyQueryTargets);
  const preparedTargets = $derived(prepareJumpTargets(rankedTargets));
  const items = $derived(rankJumpTargets(query, preparedTargets));

  function runSelected() {
    const item = items[selected];
    if (!item) return;
    onjump(item);
  }

  function handleKey(event: KeyboardEvent) {
    if (event.key === "Escape") {
      event.preventDefault();
      onclose();
    } else if (event.key === "ArrowDown") {
      event.preventDefault();
      selected = items.length ? (selected + 1) % items.length : 0;
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      selected = items.length ? (selected - 1 + items.length) % items.length : 0;
    } else if (event.key === "Enter") {
      event.preventDefault();
      runSelected();
    }
  }

  function handleOpenChange(open: boolean) {
    if (!open && visible) onclose();
  }

  function kindLabel(kind: JumpTarget["kind"]) {
    if (kind === "work-item") return "item";
    if (kind === "work-item-run") return "run";
    if (kind === "plugin-command") return "plugin";
    return kind;
  }

  function kindTone(kind: JumpTarget["kind"]) {
    if (kind === "session" || kind === "pane" || kind === "pty") return "accent";
    if (kind === "project") return "blue";
    if (kind === "work-item" || kind === "work-item-run") return "green";
    return "muted";
  }

  $effect(() => {
    if (selected >= items.length) selected = Math.max(0, items.length - 1);
  });

  $effect(() => {
    if (visible && !previousVisible) {
      query = "";
      selected = 0;
      setTimeout(() => input?.focus());
    }
    previousVisible = visible;
  });
</script>

<ModalShell
  open={visible}
  titleId="jump-palette-title"
  titleClass="sr-only"
  placement="top"
  class="max-w-[640px] overflow-hidden bg-bg-base shadow-[0_24px_80px_rgba(0,0,0,0.45)]"
  interactOutsideBehavior="close"
  onOpenChange={handleOpenChange}
>
  {#snippet heading()}
    Jump palette
  {/snippet}

  <div class="flex h-11 items-center gap-2 border-b border-hairline px-3">
    <Search size={15} class="shrink-0 text-text-muted" />
    <TextField
      bind:ref={input}
      bind:value={query}
      variant="seamless"
      placeholder="Jump to session, project, work item"
      aria-label="Jump target"
      class="h-full min-w-0 flex-1 border-transparent px-0 text-[13px] hover:border-transparent focus:border-transparent focus:bg-transparent"
      onkeydown={handleKey}
    />
  </div>

  <div class="max-h-[420px] overflow-y-auto p-1.5">
    {#if items.length === 0}
      <div class="px-2 py-6 text-center text-[13px] text-text-muted">No targets</div>
    {:else}
      <div role="listbox" aria-labelledby="jump-palette-title">
        {#each items as item, index (item.id)}
          <Button
            type="button"
            variant="ghost"
            size="lg"
            align="start"
            role="option"
            aria-selected={index === selected}
            class="!h-auto w-full min-w-0 rounded px-2.5 py-2 text-[13px] {index === selected
              ? 'bg-bg-hover text-text-primary'
              : 'text-text-secondary hover:bg-bg-hover hover:text-text-primary'}"
            onmouseenter={() => (selected = index)}
            onclick={() => onjump(item)}
          >
            <span class="flex min-w-0 flex-1 items-center gap-2">
              <Badge tone={kindTone(item.kind)} class="w-[58px] shrink-0 justify-center uppercase">
                {kindLabel(item.kind)}
              </Badge>
              <span class="min-w-0 flex-1">
                <span class="flex min-w-0 items-center gap-1.5">
                  <span class="truncate font-semibold">{item.title}</span>
                  {#if item.current}
                    <span class="shrink-0 text-[11px] font-medium text-accent">current</span>
                  {/if}
                </span>
                {#if item.subtitle || item.detail}
                  <span class="mt-0.5 block truncate text-[12px] font-normal text-text-muted">
                    {[item.subtitle, item.detail].filter(Boolean).join(" / ")}
                  </span>
                {/if}
              </span>
            </span>
          </Button>
        {/each}
      </div>
    {/if}
  </div>
</ModalShell>
