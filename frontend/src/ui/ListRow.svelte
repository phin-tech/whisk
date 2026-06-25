<script lang="ts">
  import type { Snippet } from "svelte";

  type RowElement = "button" | "div";
  type Props = {
    as?: RowElement;
    cols?: string;
    type?: "button" | "submit" | "reset";
    disabled?: boolean;
    role?: string;
    tabindex?: number;
    onclick?: (event: MouseEvent) => void;
    onkeydown?: (event: KeyboardEvent) => void;
    class?: string;
    children?: Snippet;
    [key: string]: unknown;
  };

  let {
    as = "div",
    cols = "",
    type = "button",
    disabled = false,
    role,
    tabindex,
    onclick,
    onkeydown,
    class: className = "",
    children,
    ...restProps
  }: Props = $props();

  const base = "w-full min-w-0 px-3 py-2 text-left transition-colors hover:bg-bg-surface/40";
  const gridClass = $derived(cols ? `grid ${cols} items-center gap-3` : "");
  const disabledClass = $derived(disabled ? "cursor-not-allowed opacity-50" : "");
  const classes = $derived(`${base} ${gridClass} ${disabledClass} ${className}`.trim());
  const element = $derived(as);
</script>

<svelte:element
  this={element}
  {type}
  {disabled}
  {role}
  {tabindex}
  class={classes}
  {onclick}
  {onkeydown}
  {...restProps}
>
  {@render children?.()}
</svelte:element>
