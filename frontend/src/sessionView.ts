type LayoutNodeLike = {
  kind: string;
  paneId?: string;
  direction?: string;
  children?: LayoutNodeLike[];
};

type SessionLike = {
  id: string;
  layout: LayoutNodeLike;
  panes: { [_ in string]?: { id?: string; ptyId: string } };
};

export function paneIds(node: LayoutNodeLike | undefined): string[] {
  if (!node) return [];
  if (node.kind === "leaf") return node.paneId ? [node.paneId] : [];
  return (node.children ?? []).flatMap(paneIds);
}

export function visiblePtyIds(
  sessions: SessionLike[],
  activeSessionId: string,
  activePaneId: string,
): string[] {
  const ptys: string[] = [];
  const seen = new Set<string>();
  const activeSession = sessions.find((session) => session.id === activeSessionId);
  const activePtyID = activeSession?.panes[activePaneId]?.ptyId;
  if (activePtyID) {
    ptys.push(activePtyID);
    seen.add(activePtyID);
  }
  for (const session of sessions) {
    for (const paneID of paneIds(session.layout)) {
      const ptyID = session.panes[paneID]?.ptyId;
      if (ptyID && !seen.has(ptyID)) {
        ptys.push(ptyID);
        seen.add(ptyID);
      }
    }
  }
  return ptys;
}
