import { describe, expect, it } from "vitest";
import source from "./App.svelte?raw";

describe("App notification refresh", () => {
  it("loads pending agent bridge approvals instead of clearing them", () => {
    expect(source).toContain('ListAgentBridgeApprovals({ status: "pending" })');
    expect(source).not.toContain("agentBridgeApprovals = [];");
  });
});
