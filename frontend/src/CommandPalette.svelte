<script lang="ts">
  import Search from "@lucide/svelte/icons/search";
  import type { Command } from "./commands";
  import { commandItems } from "./commands";

  export let visible = false;
  export let commands: Command[] = [];
  export let onclose: () => void;
  export let onrun: (id: string) => void;

  let query = "";
  let selected = 0;
  let input: HTMLInputElement;
  let previousVisible = false;

  $: items = commandItems(commands, query);
  $: if (selected >= items.length) selected = Math.max(0, items.length - 1);
  $: if (visible && !previousVisible) {
    query = "";
    selected = 0;
    setTimeout(() => input?.focus());
  }
  $: previousVisible = visible;

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
</script>

{#if visible}
  <div
    class="fixed inset-0 z-50 flex items-start justify-center bg-black/45 px-4 pt-[14vh]"
    role="dialog"
    aria-modal="true"
    aria-label="Command palette"
    tabindex="-1"
    on:mousedown|self={onclose}
  >
    <div class="w-full max-w-[560px] overflow-hidden rounded-lg border border-border bg-bg-base shadow-[0_24px_80px_rgba(0,0,0,0.45)]">
      <div class="flex h-11 items-center gap-2 border-b border-hairline px-3">
        <Search size={15} class="shrink-0 text-text-muted" />
        <input
          bind:this={input}
          bind:value={query}
          class="h-full min-w-0 flex-1 bg-transparent text-[13px] text-text-primary outline-none placeholder:text-text-muted"
          type="text"
          placeholder="Run command"
          on:keydown={handleKey}
        />
      </div>
      <div class="max-h-[320px] overflow-y-auto p-1.5">
        {#if items.length === 0}
          <div class="px-2 py-6 text-center text-sm text-text-muted">No commands</div>
        {:else}
          {#each items as item, index (item.id)}
            <button
              type="button"
              class="flex h-9 w-full items-center justify-between gap-3 rounded px-2.5 text-left text-[13px] transition-colors {index ===
              selected
                ? 'bg-bg-hover text-text-primary'
                : 'text-text-secondary hover:bg-bg-hover hover:text-text-primary'}"
              on:mouseenter={() => (selected = index)}
              on:click={() => onrun(item.id)}
            >
              <span class="truncate">{item.title}</span>
              {#if item.shortcut}
                <span class="shrink-0 font-mono text-[11px] text-text-muted">{item.shortcut}</span>
              {/if}
            </button>
          {/each}
        {/if}
      </div>
    </div>
  </div>
{/if}
