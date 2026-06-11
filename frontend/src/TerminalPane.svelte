<script lang="ts">
  import { onMount } from "svelte";
  import { FitAddon } from "@xterm/addon-fit";
  import { Terminal } from "@xterm/xterm";
  import "@xterm/xterm/css/xterm.css";
  import type { Pane } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import { ResizePTY, WritePTY } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";

  export let pane: Pane;
  export let output: string;
  export let focused = false;
  export let onFocus: () => void;
  export let onInput: () => void;

  let host: HTMLDivElement;
  let terminal: Terminal;
  let fitAddon: FitAddon;
  let resizeObserver: ResizeObserver;
  let written = 0;
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

  onMount(() => {
    terminal = new Terminal({
      cursorBlink: true,
      fontFamily: "SFMono-Regular, Menlo, Monaco, Consolas, monospace",
      fontSize: 13,
      lineHeight: 1.16,
      theme: {
        background: "#0b0f14",
        foreground: "#d7dde8",
        cursor: "#f3c969",
        selectionBackground: "#31445f",
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
    if (output) {
      terminal.write(output);
      written = output.length;
    }
    return () => {
      resizeObserver.disconnect();
      terminal.dispose();
    };
  });

  $: if (terminal && output.length > written) {
    terminal.write(output.slice(written));
    written = output.length;
  }
</script>

<div
  class:focused
  class="terminal-frame"
  on:click={focusTerminal}
  on:keydown={(event) => {
    if (event.key === "Enter" || event.key === " ") focusTerminal();
  }}
  role="button"
  tabindex="0"
  aria-label={`Focus pane ${pane.id}`}
>
  <div class="pane-header">
    <span>{pane.id}</span>
    <small>{pane.ptyId}</small>
  </div>
  <div bind:this={host} class="terminal-host"></div>
</div>
