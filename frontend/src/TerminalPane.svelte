<script lang="ts">
  import { onMount } from "svelte";
  import { FitAddon } from "@xterm/addon-fit";
  import Check from "@lucide/svelte/icons/check";
  import CircleStop from "@lucide/svelte/icons/circle-stop";
  import Clipboard from "@lucide/svelte/icons/clipboard";
  import X from "@lucide/svelte/icons/x";
  import { Terminal } from "@xterm/xterm";
  import "@xterm/xterm/css/xterm.css";
  import type { Pane } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import { ResizePTY } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import { terminalInputRefreshDelays, terminalInputShouldRefreshOutput, type TerminalSnapshot } from "./ptyStream";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";

  export let pane: Pane;
  export let terminalSnapshot: TerminalSnapshot | null = null;
  export let outputChunks: Uint8Array[] = [];
  export let chunkStartOffsets: number[] = [];
  export let jumpRevision = 0;
  export let bottomRevision = 0;
  export let focused = false;
  export let fontSize = 13;
  export let cursorBlink = true;
  export let onFocus: () => void;
  export let onInput: (ptyId: string) => void;
  export let onWriteInput: (ptyId: string, data: string) => Promise<void>;
  export let onClose: () => void;
  export let onKillPTY: () => void;
  export let canClose = false;

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
  let appliedSnapshotKey = "";
  let appliedJumpRevision = 0;
  let appliedBottomRevision = 0;
  let scrollToReplayStart = false;
  let terminalWriting = false;
  let pendingTerminalOperations: TerminalOperation[] = [];

  type TerminalOperation =
    | { kind: "write"; bytes: Uint8Array }
    | { kind: "scroll-top" };

  const encoder = new TextEncoder();

  function cssToken(name: string, fallback: string) {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
  }

  function terminalTheme() {
    return {
      background: cssToken("--color-terminal-surface", "rgb(9, 9, 11)"),
      foreground: cssToken("--color-terminal-foreground", "rgb(250, 250, 250)"),
      cursor: cssToken("--color-terminal-cursor", "rgb(125, 211, 252)"),
      selectionBackground: cssToken("--color-terminal-selection", "rgba(39, 39, 42, 0.72)"),
    };
  }

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

  function closePane(event: MouseEvent) {
    event.stopPropagation();
    if (!canClose) return;
    onClose();
  }

  function killPTY(event: MouseEvent) {
    event.stopPropagation();
    if (!pane.currentPtyId) return;
    onKillPTY();
  }

  function resetRenderedTerminal(nextPtyId: string) {
    pendingTerminalOperations = [];
    terminalWriting = false;
    terminal.reset();
    writtenChunks = 0;
    renderedPtyId = nextPtyId;
    appliedSnapshotKey = "";
  }

  function enqueueTerminalOperation(operation: TerminalOperation) {
    pendingTerminalOperations.push(operation);
    drainTerminalOperations();
  }

  function drainTerminalOperations() {
    if (!terminal || terminalWriting) return;
    const operation = pendingTerminalOperations.shift();
    if (!operation) return;

    if (operation.kind === "write") {
      terminalWriting = true;
      terminal.write(operation.bytes, () => {
        terminalWriting = false;
        drainTerminalOperations();
      });
      return;
    }
    if (operation.kind === "scroll-top") {
      terminal.scrollToTop();
    }
    drainTerminalOperations();
  }

  function enqueueWrite(bytes: Uint8Array) {
    if (bytes.length === 0) return;
    enqueueTerminalOperation({ kind: "write", bytes });
  }

  function enqueueWriteText(text: string | undefined) {
    if (!text) return;
    enqueueWrite(encoder.encode(text));
  }

  function writeOutputChunk(chunk: Uint8Array, _chunkStartOffset: number | undefined) {
    enqueueWrite(chunk);
  }

  function snapshotKey(nextPtyId: string, snapshot: TerminalSnapshot | null) {
    if (!snapshot) return "";
    return `${nextPtyId}:${snapshot.offset}:${snapshot.cols}:${snapshot.rows}:${snapshot.truncated ? 1 : 0}`;
  }

  function applyTerminalSnapshot(nextPtyId: string, snapshot: TerminalSnapshot | null) {
    if (!terminal || !snapshot) return;
    const key = snapshotKey(nextPtyId, snapshot);
    if (renderedPtyId === nextPtyId && appliedSnapshotKey === key) return;
    resetRenderedTerminal(nextPtyId);
    appliedSnapshotKey = key;
    enqueueWriteText(snapshot.rehydrateBeforeViewport);
    enqueueWriteText(snapshot.scrollbackAnsi);
    enqueueWriteText(snapshot.viewportAnsi);
    enqueueWriteText(snapshot.rehydrateSequences);
  }

  function replayOutputChunks(nextPtyId: string, chunks: Uint8Array[], starts: number[]) {
    if (!terminal) return;
    if (renderedPtyId !== nextPtyId || chunks.length < writtenChunks) {
      resetRenderedTerminal(nextPtyId);
    }
    if (chunks.length <= writtenChunks) return;
    for (let index = writtenChunks; index < chunks.length; index += 1) {
      writeOutputChunk(chunks[index], starts[index]);
    }
    writtenChunks = chunks.length;
  }

  function applyJumpRevision(nextPtyId: string, nextJumpRevision: number) {
    if (!terminal || appliedJumpRevision === nextJumpRevision) return;
    appliedJumpRevision = nextJumpRevision;
    scrollToReplayStart = true;
    resetRenderedTerminal(nextPtyId);
  }

  function replayAndMaybeScroll(
    nextPtyId: string,
    snapshot: TerminalSnapshot | null,
    chunks: Uint8Array[],
    starts: number[],
    nextJumpRevision: number,
  ) {
    applyJumpRevision(nextPtyId, nextJumpRevision);
    applyTerminalSnapshot(nextPtyId, snapshot);
    replayOutputChunks(nextPtyId, chunks, starts);
    if (scrollToReplayStart && chunks.length > 0) {
      enqueueTerminalOperation({ kind: "scroll-top" });
      scrollToReplayStart = false;
    }
  }

  function applyBottomRevision(nextBottomRevision: number) {
    if (!terminal || appliedBottomRevision === nextBottomRevision) return;
    appliedBottomRevision = nextBottomRevision;
    terminal.scrollToBottom();
  }

  onMount(() => {
    terminal = new Terminal({
      allowProposedApi: true,
      cursorBlink,
      fontFamily: "SFMono-Regular, Menlo, Monaco, Consolas, monospace",
      fontSize,
      lineHeight: 1.16,
      overviewRulerWidth: 8,
      theme: terminalTheme(),
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
      onWriteInput(ptyId, data)
        .then(() => {
          if (terminalInputShouldRefreshOutput()) onInput(ptyId);
          for (const delay of terminalInputRefreshDelays()) {
            window.setTimeout(() => onInput(ptyId), delay);
          }
        })
        .catch(console.error);
    });
    replayAndMaybeScroll(pane.currentPtyId ?? "", terminalSnapshot, outputChunks, chunkStartOffsets, jumpRevision);
    return () => {
      resizeObserver.disconnect();
      pendingTerminalOperations = [];
      terminal.dispose();
    };
  });

  $: if (terminal) replayAndMaybeScroll(pane.currentPtyId ?? "", terminalSnapshot, outputChunks, chunkStartOffsets, jumpRevision);
  $: if (terminal) applyBottomRevision(bottomRevision);
  $: if (focused && terminal) terminal.focus();
</script>

<div
  class:focused
  class="group flex h-full min-h-0 min-w-0 flex-col overflow-hidden border border-border-subtle/70 bg-terminal-surface text-left text-terminal-foreground outline-none transition-[filter,border-color] duration-150 {focused
    ? 'border-accent-dim brightness-105'
    : 'brightness-90 hover:brightness-100'}"
  onclick={focusTerminal}
  onkeydown={(event) => {
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
    <div class="ml-auto flex min-w-0 items-center gap-1">
      {#if pane.currentPtyId}
        <Button
          type="button"
          variant="ghost"
          size="sm"
          align="start"
          class="h-5 min-w-0 px-1 py-0.5 font-mono text-[10px]"
          aria-label={`Copy PTY id ${pane.currentPtyId}`}
          title={`Copy PTY id: ${pane.currentPtyId}`}
          onclick={copyPtyId}
          onkeydown={(event: KeyboardEvent) => event.stopPropagation()}
        >
          <span class="truncate">{pane.currentPtyId}</span>
          {#if copiedPtyId === pane.currentPtyId}
            <Check size={11} />
          {:else}
            <Clipboard size={11} />
          {/if}
        </Button>
        <IconButton
          label={`Kill PTY ${pane.currentPtyId}`}
          title={`Kill PTY ${pane.currentPtyId}`}
          tone="danger"
          size="sm"
          onclick={killPTY}
          onkeydown={(event: KeyboardEvent) => event.stopPropagation()}
        >
          <CircleStop size={12} />
        </IconButton>
      {:else}
        <small class="truncate font-mono text-[10px] text-text-muted">empty</small>
      {/if}
      <IconButton
        label={`Close pane ${pane.id}`}
        title={canClose
          ? pane.currentPtyId
            ? `Close pane ${pane.id} and kill PTY ${pane.currentPtyId}`
            : `Close pane ${pane.id}`
          : "Cannot close the last pane"}
        disabled={!canClose}
        size="sm"
        onclick={closePane}
        onkeydown={(event: KeyboardEvent) => event.stopPropagation()}
      >
        <X size={12} />
      </IconButton>
    </div>
  </div>
  <div bind:this={host} class="min-h-0 min-w-0 flex-1 overflow-hidden"></div>
</div>
