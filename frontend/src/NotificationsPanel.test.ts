import { describe, expect, it } from "vitest";
import source from "./NotificationsPanel.svelte?raw";

describe("NotificationsPanel", () => {
  it("keeps hook notification rows inside the card", () => {
    expect(source).toContain("class=\"flex w-full min-w-0 items-start");
    expect(source).toContain("max-w-[72px]");
    expect(source).toContain("break-all");
  });

  it("expands prompt rows to answer inline", () => {
    expect(source).toContain("Click to respond");
    expect(source).toContain("toggleExpanded(prompt.id)");
    expect(source).toContain("resolveOptionPrompt(prompt, option.value, index)");
    expect(source).toContain("resolveTextPrompt(prompt)");
    expect(source).toContain("promptMessages.has(hook.title)");
  });
});
