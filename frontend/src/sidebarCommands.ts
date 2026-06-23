export type SidebarId = "sessions" | "ptys" | "work" | "projects" | "notifications";
export type MainView = "session" | "work" | "projects";

export function nextSidebarAfterToggle(activeSidebar: SidebarId | null, activeMain: MainView): SidebarId | null {
  if (activeSidebar) return null;
  if (activeMain === "work") return "work";
  if (activeMain === "projects") return "projects";
  return "sessions";
}
