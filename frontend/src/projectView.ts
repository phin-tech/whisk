type ProjectLike = {
  id: string;
};

type ProjectDetailFor<T extends ProjectLike, TSession = unknown> = {
  project: T;
  workItems?: unknown[] | null;
  sessions?: TSession[] | null;
  runs?: unknown[] | null;
};

type ProjectDetailLike = ProjectDetailFor<ProjectLike> | null;

type RunLike = {
  updatedAt?: unknown;
  createdAt?: unknown;
};

type SessionLike = {
  id: string;
  name?: string;
  projectId?: string;
};

export function projectDetailCounts(detail: ProjectDetailLike) {
  return {
    workItems: detail?.workItems?.length ?? 0,
    sessions: detail?.sessions?.length ?? 0,
    runs: detail?.runs?.length ?? 0,
  };
}

export function selectedProjectDetail<T extends ProjectLike>(
  projects: T[],
  detail: ProjectDetailFor<T> | null,
  activeProjectId: string,
) {
  if (detail?.project.id === activeProjectId) return detail;
  const project = projects.find((candidate) => candidate.id === activeProjectId) ?? null;
  return project ? { project, workItems: [], sessions: [], runs: [] } : null;
}

export function sortRunsRecent<T extends RunLike>(runs: T[]) {
  return [...runs].sort((a, b) => runPriority(b) - runPriority(a) || runTime(b) - runTime(a));
}

export function sessionNameSuffix<T extends SessionLike>(session: T, sessions: T[]) {
  if (sessions.filter((candidate) => candidate.name === session.name).length < 2) return "";
  return session.id.replace(/^sess_/, "").slice(0, 6);
}

export function projectDetailWithStoreSessions<
  TDetail extends { project: ProjectLike; sessions: TSession[] },
  TSession extends SessionLike,
>(
  detail: TDetail | null,
  activeProjectId: string,
  sessions: TSession[],
) {
  if (!detail || detail.project.id !== activeProjectId) return detail;
  return {
    ...detail,
    sessions: sessions.filter((session) => session.projectId === activeProjectId),
  } as TDetail;
}

function runTime(run: RunLike) {
  const value = run.updatedAt || run.createdAt;
  if (!value) return 0;
  const parsed = value instanceof Date ? value.getTime() : new Date(String(value)).getTime();
  return Number.isNaN(parsed) ? 0 : parsed;
}

function runPriority(run: RunLike) {
  const status = "status" in run ? run.status : "";
  return status === "running" || status === "awaiting_input" || status === "queued" ? 1 : 0;
}
