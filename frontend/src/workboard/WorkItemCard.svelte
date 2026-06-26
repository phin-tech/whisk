<script lang="ts">
  import ArrowLeft from "@lucide/svelte/icons/arrow-left";
  import ArrowRight from "@lucide/svelte/icons/arrow-right";
  import Clock3 from "@lucide/svelte/icons/clock-3";
  import GitBranch from "@lucide/svelte/icons/git-branch";
  import Play from "@lucide/svelte/icons/play";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
  import type { WorkflowStage } from "../../bindings/github.com/phin-tech/whisk/internal/domain/workitem/models";
  import type { WorkItem, WorkItemRun } from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import type { WorkItemAttention, WorkItemAttentionSignal, WorkItemCardIndicator } from "../workView";
  import Button from "../ui/Button.svelte";
  import CardIndicators from "../ui/CardIndicators.svelte";
  import IconButton from "../ui/IconButton.svelte";
  import StatusDot from "../ui/StatusDot.svelte";

  type Targets = {
    previous: WorkflowStage | null;
    next: WorkflowStage | null;
    blockedNext: WorkflowStage | null;
  };

  export let item: WorkItem;
  export let targets: Targets;
  export let latestRun: WorkItemRun | null = null;
  export let terminalRun: WorkItemRun | null = null;
  export let attention: WorkItemAttention;
  export let indicators: WorkItemCardIndicator[] = [];
  export let canExecute = false;
  export let loading = false;
  export let cardRailClass: (severity: string) => string;
  export let attentionDotClass: (tone: string) => string;
  export let onOpenDetail: (item: WorkItem) => void;
  export let onOpenRunTerminal: (run: WorkItemRun) => void;
  export let onLaunchRun: (runId: string) => void;
  export let onQueueExecution: (workItemId: string) => void;
  export let onLaunchExecution: (workItemId: string) => void;
  export let onGenerateWorktree: (item: WorkItem) => void;
  export let onMovePrevious: (item: WorkItem) => void;
  export let onMoveNext: (item: WorkItem) => void;

  $: visibleAttentionSignals = attentionSignalsNotShownByIndicators(attention.signals, indicators);

  function attentionSignalsNotShownByIndicators(
    signals: WorkItemAttentionSignal[],
    cardIndicators: WorkItemCardIndicator[],
  ) {
    const hidden = new Set(
      cardIndicators.map((indicator) => {
        if (indicator.id === "run-queued") return "queued";
        if (indicator.id === "run-awaiting-input") return "awaiting-input";
        if (indicator.id === "review-gate") return "blocking-gates";
        return indicator.id;
      }),
    );
    return signals.filter((signal) => !hidden.has(signal.id));
  }
</script>

<article class="group relative min-h-[76px] overflow-hidden bg-bg-base transition-colors hover:bg-bg-surface/60 focus-within:bg-bg-surface/60">
  <div class="absolute inset-y-3 left-0 w-1 rounded-r {cardRailClass(attention.severity)}"></div>
  <div class="grid gap-2 px-3 py-3 pl-4">
    <div class="flex min-w-0 items-start gap-2">
      <Button
        variant="ghost"
        align="start"
        class="work-card-title min-w-0 flex-1 border-transparent bg-transparent p-0 text-left text-[14px] font-semibold leading-5 text-text-primary hover:bg-transparent hover:text-accent focus-visible:text-accent"
        onclick={() => onOpenDetail(item)}
      >
        <span class="font-mono text-[12px] font-medium text-text-muted">#{item.number}</span>
        {item.title}
      </Button>
      {#if terminalRun}
        <IconButton
          label="Open running terminal"
          class="animate-pulse border-green/45 bg-green/12 text-green hover:border-green"
          disabled={loading}
          onclick={() => onOpenRunTerminal(terminalRun)}
        >
          <SquareTerminal size={14} />
        </IconButton>
      {/if}
      <div class="flex shrink-0 gap-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
        {#if latestRun?.status === "queued"}
          <IconButton label="Launch queued run" class="border-blue/35 bg-blue/10 text-blue hover:border-blue" disabled={loading} onclick={() => onLaunchRun(latestRun.id)}>
            <Play size={13} />
          </IconButton>
        {/if}
        {#if canExecute}
          <IconButton label="Queue execution" class="border-blue/35 bg-blue/10 text-blue hover:border-blue" disabled={loading} onclick={() => onQueueExecution(item.id)}>
            <Clock3 size={13} />
          </IconButton>
          <IconButton label="Launch execution" class="border-green/35 bg-green/10 text-green hover:border-green" disabled={loading} onclick={() => onLaunchExecution(item.id)}>
            <Play size={13} />
          </IconButton>
        {/if}
        {#if targets.blockedNext && !item.worktree}
          <IconButton label="Generate worktree" disabled={loading} onclick={() => onGenerateWorktree(item)}>
            <GitBranch size={13} />
          </IconButton>
        {/if}
        <IconButton label="Move previous" disabled={loading || !targets.previous} onclick={() => onMovePrevious(item)}>
          <ArrowLeft size={13} />
        </IconButton>
        <IconButton
          label={targets.blockedNext ? "Generate worktree before moving" : "Move next"}
          disabled={loading || !targets.next}
          onclick={() => onMoveNext(item)}
        >
          <ArrowRight size={13} />
        </IconButton>
      </div>
    </div>

    <div class="grid min-w-0 gap-1">
      <CardIndicators {indicators} />
      {#if visibleAttentionSignals.length > 0}
        <div class="flex min-w-0 flex-wrap items-center gap-1.5">
          {#each visibleAttentionSignals as signal (signal.id)}
            <span class="inline-flex min-w-0 items-center gap-1 text-[12px]">
              <span class={attentionDotClass(signal.tone)}>●</span>
              <span class="truncate text-text-muted">{signal.label}</span>
            </span>
          {/each}
        </div>
      {:else if indicators.length === 0}
        <StatusDot status={item.runState || "idle"} label={item.runState || "Idle"} showLabel class="text-[12px]" />
      {/if}
    </div>
  </div>
</article>
