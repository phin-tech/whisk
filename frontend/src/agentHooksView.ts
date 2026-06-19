import type { AgentBridgeEvent, AgentHookIntegration } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";

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

export function agentHookDebugRows(events: AgentBridgeEvent[]) {
  return [...events]
    .sort((left, right) => timestamp(right.createdAt) - timestamp(left.createdAt))
    .map((event) => ({
      id: event.id,
      provider: event.provider,
      title: event.eventName,
      message: event.message || event.notificationType || event.toolName || event.eventName,
      meta: `${event.sessionId || "unowned"} / ${event.ptyId || "no pty"}`,
      createdAt: String(event.createdAt || ""),
    }));
}

function timestamp(value: unknown) {
  if (!value) return 0;
  const parsed = Date.parse(String(value));
  return Number.isFinite(parsed) ? parsed : 0;
}
