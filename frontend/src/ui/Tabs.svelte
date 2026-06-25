<script lang="ts">
  import type { Component, Snippet } from "svelte";

  type Tab = {
    id: string;
    label: string;
    count?: number;
    icon?: Component;
  };
  type Props = {
    tabs: Tab[];
    active?: string;
    onchange?: (tabId: string) => void;
    class?: string;
    children?: Snippet;
  };

  let { tabs, active = $bindable(""), onchange = () => {}, class: className = "", children }: Props =
    $props();

  const classes = $derived(`shrink-0 border-b border-hairline px-5 ${className}`.trim());

  function select(tabId: string) {
    active = tabId;
    onchange(tabId);
  }
</script>

<div class={classes}>
  <div class="flex min-h-10 items-end gap-1 overflow-x-auto">
    {#each tabs as tab (tab.id)}
      {@const Icon = tab.icon}
      <button
        type="button"
        class="inline-flex h-10 items-center gap-1.5 border-b px-3 text-[12px] font-medium transition-colors {active ===
        tab.id
          ? 'border-accent text-text-primary'
          : 'border-transparent text-text-muted hover:text-text-primary'}"
        onclick={() => select(tab.id)}
      >
        {#if Icon}
          <Icon size={14} />
        {/if}
        <span>{tab.label}</span>
        {#if tab.count && tab.count > 0}
          <span class="font-mono text-[10px] text-text-muted">{tab.count}</span>
        {/if}
      </button>
    {/each}
    {@render children?.()}
  </div>
</div>
