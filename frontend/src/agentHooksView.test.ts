import { describe, expect, it } from "vitest";
import { agentHookIntegrationFor, upsertAgentHookIntegration } from "./agentHooksView";

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
});
