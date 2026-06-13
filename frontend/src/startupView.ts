export type StartupView = "sessions" | "kanban";
export type StartupMainView = "session" | "work";
export type StartupSidebar = "sessions" | "work";

export type StartupTarget = {
  main: StartupMainView;
  sidebar: StartupSidebar;
};

export function normalizeStartupView(value: unknown): StartupView {
  return value === "kanban" ? "kanban" : "sessions";
}

export function startupTarget(view: StartupView): StartupTarget {
  if (view === "kanban") {
    return { main: "work", sidebar: "work" };
  }
  return { main: "session", sidebar: "sessions" };
}
