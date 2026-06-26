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

type PtyBookmarkLike = {
  id: string;
  ptyId: string;
  sessionId?: string;
  paneId?: string;
  offset: number;
  kind?: string;
  label?: string;
};

export type PtyBookmarkRow = {
  id: string;
  ptyId: string;
  label: string;
  offset: number;
  offsetLabel: string;
  detail: string;
  kind: string;
};

export type OutputReplayState = {
  outputChunks: Record<string, string[]>;
  offsets: Record<string, number>;
  jumpRevisions: Record<string, number>;
};

export type BookmarkJumpTarget = {
  sessionId: string;
  paneId: string;
  ptyId: string;
  offset: number;
};

export type BookmarkDirection = "previous" | "next";

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
  if (!node) return [];
  if (node.kind === "leaf") return node.paneId ? [node.paneId] : [];
  return (node.children ?? []).flatMap(paneIds);
}

export function sessionWindows<T extends WindowLike>(
  session: { windows: { [_ in string]?: T } } | null | undefined,
): T[] {
  return Object.values(session?.windows ?? {}).filter((window): window is T => Boolean(window));
}

export function firstPaneId(session: SessionLike | null | undefined): string {
  for (const window of sessionWindows(session)) {
    const [paneId] = paneIds(window.layout);
    if (paneId) return paneId;
  }
  return "";
}

export function activeWindow<T extends WindowLike>(
  session: { windows: { [_ in string]?: T } } | null | undefined,
  paneId: string,
): T | null {
  const windows = sessionWindows(session);
  const containing = windows.find((window) => paneIds(window.layout).includes(paneId));
  return containing ?? windows[0] ?? null;
}

export function visiblePtyIds(
  sessions: SessionLike[],
  activeSessionId: string,
  activePaneId: string,
): string[] {
  const ptys: string[] = [];
  const seen = new Set<string>();
  const activeSession = sessions.find((session) => session.id === activeSessionId);
  const activePtyID = activeSession?.panes[activePaneId]?.currentPtyId;
  if (activePtyID) {
    ptys.push(activePtyID);
    seen.add(activePtyID);
  }
  for (const session of sessions) {
    for (const window of sessionWindows(session)) {
      for (const paneID of paneIds(window.layout)) {
        const ptyID = session.panes[paneID]?.currentPtyId;
        if (ptyID && !seen.has(ptyID)) {
          ptys.push(ptyID);
          seen.add(ptyID);
        }
      }
    }
  }
  return ptys;
}

export function closePaneRequest(
  session: SessionLike | null | undefined,
  windowId: string,
  paneId: string,
): ClosePaneRequestLike | null {
  const window = session?.windows[windowId];
  const pane = session?.panes[paneId];
  const windowPaneIds = paneIds(window?.layout);
  if (!session || !window || !pane || windowPaneIds.length <= 1) return null;
  if (!windowPaneIds.includes(paneId)) return null;
  return { sessionId: session.id, windowId, paneId };
}

export function closePaneTarget(
  session: SessionLike | null | undefined,
  windowId: string,
  paneId: string,
): ClosePaneTargetLike | null {
  const window = session?.windows[windowId];
  const pane = session?.panes[paneId];
  const windowPaneIds = paneIds(window?.layout);
  if (!session || !window || !pane || !windowPaneIds.includes(paneId)) return null;
  const ptyId = pane.currentPtyId ?? "";
  if (windowPaneIds.length <= 1) return { kind: "session", sessionId: session.id, ptyId };
  return { kind: "pane", request: { sessionId: session.id, windowId, paneId }, ptyId };
}

export function killPTYRequest(pane: PaneLike | null | undefined): KillPTYRequestLike | null {
  return pane?.currentPtyId ? { ptyId: pane.currentPtyId } : null;
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

export function ptyBookmarkRowsByPty(bookmarks: PtyBookmarkLike[]) {
  const grouped: Record<string, PtyBookmarkRow[]> = {};
  for (const bookmark of [...bookmarks].sort((a, b) => a.ptyId.localeCompare(b.ptyId) || a.offset - b.offset || a.id.localeCompare(b.id))) {
    const kind = bookmark.kind ?? "";
    const kindLabel = kind.trim().replace(/[_-]+/g, " ");
    const label = bookmark.label?.trim() || kindLabel || `offset ${bookmark.offset}`;
    const row = {
      id: bookmark.id,
      ptyId: bookmark.ptyId,
      label,
      offset: bookmark.offset,
      offsetLabel: `@${bookmark.offset}`,
      detail: `${bookmark.sessionId || "unowned"} / ${bookmark.paneId || "detached"}`,
      kind,
    };
    grouped[bookmark.ptyId] = [...(grouped[bookmark.ptyId] ?? []), row];
  }
  return grouped;
}

export async function safeBookmarksByPty<T extends { id: string }, B>(
  ptys: T[],
  load: (ptyId: string) => Promise<B[]>,
): Promise<Record<string, B[]>> {
  const entries = await Promise.all(
    ptys.map(async (pty) => {
      try {
        return [pty.id, await load(pty.id)] as const;
      } catch {
        return [pty.id, []] as const;
      }
    }),
  );
  return Object.fromEntries(entries);
}

export function nextBookmarkTarget<T extends PtyBookmarkLike>(
  bookmarks: T[],
  activeBookmarkId: string,
  currentOffset: number,
  direction: BookmarkDirection,
): T | null {
  const sorted = [...bookmarks].sort((a, b) => a.offset - b.offset || a.id.localeCompare(b.id));
  if (sorted.length === 0) return null;

  const activeIndex = sorted.findIndex((bookmark) => bookmark.id === activeBookmarkId);
  if (activeIndex >= 0) {
    const delta = direction === "next" ? 1 : -1;
    return sorted[(activeIndex + delta + sorted.length) % sorted.length];
  }

  const offset = Math.max(0, currentOffset);
  if (direction === "next") {
    return sorted.find((bookmark) => bookmark.offset > offset) ?? sorted[0];
  }
  for (let i = sorted.length - 1; i >= 0; i -= 1) {
    if (sorted[i].offset < offset) return sorted[i];
  }
  return sorted[sorted.length - 1];
}

export function latestPromptJumpPointTarget<T extends PtyBookmarkLike>(bookmarks: T[]): T | null {
  const sorted = bookmarks
    .filter((bookmark) => bookmark.kind === "jump_point")
    .sort((a, b) => b.offset - a.offset || b.id.localeCompare(a.id));
  return sorted[0] ?? null;
}

export function bookmarkJumpTarget(
  sessions: SessionLike[],
  bookmark: PtyBookmarkLike,
): BookmarkJumpTarget | null {
  const exactSession = sessions.find((session) => session.id === bookmark.sessionId);
  const exactPane = exactSession?.panes[bookmark.paneId ?? ""];
  if (exactSession && exactPane?.currentPtyId === bookmark.ptyId) {
    return {
      sessionId: exactSession.id,
      paneId: exactPane.id || bookmark.paneId || "",
      ptyId: bookmark.ptyId,
      offset: Math.max(0, bookmark.offset),
    };
  }

  for (const session of sessions) {
    for (const [paneId, pane] of Object.entries(session.panes)) {
      if (pane?.currentPtyId === bookmark.ptyId) {
        return {
          sessionId: session.id,
          paneId: pane.id || paneId,
          ptyId: bookmark.ptyId,
          offset: Math.max(0, bookmark.offset),
        };
      }
    }
  }
  return null;
}

export function resetOutputReplayForBookmark(
  state: OutputReplayState,
  bookmark: Pick<PtyBookmarkLike, "ptyId" | "offset">,
): OutputReplayState {
  const offset = Math.max(0, bookmark.offset);
  return {
    outputChunks: { ...state.outputChunks, [bookmark.ptyId]: [] },
    offsets: { ...state.offsets, [bookmark.ptyId]: offset },
    jumpRevisions: {
      ...state.jumpRevisions,
      [bookmark.ptyId]: (state.jumpRevisions[bookmark.ptyId] ?? 0) + 1,
    },
  };
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
	return {
		sessions: event.type === "session.changed",
		ptys: event.type === "pty.changed",
		outputPtyId: event.type === "pty.output" ? (event.ptyId ?? null) : null,
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
