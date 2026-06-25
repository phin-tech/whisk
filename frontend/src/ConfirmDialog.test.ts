import { describe, expect, it } from "vitest";
import source from "./ConfirmDialog.svelte?raw";

describe("ConfirmDialog", () => {
  it("uses the local Bits-backed UI layer for dialog controls", () => {
    expect(source).toContain('from "./ui/ModalShell.svelte"');
    expect(source).toContain('from "./ui/Button.svelte"');
    expect(source).toContain('from "./ui/Checkbox.svelte"');
    expect(source).toContain("$props()");
    expect(source).toContain("$state(false)");
    expect(source).toContain("$effect(");
    expect(source).toContain("{#snippet heading()}");
    expect(source).toContain("<ModalShell");
    expect(source).toContain("<Button");
    expect(source).toContain("<Checkbox");
    expect(source).not.toMatch(/<button\b/);
    expect(source).not.toMatch(/<input\b/);
    expect(source).not.toMatch(/\bexport let\b/);
    expect(source).not.toMatch(/\$:/);
    expect(source).not.toMatch(/\son:[a-z]/);
  });

  it("supports keyboard confirm/cancel and an optional dont-ask-again checkbox", () => {
    expect(source).toContain('titleId="confirm-dialog-title"');
    expect(source).toContain('interactOutsideBehavior="ignore"');
    expect(source).toContain("onOpenChange={handleOpenChange}");
    expect(source).toContain("onEscapeKeydown={handleEscape}");
    expect(source).toContain("onkeydown={handleKey}");
    expect(source).toMatch(/!open && visible[\s\S]*oncancel\(\)/);
    expect(source).toMatch(/function handleEscape[\s\S]*oncancel\(\)/);
    expect(source).toMatch(/event\.key !== "Enter"[\s\S]*onconfirm\(checked\)/);
    expect(source).toContain("bind:checked");
  });

  it("keeps destructive prompts compact", () => {
    expect(source).toContain("max-w-[300px]");
    expect(source).not.toContain("max-w-[520px]");
    expect(source).not.toContain("max-w-[420px]");
    expect(source).not.toContain("text-[24px]");
  });
});
