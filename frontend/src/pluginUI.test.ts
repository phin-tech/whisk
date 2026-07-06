import { describe, expect, it } from "vitest";
import {
  derivePluginCommandDescriptors,
  derivePluginCommandJumpTargets,
  derivePluginUIContributionScope,
  pluginUIContributionScopeKey,
} from "./pluginUI";

describe("pluginUI", () => {
  it("derives a scoped contribution request from the active UI context", () => {
    const scope = derivePluginUIContributionScope({
      activeProjectId: " proj_01 ",
      openWorkItemId: " wi_01 ",
      activeSessionId: " sess_01 ",
      activePaneId: " pane_01 ",
      activePtyId: " pty_01 ",
    });

    expect(scope).toEqual({
      projectId: "proj_01",
      workItemId: "wi_01",
      sessionId: "sess_01",
      paneId: "pane_01",
      ptyId: "pty_01",
    });
    expect(pluginUIContributionScopeKey(scope)).toBe(
      "projectId=proj_01|workItemId=wi_01|runId=|sessionId=sess_01|paneId=pane_01|ptyId=pty_01|gateReportId=|phase=",
    );
  });

  it("maps enabled trusted plugin commands to command palette descriptors", () => {
    const descriptors = derivePluginCommandDescriptors({
      scope: { projectId: "proj_01", workItemId: "wi_01" },
      plugins: [
        {
          pluginId: "github",
          name: "GitHub",
          trusted: true,
          enabled: true,
          version: "1.0.0",
          commands: [{ id: "review", label: "Review PR", scope: "workItem" }],
        },
        {
          pluginId: "linear",
          name: "Linear",
          trusted: false,
          enabled: true,
          version: "1.0.0",
          commands: [{ id: "sync", label: "Sync issue", scope: "project" }],
        },
      ],
    });

    expect(descriptors).toEqual([
      {
        id: "plugin-command:github:review",
        title: "GitHub: Review PR",
        pluginId: "github",
        pluginName: "GitHub",
        commandId: "review",
        commandLabel: "Review PR",
        commandScope: "workItem",
        contributionScope: { projectId: "proj_01", workItemId: "wi_01" },
      },
    ]);
  });

  it("maps plugin commands to searchable jump palette targets", () => {
    const targets = derivePluginCommandJumpTargets({
      scope: { projectId: "proj_01", workItemId: "wi_01", phase: "review" },
      plugins: [
        {
          pluginId: "github",
          name: "GitHub",
          trusted: true,
          enabled: true,
          version: "1.0.0",
          commands: [{ id: "review", label: "Review PR", scope: "workItem" }],
        },
      ],
    });

    expect(targets).toEqual([
      {
        id: "plugin-command:github:review",
        kind: "plugin-command",
        title: "GitHub: Review PR",
        subtitle: "Plugin command",
        detail: "workItem",
        keywords: ["github", "GitHub", "review", "Review PR", "workItem", "proj_01", "wi_01", "review"],
        payload: { kind: "plugin-command", pluginId: "github", commandId: "review" },
      },
    ]);
  });
});
