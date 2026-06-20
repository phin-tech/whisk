import { describe, expect, it } from "vitest";
import source from "./ConfirmDialog.svelte?raw";

describe("ConfirmDialog", () => {
  it("supports keyboard confirm/cancel and an optional dont-ask-again checkbox", () => {
    expect(source).toContain('role="dialog"');
    expect(source).toContain('aria-modal="true"');
    expect(source).toContain('tabindex="-1"');
    expect(source).toContain("dialog?.focus()");
    expect(source).toContain("on:keydown={handleKey}");
    expect(source).toMatch(/event\.key === "Escape"[\s\S]*oncancel\(\)/);
    expect(source).toMatch(/event\.key === "Enter"[\s\S]*onconfirm\(checked\)/);
    expect(source).toContain('type="checkbox"');
  });

  it("keeps destructive prompts compact", () => {
    expect(source).toContain("max-w-[300px]");
    expect(source).not.toContain("max-w-[520px]");
    expect(source).not.toContain("max-w-[420px]");
    expect(source).not.toContain("text-[24px]");
  });
});
