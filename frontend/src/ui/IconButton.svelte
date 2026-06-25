<script lang="ts">
  import type { Snippet } from "svelte";
  import Button from "./Button.svelte";

  type Props = {
    label: string;
    title?: string;
    tone?: "default" | "danger";
    size?: "sm" | "md";
    disabled?: boolean;
    type?: "button" | "submit" | "reset";
    onclick?: (event: MouseEvent) => void;
    class?: string;
    children?: Snippet;
    [key: string]: unknown;
  };

  let {
    label,
    title = label,
    tone = "default",
    size = "md",
    disabled = false,
    type = "button",
    onclick,
    class: className = "",
    children,
    ...restProps
  }: Props = $props();

  const variant = $derived(tone === "danger" ? "danger-ghost" : "ghost");
  const buttonSize = $derived(size === "sm" ? "icon-sm" : "icon");
</script>

<Button
  {type}
  {disabled}
  {title}
  {variant}
  size={buttonSize}
  aria-label={label}
  class={className}
  {onclick}
  {...restProps}
>
  {@render children?.()}
</Button>
