<script lang="ts">
  import type { Snippet } from "svelte";
  import type { NextStepView } from "../workView";
  import Button from "./Button.svelte";

  type RunnableNextStep = NextStepView & { run?: () => void };
  type Props = {
    step: RunnableNextStep;
    disabled?: boolean;
    onrun?: () => void;
    class?: string;
    controls?: Snippet;
    [key: string]: unknown;
  };

  let {
    step,
    disabled = false,
    onrun,
    class: className = "",
    controls,
    ...restProps
  }: Props = $props();

  const classes = $derived(
    `flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-hairline bg-bg-base/60 px-5 py-2.5 ${className}`.trim(),
  );
  const toneClass = $derived(
    step.tone === "accent"
      ? "border-green/40 bg-green/15 text-green hover:border-green"
      : step.tone === "primary"
        ? "border-amber/45 bg-amber/12 text-amber hover:border-amber"
        : "border-accent-dim bg-accent-dim text-text-primary hover:border-accent hover:text-accent",
  );

  function run() {
    if (onrun) {
      onrun();
      return;
    }
    step.run?.();
  }
</script>

<div class={classes} {...restProps}>
  <div class="flex min-w-0 items-center gap-2">
    <span class="shrink-0 text-[10px] font-semibold uppercase tracking-widest text-text-muted">Next</span>
    <span class="min-w-0 text-[13px] leading-5 text-text-primary">{step.message}</span>
  </div>
  <div class="flex shrink-0 flex-wrap items-center gap-2">
    {@render controls?.()}
    {#if step.label}
      <Button
        type="button"
        variant="outline"
        class={`h-8 justify-center px-3 text-[12px] ${toneClass}`}
        {disabled}
        onclick={run}
      >
        {step.label}
      </Button>
    {/if}
  </div>
</div>
