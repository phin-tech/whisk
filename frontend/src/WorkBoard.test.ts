import { describe, expect, it } from "vitest";
import source from "./WorkBoard.svelte?raw";

describe("WorkBoard", () => {
  it("edits work item text with explicit save and cancel callbacks", () => {
    expect(source).toContain("export let onUpdateWorkItem");
    expect(source).toContain("function openDetail");
    expect(source).toContain("function saveDetail");
    expect(source).toContain("function resetDetailDraft");
    expect(source).toContain('aria-label="Work item title"');
    expect(source).toContain('aria-label="Work item description"');
    expect(source).toContain("onUpdateWorkItem({");
    expect(source).toContain("Cancel");
    expect(source).toContain("Save");
  });
});
