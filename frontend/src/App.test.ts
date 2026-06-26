import { describe, expect, it } from "vitest";
import source from "./App.svelte?raw";

describe("App notification refresh", () => {
  it("loads pending agent bridge approvals instead of clearing them", () => {
    expect(source).toContain('ListAgentBridgeApprovals({ status: "pending" })');
    expect(source).not.toContain("agentBridgeApprovals = [];");
  });

  it("loads pty bookmarks and resets output replay when jumping to one", () => {
    expect(source).toContain("ListPTYBookmarks");
    expect(source).toContain("bookmarksByPty");
    expect(source).toContain("jumpToBookmark");
    expect(source).toContain("resetOutputReplayForBookmark");
    expect(source).toContain("bookmarkJumpTarget");
  });
});
