<script lang="ts">
  import { onMount } from "svelte";
  import { FitAddon } from "@xterm/addon-fit";
  import BookmarkPlus from "@lucide/svelte/icons/bookmark-plus";
  import Check from "@lucide/svelte/icons/check";
  import CircleStop from "@lucide/svelte/icons/circle-stop";
  import Clipboard from "@lucide/svelte/icons/clipboard";
  import X from "@lucide/svelte/icons/x";
  import { Terminal, type IDecoration, type IMarker } from "@xterm/xterm";
  import "@xterm/xterm/css/xterm.css";
  import type { Pane } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type { PTYBookmark } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { ResizePTY } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import { bookmarkMarkerPoints, type BookmarkJumpRequest } from "./ptyMarkers";
  import { terminalInputRefreshDelays, terminalInputShouldRefreshOutput } from "./ptyStream";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";

  export let pane: Pane;
  export let outputChunks: string[] = [];
  export let chunkStartOffsets: number[] = [];
  export let bookmarks: PTYBookmark[] = [];
  export let bookmarkJumpRequest: BookmarkJumpRequest | null = null;
  export let jumpRevision = 0;
  export let bottomRevision = 0;
  export let focused = false;
  export let fontSize = 13;
  export let cursorBlink = true;
  export let onFocus: () => void;
  export let onInput: (ptyId: string) => void;
  export let onWriteInput: (ptyId: string, data: string) => Promise<void>;
  export let onAddBookmark: (ptyId: string) => void;
  export let onBookmark: (bookmark: PTYBookmark) => void;
  export let onBookmarkReplayFallback: (bookmark: PTYBookmark) => void = () => {};
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
  let appliedBottomRevision = 0;
  let appliedBookmarkJumpRevision = 0;
  let appliedBookmarkMarkerSignature = "";
  let scrollToReplayStart = false;
  let terminalWriting = false;
  let pendingTerminalOperations: TerminalOperation[] = [];
  let bookmarkMarkers = new Map<string, IMarker>();
  let bookmarkDecorations = new Map<string, IDecoration>();

  type TerminalOperation =
    | { kind: "write"; bytes: Uint8Array }
    | { kind: "marker"; bookmarkId: string }
    | { kind: "scroll-top" }
    | { kind: "scroll-marker"; bookmarkId: string };

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

  function terminalBookmarkRulerColor() {
    return cssToken("--terminal-bookmark-ruler-color", cssToken("--color-accent", "rgb(125, 211, 252)"));
  }

  function terminalBookmarkCellBackgroundColor() {
    return cssToken("--color-accent-dim", "rgb(14, 165, 233)");
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

  function bookmarkById(bookmarkId: string) {
    return bookmarks.find((candidate) => candidate.id === bookmarkId) ?? null;
  }

  function clickBookmarkDecoration(event: MouseEvent | KeyboardEvent, bookmarkId: string) {
    event.stopPropagation();
    event.preventDefault();
    const bookmark = bookmarkById(bookmarkId);
    if (bookmark) onBookmark(bookmark);
  }

  function addBookmark(event: MouseEvent) {
    event.stopPropagation();
    if (pane.currentPtyId) onAddBookmark(pane.currentPtyId);
  }

  function base64Bytes(chunk: string) {
    if (!chunk) return;
    const binary = atob(chunk);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i += 1) {
      bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
  }

  function liveMarker(bookmarkId: string) {
    const marker = bookmarkMarkers.get(bookmarkId);
    return marker && !marker.isDisposed && marker.line >= 0 ? marker : null;
  }

  function liveBookmarkMarkerIds() {
    const ids = new Set<string>();
    for (const [bookmarkId, marker] of bookmarkMarkers) {
      if (marker.isDisposed || marker.line < 0) {
        bookmarkMarkers.delete(bookmarkId);
      } else {
        ids.add(bookmarkId);
      }
    }
    return ids;
  }

  function clearBookmarkMarkers() {
    for (const marker of bookmarkMarkers.values()) marker.dispose();
    bookmarkMarkers.clear();
    clearBookmarkDecorations();
  }

  function clearBookmarkDecorations() {
    for (const decoration of bookmarkDecorations.values()) decoration.dispose();
    bookmarkDecorations.clear();
  }

  function registerBookmarkDecoration(bookmarkId: string, marker: IMarker) {
    if (!terminal || bookmarkDecorations.has(bookmarkId)) return;
    const decoration = terminal.registerDecoration({
      marker,
      anchor: "left",
      width: 1,
      backgroundColor: terminalBookmarkCellBackgroundColor(),
      foregroundColor: terminalBookmarkRulerColor(),
      layer: "top",
      overviewRulerOptions: {
        color: terminalBookmarkRulerColor(),
        position: "full",
      },
    });
    if (!decoration) return;
    bookmarkDecorations.set(bookmarkId, decoration);
    decoration.onRender((element) => {
      const bookmark = bookmarkById(bookmarkId);
      element.classList.add("terminal-bookmark-decoration");
      element.title = bookmark?.label ? `${bookmark.label} @${bookmark.offset}` : "Bookmark";
      element.setAttribute("role", "button");
      element.setAttribute("aria-label", bookmark?.label ? `Open bookmark ${bookmark.label}` : "Open bookmark");
      element.tabIndex = 0;
      element.onclick = (event: MouseEvent) => clickBookmarkDecoration(event, bookmarkId);
      element.onkeydown = (event: KeyboardEvent) => {
        if (event.key === "Enter" || event.key === " ") clickBookmarkDecoration(event, bookmarkId);
      };
    });
    decoration.onDispose(() => {
      if (bookmarkDecorations.get(bookmarkId) === decoration) bookmarkDecorations.delete(bookmarkId);
    });
  }

  function registerBookmarkMarker(bookmarkId: string) {
    if (!terminal || liveMarker(bookmarkId)) return;
    const marker = terminal.registerMarker(0);
    bookmarkMarkers.set(bookmarkId, marker);
    registerBookmarkDecoration(bookmarkId, marker);
    marker.onDispose(() => {
      if (bookmarkMarkers.get(bookmarkId) === marker) bookmarkMarkers.delete(bookmarkId);
    });
  }

  function resetRenderedTerminal(nextPtyId: string) {
    pendingTerminalOperations = [];
    terminalWriting = false;
    clearBookmarkMarkers();
    terminal.reset();
    writtenChunks = 0;
    renderedPtyId = nextPtyId;
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
    if (operation.kind === "marker") {
      registerBookmarkMarker(operation.bookmarkId);
    } else if (operation.kind === "scroll-top") {
      terminal.scrollToTop();
    } else {
      const marker = liveMarker(operation.bookmarkId);
      if (marker) terminal.scrollToLine(marker.line);
    }
    drainTerminalOperations();
  }

  function enqueueWrite(bytes: Uint8Array) {
    if (bytes.length === 0) return;
    enqueueTerminalOperation({ kind: "write", bytes });
  }

  function writeBase64Chunk(chunk: string, chunkStartOffset: number | undefined) {
    const bytes = base64Bytes(chunk);
    if (!bytes) return;
    const markerPoints = Number.isFinite(chunkStartOffset)
      ? bookmarkMarkerPoints(bookmarks, liveBookmarkMarkerIds(), chunkStartOffset ?? 0, bytes.length)
      : [];
    let cursor = 0;
    for (const point of markerPoints) {
      if (point.byteIndex > cursor) enqueueWrite(bytes.subarray(cursor, point.byteIndex));
      enqueueTerminalOperation({ kind: "marker", bookmarkId: point.bookmarkId });
      cursor = point.byteIndex;
    }
    enqueueWrite(bytes.subarray(cursor));
  }

  function replayOutputChunks(nextPtyId: string, chunks: string[], starts: number[]) {
    if (!terminal) return;
    if (renderedPtyId !== nextPtyId || chunks.length < writtenChunks) {
      resetRenderedTerminal(nextPtyId);
      appliedBookmarkMarkerSignature = "";
    }
    if (chunks.length <= writtenChunks) return;
    for (let index = writtenChunks; index < chunks.length; index += 1) {
      writeBase64Chunk(chunks[index], starts[index]);
    }
    writtenChunks = chunks.length;
  }

  function applyJumpRevision(nextPtyId: string, nextJumpRevision: number) {
    if (!terminal || appliedJumpRevision === nextJumpRevision) return;
    appliedJumpRevision = nextJumpRevision;
    scrollToReplayStart = true;
    resetRenderedTerminal(nextPtyId);
    appliedBookmarkMarkerSignature = "";
  }

  function replayAndMaybeScroll(nextPtyId: string, chunks: string[], starts: number[], nextJumpRevision: number) {
    applyJumpRevision(nextPtyId, nextJumpRevision);
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

  function bookmarkOffsetIsRendered(bookmark: PTYBookmark, chunks: string[], starts: number[]) {
    for (let index = 0; index < chunks.length; index += 1) {
      const start = starts[index];
      if (!Number.isFinite(start)) continue;
      const bytes = base64Bytes(chunks[index]);
      if (!bytes) continue;
      const end = Math.floor(start) + bytes.length;
      if (bookmark.offset >= Math.floor(start) && bookmark.offset <= end) return true;
    }
    return false;
  }

  function bookmarkMarkerSignature(nextPtyId: string, chunks: string[], starts: number[], nextBookmarks: PTYBookmark[]) {
    const renderedBookmarks = nextBookmarks
      .filter((bookmark) => bookmarkOffsetIsRendered(bookmark, chunks, starts))
      .map((bookmark) => `${bookmark.id}:${bookmark.offset}`)
      .sort()
      .join(",");
    if (!nextPtyId || !renderedBookmarks) return "";
    return `${nextPtyId}|${chunks.length}|${starts.join(",")}|${renderedBookmarks}`;
  }

  function replayRenderedChunksForBookmarkMarkers(nextPtyId: string, chunks: string[], starts: number[], nextBookmarks: PTYBookmark[]) {
    if (!terminal || !nextPtyId || chunks.length === 0 || terminalWriting || pendingTerminalOperations.length > 0) return;
    const signature = bookmarkMarkerSignature(nextPtyId, chunks, starts, nextBookmarks);
    if (!signature || appliedBookmarkMarkerSignature === signature) return;
    const needsReplay = nextBookmarks.some((bookmark) => bookmarkOffsetIsRendered(bookmark, chunks, starts) && !liveMarker(bookmark.id));
    if (!needsReplay) {
      appliedBookmarkMarkerSignature = signature;
      return;
    }
    resetRenderedTerminal(nextPtyId);
    replayOutputChunks(nextPtyId, chunks, starts);
    appliedBookmarkMarkerSignature = signature;
  }

  function syncCurrentEndMarkers(chunks: string[], starts: number[], nextBookmarks: PTYBookmark[]) {
    if (!terminal || chunks.length === 0) return;
    const lastIndex = chunks.length - 1;
    const lastStart = starts[lastIndex];
    if (!Number.isFinite(lastStart)) return;
    const lastBytes = base64Bytes(chunks[lastIndex]);
    if (!lastBytes) return;
    const endOffset = Math.floor(lastStart) + lastBytes.length;
    for (const bookmark of nextBookmarks) {
      if (bookmark.offset === endOffset && !liveMarker(bookmark.id)) {
        enqueueTerminalOperation({ kind: "marker", bookmarkId: bookmark.id });
      }
    }
  }

  function applyBookmarkJumpRequest(request: BookmarkJumpRequest | null) {
    if (!terminal || !request || appliedBookmarkJumpRevision === request.revision) return;
    appliedBookmarkJumpRevision = request.revision;
    const marker = liveMarker(request.bookmarkId);
    if (marker) {
      enqueueTerminalOperation({ kind: "scroll-marker", bookmarkId: request.bookmarkId });
      return;
    }
    const bookmark = bookmarks.find((candidate) => candidate.id === request.bookmarkId);
    if (bookmark) onBookmarkReplayFallback(bookmark);
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
    replayAndMaybeScroll(pane.currentPtyId ?? "", outputChunks, chunkStartOffsets, jumpRevision);
    return () => {
      resizeObserver.disconnect();
      pendingTerminalOperations = [];
      clearBookmarkMarkers();
      terminal.dispose();
    };
  });

  $: if (terminal) replayAndMaybeScroll(pane.currentPtyId ?? "", outputChunks, chunkStartOffsets, jumpRevision);
  $: if (terminal) replayRenderedChunksForBookmarkMarkers(pane.currentPtyId ?? "", outputChunks, chunkStartOffsets, bookmarks);
  $: if (terminal) syncCurrentEndMarkers(outputChunks, chunkStartOffsets, bookmarks);
  $: if (terminal) applyBookmarkJumpRequest(bookmarkJumpRequest);
  $: if (terminal) applyBottomRevision(bottomRevision);
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
          label={`Add bookmark for ${pane.currentPtyId}`}
          title={`Add bookmark for ${pane.currentPtyId}`}
          size="sm"
          onclick={addBookmark}
          onkeydown={(event: KeyboardEvent) => event.stopPropagation()}
        >
          <BookmarkPlus size={12} />
        </IconButton>
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

<style>
  :global(.xterm .terminal-bookmark-decoration) {
    cursor: pointer;
    pointer-events: auto;
    position: relative;
  }

  :global(.xterm .terminal-bookmark-decoration::before) {
    background: var(--terminal-bookmark-ruler-color);
    border-radius: 999px;
    bottom: 15%;
    box-shadow: 0 0 0 1px var(--color-bg-deep), 0 0 8px var(--color-accent-dim);
    content: "";
    left: 0;
    position: absolute;
    top: 15%;
    width: 3px;
  }
</style>
