<script lang="ts">
  import type { Snippet } from "svelte";

  type Props = {
    label: string;
    class?: string;
    valueClass?: string;
    children?: Snippet;
    [key: string]: unknown;
  };

  let {
    label,
    class: className = "",
    valueClass = "",
    children,
    ...restProps
  }: Props = $props();

  const classes = $derived(`flex items-center justify-between gap-2 px-3 py-2 ${className}`.trim());
  const valueClasses = $derived(`flex min-w-0 max-w-[60%] items-center gap-2 ${valueClass}`.trim());
</script>

<div class={classes} {...restProps}>
  <span class="text-[11px] font-semibold uppercase text-text-muted">{label}</span>
  <div class={valueClasses}>
    {@render children?.()}
  </div>
</div>
