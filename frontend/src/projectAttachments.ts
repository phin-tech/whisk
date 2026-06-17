import type { MetadataValue } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";

export type ProjectAttachmentLike = {
  id: string;
  kind: string;
  title?: string;
  path?: string;
  url?: string;
  note?: string;
  provider?: string;
  target?: string;
  includeInContext?: boolean;
};

export type ProjectAttachmentFormValues = {
  title: string;
  target: string;
  note: string;
  provider: string;
  includeInContext: boolean;
};

export type ProjectAttachmentUpdatePayload = {
  projectId: string;
  title: string;
  path: string;
  url: string;
  note: string;
  provider: string;
  target: string;
  includeInContext: boolean;
  meta?: Record<string, MetadataValue>;
};

export function parseGitHubIssueURL(value: string) {
  try {
    const url = new URL(value.trim());
    if (url.hostname !== "github.com") return null;
    const parts = url.pathname.split("/").filter(Boolean);
    if (parts.length !== 4 || parts[2] !== "issues" || !/^\d+$/.test(parts[3])) return null;
    return {
      repo: `${parts[0]}/${parts[1]}`,
      issue: parts[3],
      url: `https://github.com/${parts[0]}/${parts[1]}/issues/${parts[3]}`,
      target: `${parts[0]}/${parts[1]}#${parts[3]}`,
    };
  } catch {
    return null;
  }
}

export function isGitHubIssueAttachment(attachment: { kind?: string; provider?: string }) {
  return attachment.kind === "external" && attachment.provider === "github";
}

export function projectAttachmentEditValues(attachment: ProjectAttachmentLike): ProjectAttachmentFormValues {
  return {
    title: attachment.title ?? "",
    target: isGitHubIssueAttachment(attachment) ? (attachment.url ?? "") : (attachment.path || attachment.url || attachment.target || ""),
    note: attachment.note ?? "",
    provider: attachment.provider || "github",
    includeInContext: Boolean(attachment.includeInContext),
  };
}

export function buildProjectAttachmentUpdate(
  projectId: string,
  kind: string,
  values: ProjectAttachmentFormValues,
): ProjectAttachmentUpdatePayload | null {
  const target = values.target.trim();
  const note = values.note.trim();
  const provider = values.provider.trim();

  if ((kind === "file" || kind === "url" || kind === "external") && !target) return null;
  if (kind === "note" && !note) return null;

  if (kind === "external" && provider === "github") {
    const parsed = parseGitHubIssueURL(target);
    if (!parsed) return null;
    return {
      projectId,
      title: values.title.trim(),
      path: "",
      url: parsed.url,
      note: "",
      provider: "github",
      target: parsed.target,
      includeInContext: values.includeInContext,
      meta: {
        "github/type": { type: "string", string: "issue" },
        "github/repo": { type: "string", string: parsed.repo },
        "github/number": { type: "number", number: Number(parsed.issue) },
      },
    };
  }

  return {
    projectId,
    title: values.title.trim(),
    path: kind === "file" ? target : "",
    url: kind === "url" ? target : "",
    note: kind === "note" ? note : "",
    provider: kind === "external" ? provider : "",
    target: kind === "external" ? target : "",
    includeInContext: values.includeInContext,
  };
}
