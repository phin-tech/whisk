<script lang="ts">
  import type { Snippet } from "svelte";
  import TextField from "./TextField.svelte";

  type Meta = {
    value: string | number;
    label: string;
  };
  type Props = {
    value?: string;
    meta?: Meta[];
    disabled?: boolean;
    ariaLabel?: string;
    oncommit?: () => void;
    class?: string;
    icon?: Snippet;
    actions?: Snippet;
    children?: Snippet;
    [key: string]: unknown;
  };

  let {
    value = $bindable(""),
    meta = [],
    disabled = false,
    ariaLabel = "Panel title",
    oncommit = () => {},
    class: className = "",
    icon,
    actions,
    children,
    ...restProps
  }: Props = $props();

  const classes = $derived(
    `shrink-0 border-b border-hairline bg-bg-deep px-4 py-2.5 ${className}`.trim(),
  );
</script>

<header class={classes} {...restProps}>
  <div class="flex min-w-0 items-center gap-2">
    {@render icon?.()}
    <TextField
      variant="seamless"
      bind:value
      {disabled}
      aria-label={ariaLabel}
      class="min-w-0 flex-1 px-1 text-[15px] font-medium"
      onblur={oncommit}
    />
    {@render actions?.()}
  </div>
  {#if meta.length > 0}
    <div class="mt-1 flex items-center gap-1.5 text-[11px] text-text-muted">
      {#each meta as item, index (`${item.label}:${item.value}`)}
        {#if index > 0}
          <span class="opacity-40">·</span>
        {/if}
        <span class="font-mono">{item.value}</span><span>{item.label}</span>
      {/each}
    </div>
  {/if}
  {@render children?.()}
</header>
