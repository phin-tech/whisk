<script lang="ts">
  import { onMount } from "svelte";
  import { FitAddon } from "@xterm/addon-fit";
  import BookmarkIcon from "@lucide/svelte/icons/bookmark";
  import Check from "@lucide/svelte/icons/check";
  import CircleStop from "@lucide/svelte/icons/circle-stop";
  import Clipboard from "@lucide/svelte/icons/clipboard";
  import X from "@lucide/svelte/icons/x";
  import { Terminal } from "@xterm/xterm";
  import "@xterm/xterm/css/xterm.css";
  import type { Pane } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type { PTYBookmark } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { ResizePTY } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import { terminalInputRefreshDelays, terminalInputShouldRefreshOutput } from "./ptyStream";
  import { ptyBookmarkRowsByPty } from "./sessionView";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";

  export let pane: Pane;
  export let outputChunks: string[] = [];
  export let bookmarks: PTYBookmark[] = [];
  export let jumpRevision = 0;
  export let focused = false;
  export let fontSize = 13;
  export let cursorBlink = true;
  export let onFocus: () => void;
  export let onInput: (ptyId: string) => void;
  export let onWriteInput: (ptyId: string, data: string) => Promise<void>;
  export let onBookmark: (bookmark: PTYBookmark) => void;
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
  let appliedJumpRevision = 0;
  let scrollToReplayStart = false;

  $: bookmarkRows = pane.currentPtyId
    ? (ptyBookmarkRowsByPty(bookmarks)[pane.currentPtyId] ?? [])
    : [];

  function cssToken(name: string, fallback: string) {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
  }

  function terminalTheme() {
    return {
      background: cssToken("--color-bg-deep", "rgb(9, 9, 11)"),
      foreground: cssToken("--color-text-primary", "rgb(250, 250, 250)"),
      cursor: cssToken("--color-accent", "rgb(125, 211, 252)"),
      selectionBackground: cssToken("--color-bg-active", "rgba(39, 39, 42, 0.72)"),
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

  function jumpBookmark(event: MouseEvent, bookmarkId: string) {
    event.stopPropagation();
    const bookmark = bookmarks.find((candidate) => candidate.id === bookmarkId);
    if (bookmark) onBookmark(bookmark);
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

  function applyJumpRevision(nextPtyId: string, nextJumpRevision: number) {
    if (!terminal || appliedJumpRevision === nextJumpRevision) return;
    appliedJumpRevision = nextJumpRevision;
    scrollToReplayStart = true;
    terminal.reset();
    writtenChunks = 0;
    renderedPtyId = nextPtyId;
  }

  function replayAndMaybeScroll(nextPtyId: string, chunks: string[], nextJumpRevision: number) {
    applyJumpRevision(nextPtyId, nextJumpRevision);
    replayOutputChunks(nextPtyId, chunks);
    if (scrollToReplayStart && chunks.length > 0) {
      terminal.scrollToTop();
      scrollToReplayStart = false;
    }
  }

  onMount(() => {
    terminal = new Terminal({
      cursorBlink,
      fontFamily: "SFMono-Regular, Menlo, Monaco, Consolas, monospace",
      fontSize,
      lineHeight: 1.16,
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
    replayAndMaybeScroll(pane.currentPtyId ?? "", outputChunks, jumpRevision);
    return () => {
      resizeObserver.disconnect();
      terminal.dispose();
    };
  });

  $: if (terminal) replayAndMaybeScroll(pane.currentPtyId ?? "", outputChunks, jumpRevision);
  $: if (focused && terminal) terminal.focus();
</script>

<div
  class:focused
  class="group flex h-full min-h-0 min-w-0 flex-col overflow-hidden border border-border-subtle/70 bg-bg-deep text-left text-text-primary outline-none transition-[filter,border-color] duration-150 {focused
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
  {#if pane.currentPtyId && bookmarkRows.length > 0}
    <div
      class="flex h-7 shrink-0 items-center gap-1 border-b border-hairline bg-bg-base/80 px-2 text-[10px]"
    >
      <BookmarkIcon size={12} class="shrink-0 text-text-muted" />
      <div class="app-scrollbar flex min-w-0 flex-1 gap-1 overflow-x-auto">
        {#each bookmarkRows as bookmark (bookmark.id)}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            align="start"
            class="h-5 max-w-[180px] shrink-0 gap-1 rounded border border-border-subtle/60 bg-bg-surface/35 px-1.5 py-0 text-[10px]"
            aria-label={`Jump to bookmark ${bookmark.label}`}
            title={`Jump to bookmark ${bookmark.label} ${bookmark.offsetLabel}`}
            onclick={(event: MouseEvent) => jumpBookmark(event, bookmark.id)}
            onkeydown={(event: KeyboardEvent) => event.stopPropagation()}
          >
            <span class="min-w-0 truncate">{bookmark.label}</span>
            <span class="font-mono text-text-muted">{bookmark.offsetLabel}</span>
          </Button>
        {/each}
      </div>
    </div>
  {/if}
  <div bind:this={host} class="min-h-0 min-w-0 flex-1 overflow-hidden"></div>
</div>
