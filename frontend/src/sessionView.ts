import {
  currentPtyIdOf,
  idString,
  parsePaneId,
  parsePtyId,
  parseSessionId,
  parseWindowId,
  sessionIdOf,
  type PaneId,
  type PtyId,
  type SessionId,
  type WindowId,
} from "./ids";

type LayoutNodeLike = {
  kind: string;
  paneId?: string;
  direction?: string;
  children?: LayoutNodeLike[];
};

type SessionLike = {
  id: string;
  projectId?: string;
  name?: string;
  rootDir?: string;
  windows: { [_ in string]?: WindowLike };
  panes: { [_ in string]?: { id?: string; currentPtyId?: string | null } };
};

type PaneLike = {
  id?: string;
  currentPtyId?: string | null;
};

type ProjectLike = {
  id: string;
  name: string;
};

export type SessionGroupMode = "recent" | "project" | "folder";

export type SessionGroup<T extends SessionLike> = {
  id: string;
  title: string;
  sessions: T[];
};

type WindowLike = {
  id: string;
  layout: LayoutNodeLike;
};

type PtyInfoLike = {
  id: string;
  workingDir: string;
  cols: number;
  rows: number;
  running: boolean;
  status?: string;
  sessionId: string;
  paneId: string;
};

type RuntimeEventLike = {
  type: string;
  seq?: number;
  ptyId?: string;
  offset?: number;
};

type PtyHistorySummaryLike = {
  ptyId: string;
  sessionId?: string;
  paneId?: string;
  workingDir?: string;
  createdAt?: string;
  exitCode?: number | null;
};

export type ClosePaneRequestLike = {
  sessionId: string;
  windowId: string;
  paneId: string;
};

export type KillPTYRequestLike = {
  ptyId: string;
};

export type ClosePaneTargetLike =
  | { kind: "pane"; request: ClosePaneRequestLike; ptyId: string }
  | { kind: "session"; sessionId: string; ptyId: string };

export function paneIds(node: LayoutNodeLike | undefined): string[] {
  return typedPaneIds(node).map(idString);
}

function typedPaneIds(node: LayoutNodeLike | undefined): PaneId[] {
  if (!node) return [];
  if (node.kind === "leaf") {
    const paneId = parsePaneId(node.paneId);
    return paneId ? [paneId] : [];
  }
  return (node.children ?? []).flatMap(typedPaneIds);
}

export function sessionWindows<T extends WindowLike>(
  session: { windows: { [_ in string]?: T } } | null | undefined,
): T[] {
  return Object.values(session?.windows ?? {}).filter((window): window is T => Boolean(window));
}

export function firstPaneId(session: SessionLike | null | undefined): string {
  for (const window of sessionWindows(session)) {
    const [paneId] = typedPaneIds(window.layout);
    if (paneId) return idString(paneId);
  }
  return "";
}

export function activeWindow<T extends WindowLike>(
  session: { windows: { [_ in string]?: T } } | null | undefined,
  paneId: string,
): T | null {
  const windows = sessionWindows(session);
  const targetPaneId = parsePaneId(paneId);
  const containing = targetPaneId
    ? windows.find((window) => hasPaneId(typedPaneIds(window.layout), targetPaneId))
    : null;
  return containing ?? windows[0] ?? null;
}

export function visiblePtyIds(
  sessions: SessionLike[],
  activeSessionId: string,
  activePaneId: string,
): string[] {
  const ptys: PtyId[] = [];
  const seen = new Set<PtyId>();
  const activeSession = findSessionById(sessions, parseSessionId(activeSessionId));
  const activePane = paneById(activeSession, parsePaneId(activePaneId));
  const activePtyId = currentPtyIdOf(activePane);
  if (activePtyId) {
    ptys.push(activePtyId);
    seen.add(activePtyId);
  }
  for (const session of sessions) {
    for (const window of sessionWindows(session)) {
      for (const paneId of typedPaneIds(window.layout)) {
        const ptyId = currentPtyIdOf(paneById(session, paneId));
        if (ptyId && !seen.has(ptyId)) {
          ptys.push(ptyId);
          seen.add(ptyId);
        }
      }
    }
  }
  return ptys.map(idString);
}

export function closePaneRequest(
  session: SessionLike | null | undefined,
  windowId: string,
  paneId: string,
): ClosePaneRequestLike | null {
  const sessionId = sessionIdOf(session);
  const typedWindowId = parseWindowId(windowId);
  const typedPaneId = parsePaneId(paneId);
  const window = windowById(session, typedWindowId);
  const pane = paneById(session, typedPaneId);
  const windowPaneIds = typedPaneIds(window?.layout);
  if (!sessionId || !typedWindowId || !typedPaneId || !window || !pane || windowPaneIds.length <= 1) {
    return null;
  }
  if (!hasPaneId(windowPaneIds, typedPaneId)) return null;
  return {
    sessionId: idString(sessionId),
    windowId: idString(typedWindowId),
    paneId: idString(typedPaneId),
  };
}

export function closePaneTarget(
  session: SessionLike | null | undefined,
  windowId: string,
  paneId: string,
): ClosePaneTargetLike | null {
  const sessionId = sessionIdOf(session);
  const typedWindowId = parseWindowId(windowId);
  const typedPaneId = parsePaneId(paneId);
  const window = windowById(session, typedWindowId);
  const pane = paneById(session, typedPaneId);
  const windowPaneIds = typedPaneIds(window?.layout);
  if (!sessionId || !typedWindowId || !typedPaneId || !window || !pane || !hasPaneId(windowPaneIds, typedPaneId)) {
    return null;
  }
  const ptyId = currentPtyIdOf(pane);
  if (windowPaneIds.length <= 1) {
    return { kind: "session", sessionId: idString(sessionId), ptyId: ptyId ? idString(ptyId) : "" };
  }
  return {
    kind: "pane",
    request: {
      sessionId: idString(sessionId),
      windowId: idString(typedWindowId),
      paneId: idString(typedPaneId),
    },
    ptyId: ptyId ? idString(ptyId) : "",
  };
}

export function killPTYRequest(pane: PaneLike | null | undefined): KillPTYRequestLike | null {
  const ptyId = currentPtyIdOf(pane);
  return ptyId ? { ptyId: idString(ptyId) } : null;
}

export function ptyRowsFromInventory(ptys: PtyInfoLike[]) {
  return ptys.map((pty) => ({
    id: pty.id,
    title: pty.id,
    subtitle: `${pty.sessionId || "unowned"} / ${pty.paneId || "detached"}`,
    detail: `${pty.workingDir || "."} / ${pty.cols}x${pty.rows}`,
    running: pty.running,
    status: pty.status || (pty.running ? "running" : "exited"),
    canDelete: !pty.running,
  }));
}

export function ptyHistoryRows(history: PtyHistorySummaryLike[]) {
  return history.map((item) => ({
    id: item.ptyId,
    title: item.ptyId,
    subtitle: `${item.sessionId || "unowned"} / ${item.paneId || "detached"}`,
    detail: item.workingDir || ".",
    createdAt: item.createdAt || "",
    exitCode: item.exitCode ?? null,
  }));
}

export function sessionGroups<T extends SessionLike>(
  sessions: T[],
  projects: ProjectLike[],
  mode: SessionGroupMode,
  query: string,
): SessionGroup<T>[] {
  const needle = query.trim().toLowerCase();
  const filtered = sessions.filter((session) => {
    if (!needle) return true;
    return `${session.name ?? ""} ${session.rootDir ?? ""} ${session.projectId ?? ""}`
      .toLowerCase()
      .includes(needle);
  });
  if (mode === "recent") {
    return filtered.length === 0 ? [] : [{ id: "recent", title: "Recent", sessions: filtered }];
  }

  const projectNames = new Map(projects.map((project) => [project.id, project.name]));
  const groups = new Map<string, SessionGroup<T>>();
  for (const session of filtered) {
    const key = mode === "project" ? session.projectId || "none" : session.rootDir || ".";
    const title =
      mode === "project" ? (session.projectId ? (projectNames.get(session.projectId) ?? session.projectId) : "No project") : key;
    if (!groups.has(key)) groups.set(key, { id: key, title, sessions: [] });
    groups.get(key)?.sessions.push(session);
  }
  return Array.from(groups.values());
}

export function runtimeRefreshTargets(event: RuntimeEventLike) {
  const outputPtyId = event.type === "pty.output" ? parsePtyId(event.ptyId) : null;
	return {
		sessions: event.type === "session.changed",
		ptys: event.type === "pty.changed",
		outputPtyId: outputPtyId ? idString(outputPtyId) : null,
		work: event.type === "workitems.changed" || event.type === "status.changed",
		statusEvents: event.type === "status.changed",
		agentBridgeApprovals: event.type === "agent_bridge_approvals.changed" || event.type === "agent_prompts.changed",
		agentHookEvents: event.type === "agent_hook_events.changed",
	};
}

export function isStalePTYError(err: unknown) {
  const message = err instanceof Error ? err.message : String(err);
  if (/\bpty\s+\S+\s+not found\b/.test(message)) return true;
  try {
    const parsed = JSON.parse(message) as { message?: unknown };
    return typeof parsed.message === "string" && /\bpty\s+\S+\s+not found\b/.test(parsed.message);
  } catch {
    return false;
  }
}

function findSessionById<T extends SessionLike>(
  sessions: readonly T[],
  sessionId: SessionId | null,
): T | null {
  if (!sessionId) return null;
  return sessions.find((session) => sessionIdOf(session) === sessionId) ?? null;
}

function windowById<T extends WindowLike>(
  session: { windows: { [_ in string]?: T } } | null | undefined,
  windowId: WindowId | null,
): T | null {
  return windowId ? session?.windows[idString(windowId)] ?? null : null;
}

function paneById(
  session: { panes: { [_ in string]?: PaneLike | null | undefined } } | null | undefined,
  paneId: PaneId | null,
): PaneLike | null {
  return paneId ? session?.panes[idString(paneId)] ?? null : null;
}

function hasPaneId(paneIds: readonly PaneId[], paneId: PaneId) {
  return paneIds.some((candidate) => candidate === paneId);
}
