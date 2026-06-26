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

  it("renders bookmark jump controls and scrolls to the replay start after a jump", () => {
    expect(source).toContain("export let bookmarks");
    expect(source).toContain("export let jumpRevision");
    expect(source).toContain("onBookmark");
    expect(source).toContain("scrollToTop");
    expect(source).toContain('replayAndMaybeScroll(pane.currentPtyId ?? "", outputChunks, jumpRevision)');
  });
});
