import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
import type { Project } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
import { sessionGroups, type SessionGroupMode } from "./sessionView";

export type SessionsPanelRow =
  | {
      kind: "group";
      key: string;
      groupId: string;
      title: string;
      count: number;
      collapsed: boolean;
    }
  | {
      kind: "session";
      key: string;
      session: Session;
      active: boolean;
      confirmingClose: boolean;
    };

export type SessionsPanelInput = {
  sessions: Session[];
  projects: Project[];
  groupMode: SessionGroupMode;
  query: string;
  collapsedGroupIds: Set<string>;
  activeSessionId: string;
  confirmingSessionId: string;
};

export function deriveSessionsPanelRows(input: SessionsPanelInput): SessionsPanelRow[] {
  const {
    sessions,
    projects,
    groupMode,
    query,
    collapsedGroupIds,
    activeSessionId,
    confirmingSessionId,
  } = input;

  const groups = sessionGroups(sessions, projects, groupMode, query);
  const rows: SessionsPanelRow[] = [];

  for (const group of groups) {
    const collapsed = collapsedGroupIds.has(group.id);
    rows.push({
      kind: "group",
      key: `group:${group.id}`,
      groupId: group.id,
      title: group.title,
      count: group.sessions.length,
      collapsed,
    });
    if (!collapsed) {
      for (const session of group.sessions) {
        rows.push({
          kind: "session",
          key: `session:${session.id}`,
          session,
          active: session.id === activeSessionId,
          confirmingClose: session.id === confirmingSessionId,
        });
      }
    }
  }

  return rows;
}

export const SESSION_HEADER_HEIGHT = 28;
export const SESSION_ROW_HEIGHT = 52;

export function sessionRowHeight(row: SessionsPanelRow): number {
  return row.kind === "group" ? SESSION_HEADER_HEIGHT : SESSION_ROW_HEIGHT;
}
