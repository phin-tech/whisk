declare const idBrand: unique symbol;

export type Brand<K extends string> = string & { readonly [idBrand]: K };

export type SessionId = Brand<"SessionId">;
export type WindowId = Brand<"WindowId">;
export type PaneId = Brand<"PaneId">;
export type PtyId = Brand<"PtyId">;
export type ProjectId = Brand<"ProjectId">;
export type WorkItemId = Brand<"WorkItemId">;
export type RunId = Brand<"RunId">;
export type PaneKey = `${SessionId}:${PaneId}`;

export type PaneKeyParts = {
  sessionId: SessionId;
  paneId: PaneId;
};

type AnyId = Brand<string>;

export function parseSessionId(value: unknown): SessionId | null {
  return parseId<"SessionId">(value);
}

export function requireSessionId(value: unknown, field = "sessionId"): SessionId {
  return requireId<"SessionId">(value, field);
}

export function unsafeSessionId(value: string): SessionId {
  return unsafeId<"SessionId">(value);
}

export function parseWindowId(value: unknown): WindowId | null {
  return parseId<"WindowId">(value);
}

export function requireWindowId(value: unknown, field = "windowId"): WindowId {
  return requireId<"WindowId">(value, field);
}

export function unsafeWindowId(value: string): WindowId {
  return unsafeId<"WindowId">(value);
}

export function parsePaneId(value: unknown): PaneId | null {
  return parseId<"PaneId">(value);
}

export function requirePaneId(value: unknown, field = "paneId"): PaneId {
  return requireId<"PaneId">(value, field);
}

export function unsafePaneId(value: string): PaneId {
  return unsafeId<"PaneId">(value);
}

export function parsePtyId(value: unknown): PtyId | null {
  return parseId<"PtyId">(value);
}

export function requirePtyId(value: unknown, field = "ptyId"): PtyId {
  return requireId<"PtyId">(value, field);
}

export function unsafePtyId(value: string): PtyId {
  return unsafeId<"PtyId">(value);
}

export function parseProjectId(value: unknown): ProjectId | null {
  return parseId<"ProjectId">(value);
}

export function requireProjectId(value: unknown, field = "projectId"): ProjectId {
  return requireId<"ProjectId">(value, field);
}

export function unsafeProjectId(value: string): ProjectId {
  return unsafeId<"ProjectId">(value);
}

export function parseWorkItemId(value: unknown): WorkItemId | null {
  return parseId<"WorkItemId">(value);
}

export function requireWorkItemId(value: unknown, field = "workItemId"): WorkItemId {
  return requireId<"WorkItemId">(value, field);
}

export function unsafeWorkItemId(value: string): WorkItemId {
  return unsafeId<"WorkItemId">(value);
}

export function parseRunId(value: unknown): RunId | null {
  return parseId<"RunId">(value);
}

export function requireRunId(value: unknown, field = "runId"): RunId {
  return requireId<"RunId">(value, field);
}

export function unsafeRunId(value: string): RunId {
  return unsafeId<"RunId">(value);
}

export function idString(id: AnyId): string {
  return id;
}

export function optionalIdString(id: AnyId | null | undefined): string {
  return id ?? "";
}

export function sessionIdOf(value: { id?: string | null } | null | undefined): SessionId | null {
  return parseSessionId(value?.id);
}

export function paneIdOf(value: { id?: string | null } | null | undefined): PaneId | null {
  return parsePaneId(value?.id);
}

export function currentPtyIdOf(
  value: { currentPtyId?: string | null } | null | undefined,
): PtyId | null {
  return parsePtyId(value?.currentPtyId);
}

export function layoutPaneIdOf(value: { paneId?: string | null } | null | undefined): PaneId | null {
  return parsePaneId(value?.paneId);
}

export function ptyIdOf(
  value: { id?: string | null; ptyId?: string | null } | null | undefined,
): PtyId | null {
  if (!value) return null;
  return parsePtyId(value.id) ?? parsePtyId(value.ptyId);
}

export function projectIdOf(
  value: { id?: string | null; projectId?: string | null } | null | undefined,
): ProjectId | null {
  if (!value) return null;
  return parseProjectId(value.id) ?? parseProjectId(value.projectId);
}

export function workItemIdOf(
  value: { id?: string | null; workItemId?: string | null } | null | undefined,
): WorkItemId | null {
  if (!value) return null;
  return parseWorkItemId(value.id) ?? parseWorkItemId(value.workItemId);
}

export function runIdOf(
  value: { id?: string | null; runId?: string | null } | null | undefined,
): RunId | null {
  if (!value) return null;
  return parseRunId(value.id) ?? parseRunId(value.runId);
}

export function paneKey(sessionId: SessionId, paneId: PaneId): PaneKey {
  assertPaneKeyPart("sessionId", sessionId);
  assertPaneKeyPart("paneId", paneId);
  return `${sessionId}:${paneId}` as PaneKey;
}

export function parsePaneKey(value: unknown): PaneKeyParts | null {
  if (typeof value !== "string" || value !== value.trim()) return null;
  const parts = value.split(":");
  if (parts.length !== 2) return null;
  const [sessionIdPart, paneIdPart] = parts;
  if (!isPaneKeyPart(sessionIdPart) || !isPaneKeyPart(paneIdPart)) return null;
  const sessionId = parseSessionId(sessionIdPart);
  const paneId = parsePaneId(paneIdPart);
  if (!sessionId || !paneId) return null;
  return { sessionId, paneId };
}

export function requirePaneKey(value: unknown, field = "paneKey"): PaneKey {
  const parsed = parsePaneKey(value);
  if (!parsed) throw new Error(`${field} must be a sessionId:paneId string`);
  return paneKey(parsed.sessionId, parsed.paneId);
}

export function splitPaneKey(key: PaneKey): PaneKeyParts {
  return requirePaneKeyParts(key);
}

function parseId<K extends string>(value: unknown): Brand<K> | null {
  if (typeof value !== "string") return null;
  const trimmed = value.trim();
  if (!trimmed) return null;
  return trimmed as Brand<K>;
}

function requireId<K extends string>(value: unknown, field: string): Brand<K> {
  const parsed = parseId<K>(value);
  if (!parsed) throw new Error(`${field} must be a non-empty string`);
  return parsed;
}

function unsafeId<K extends string>(value: string): Brand<K> {
  return value as Brand<K>;
}

function requirePaneKeyParts(key: PaneKey): PaneKeyParts {
  const parsed = parsePaneKey(key);
  if (!parsed) throw new Error("paneKey must be a sessionId:paneId string");
  return parsed;
}

function assertPaneKeyPart(field: string, value: AnyId) {
  const stringValue = idString(value);
  if (!stringValue.trim()) throw new Error(`${field} must be a non-empty string`);
  if (stringValue !== stringValue.trim()) {
    throw new Error(`${field} must not contain surrounding whitespace`);
  }
  if (stringValue.includes(":")) throw new Error(`${field} must not contain ':'`);
}

function isPaneKeyPart(value: string) {
  return Boolean(value) && value === value.trim() && !value.includes(":");
}
