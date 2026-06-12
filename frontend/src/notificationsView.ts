import { firstPaneId } from "./sessionView";

type StatusEventLike = {
  id: string;
  kind: string;
  message?: string;
  sessionId?: string;
  ptyId?: string;
  workItemId?: string;
  requiresAttention?: boolean;
  createdAt?: string | null;
  readAt?: unknown;
};

type LayoutNodeLike = {
  kind: string;
  paneId?: string;
  direction?: string;
  children?: LayoutNodeLike[];
};

type SessionLike = {
  id: string;
  windows: { [_ in string]?: { id: string; layout: LayoutNodeLike } };
  panes: { [_ in string]?: { id?: string; currentPtyId?: string | null } };
};

export function notificationBadgeCount(events: StatusEventLike[]) {
  return events.filter((event) => event.requiresAttention && !event.readAt).length;
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

export function targetForStatusEvent(event: StatusEventLike, sessions: SessionLike[]) {
  if (event.sessionId || event.ptyId) {
    const session =
      sessions.find((candidate) => candidate.id === event.sessionId) ??
      sessions.find((candidate) =>
        Object.values(candidate.panes).some((pane) => pane?.currentPtyId === event.ptyId),
      );
    if (session) {
      const paneId =
        Object.entries(session.panes).find(([, pane]) => pane?.currentPtyId === event.ptyId)?.[0] ??
        firstPaneId(session);
      return { main: "session" as const, sessionId: session.id, paneId };
    }
  }
  if (event.workItemId) return { main: "work" as const, sessionId: "", paneId: "" };
  return { main: "session" as const, sessionId: event.sessionId || "", paneId: "" };
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
