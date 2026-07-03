import type { AgentBridgeEvent, AgentHookIntegration } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
import { firstPaneId } from "./sessionView";

type SessionLike = {
  id: string;
  name?: string;
  rootDir?: string;
  windows: { [_ in string]?: { id: string; layout: LayoutNodeLike } };
  panes: { [_ in string]?: { id?: string; currentPtyId?: string | null; workingDir?: string } };
};

type LayoutNodeLike = {
  kind: string;
  paneId?: string;
  children?: LayoutNodeLike[];
};

export function normalizeAgentHookIntegration(
  integration: AgentHookIntegration,
  provider: string,
): AgentHookIntegration {
  return {
    ...integration,
    provider: integration.provider || provider,
    state: integration.state || stateForStatus(integration.status),
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
      state: "not_installed",
      status: "missing",
      latestVersion: "",
      helperPath: "",
      configPath: "",
      manifestPath: "",
    }
  );
}

function stateForStatus(status: string) {
  if (status === "current") return "installed";
  if (["modified", "outdated", "untrusted"].includes(status)) return "partial";
  if (status === "unavailable") return "error";
  return "not_installed";
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

export function agentHookNotificationRows(events: AgentBridgeEvent[], sessions: SessionLike[] = []) {
  const rows = events
    .filter(isAgentHookNotification)
    .sort((left, right) => timestamp(right.createdAt) - timestamp(left.createdAt))
    .map((event) => notificationRow(event, sessions));
  const specificContexts = new Set(rows.filter((row) => !row.generic).map((row) => contextKey(row)));
  const seen = new Set<string>();
  return rows
    .filter((row) => !(row.generic && specificContexts.has(contextKey(row))))
    .filter((row) => {
      const key = `${contextKey(row)}\u0000${row.title}\u0000${row.message}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    })
    .map(({ generic, ...row }) => row);
}

export function isAgentHookNotification(event: AgentBridgeEvent) {
  return event.eventName === "Notification" || isAgentQuestion(event);
}

export function agentHookNotificationClickTarget(event: AgentBridgeEvent, sessions: SessionLike[]) {
  const meta = whiskMetadata(event);
  const sessionId = event.sessionId || meta.sessionId;
  const target = eventSessionPane(event, sessions);
  if (!target) return { main: "session" as const, sessionId, paneId: "", readEventId: event.id };
  return { main: "session" as const, sessionId: target.session.id, paneId: target.paneId, readEventId: event.id };
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

function eventMeta(event: AgentBridgeEvent, sessions: SessionLike[] = []) {
  const target = eventSessionPane(event, sessions);
  if (!target) return `${event.sessionId || "unowned"} / ${event.ptyId || "no pty"}`;
  return [
    target.session.name || target.session.id,
    target.paneId,
    target.pane?.workingDir || target.session.rootDir,
  ]
    .filter(Boolean)
    .join(" / ");
}

function notificationRow(event: AgentBridgeEvent, sessions: SessionLike[]) {
  const message = event.message || questionMessage(event) || event.notificationType || event.toolName || event.eventName;
  const fallbackTitle = isAgentQuestion(event) ? `${providerName(event.provider)} question` : "Agent notification";
  const label = event.title || fallbackTitle;
  const generic = isGenericNotification(message);
  const title = generic || isGenericLabel(label) ? message : label;
  return {
    id: event.id,
    provider: event.provider,
    title,
    message: title === message ? label : message,
    meta: eventMeta(event, sessions),
    createdAt: String(event.createdAt || ""),
    generic,
  };
}

function contextKey(row: { provider: string; meta: string }) {
  return `${row.provider}\u0000${row.meta}`;
}

function isGenericLabel(value: string) {
  return /^(agent notification|.+ approval|.+ notification|.+ question)$/i.test(value.trim());
}

function isGenericNotification(value: string) {
  return /needs your permission/i.test(value);
}

function eventSessionPane(event: AgentBridgeEvent, sessions: SessionLike[]) {
  const meta = whiskMetadata(event);
  const sessionId = event.sessionId || meta.sessionId;
  const ptyId = event.ptyId || meta.ptyId;
  const session =
    sessions.find((candidate) => candidate.id === sessionId) ??
    sessions.find((candidate) => Object.values(candidate.panes).some((pane) => pane?.currentPtyId === ptyId));
  if (!session) return null;
  const paneEntry = Object.entries(session.panes).find(([, pane]) => pane?.currentPtyId === ptyId);
  const paneId = paneEntry?.[0] ?? firstPaneId(session);
  return { session, paneId, pane: paneEntry?.[1] ?? session.panes[paneId] };
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
