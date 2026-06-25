<script lang="ts">
  import type { Snippet } from "svelte";
  import Button from "./Button.svelte";

  type Props = {
    active?: boolean;
    tone?: "default" | "danger";
    disabled?: boolean;
    onclick?: (event: MouseEvent) => void;
    class?: string;
    children?: Snippet;
    [key: string]: unknown;
  };

  let {
    active = false,
    tone = "default",
    disabled = false,
    onclick,
    class: className = "",
    children,
    ...restProps
  }: Props = $props();

  const toneClass = $derived(
    tone === "danger" ? "text-red hover:bg-red/10 hover:text-red" : "text-text-secondary hover:bg-bg-surface/80 hover:text-text-primary",
  );
  const activeClass = $derived(active ? "text-accent" : "");
  const classes = $derived(
    `h-auto w-full rounded-none border-transparent px-3 py-1.5 text-[12px] font-medium ${toneClass} ${activeClass} ${className}`.trim(),
  );
</script>

<Button
  type="button"
  variant={tone === "danger" ? "danger-ghost" : "ghost"}
  size="md"
  align="start"
  role="menuitem"
  {disabled}
  class={classes}
  {onclick}
  {...restProps}
>
  {@render children?.()}
</Button>
