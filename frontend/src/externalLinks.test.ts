import { describe, expect, it } from "vitest";
import { externalAttachmentURL, openExternalURL } from "./externalLinks";

describe("externalLinks", () => {
  it("uses only stored attachment urls as external link targets", () => {
    expect(externalAttachmentURL({ url: " https://github.com/phin-tech/roux-next-gen/issues/123 " })).toBe(
      "https://github.com/phin-tech/roux-next-gen/issues/123",
    );
    expect(externalAttachmentURL({})).toBe("");
  });

  it("opens external links through the native Wails browser bridge", async () => {
    const opened: string[] = [];

    await openExternalURL(
      " https://github.com/phin-tech/roux-next-gen/issues/123 ",
      async (url) => {
        opened.push(url);
      },
      () => {
        throw new Error("fallback should not run");
      },
    );

    expect(opened).toEqual(["https://github.com/phin-tech/roux-next-gen/issues/123"]);
  });

  it("falls back to window opening when the native bridge is unavailable", async () => {
    const fallbackOpened: string[] = [];

    await openExternalURL(
      "https://github.com/phin-tech/roux-next-gen/issues/123",
      async () => {
        throw new Error("runtime unavailable");
      },
      (url) => {
        fallbackOpened.push(url);
      },
    );

    expect(fallbackOpened).toEqual(["https://github.com/phin-tech/roux-next-gen/issues/123"]);
  });
});
