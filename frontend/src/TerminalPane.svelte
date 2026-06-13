<script lang="ts">
  import { onMount } from "svelte";
  import { FitAddon } from "@xterm/addon-fit";
  import Check from "@lucide/svelte/icons/check";
  import Clipboard from "@lucide/svelte/icons/clipboard";
  import { Terminal } from "@xterm/xterm";
  import "@xterm/xterm/css/xterm.css";
  import type { Pane } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import { ResizePTY, WritePTY } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";

  export let pane: Pane;
  export let outputChunks: string[] = [];
  export let focused = false;
  export let fontSize = 13;
  export let cursorBlink = true;
  export let onFocus: () => void;
  export let onInput: (ptyId: string) => void;

  let host: HTMLDivElement;
  let terminal: Terminal;
  let fitAddon: FitAddon;
  let resizeObserver: ResizeObserver;
  let writtenChunks = 0;
  let lastCols = 0;
  let lastRows = 0;
  let copiedPtyId = "";
  let copiedTimer: ReturnType<typeof setTimeout> | null = null;
  let renderedPtyId = "";

  function fitAndResize() {
    if (!pane.currentPtyId || !terminal || !fitAddon || !host.offsetWidth || !host.offsetHeight) return;
    fitAddon.fit();
    const { cols, rows } = terminal;
    if (cols === lastCols && rows === lastRows) return;
    lastCols = cols;
    lastRows = rows;
    ResizePTY({ ptyId: pane.currentPtyId, cols, rows }).catch(console.error);
  }

  function focusTerminal() {
    onFocus();
    terminal?.focus();
  }

  async function copyPtyId(event: MouseEvent) {
    event.stopPropagation();
    if (!pane.currentPtyId) return;
    await navigator.clipboard.writeText(pane.currentPtyId);
    copiedPtyId = pane.currentPtyId;
    if (copiedTimer) clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      copiedPtyId = "";
      copiedTimer = null;
    }, 1200);
  }

  function writeBase64Chunk(chunk: string) {
    if (!chunk) return;
    const binary = atob(chunk);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i += 1) {
      bytes[i] = binary.charCodeAt(i);
    }
    terminal.write(bytes);
  }

  function replayOutputChunks(nextPtyId: string, chunks: string[]) {
    if (!terminal) return;
    if (renderedPtyId !== nextPtyId || chunks.length < writtenChunks) {
      terminal.reset();
      writtenChunks = 0;
      renderedPtyId = nextPtyId;
    }
    if (chunks.length <= writtenChunks) return;
    for (const chunk of chunks.slice(writtenChunks)) {
      writeBase64Chunk(chunk);
    }
    writtenChunks = chunks.length;
  }

  onMount(() => {
    terminal = new Terminal({
      cursorBlink,
      fontFamily: "SFMono-Regular, Menlo, Monaco, Consolas, monospace",
      fontSize,
      lineHeight: 1.16,
      theme: {
        background: "#09090b",
        foreground: "#fafafa",
        cursor: "#7dd3fc",
        selectionBackground: "#27272acc",
      },
    });
    fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(host);
    fitAndResize();
    resizeObserver = new ResizeObserver(fitAndResize);
    resizeObserver.observe(host);
    window.requestAnimationFrame(fitAndResize);
    terminal.onData((data) => {
      const ptyId = pane.currentPtyId;
      if (!ptyId) return;
      WritePTY({ ptyId, data })
        .then(() => {
          onInput(ptyId);
          window.setTimeout(() => onInput(ptyId), 50);
          window.setTimeout(() => onInput(ptyId), 200);
        })
        .catch(console.error);
    });
    replayOutputChunks(pane.currentPtyId ?? "", outputChunks);
    return () => {
      resizeObserver.disconnect();
      terminal.dispose();
    };
  });

  $: if (terminal) replayOutputChunks(pane.currentPtyId ?? "", outputChunks);
</script>

<div
  class:focused
  class="group flex h-full min-h-0 min-w-0 flex-col overflow-hidden border border-border-subtle/70 bg-bg-deep text-left text-text-primary outline-none transition-[filter,border-color] duration-150 {focused
    ? 'border-accent-dim brightness-105'
    : 'brightness-90 hover:brightness-100'}"
  on:click={focusTerminal}
  on:keydown={(event) => {
    if (event.key === "Enter" || event.key === " ") focusTerminal();
  }}
  role="button"
  tabindex="0"
  aria-label={`Focus pane ${pane.id}`}
>
  <div
    class="flex h-7 shrink-0 items-center justify-between gap-2 border-b border-hairline bg-bg-base/95 px-2 text-[11px]"
  >
    <span class="truncate font-medium text-text-secondary">{pane.id}</span>
    {#if pane.currentPtyId}
      <button
        type="button"
        class="inline-flex min-w-0 items-center gap-1 rounded border border-transparent px-1 py-0.5 font-mono text-[10px] text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-surface hover:text-text-primary focus:outline-none focus:ring-1 focus:ring-accent"
        aria-label={`Copy PTY id ${pane.currentPtyId}`}
        title={`Copy PTY id: ${pane.currentPtyId}`}
        on:click={copyPtyId}
        on:keydown|stopPropagation
      >
        <span class="truncate">{pane.currentPtyId}</span>
        {#if copiedPtyId === pane.currentPtyId}
          <Check size={11} />
        {:else}
          <Clipboard size={11} />
        {/if}
      </button>
    {:else}
      <small class="truncate font-mono text-[10px] text-text-muted">empty</small>
    {/if}
  </div>
  <div bind:this={host} class="min-h-0 min-w-0 flex-1 overflow-hidden"></div>
</div>
