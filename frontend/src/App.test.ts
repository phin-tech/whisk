import { describe, expect, it } from "vitest";
import source from "./App.svelte?raw";

function functionBody(name: string) {
  const match = source.match(new RegExp(`function ${name}\\([^)]*\\) \\{([\\s\\S]*?)\\n  \\}`));
  return match?.[1] ?? "";
}

describe("App notification refresh", () => {
  it("wires the jump palette as client-owned navigation over loaded read models", () => {
    expect(source).toContain('import JumpPalette from "./JumpPalette.svelte"');
    expect(source).toContain('import { deriveJumpTargets } from "./jumpTargets"');
    expect(source).toContain('id: "jumpPalette.open"');
    expect(source).toContain("function openJumpPalette()");
    expect(source).toContain("function jumpToTarget(target: JumpTarget)");
    expect(source).toContain("function recordJumpRecent(targetId: string)");
    expect(source).toContain("recentTargetIds={recentTargetIds}");
    expect(source).toContain("selectPaneTarget(payload.sessionId, payload.paneId)");
    expect(source).toContain("selectPaneTarget(target.sessionId, target.paneId)");
    expect(source).toContain("if (!selected) return false");
    expect(source).toContain('navigateTo("work", { openItemId: payload.workItemId })');
    expect(source).toContain("if (activated) recordJumpRecent(target.id)");
    expect(source).toContain("<JumpPalette");
    expect(source).not.toContain("CreateJumpTarget");
    expect(source).not.toContain("PersistJump");
  });

  it("wires plugin UI contributions into command and jump palettes through the daemon read model", () => {
    expect(source).toContain("ListUIContributions");
    expect(source).toContain('} from "./pluginUI"');
    expect(source).toContain("let uiContributions");
    expect(source).toContain("derivePluginCommandDescriptors(uiContributions)");
    expect(source).toContain("derivePluginCommandJumpTargets(uiContributions)");
    expect(source).toContain("async function refreshUIContributionsForScope");
    expect(source).toContain("function activatePluginCommand");
    expect(source).toContain("payload.kind === \"plugin-command\"");
  });

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

  it("requests and applies daemon terminal snapshots before output deltas", () => {
    expect(source).toContain("let terminalSnapshots");
    expect(source).toContain("function applyPTYTerminalSnapshot");
    expect(source).toMatch(/const snapshot = await Output\(\{[\s\S]*snapshot: true/);
    expect(source).toContain("ptySnapshotFromTextFrame(frame)");
    expect(source).toContain("outputChunks = {");
    expect(source).toContain("[ptyId]: []");
    expect(source).toContain("terminalSnapshots = { ...terminalSnapshots, [ptyId]: snapshot }");
    expect(source).toContain("ptyAttachWebSocketURL(address, ptyId, offsets[ptyId] ?? 0, daemonControlToken, true, true)");
  });

  it("guards stale PTY async continuations before recreating stream state", () => {
    expect(source).toContain("terminalStreamHasLivePty");
    expect(source).toContain("function canRefreshPTYOutput(ptyId: string)");
    expect(source).toContain("function canOpenPTYStream(ptyId: string)");
    expect(source).toContain("function canContinuePTYStream(ptyId: string)");
    expect(source).toMatch(/async function refreshOutput\(ptyId: string\) \{\s+if \(!canRefreshPTYOutput\(ptyId\)\) return;/);
    expect(source).toMatch(/if \(inFlightGeneration === generation\) \{[\s\S]*if \(!canRefreshPTYOutput\(ptyId\)\) return;[\s\S]*outputFetchAgain\.add\(ptyId\);/);
    expect(source).toMatch(/const snapshot = await Output\(\{[\s\S]*if \(!isCurrentDaemonGeneration\(daemonLink, generation\) \|\| !canRefreshPTYOutput\(ptyId\)\) return;[\s\S]*outputChunks =/);
    expect(source).toMatch(/const address = await loadDaemonAddress\(\);[\s\S]*if \(!address \|\| !isCurrentDaemonGeneration\(daemonLink, generation\) \|\| !canContinuePTYStream\(ptyId\)\) return;[\s\S]*ptyStreams =/);
    expect(source).toMatch(/socket\.onmessage = \(event\) => \{[\s\S]*!canContinuePTYStream\(ptyId\)[\s\S]*return;/);
  });

  it("opens PTY streams from the visible set even while PTY inventory is stale", () => {
    expect(source).toContain("$: if (settingsLoaded) syncPTYStreams(visiblePTYIds)");

    const openGuard = functionBody("canOpenPTYStream");
    expect(openGuard).toContain("daemonLink.canUseDaemon");
    expect(openGuard).toContain("isVisiblePty(ptyId)");
    expect(openGuard).not.toContain("isLivePty");
    expect(openGuard).not.toContain("canRefreshPTYOutput");

    const continuationGuard = functionBody("canContinuePTYStream");
    expect(continuationGuard).toContain("canOpenPTYStream(ptyId)");
    expect(continuationGuard).toContain("!ptyInventoryLoaded");
    expect(continuationGuard).toContain("loadingPtys");
    expect(continuationGuard).toContain("isLivePty(ptyId)");
  });
});
