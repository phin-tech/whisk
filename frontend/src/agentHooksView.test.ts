import { describe, expect, it } from "vitest";
import { agentHookDebugRows, agentHookIntegrationFor, upsertAgentHookIntegration } from "./agentHooksView";

describe("agent hook integration view state", () => {
  it("uses the clicked provider when an action response omits provider", () => {
    const integrations = upsertAgentHookIntegration(
      [],
      {
        provider: "",
        status: "current",
        latestVersion: "1.1.0",
        helperPath: "/helper",
        configPath: "/settings.json",
        manifestPath: "/manifest.json",
      },
      "claude",
    );

    expect(agentHookIntegrationFor(integrations, "claude")).toMatchObject({
      provider: "claude",
      status: "current",
      helperPath: "/helper",
    });
  });

  it("replaces the existing provider row", () => {
    const integrations = upsertAgentHookIntegration(
      [
        {
          provider: "claude",
          status: "missing",
          latestVersion: "",
          helperPath: "",
          configPath: "",
          manifestPath: "",
        },
      ],
      {
        provider: "claude",
        status: "current",
        latestVersion: "1.1.0",
        helperPath: "/helper",
        configPath: "/settings.json",
        manifestPath: "/manifest.json",
      },
      "claude",
    );

    expect(integrations).toHaveLength(1);
    expect(integrations[0].status).toBe("current");
  });

  it("formats debug hook events newest first", () => {
    expect(
      agentHookDebugRows([
        {
          id: "old",
          provider: "claude",
          eventName: "PostToolUse",
          toolName: "Bash",
          status: "pending",
          createdAt: "2026-06-11T12:01:00Z",
        },
        {
          id: "new",
          provider: "codex",
          eventName: "Notification",
          message: "Task finished",
          sessionId: "sess_01",
          ptyId: "pty_01",
          status: "pending",
          createdAt: "2026-06-11T12:03:00Z",
        },
      ]),
    ).toEqual([
      {
        id: "new",
        provider: "codex",
        title: "Notification",
        message: "Task finished",
        meta: "sess_01 / pty_01",
        createdAt: "2026-06-11T12:03:00Z",
      },
      {
        id: "old",
        provider: "claude",
        title: "PostToolUse",
        message: "Bash",
        meta: "unowned / no pty",
        createdAt: "2026-06-11T12:01:00Z",
      },
    ]);
  });
});
