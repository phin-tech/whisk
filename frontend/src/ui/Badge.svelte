<script lang="ts">
  import type { Snippet } from "svelte";

  type Tone = "muted" | "accent" | "green" | "blue" | "amber" | "red";
  type Props = {
    tone?: Tone;
    label?: string;
    class?: string;
    children?: Snippet;
    [key: string]: unknown;
  };

  let { tone = "muted", label = "", class: className = "", children, ...restProps }: Props = $props();

  const toneClass = $derived(
    tone === "accent"
      ? "border-accent/35 bg-accent-dim/10 text-accent"
      : tone === "green"
        ? "border-green/30 bg-green/10 text-green"
        : tone === "blue"
          ? "border-blue/30 bg-blue/10 text-blue"
          : tone === "amber"
            ? "border-amber/30 bg-amber/10 text-amber"
            : tone === "red"
              ? "border-red/30 bg-red/10 text-red"
              : "border-border-subtle bg-bg-surface/60 text-text-muted",
  );
  const classes = $derived(
    `inline-flex h-5 max-w-full items-center rounded border px-1.5 text-[11px] font-medium ${toneClass} ${className}`.trim(),
  );
</script>

<span class={classes} {...restProps}>
  <span class="min-w-0 truncate">
    {#if children}
      {@render children?.()}
    {:else}
      {label}
    {/if}
  </span>
</span>
