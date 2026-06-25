<script lang="ts">
  import type { Snippet } from "svelte";

  type Props = {
    title: string;
    eyebrow?: string;
    description?: string;
    class?: string;
    children?: Snippet;
    [key: string]: unknown;
  };

  let {
    title,
    eyebrow = "",
    description = "",
    class: className = "",
    children,
    ...restProps
  }: Props = $props();

  const classes = $derived(`flex min-w-0 items-start justify-between gap-3 ${className}`.trim());
</script>

<header class={classes} {...restProps}>
  <div class="min-w-0">
    {#if eyebrow}
      <p class="mb-1 text-[11px] font-semibold uppercase text-text-muted">{eyebrow}</p>
    {/if}
    <h2 class="truncate text-[11px] font-semibold uppercase text-text-muted">{title}</h2>
    {#if description}
      <p class="mt-1 text-[12px] leading-5 text-text-secondary">{description}</p>
    {/if}
  </div>
  {#if children}
    <div class="flex shrink-0 items-center gap-1">
      {@render children?.()}
    </div>
  {/if}
</header>
