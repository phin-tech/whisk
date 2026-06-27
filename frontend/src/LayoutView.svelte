<script lang="ts">
  import type { LayoutNode, Pane } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import TerminalPane from "./TerminalPane.svelte";

  export let node: LayoutNode;
  export let panes: { [_ in string]?: Pane };
  export let outputChunks: Record<string, string[]>;
  export let outputChunkStartOffsets: Record<string, number[]> = {};
  export let bottomJumpRevisions: Record<string, number> = {};
  export let activePaneId: string;
  export let terminalFontSize = 13;
  export let terminalCursorBlink = true;
  export let onFocus: (paneId: string) => void;
  export let onInput: (ptyId: string) => void;
  export let onWriteInput: (ptyId: string, data: string) => Promise<void>;
  export let onClose: (paneId: string) => void;
  export let onKillPTY: (paneId: string) => void;
  export let canClose: (paneId: string) => boolean;
</script>

{#if node.kind === "leaf"}
  {@const pane = panes[node.paneId ?? ""]}
  {#if pane}
    {#key pane.currentPtyId ?? pane.id}
      <TerminalPane
        pane={pane}
        outputChunks={pane.currentPtyId ? (outputChunks[pane.currentPtyId] ?? []) : []}
        chunkStartOffsets={pane.currentPtyId ? (outputChunkStartOffsets[pane.currentPtyId] ?? []) : []}
        jumpRevision={0}
        bottomRevision={pane.currentPtyId ? (bottomJumpRevisions[pane.currentPtyId] ?? 0) : 0}
        focused={activePaneId === pane.id}
        fontSize={terminalFontSize}
        cursorBlink={terminalCursorBlink}
        onFocus={() => onFocus(pane.id)}
        onInput={onInput}
        onWriteInput={onWriteInput}
        onClose={() => onClose(pane.id)}
        onKillPTY={() => onKillPTY(pane.id)}
        canClose={canClose(pane.id)}
      />
    {/key}
  {/if}
{:else}
  <div
    class="flex min-h-0 min-w-0 flex-1 gap-px bg-hairline {node.direction ===
    'horizontal'
      ? 'flex-row'
      : 'flex-col'}"
  >
    {#each node.children ?? [] as child}
      <div class="flex min-h-0 min-w-0 flex-1 flex-col">
      <svelte:self
        node={child}
        panes={panes}
        {outputChunks}
        {outputChunkStartOffsets}
        {bottomJumpRevisions}
        activePaneId={activePaneId}
        {terminalFontSize}
        {terminalCursorBlink}
        onFocus={onFocus}
        onInput={onInput}
        onWriteInput={onWriteInput}
        onClose={onClose}
        onKillPTY={onKillPTY}
        canClose={canClose}
      />
      </div>
    {/each}
  </div>
{/if}
