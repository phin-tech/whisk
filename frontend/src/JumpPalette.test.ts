import { describe, expect, it } from "vitest";
import source from "./JumpPalette.svelte?raw";

describe("JumpPalette", () => {
  it("uses the jump ranking foundation and local UI layer", () => {
    expect(source).toContain('from "./jumpFilter"');
    expect(source).toContain('from "./jumpRecents"');
    expect(source).toContain("reconcileJumpRecents(");
    expect(source).toContain("applyRecentJumpTargets(targets, validRecentTargetIds)");
    expect(source).toContain("query.trim() ? targets : emptyQueryTargets");
    expect(source).toContain("prepareJumpTargets(rankedTargets)");
    expect(source).toContain("rankJumpTargets(query, preparedTargets)");
    expect(source).toContain('from "./ui/ModalShell.svelte"');
    expect(source).toContain('from "./ui/Button.svelte"');
    expect(source).toContain('from "./ui/TextField.svelte"');
    expect(source).toContain('from "./ui/Badge.svelte"');
    expect(source).not.toMatch(/<(button|input|textarea|select)\b/);
  });

  it("supports the command-palette keyboard contract for jump targets", () => {
    expect(source).toContain('event.key === "Escape"');
    expect(source).toContain('event.key === "ArrowDown"');
    expect(source).toContain('event.key === "ArrowUp"');
    expect(source).toContain('event.key === "Enter"');
    expect(source).toContain("runSelected()");
    expect(source).toContain("onjump(item)");
    expect(source).toContain("onclose()");
    expect(source).toContain("role=\"listbox\"");
    expect(source).toContain("role=\"option\"");
    expect(source).toContain("aria-selected={index === selected}");
  });

  it("resets query and highlight when opened", () => {
    expect(source).toMatch(/if \(visible && !previousVisible\)[\s\S]*query = ""/);
    expect(source).toMatch(/if \(visible && !previousVisible\)[\s\S]*selected = 0/);
    expect(source).toContain("setTimeout(() => input?.focus())");
    expect(source).toContain('aria-label="Jump target"');
  });
});
