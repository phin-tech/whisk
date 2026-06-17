import { describe, expect, it } from "vitest";
import {
  buildProjectAttachmentUpdate,
  parseGitHubIssueURL,
  projectAttachmentEditValues,
} from "./projectAttachments";

describe("projectAttachments", () => {
  it("parses GitHub issue urls into canonical attachment identity", () => {
    expect(parseGitHubIssueURL(" https://github.com/phin-tech/roux-next-gen/issues/123 ")).toEqual({
      repo: "phin-tech/roux-next-gen",
      issue: "123",
      url: "https://github.com/phin-tech/roux-next-gen/issues/123",
      target: "phin-tech/roux-next-gen#123",
    });
    expect(parseGitHubIssueURL("https://github.com/phin-tech/roux-next-gen/pull/123")).toBeNull();
  });

  it("prefills GitHub issue edits from the user-facing url", () => {
    expect(
      projectAttachmentEditValues({
        id: "att_01",
        kind: "external",
        provider: "github",
        target: "phin-tech/roux-next-gen#122",
        url: "https://github.com/phin-tech/roux-next-gen/issues/123",
        title: "Issue",
        includeInContext: true,
      }),
    ).toEqual({
      title: "Issue",
      target: "https://github.com/phin-tech/roux-next-gen/issues/123",
      note: "",
      provider: "github",
      includeInContext: true,
    });
  });

  it("builds a GitHub issue update from the edited url", () => {
    expect(
      buildProjectAttachmentUpdate("proj_01", "external", {
        title: "Updated issue",
        target: "https://github.com/phin-tech/roux-next-gen/issues/456",
        note: "",
        provider: "github",
        includeInContext: true,
      }),
    ).toEqual({
      projectId: "proj_01",
      title: "Updated issue",
      path: "",
      url: "https://github.com/phin-tech/roux-next-gen/issues/456",
      note: "",
      provider: "github",
      target: "phin-tech/roux-next-gen#456",
      includeInContext: true,
      meta: {
        "github/type": { type: "string", string: "issue" },
        "github/repo": { type: "string", string: "phin-tech/roux-next-gen" },
        "github/number": { type: "number", number: 456 },
      },
    });
  });

  it("rejects invalid GitHub issue updates", () => {
    expect(
      buildProjectAttachmentUpdate("proj_01", "external", {
        title: "",
        target: "https://github.com/phin-tech/roux-next-gen/pull/456",
        note: "",
        provider: "github",
        includeInContext: true,
      }),
    ).toBeNull();
  });
});
