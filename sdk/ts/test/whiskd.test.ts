import createClient from "openapi-fetch";
import { describe, expect, inject, it } from "vitest";
import type { paths } from "../whiskd";

// End-to-end: the generated TS types + openapi-fetch against a live whiskd.
// Mirrors the Python suite so both clients are proven against real wire behavior.
const baseUrl = inject("baseUrl");

describe.skipIf(!baseUrl)("whiskd headless TS client", () => {
  const client = createClient<paths>({ baseUrl });

  it("compatibility handshake", async () => {
    const { data, error } = await client.GET("/v1/compat");
    expect(error).toBeUndefined();
    expect(data!.apiVersion).toBeGreaterThanOrEqual(1);
    expect(data!.gitSha).not.toEqual("");
  });

  it("sessions start empty", async () => {
    const { data, error } = await client.GET("/v1/sessions");
    expect(error).toBeUndefined();
    expect(data).toEqual([]);
  });

  it("work item round trip", async () => {
    const project = await client.POST("/v1/projects", {
      body: { name: "TS Integration", rootDir: process.cwd() },
    });
    expect(project.error).toBeUndefined();
    const projectId = project.data!.id;
    expect(projectId).toBeTruthy();

    const item = await client.POST("/v1/work-items", {
      body: { projectId, title: "hello from ts" },
    });
    expect(item.error).toBeUndefined();
    expect(item.data!.projectId).toEqual(projectId);
    expect(item.data!.number).toBeGreaterThanOrEqual(1);

    const list = await client.GET("/v1/work-items", {
      params: { query: { projectId } },
    });
    expect(list.error).toBeUndefined();
    expect(list.data!.some((i) => i.id === item.data!.id)).toBe(true);
  });
});
