import type { AgentHookIntegration } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";

export function normalizeAgentHookIntegration(
  integration: AgentHookIntegration,
  provider: string,
): AgentHookIntegration {
  return {
    ...integration,
    provider: integration.provider || provider,
  };
}

export function upsertAgentHookIntegration(
  integrations: AgentHookIntegration[],
  integration: AgentHookIntegration,
  provider: string,
): AgentHookIntegration[] {
  const normalized = normalizeAgentHookIntegration(integration, provider);
  return [
    normalized,
    ...integrations.filter((item) => item.provider !== normalized.provider),
  ];
}

export function agentHookIntegrationFor(
  integrations: AgentHookIntegration[],
  provider: string,
): AgentHookIntegration {
  return (
    integrations.find((integration) => integration.provider === provider) ?? {
      provider,
      status: "missing",
      latestVersion: "",
      helperPath: "",
      configPath: "",
      manifestPath: "",
    }
  );
}
