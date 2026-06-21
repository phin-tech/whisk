import { describe, expect, it } from "vitest";
import source from "./NotificationsPanel.svelte?raw";

describe("NotificationsPanel", () => {
  it("keeps hook notification rows inside the card", () => {
    expect(source).toContain("class=\"flex w-full min-w-0 items-start");
    expect(source).toContain("max-w-[72px]");
    expect(source).toContain("break-all");
  });
});
