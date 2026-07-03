import { describe, expect, it } from "vitest";
import source from "./TerminalPane.svelte?raw";

describe("TerminalPane", () => {
  it("focuses xterm when the pane becomes focused", () => {
    expect(source).toMatch(/if \(focused && terminal\)[\s\S]*terminal\.focus\(\)/);
  });

  it("themes xterm from design-system tokens instead of hardcoded hex", () => {
    const withoutSvelteBlocks = source.replace(/\{#[a-z]+/g, "");
    expect(withoutSvelteBlocks).not.toMatch(/#[0-9a-fA-F]{3,8}/);
    expect(source).toContain('cssToken("--color-terminal-surface"');
    expect(source).toContain('cssToken("--color-terminal-foreground"');
    expect(source).toContain('cssToken("--color-terminal-cursor"');
    expect(source).toContain('cssToken("--color-terminal-selection"');
    expect(source).toContain("theme: terminalTheme()");
  });

  it("uses local button primitives for terminal pane actions", () => {
    expect(source).toContain('from "./ui/Button.svelte"');
    expect(source).toContain('from "./ui/IconButton.svelte"');
    expect(source).not.toMatch(/<button\b/);
  });

  it("does not expose bookmark UI or marker plumbing", () => {
    expect(source).not.toContain("bookmark-plus");
    expect(source).not.toContain("export let bookmarks");
    expect(source).not.toContain("export let bookmarkJumpRequest");
    expect(source).not.toContain("onAddBookmark");
    expect(source).not.toContain("onBookmark");
    expect(source).not.toContain("onBookmarkReplayFallback");
    expect(source).not.toContain("Add bookmark for");
    expect(source).not.toContain("bookmarkMarkerPoints");
    expect(source).not.toContain("registerDecoration");
    expect(source).not.toContain("terminal-bookmark-decoration");
    expect(source).not.toContain("createBookmarkFromTerminalClick");
    expect(source).not.toContain("outputLineOffsetPoints");
    expect(source).toContain("export let chunkStartOffsets");
    expect(source).toContain("export let jumpRevision");
    expect(source).toContain("scrollToTop");
    expect(source).toContain('replayAndMaybeScroll(pane.currentPtyId ?? "", outputChunks, chunkStartOffsets, jumpRevision)');
  });

  it("scrolls back to the terminal bottom when requested", () => {
    expect(source).toContain("export let bottomRevision");
    expect(source).toContain("applyBottomRevision");
    expect(source).toContain("scrollToBottom");
  });

});
