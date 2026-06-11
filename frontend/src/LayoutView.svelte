<script lang="ts">
  import type { LayoutNode, Pane } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import TerminalPane from "./TerminalPane.svelte";

  export let node: LayoutNode;
  export let panes: { [_ in string]?: Pane };
  export let outputs: Record<string, string>;
  export let activePaneId: string;
  export let onFocus: (paneId: string) => void;
  export let onInput: () => void;
</script>

{#if node.kind === "leaf"}
  {@const pane = panes[node.paneId ?? ""]}
  {#if pane}
    <TerminalPane
      pane={pane}
      output={outputs[pane.ptyId] ?? ""}
      focused={activePaneId === pane.id}
      onFocus={() => onFocus(pane.id)}
      onInput={onInput}
    />
  {/if}
{:else}
  <div class:row={node.direction === "horizontal"} class:column={node.direction === "vertical"} class="split">
    {#each node.children ?? [] as child}
      <svelte:self
        node={child}
        panes={panes}
        outputs={outputs}
        activePaneId={activePaneId}
        onFocus={onFocus}
        onInput={onInput}
      />
    {/each}
  </div>
{/if}
