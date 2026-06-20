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
      title: event.title || event.eventName,
      message: event.message || event.notificationType || event.toolName || event.eventName,
      meta: eventMeta(event),
      createdAt: String(event.createdAt || ""),
    }));
}

export function agentHookNotificationRows(events: AgentBridgeEvent[]) {
  return events
    .filter(isAgentHookNotification)
    .sort((left, right) => timestamp(right.createdAt) - timestamp(left.createdAt))
    .map((event) => ({
      id: event.id,
      provider: event.provider,
      title: event.title || (isAgentQuestion(event) ? `${providerName(event.provider)} question` : "Agent notification"),
      message: event.message || questionMessage(event) || event.notificationType || event.toolName || event.eventName,
      meta: eventMeta(event),
      createdAt: String(event.createdAt || ""),
    }));
}

export function isAgentHookNotification(event: AgentBridgeEvent) {
  return event.eventName === "Notification" || isAgentQuestion(event);
}

export function agentHookDebugDetailRows(event: AgentBridgeEvent) {
  const meta = whiskMetadata(event);
  return [
    detail("Agent", event.agent || meta.actor || meta.provider || event.provider),
    detail("Event", event.eventName),
    detail("Kind", event.kind),
    detail("Tool", event.toolName),
    detail("Session", meta.sessionId || event.sessionId),
    detail("Provider session", event.providerSessionId),
    detail("PTY", meta.ptyId || event.ptyId),
    detail("CWD", event.cwd || meta.cwd || rawString(event.raw, "cwd")),
    detail("Project", meta.projectId),
    detail("Project root", meta.projectRoot),
    detail("Work item", meta.workItemId),
    detail("Run", meta.runId),
    detail("Result", event.result),
    detail("Status", event.status),
    detail("Created", event.createdAt),
    detail("Raw", event.raw ? JSON.stringify(event.raw) : ""),
  ].filter((row): row is { label: string; value: string } => row !== null);
}

function eventMeta(event: AgentBridgeEvent) {
  return `${event.sessionId || "unowned"} / ${event.ptyId || "no pty"}`;
}

function isAgentQuestion(event: AgentBridgeEvent) {
  return event.kind === "question" || event.toolName === "AskUserQuestion";
}

function questionMessage(event: AgentBridgeEvent) {
  const questions = event.raw?.tool_input?.questions;
  if (!Array.isArray(questions)) return "";
  const first = questions[0];
  return first && typeof first === "object" && typeof first.question === "string" ? first.question : "";
}

function providerName(provider: string) {
  if (!provider) return "Agent";
  return provider.charAt(0).toUpperCase() + provider.slice(1);
}

function whiskMetadata(event: AgentBridgeEvent): Record<string, string> {
  const raw = event.raw?.whisk;
  if (!raw || typeof raw !== "object") return {};
  return Object.fromEntries(
    Object.entries(raw)
      .filter(([, value]) => typeof value === "string" && value !== "")
      .map(([key, value]) => [key, value as string]),
  );
}

function rawString(raw: AgentBridgeEvent["raw"], key: string) {
  const value = raw?.[key];
  return typeof value === "string" ? value : "";
}

function detail(label: string, value: unknown) {
  if (value === undefined || value === null || value === "") return null;
  return { label, value: String(value) };
}

function timestamp(value: unknown) {
  if (!value) return 0;
  const parsed = Date.parse(String(value));
  return Number.isFinite(parsed) ? parsed : 0;
}
