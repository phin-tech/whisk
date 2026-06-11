<script lang="ts">
  import { onMount } from "svelte";
  import { FitAddon } from "@xterm/addon-fit";
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
  export let onInput: () => void;

  let host: HTMLDivElement;
  let terminal: Terminal;
  let fitAddon: FitAddon;
  let resizeObserver: ResizeObserver;
  let writtenChunks = 0;
  let lastCols = 0;
  let lastRows = 0;

  function fitAndResize() {
    if (!terminal || !fitAddon || !host.offsetWidth || !host.offsetHeight) return;
    fitAddon.fit();
    const { cols, rows } = terminal;
    if (cols === lastCols && rows === lastRows) return;
    lastCols = cols;
    lastRows = rows;
    ResizePTY({ ptyId: pane.ptyId, cols, rows }).catch(console.error);
  }

  function focusTerminal() {
    onFocus();
    terminal?.focus();
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
      WritePTY({ ptyId: pane.ptyId, data })
        .then(() => {
          onInput();
          window.setTimeout(onInput, 16);
        })
        .catch(console.error);
    });
    if (outputChunks.length > 0) {
      for (const chunk of outputChunks) writeBase64Chunk(chunk);
      writtenChunks = outputChunks.length;
    }
    return () => {
      resizeObserver.disconnect();
      terminal.dispose();
    };
  });

  $: if (terminal && outputChunks.length > writtenChunks) {
    for (const chunk of outputChunks.slice(writtenChunks)) {
      writeBase64Chunk(chunk);
    }
    writtenChunks = outputChunks.length;
  }
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
    <small class="truncate font-mono text-[10px] text-text-muted">{pane.ptyId}</small>
  </div>
  <div bind:this={host} class="min-h-0 min-w-0 flex-1 overflow-hidden"></div>
</div>
