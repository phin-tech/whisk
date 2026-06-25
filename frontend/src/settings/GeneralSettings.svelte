<script lang="ts">
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Button from "../ui/Button.svelte";

  type StartupView = "sessions" | "kanban";
  type RailSide = "left" | "right";
  type Props = {
    railSide: RailSide;
    startupView: StartupView;
    onRailSide: (side: RailSide) => void;
    onStartupView: (view: StartupView) => void;
    onRunOnboarding: () => void;
  };

  let { railSide, startupView, onRailSide, onStartupView, onRunOnboarding }: Props = $props();
</script>

<div class="rounded-xl border border-border-subtle bg-bg-surface/35 p-3">
  <div class="flex items-start justify-between gap-3">
    <div>
      <div class="text-[13px]">Theme</div>
      <div class="mt-0.5 text-[11px] text-text-muted">Refined Zinc</div>
    </div>
    <div class="rounded border border-border bg-bg-deep px-2 py-1 text-[12px] text-text-secondary">
      Default
    </div>
  </div>
</div>

<div class="mt-4 flex items-center justify-between gap-3 py-2">
  <div>
    <div class="text-[13px]">Open to</div>
    <div class="mt-0.5 text-[11px] text-text-muted">Initial workspace after launch.</div>
  </div>
  <div class="flex overflow-hidden rounded border border-border bg-bg-deep">
    {#each [{ id: "sessions", label: "Sessions" }, { id: "kanban", label: "Kanban" }] as option}
      <Button
        variant={startupView === option.id ? "primary" : "ghost"}
        size="sm"
        class="h-7 rounded-none border-transparent text-[11px] {startupView === option.id ? '' : 'bg-transparent'}"
        aria-pressed={startupView === option.id}
        onclick={() => onStartupView(option.id as StartupView)}
      >
        {option.label}
      </Button>
    {/each}
  </div>
</div>

<div class="mt-4 flex items-center justify-between gap-3 py-2">
  <div>
    <div class="text-[13px]">Onboarding</div>
    <div class="mt-0.5 text-[11px] text-text-muted">Agent hooks, skills, and plugin trust.</div>
  </div>
  <Button size="sm" onclick={onRunOnboarding}>
    <RefreshCw size={13} />
    <span>Re-run</span>
  </Button>
</div>

<div class="mt-4 flex items-center justify-between py-2">
  <div>
    <div class="text-[13px]">Activity rail</div>
    <div class="mt-0.5 text-[11px] text-text-muted">Position of the icon rail and sidebar dock.</div>
  </div>
  <div class="flex overflow-hidden rounded border border-border bg-bg-deep">
    {#each ["left", "right"] as side}
      <Button
        variant={railSide === side ? "primary" : "ghost"}
        size="sm"
        class="h-7 rounded-none border-transparent text-[11px] {railSide === side ? '' : 'bg-transparent'}"
        aria-pressed={railSide === side}
        onclick={() => onRailSide(side as RailSide)}
      >
        {side}
      </Button>
    {/each}
  </div>
</div>
