<script lang="ts">
  import { Button as BitsButton } from "bits-ui";
  import type { Snippet } from "svelte";

  type Variant = "primary" | "outline" | "ghost" | "danger" | "danger-ghost";
  type Size = "sm" | "md" | "lg" | "icon" | "icon-sm";
  type Align = "center" | "start" | "between";
  type Props = {
    variant?: Variant;
    size?: Size;
    align?: Align;
    type?: "button" | "submit" | "reset";
    href?: string;
    disabled?: boolean;
    onclick?: (event: MouseEvent) => void;
    class?: string;
    children?: Snippet;
    [key: string]: unknown;
  };

  let {
    variant = "outline",
    size = "md",
    align = "center",
    type = "button",
    href,
    disabled = false,
    onclick,
    class: className = "",
    children,
    ...restProps
  }: Props = $props();

  const base =
    "inline-flex items-center gap-1 rounded border font-semibold transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-not-allowed disabled:opacity-50";

  const variantClass = $derived(
    variant === "primary"
      ? "border-accent-dim bg-accent-dim text-text-primary hover:border-accent hover:text-accent"
      : variant === "ghost"
        ? "border-transparent bg-transparent text-text-muted hover:border-accent/40 hover:bg-accent-dim/10 hover:text-accent"
        : variant === "danger"
          ? "border-red/35 bg-red/10 text-red hover:border-red"
          : variant === "danger-ghost"
            ? "border-transparent bg-transparent text-text-muted hover:border-red/40 hover:bg-red/10 hover:text-red"
            : "border-border-subtle bg-bg-surface/60 text-text-secondary hover:border-accent hover:text-accent",
  );

  const sizeClass = $derived(
    size === "icon"
      ? "h-7 w-7 p-0"
      : size === "icon-sm"
        ? "h-5 w-5 p-0"
        : size === "lg"
          ? "h-9 px-3 text-[12px]"
          : size === "sm"
            ? "h-7 px-2 text-[11px]"
            : "h-8 px-2.5 text-[12px]",
  );
  const alignClass = $derived(
    align === "between" ? "justify-between text-left" : align === "start" ? "justify-start text-left" : "justify-center",
  );

  const classes = $derived(`${base} ${variantClass} ${sizeClass} ${alignClass} ${className}`.trim());
</script>

{#if href}
  <BitsButton.Root {href} {disabled} class={classes} {onclick} {...restProps}>
    {@render children?.()}
  </BitsButton.Root>
{:else}
  <BitsButton.Root {type} {disabled} class={classes} {onclick} {...restProps}>
    {@render children?.()}
  </BitsButton.Root>
{/if}
