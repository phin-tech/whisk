import type { JumpTarget } from "./jumpFilter";

type LayoutNodeLike = {
  kind: string;
  paneId?: string;
  children?: readonly LayoutNodeLike[];
};

type WindowLike = {
  id: string;
  name?: string;
  layout?: LayoutNodeLike;
};

type PaneLike = {
  id?: string;
  windowId?: string;
  currentPtyId?: string | null;
  workingDir?: string;
};

export type SessionLike = {
  id: string;
  projectId?: string;
  name?: string;
  rootDir?: string;
  windows?: { [_ in string]?: WindowLike };
  panes?: { [_ in string]?: PaneLike };
};

export type ProjectLike = {
  id: string;
  name?: string;
  slug?: string;
  rootDir?: string;
  description?: string;
};

export type PTYInfoLike = {
  id: string;
  workingDir?: string;
  running?: boolean;
  status?: string;
  sessionId?: string;
  windowId?: string;
  paneId?: string;
};

export type WorkItemLike = {
  id: string;
  projectId: string;
  number: number;
  title: string;
  stageId?: string;
  runState?: string;
  bodyMarkdown?: string;
};

export type WorkItemRunLike = {
  id: string;
  projectId: string;
  workItemId: string;
  status?: string;
  preset?: string;
  sessionId?: string;
  ptyId?: string;
  createdAt?: unknown;
  updatedAt?: unknown;
};

export type JumpTargetsInput = {
  sessions?: readonly SessionLike[];
  activeSessionId?: string;
  activePaneId?: string;
  ptys?: readonly PTYInfoLike[];
  projects?: readonly ProjectLike[];
  activeProjectId?: string;
  workItems?: readonly WorkItemLike[];
  workItemRuns?: readonly WorkItemRunLike[];
  openWorkItemId?: string;
};

type PaneEntry = {
  paneId: string;
  pane: PaneLike;
  window: WindowLike | null;
};

type AttachedPTYTarget = {
  session: SessionLike;
  paneId: string;
  windowId: string;
};

export function deriveJumpTargets(input: JumpTargetsInput): JumpTarget[] {
  const sessions = input.sessions ?? [];
  const ptys = input.ptys ?? [];
  const projects = input.projects ?? [];
  const workItems = input.workItems ?? [];
  const workItemRuns = input.workItemRuns ?? [];

  const projectsById = new Map(projects.map((project) => [project.id, project]));
  const sessionsById = new Map(sessions.map((session) => [session.id, session]));
  const ptysById = new Map(ptys.map((pty) => [pty.id, pty]));
  const workItemsById = new Map(workItems.map((item) => [item.id, item]));
  const activePtyId = currentPtyId(sessions, input.activeSessionId ?? "", input.activePaneId ?? "");

  const targets: JumpTarget[] = [];

  for (const session of sessions) {
    if (!session.id) continue;
    const project = session.projectId ? projectsById.get(session.projectId) : undefined;
    targets.push({
      id: `session:${session.id}`,
      kind: "session",
      title: sessionTitle(session),
      subtitle: projectLabel(project),
      detail: session.rootDir,
      keywords: compact([
        session.id,
        session.projectId,
        session.rootDir,
        project?.id,
        project?.name,
        project?.slug,
        project?.rootDir,
      ]),
      current: session.id === input.activeSessionId,
      payload: {
        kind: "session",
        sessionId: session.id,
        ...(session.projectId ? { projectId: session.projectId } : {}),
      },
    });
  }

  for (const session of sessions) {
    for (const entry of orderedPanes(session)) {
      const ptyId = entry.pane.currentPtyId ?? "";
      const pty = ptyId ? ptysById.get(ptyId) : undefined;
      const windowId = entry.window?.id || entry.pane.windowId || pty?.windowId || "";
      const active = session.id === input.activeSessionId && entry.paneId === input.activePaneId;
      targets.push({
        id: `pane:${session.id}:${entry.paneId}`,
        kind: "pane",
        title: `${sessionTitle(session)} pane ${shortId(entry.paneId)}`,
        subtitle: pty?.workingDir || entry.pane.workingDir || session.rootDir,
        detail: compact([entry.window?.name, pty ? `PTY ${shortId(pty.id)}` : "", pty?.status]).join(" / "),
        keywords: compact([
          entry.paneId,
          entry.pane.currentPtyId ?? "",
          entry.pane.workingDir,
          entry.window?.id,
          entry.window?.name,
          session.id,
          session.name,
          session.rootDir,
          pty?.id,
          pty?.workingDir,
          pty?.status,
        ]),
        current: active,
        payload: {
          kind: "pane",
          sessionId: session.id,
          ...(windowId ? { windowId } : {}),
          paneId: entry.paneId,
          ...(ptyId ? { ptyId } : {}),
        },
      });
    }
  }

  for (const pty of ptys) {
    if (!pty.id) continue;
    const attached = attachedPaneForPTY(pty, sessions, sessionsById);
    if (!attached) continue;
    const session = attached.session;
    const project = session?.projectId ? projectsById.get(session.projectId) : undefined;
    const active =
      pty.id === activePtyId ||
      Boolean(session.id === input.activeSessionId && attached.paneId === input.activePaneId);
    targets.push({
      id: `pty:${pty.id}`,
      kind: "pty",
      title: `PTY ${shortId(pty.id)}`,
      subtitle: pty.workingDir,
      detail: compact([sessionTitle(session), pty.status || (pty.running ? "running" : "")]).join(" / "),
      keywords: compact([
        pty.id,
        pty.workingDir,
        pty.status,
        pty.sessionId,
        pty.windowId,
        pty.paneId,
        session?.name,
        session?.rootDir,
        project?.name,
        project?.slug,
      ]),
      current: active,
      payload: {
        kind: "pty",
        ptyId: pty.id,
        sessionId: session.id,
        ...(attached.windowId ? { windowId: attached.windowId } : {}),
        paneId: attached.paneId,
      },
    });
  }

  for (const project of projects) {
    if (!project.id) continue;
    targets.push({
      id: `project:${project.id}`,
      kind: "project",
      title: projectTitle(project),
      subtitle: project.slug,
      detail: project.rootDir,
      keywords: compact([project.id, project.slug, project.rootDir, project.description]),
      current: project.id === input.activeProjectId,
      payload: { kind: "project", projectId: project.id },
    });
  }

  for (const item of activeProjectItems(workItems, input.activeProjectId ?? "")) {
    targets.push({
      id: `work-item:${item.id}`,
      kind: "work-item",
      title: workItemTitle(item),
      subtitle: compact([item.stageId, item.runState]).join(" / "),
      detail: projectTitle(projectsById.get(item.projectId)),
      keywords: compact([
        item.id,
        item.projectId,
        item.number ? String(item.number) : "",
        item.number ? `#${item.number}` : "",
        item.stageId,
        item.runState,
        item.bodyMarkdown,
      ]),
      current: item.id === input.openWorkItemId,
      payload: { kind: "work-item", projectId: item.projectId, workItemId: item.id },
    });
  }

  for (const run of activeProjectRuns(workItemRuns, input.activeProjectId ?? "")) {
    if (!run.sessionId && !run.ptyId) continue;
    const item = workItemsById.get(run.workItemId);
    const session = run.sessionId ? sessionsById.get(run.sessionId) : undefined;
    const projectId = run.projectId || item?.projectId || "";
    const active = Boolean(
      (run.ptyId && run.ptyId === activePtyId) || run.workItemId === input.openWorkItemId,
    );
    targets.push({
      id: `work-item-run:${run.id}`,
      kind: "work-item-run",
      title: item ? `Run for ${workItemTitle(item)}` : `Run ${shortId(run.id)}`,
      subtitle: compact([run.status, run.preset]).join(" / "),
      detail: compact([sessionTitle(session), run.ptyId]).join(" / "),
      keywords: compact([
        run.id,
        run.projectId,
        run.workItemId,
        run.status,
        run.preset,
        run.sessionId,
        run.ptyId,
        item?.number ? String(item.number) : "",
        item?.number ? `#${item.number}` : "",
        item?.title,
      ]),
      current: active,
      payload: {
        kind: "work-item-run",
        projectId,
        workItemId: run.workItemId,
        runId: run.id,
        ...(run.sessionId ? { sessionId: run.sessionId } : {}),
        ...(run.ptyId ? { ptyId: run.ptyId } : {}),
      },
    });
  }

  return currentTargetsFirst(targets);
}

function activeProjectItems(items: readonly WorkItemLike[], activeProjectId: string): WorkItemLike[] {
  if (!activeProjectId) return [...items];
  return items.filter((item) => item.projectId === activeProjectId);
}

function activeProjectRuns(runs: readonly WorkItemRunLike[], activeProjectId: string): WorkItemRunLike[] {
  if (!activeProjectId) return [...runs];
  return runs.filter((run) => run.projectId === activeProjectId);
}

function currentTargetsFirst(targets: JumpTarget[]): JumpTarget[] {
  return targets
    .map((target, index) => ({ target, index }))
    .sort((a, b) => {
      if (Boolean(a.target.current) !== Boolean(b.target.current)) return a.target.current ? -1 : 1;
      return a.index - b.index;
    })
    .map((entry) => entry.target);
}

function orderedPanes(session: SessionLike): PaneEntry[] {
  const result: PaneEntry[] = [];
  const seen = new Set<string>();
  const panes = session.panes ?? {};
  const windows = Object.values(session.windows ?? {}).filter((window): window is WindowLike => Boolean(window));

  for (const window of windows) {
    for (const paneId of paneIds(window.layout)) {
      const pane = panes[paneId];
      if (!pane || seen.has(paneId)) continue;
      result.push({ paneId: pane.id || paneId, pane: { ...pane, id: pane.id || paneId }, window });
      seen.add(paneId);
    }
  }

  for (const [paneKey, pane] of Object.entries(panes)) {
    if (!pane || seen.has(paneKey)) continue;
    const paneId = pane.id || paneKey;
    result.push({
      paneId,
      pane: { ...pane, id: paneId },
      window: pane.windowId ? windows.find((candidate) => candidate.id === pane.windowId) ?? null : null,
    });
    seen.add(paneKey);
  }

  return result;
}

function paneIds(node: LayoutNodeLike | undefined): string[] {
  if (!node) return [];
  if (node.kind === "leaf") return node.paneId ? [node.paneId] : [];
  return (node.children ?? []).flatMap(paneIds);
}

function currentPtyId(sessions: readonly SessionLike[], activeSessionId: string, activePaneId: string): string {
  if (!activeSessionId || !activePaneId) return "";
  const session = sessions.find((candidate) => candidate.id === activeSessionId);
  return session?.panes?.[activePaneId]?.currentPtyId ?? "";
}

function attachedPaneForPTY(
  pty: PTYInfoLike,
  sessions: readonly SessionLike[],
  sessionsById: ReadonlyMap<string, SessionLike>,
): AttachedPTYTarget | null {
  const candidateSessions = pty.sessionId
    ? [sessionsById.get(pty.sessionId)].filter((session): session is SessionLike => Boolean(session))
    : sessions;

  for (const session of candidateSessions) {
    const entries = orderedPanes(session);
    const preferred = pty.paneId
      ? entries.find((entry) => entry.paneId === pty.paneId || entry.pane.id === pty.paneId)
      : undefined;
    const match =
      preferred?.pane.currentPtyId === pty.id
        ? preferred
        : entries.find((entry) => entry.pane.currentPtyId === pty.id);
    if (!match) continue;
    return {
      session,
      paneId: match.paneId,
      windowId: match.window?.id || match.pane.windowId || pty.windowId || "",
    };
  }

  return null;
}

function sessionTitle(session: SessionLike | null | undefined): string {
  if (!session) return "";
  return session.name || basename(session.rootDir) || session.id;
}

function projectTitle(project: ProjectLike | null | undefined): string {
  if (!project) return "";
  return project.name || project.slug || basename(project.rootDir) || project.id;
}

function projectLabel(project: ProjectLike | null | undefined): string | undefined {
  if (!project) return undefined;
  return projectTitle(project);
}

function workItemTitle(item: WorkItemLike): string {
  const prefix = item.number ? `#${item.number} ` : "";
  return `${prefix}${item.title || item.id}`;
}

function basename(value: string | undefined): string {
  if (!value) return "";
  let end = value.length;
  while (end > 0 && (value[end - 1] === "/" || value[end - 1] === "\\")) end -= 1;
  const trimmed = value.slice(0, end);
  const slash = Math.max(trimmed.lastIndexOf("/"), trimmed.lastIndexOf("\\"));
  return slash >= 0 ? trimmed.slice(slash + 1) : trimmed;
}

function shortId(value: string): string {
  for (const prefix of ["sess_", "pane_", "pty_", "proj_", "item_", "run_"]) {
    if (value.startsWith(prefix)) return value.slice(prefix.length, prefix.length + 8);
  }
  return value.slice(0, 12);
}

function compact(values: readonly (string | null | undefined | false)[]): string[] {
  return values.map((value) => String(value || "").trim()).filter(Boolean);
}
