import { describe, expect, it } from "vitest";
import {
  agentHookDebugDetailRows,
  agentHookDebugRows,
  agentHookIntegrationFor,
  agentHookNotificationRows,
  agentHookNotificationClickTarget,
  upsertAgentHookIntegration,
} from "./agentHooksView";

const sessions = [
  {
    id: "sess_01",
    rootDir: "/repo",
    windows: {
      win_01: {
        id: "win_01",
        layout: { kind: "split", children: [{ kind: "leaf", paneId: "pane_01" }, { kind: "leaf", paneId: "pane_02" }] },
      },
    },
    panes: {
      pane_01: { id: "pane_01", currentPtyId: "pty_01" },
      pane_02: { id: "pane_02", currentPtyId: "pty_02" },
    },
  },
];

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
          title: "Codex prompt",
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
        title: "Codex prompt",
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

  it("keeps provider notification prompts in the notification surface", () => {
    expect(
      agentHookNotificationRows([
        {
          id: "task",
          provider: "codex",
          eventName: "Notification",
          message: "What would you like to work on?",
          notificationType: "task",
          sessionId: "sess_01",
          ptyId: "pty_01",
          status: "pending",
          createdAt: "2026-06-11T12:03:00Z",
        },
        {
          id: "tool",
          provider: "codex",
          eventName: "PostToolUse",
          toolName: "Bash",
          status: "pending",
          createdAt: "2026-06-11T12:04:00Z",
        },
      ]),
    ).toEqual([
      {
        id: "task",
        provider: "codex",
        title: "Agent notification",
        message: "What would you like to work on?",
        meta: "sess_01 / pty_01",
        createdAt: "2026-06-11T12:03:00Z",
      },
    ]);
  });

  it("keeps Claude AskUserQuestion prompts in the notification surface", () => {
    expect(
      agentHookNotificationRows([
        {
          id: "question",
          provider: "claude",
          eventName: "PermissionRequest",
          toolName: "AskUserQuestion",
          status: "pending",
          createdAt: "2026-06-20T12:12:51Z",
          raw: {
            tool_input: {
              questions: [
                {
                  question: "What kind of project is Tavern Keeper?",
                  options: [{ label: "Game" }, { label: "Web app" }],
                },
              ],
            },
          },
        },
      ]),
    ).toEqual([
      {
        id: "question",
        provider: "claude",
        title: "Claude question",
        message: "What kind of project is Tavern Keeper?",
        meta: "unowned / no pty",
        createdAt: "2026-06-20T12:12:51Z",
      },
    ]);
  });

  it("targets the exact triggering pane and exposes the event to mark read", () => {
    expect(
      agentHookNotificationClickTarget(
        {
          id: "hook_01",
          provider: "codex",
          eventName: "Notification",
          message: "Need input",
          sessionId: "sess_01",
          ptyId: "pty_02",
          status: "pending",
          createdAt: "2026-06-20T12:12:51Z",
        },
        sessions,
      ),
    ).toEqual({
      main: "session",
      sessionId: "sess_01",
      paneId: "pane_02",
      readEventId: "hook_01",
    });
  });

  it("uses daemon-normalized hook metadata when available", () => {
    expect(
      agentHookDebugDetailRows({
        id: "hook",
        provider: "codex",
        agent: "codex",
        kind: "prompt",
        title: "Codex prompt",
        eventName: "UserPromptSubmit",
        message: "Implement this.",
        sessionId: "whisk_session_01",
        providerSessionId: "codex_session_01",
        ptyId: "pty_01",
        cwd: "/repo",
        status: "pending",
        createdAt: "2026-06-11T12:04:00Z",
      }),
    ).toContainEqual({ label: "Provider session", value: "codex_session_01" });
  });

  it("builds debug hook event details including cwd from raw payload", () => {
    expect(
      agentHookDebugDetailRows({
        id: "hook",
        provider: "codex",
        eventName: "PostToolUse",
        toolName: "Bash",
        sessionId: "sess_01",
        ptyId: "pty_01",
        result: "logged",
        status: "pending",
        createdAt: "2026-06-11T12:04:00Z",
        raw: {
          cwd: "/repo/tavern-keeper",
          tool_input: { command: "npm test" },
        },
      }),
    ).toEqual([
      { label: "Agent", value: "codex" },
      { label: "Event", value: "PostToolUse" },
      { label: "Tool", value: "Bash" },
      { label: "Session", value: "sess_01" },
      { label: "PTY", value: "pty_01" },
      { label: "CWD", value: "/repo/tavern-keeper" },
      { label: "Result", value: "logged" },
      { label: "Status", value: "pending" },
      { label: "Created", value: "2026-06-11T12:04:00Z" },
      { label: "Raw", value: "{\"cwd\":\"/repo/tavern-keeper\",\"tool_input\":{\"command\":\"npm test\"}}" },
    ]);
  });
});
