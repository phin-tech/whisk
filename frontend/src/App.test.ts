import { describe, expect, it } from "vitest";
import source from "./App.svelte?raw";

describe("App notification refresh", () => {
  it("loads pending agent bridge approvals instead of clearing them", () => {
    expect(source).toContain('ListAgentBridgeApprovals({ status: "pending" })');
    expect(source).not.toContain("agentBridgeApprovals = [];");
  });

  it("does not expose bookmark bindings or commands", () => {
    expect(source).not.toContain("ListPTYBookmarks");
    expect(source).not.toContain("AddPTYBookmark");
    expect(source).not.toContain("bookmarksByPty");
    expect(source).not.toContain("jumpToBookmark");
    expect(source).not.toContain("bookmarkJumpRequests");
    expect(source).toContain("outputChunkStartOffsets");
    expect(source).not.toContain("createPTYBookmark");
    expect(source).not.toContain("jumpBookmarkByDirection");
    expect(source).not.toContain("jumpToLastPrompt");
    expect(source).toContain("jumpToBottom");
    expect(source).toContain("bottomJumpRevisions");
    expect(source).not.toContain("replayBookmarkFromOffset");
    expect(source).not.toContain("latestPromptJumpPointTarget");
    expect(source).not.toContain("resetOutputReplayForBookmark");
    expect(source).not.toContain("bookmarkJumpTarget");
    expect(source).not.toContain("bookmark.add");
    expect(source).not.toContain("bookmark.previous");
    expect(source).not.toContain("bookmark.next");
    expect(source).not.toContain("bookmark.lastPrompt");
    expect(source).toContain("terminal.bottom");
  });

  it("keeps PTY output snapshots out of the steady polling path", () => {
    expect(source).not.toContain("outputReconcileTimer");
    expect(source).toContain("hasActivePTYStream(targets.outputPtyId)");
    expect(source).toContain("reconnectBackoffDelayMs");
  });
});
