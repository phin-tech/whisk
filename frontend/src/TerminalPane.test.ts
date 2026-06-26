import { describe, expect, it } from "vitest";
import source from "./TerminalPane.svelte?raw";

describe("TerminalPane", () => {
  it("focuses xterm when the pane becomes focused", () => {
    expect(source).toMatch(/if \(focused && terminal\)[\s\S]*terminal\.focus\(\)/);
  });

  it("themes xterm from design-system tokens instead of hardcoded hex", () => {
    const withoutSvelteBlocks = source.replace(/\{#[a-z]+/g, "");
    expect(withoutSvelteBlocks).not.toMatch(/#[0-9a-fA-F]{3,8}/);
    expect(source).toContain('cssToken("--color-bg-deep"');
    expect(source).toContain('cssToken("--color-text-primary"');
    expect(source).toContain('cssToken("--color-accent"');
    expect(source).toContain('cssToken("--color-bg-active"');
    expect(source).toContain("theme: terminalTheme()");
  });

  it("uses local button primitives for terminal pane actions", () => {
    expect(source).toContain('from "./ui/Button.svelte"');
    expect(source).toContain('from "./ui/IconButton.svelte"');
    expect(source).not.toMatch(/<button\b/);
  });

  it("uses xterm as the bookmark GUI and scrolls to the replay start after a jump", () => {
    expect(source).toContain("bookmark-plus");
    expect(source).toContain("export let bookmarks");
    expect(source).toContain("export let bookmarkJumpRequest");
    expect(source).toContain("export let chunkStartOffsets");
    expect(source).toContain("export let jumpRevision");
    expect(source).toContain("onAddBookmark");
    expect(source).toContain("onBookmark");
    expect(source).toContain("onBookmarkReplayFallback");
    expect(source).toContain("Add bookmark for");
    expect(source).toContain("scrollToTop");
    expect(source).toContain("registerMarker");
    expect(source).toContain("scrollToLine");
    expect(source).toContain("bookmarkMarkerPoints");
    expect(source).toContain('replayAndMaybeScroll(pane.currentPtyId ?? "", outputChunks, chunkStartOffsets, jumpRevision)');
    expect(source).not.toContain("ptyBookmarkRowsByPty");
    expect(source).not.toContain("bookmarkRows");
    expect(source).not.toContain("Jump to bookmark");
  });

  it("renders visible xterm decorations for bookmark markers", () => {
    expect(source).toContain("allowProposedApi: true");
    expect(source).toContain("overviewRulerWidth");
    expect(source).toContain("registerDecoration");
    expect(source).toContain("bookmarkDecorations");
    expect(source).toContain("terminal-bookmark-decoration");
    expect(source).toContain("terminal-bookmark-ruler-color");
    expect(source).toContain("clearBookmarkDecorations");
    expect(source).toContain("clickBookmarkDecoration");
    expect(source).toContain("element.onclick");
    expect(source).toContain("element.setAttribute(\"role\", \"button\")");
    expect(source).toContain("element.tabIndex = 0");
    expect(source).toContain("cursor: pointer");
  });

  it("replays rendered chunks when bookmarks arrive after output", () => {
    expect(source).toContain("replayRenderedChunksForBookmarkMarkers");
    expect(source).toContain("appliedBookmarkMarkerSignature");
    expect(source).toContain("bookmarkOffsetIsRendered");
    expect(source).toContain("resetRenderedTerminal");
    expect(source).toContain('replayRenderedChunksForBookmarkMarkers(pane.currentPtyId ?? "", outputChunks, chunkStartOffsets, bookmarks)');
    expect(source).toContain("syncCurrentEndMarkers(outputChunks, chunkStartOffsets, bookmarks)");
  });

  it("scrolls back to the terminal bottom when requested", () => {
    expect(source).toContain("export let bottomRevision");
    expect(source).toContain("applyBottomRevision");
    expect(source).toContain("scrollToBottom");
  });
});
