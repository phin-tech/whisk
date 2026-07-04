import {
  currentPtyIdOf,
  idString,
  parsePaneId,
  parseProjectId,
  parsePtyId,
  parseRunId,
  parseSessionId,
  parseWorkItemId,
  sessionIdOf,
  type PaneId,
  type PtyId,
  type SessionId,
} from "./ids";
import { firstPaneId } from "./sessionView";

type StatusEventLike = {
  id: string;
  scope?: string;
  kind: string;
  message?: string;
  actor?: string;
  projectId?: string;
  sessionId?: string;
  ptyId?: string;
  workItemId?: string;
  runId?: string;
  paneId?: string;
  requiresAttention?: boolean;
  createdAt?: string | null;
  readAt?: unknown;
};

type AgentHookEventLike = {
  id?: string;
  kind?: string;
  eventName?: string;
  toolName?: string;
};

type AgentPromptLike = {
  id?: string;
  status?: string;
};

type LayoutNodeLike = {
  kind: string;
  paneId?: string;
  direction?: string;
  children?: LayoutNodeLike[];
};

type SessionLike = {
  id: string;
  rootDir?: string;
  windows: { [_ in string]?: { id: string; layout: LayoutNodeLike } };
  panes: { [_ in string]?: { id?: string; currentPtyId?: string | null } };
};

export function notificationBadgeCount(events: StatusEventLike[]) {
  return events.filter((event) => event.requiresAttention && !event.readAt).length;
}

export function notificationSurfaceCount(
  events: StatusEventLike[],
  prompts: AgentPromptLike[],
  agentHookEvents: AgentHookEventLike[],
) {
  return notificationBadgeCount(events) + prompts.filter((prompt) => prompt.status === "pending").length + agentHookEvents.filter(isAgentHookNotification).length;
}

export function notificationClearEnabled(
  events: StatusEventLike[],
  agentHookEvents: AgentHookEventLike[],
) {
  return events.length > 0 || agentHookEvents.some(isAgentHookNotification);
}

function isAgentHookNotification(event: AgentHookEventLike) {
  return event.eventName === "Notification" || event.kind === "question" || event.toolName === "AskUserQuestion";
}

export function notificationRows(events: StatusEventLike[]) {
  return [...events]
    .sort((left, right) => {
      if (Boolean(left.requiresAttention) !== Boolean(right.requiresAttention)) {
        return left.requiresAttention ? -1 : 1;
      }
      return timestamp(right.createdAt) - timestamp(left.createdAt);
    })
    .map((event) => ({
      id: event.id,
      title: labelForKind(event.kind),
      message: event.message || labelForKind(event.kind),
      meta: event.sessionId || event.ptyId ? `${event.sessionId || "unowned"} / ${event.ptyId || "no pty"}` : "No terminal",
      tone: toneForKind(event.kind),
    }));
}

export function notificationDetailRows(event: StatusEventLike, sessions: SessionLike[]) {
  const session = sessionForEvent(statusEventIds(event), sessions)?.session ?? null;
  return [
    detail("Agent", event.actor),
    detail("Session", event.sessionId),
    detail("Pane", event.paneId),
    detail("PTY", event.ptyId),
    detail("CWD", session?.rootDir),
    detail("Project", event.projectId),
    detail("Work item", event.workItemId),
    detail("Run", event.runId),
    detail("Kind", event.kind),
    detail("Scope", event.scope),
    detail("Created", event.createdAt),
  ].filter((row): row is { label: string; value: string } => row !== null);
}

export function targetForStatusEvent(event: StatusEventLike, sessions: SessionLike[]) {
  const ids = statusEventIds(event);
  if (ids.sessionId || ids.ptyId) {
    const match = sessionForEvent(ids, sessions);
    if (match) {
      const explicitPaneId = ids.paneId && match.session.panes[idString(ids.paneId)] ? ids.paneId : null;
      const matchedPaneId = explicitPaneId ?? paneIdForPty(match.session, ids.ptyId) ?? parsePaneId(firstPaneId(match.session));
      return {
        main: "session" as const,
        sessionId: idString(match.sessionId),
        paneId: matchedPaneId ? idString(matchedPaneId) : "",
      };
    }
  }
  if (ids.workItemId) return { main: "work" as const, sessionId: "", paneId: "" };
  return { main: "session" as const, sessionId: ids.sessionId ? idString(ids.sessionId) : "", paneId: "" };
}

type StatusEventIds = {
  projectId: ReturnType<typeof parseProjectId>;
  sessionId: SessionId | null;
  ptyId: PtyId | null;
  workItemId: ReturnType<typeof parseWorkItemId>;
  runId: ReturnType<typeof parseRunId>;
  paneId: PaneId | null;
};

function statusEventIds(event: StatusEventLike): StatusEventIds {
  return {
    projectId: parseProjectId(event.projectId),
    sessionId: parseSessionId(event.sessionId),
    ptyId: parsePtyId(event.ptyId),
    workItemId: parseWorkItemId(event.workItemId),
    runId: parseRunId(event.runId),
    paneId: parsePaneId(event.paneId),
  };
}

function sessionForEvent(ids: StatusEventIds, sessions: SessionLike[]) {
  if (ids.sessionId) {
    const session = sessions.find((candidate) => sessionIdOf(candidate) === ids.sessionId);
    if (session) return { session, sessionId: ids.sessionId };
  }

  if (ids.ptyId) {
    for (const session of sessions) {
      const sessionId = sessionIdOf(session);
      if (sessionId && paneIdForPty(session, ids.ptyId)) return { session, sessionId };
    }
  }

  return null;
}

function paneIdForPty(session: SessionLike, ptyId: PtyId | null): PaneId | null {
  if (!ptyId) return null;
  for (const [paneKey, pane] of Object.entries(session.panes)) {
    const panePtyId = currentPtyIdOf(pane);
    if (panePtyId === ptyId) return parsePaneId(pane?.id) ?? parsePaneId(paneKey);
  }
  return null;
}

function detail(label: string, value: unknown) {
  if (value === undefined || value === null || value === "") return null;
  return { label, value: String(value) };
}

function labelForKind(kind: string) {
  switch (kind) {
    case "question":
      return "Question";
    case "blocked":
      return "Blocked";
    case "done":
      return "Done";
    default:
      return "Status";
  }
}

function toneForKind(kind: string) {
  switch (kind) {
    case "question":
      return "attention";
    case "blocked":
      return "warning";
    case "done":
      return "done";
    default:
      return "neutral";
  }
}

function timestamp(value: string | null | undefined) {
  if (!value) return 0;
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : 0;
}
