<script lang="ts">
  import { onMount } from "svelte";
  import type { WorkItem, WorkItemRun } from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import List from "../ui/List.svelte";
  import WorkItemCard from "./WorkItemCard.svelte";
  import type { WorkBoardCardView } from "./work-board-state";
  import { deriveWorkBoardCardWindow } from "./work-board-virtualization";

  const ROW_HEIGHT = 128;
  const OVERSCAN = 4;

  export let stageName = "";
  export let cards: WorkBoardCardView[] = [];
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

  let viewport: HTMLDivElement;
  let viewportHeight = ROW_HEIGHT * 8;
  let scrollOffset = 0;

  $: virtualWindow = deriveWorkBoardCardWindow({
    cards,
    rowHeight: ROW_HEIGHT,
    viewportHeight,
    scrollOffset,
    overscan: OVERSCAN,
  });

  function measureViewport() {
    if (!viewport) return;
    viewportHeight = viewport.clientHeight;
    scrollOffset = viewport.scrollTop;
  }

  onMount(() => {
    measureViewport();
    const resizeObserver = new ResizeObserver(measureViewport);
    resizeObserver.observe(viewport);
    return () => resizeObserver.disconnect();
  });
</script>

<div
  bind:this={viewport}
  class="app-scrollbar min-h-0 flex-1 overflow-y-auto"
  aria-label={`Work items in ${stageName}`}
  data-work-board-virtual-list
  onscroll={measureViewport}
>
  <List class="relative min-w-0" style={`height: ${virtualWindow.totalHeight}px;`}>
    {#each virtualWindow.cards as virtualCard (virtualCard.key)}
      <div
        class="absolute left-0 right-0 overflow-hidden bg-bg-base"
        style={`transform: translateY(${virtualCard.offsetTop}px); height: ${virtualCard.height}px;`}
        data-work-card-virtual-row
        data-work-card-key={virtualCard.key}
        data-work-card-index={virtualCard.index}
      >
        <WorkItemCard
          item={virtualCard.card.item}
          targets={virtualCard.card.targets}
          latestRun={virtualCard.card.latestRun}
          terminalRun={virtualCard.card.terminalRun}
          attention={virtualCard.card.attention}
          indicators={virtualCard.card.indicators}
          canExecute={virtualCard.card.canExecute}
          {loading}
          {cardRailClass}
          {attentionDotClass}
          {onOpenDetail}
          {onOpenRunTerminal}
          {onLaunchRun}
          {onQueueExecution}
          {onLaunchExecution}
          {onGenerateWorktree}
          onMovePrevious={onMovePrevious}
          onMoveNext={onMoveNext}
        />
      </div>
    {/each}
  </List>
</div>
