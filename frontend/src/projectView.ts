type ProjectLike = {
  id: string;
};

type ProjectDetailFor<T extends ProjectLike> = {
  project: T;
  workItems?: unknown[] | null;
  sessions?: unknown[] | null;
  runs?: unknown[] | null;
};

type ProjectDetailLike = ProjectDetailFor<ProjectLike> | null;

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
