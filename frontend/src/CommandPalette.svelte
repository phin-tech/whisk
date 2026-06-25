<script lang="ts">
  import Search from "@lucide/svelte/icons/search";
  import type { Command } from "./commands";
  import { commandItems } from "./commands";
  import Button from "./ui/Button.svelte";
  import ModalShell from "./ui/ModalShell.svelte";
  import TextField from "./ui/TextField.svelte";

  type Props = {
    visible?: boolean;
    commands?: Command[];
    onclose: () => void;
    onrun: (id: string) => void;
  };

  let { visible = false, commands = [], onclose, onrun }: Props = $props();

  let query = $state("");
  let selected = $state(0);
  let input = $state<HTMLInputElement | null>(null);
  let previousVisible = $state(false);

  const items = $derived(commandItems(commands, query));

  function runSelected() {
    const item = items[selected];
    if (!item) return;
    onrun(item.id);
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
  titleId="command-palette-title"
  titleClass="sr-only"
  placement="top"
  class="max-w-[560px] overflow-hidden bg-bg-base shadow-[0_24px_80px_rgba(0,0,0,0.45)]"
  interactOutsideBehavior="close"
  onOpenChange={handleOpenChange}
>
  {#snippet heading()}
    Command palette
  {/snippet}

  <div class="flex h-11 items-center gap-2 border-b border-hairline px-3">
    <Search size={15} class="shrink-0 text-text-muted" />
    <TextField
      bind:ref={input}
      bind:value={query}
      variant="seamless"
      placeholder="Run command"
      class="h-full min-w-0 flex-1 border-transparent px-0 text-[13px] hover:border-transparent focus:border-transparent focus:bg-transparent"
      onkeydown={handleKey}
    />
  </div>

  <div class="max-h-[320px] overflow-y-auto p-1.5">
    {#if items.length === 0}
      <div class="px-2 py-6 text-center text-[13px] text-text-muted">No commands</div>
    {:else}
      {#each items as item, index (item.id)}
        <Button
          type="button"
          variant="ghost"
          size="lg"
          align="between"
          class="h-9 w-full rounded px-2.5 text-[13px] font-medium {index === selected
            ? 'bg-bg-hover text-text-primary'
            : 'text-text-secondary hover:bg-bg-hover hover:text-text-primary'}"
          onmouseenter={() => (selected = index)}
          onclick={() => onrun(item.id)}
        >
          <span class="truncate">{item.title}</span>
          {#if item.shortcut}
            <span class="shrink-0 font-mono text-[11px] text-text-muted">{item.shortcut}</span>
          {/if}
        </Button>
      {/each}
    {/if}
  </div>
</ModalShell>
