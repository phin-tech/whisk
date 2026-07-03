import createClient from "openapi-fetch";
import { describe, expect, inject, it } from "vitest";
import { whiskdClientOptions } from "../../../ts/client";
import type { paths } from "../../../ts/whiskd";

// End-to-end: the generated TS types + openapi-fetch against a live whiskd.
// Mirrors the Python suite so both clients are proven against real wire behavior.
const baseUrl = inject("baseUrl");
const stateHome = inject("stateHome");
if (stateHome) process.env.XDG_STATE_HOME = stateHome;

describe.skipIf(!baseUrl)("whiskd headless TS client", () => {
  const client = createClient<paths>(whiskdClientOptions({ baseUrl }));

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

  it("daemon clear resets work item state", async () => {
    const project = await client.POST("/v1/projects", {
      body: { name: "TS Clear", rootDir: process.cwd() },
    });
    expect(project.error).toBeUndefined();

    const item = await client.POST("/v1/work-items", {
      body: { projectId: project.data!.id, title: "clear me" },
    });
    expect(item.error).toBeUndefined();

    const cleared = await client.POST("/v1/daemon/clear", { body: {} });
    expect(cleared.error).toBeUndefined();
    expect(cleared.data!.projectsCleared).toBeGreaterThanOrEqual(1);
    expect(cleared.data!.workItemsCleared).toBeGreaterThanOrEqual(1);

    const list = await client.GET("/v1/work-items");
    expect(list.error).toBeUndefined();
    expect(list.data).toEqual([]);
  });

  it("work item round trip", async () => {
    const project = await client.POST("/v1/projects", {
      body: { name: "TS Integration", rootDir: process.cwd() },
    });
    expect(project.error).toBeUndefined();
    const projectId = project.data!.id;
    expect(projectId).toBeTruthy();

    const attached = await client.POST("/v1/projects/{projectID}/attachments", {
      params: { path: { projectID: projectId } },
      body: {
        projectId,
        kind: "note",
        title: "Context note",
        note: "remember this",
        includeInContext: true,
      },
    });
    expect(attached.error).toBeUndefined();
    expect(attached.data!.attachments).toHaveLength(1);
    const context = await client.GET("/v1/projects/{projectID}/context", {
      params: { path: { projectID: projectId } },
    });
    expect(context.error).toBeUndefined();
    expect(context.data!.items![0].content).toEqual("remember this");

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

  it("workflow round trip", async () => {
    const project = await client.POST("/v1/projects", {
      body: { name: "TS Workflow", rootDir: process.cwd() },
    });
    expect(project.error).toBeUndefined();

    const item = await client.POST("/v1/work-items", {
      body: { projectId: project.data!.id, title: "workflow from ts" },
    });
    expect(item.error).toBeUndefined();
    const workItemId = item.data!.id;

    const planning = await client.POST("/v1/work-items/{workItemID}/start-planning", {
      params: { path: { workItemID: workItemId } },
      body: { workItemId: workItemId, actor: "vitest" },
    });
    expect(planning.error).toBeUndefined();

    const draft = await client.POST("/v1/work-items/{workItemID}/plan-drafts", {
      params: { path: { workItemID: workItemId } },
      body: {
        workItemId: workItemId,
        runId: planning.data!.id,
        title: "Test plan",
        body: "1. Change the code\n2. Run tests",
        actor: "vitest",
      },
    });
    expect(draft.error).toBeUndefined();
    expect(draft.data!.kind).toEqual("plan");
    expect(draft.data!.status).toEqual("draft");

    const ready = await client.POST("/v1/work-items/{workItemID}/approve-plan", {
      params: { path: { workItemID: workItemId } },
      body: { workItemId: workItemId, artifactId: draft.data!.id, actor: "human" },
    });
    expect(ready.error).toBeUndefined();
    expect(ready.data!.stageId).toEqual("ready");

    const execution = await client.POST("/v1/work-items/{workItemID}/start-execution", {
      params: { path: { workItemID: workItemId } },
      body: { workItemId: workItemId, actor: "vitest" },
    });
    expect(execution.error).toBeUndefined();

    const question = await client.POST("/v1/questions", {
      body: {
        workItemId: workItemId,
        runId: execution.data!.id,
        prompt: "Which branch should I use?",
        actor: "agent",
      },
    });
    expect(question.error).toBeUndefined();
    expect(question.data!.status).toEqual("open");

    const answered = await client.POST("/v1/questions/{questionID}/answer", {
      params: { path: { questionID: question.data!.id } },
      body: { id: question.data!.id, answer: "Use the current branch.", actor: "human" },
    });
    expect(answered.error).toBeUndefined();
    expect(answered.data!.status).toEqual("answered");

    const questions = await client.GET("/v1/questions", {
      params: { query: { workItemId: workItemId } },
    });
    expect(questions.error).toBeUndefined();
    expect(questions.data).toHaveLength(1);
    expect(questions.data![0].id).toEqual(question.data!.id);

    const review = await client.POST("/v1/work-items/{workItemID}/complete-execution", {
      params: { path: { workItemID: workItemId } },
      body: {
        workItemId: workItemId,
        runId: execution.data!.id,
        message: "ready for review",
        actor: "vitest",
      },
    });
    expect(review.error).toBeUndefined();
    expect(review.data!.stageId).toEqual("review");

    const feedback = await client.POST("/v1/work-items/{workItemID}/review-feedback", {
      params: { path: { workItemID: workItemId } },
      body: {
        workItemId: workItemId,
        runId: execution.data!.id,
        body: "Please tighten the assertions.",
        actor: "reviewer",
      },
    });
    expect(feedback.error).toBeUndefined();
    expect(feedback.data!.kind).toEqual("feedback");

    const artifacts = await client.GET("/v1/artifacts", {
      params: { query: { workItemId: workItemId } },
    });
    expect(artifacts.error).toBeUndefined();
    expect(artifacts.data!.map((artifact) => artifact.kind).sort()).toEqual(["feedback", "plan"]);

    const secondReview = await client.POST("/v1/work-items/{workItemID}/complete-execution", {
      params: { path: { workItemID: workItemId } },
      body: {
        workItemId: workItemId,
        runId: execution.data!.id,
        message: "ready after feedback",
        actor: "vitest",
      },
    });
    expect(secondReview.error).toBeUndefined();
    expect(secondReview.data!.stageId).toEqual("review");

    const gates = await client.GET("/v1/gate-reports", {
      params: { query: { workItemId: workItemId } },
    });
    expect(gates.error).toBeUndefined();
    expect(gates.data).toHaveLength(1);
    expect(gates.data![0].blocking).toEqual(true);
    expect(gates.data![0].status).toEqual("pending");

    const blockedDone = await client.POST("/v1/work-items/{workItemID}/approve-done", {
      params: { path: { workItemID: workItemId } },
      body: { workItemId: workItemId, actor: "human" },
    });
    expect(blockedDone.data).toBeUndefined();
    expect(blockedDone.error).toBeDefined();

    const passedGate = await client.POST("/v1/gate-reports/{gateReportID}/complete", {
      params: { path: { gateReportID: gates.data![0].id } },
      body: { id: gates.data![0].id, status: "passed", actor: "vitest" },
    });
    expect(passedGate.error).toBeUndefined();
    expect(passedGate.data!.status).toEqual("passed");

    const done = await client.POST("/v1/work-items/{workItemID}/approve-done", {
      params: { path: { workItemID: workItemId } },
      body: { workItemId: workItemId, reason: "review gate passed", actor: "human" },
    });
    expect(done.error).toBeUndefined();
    expect(done.data!.stageId).toEqual("done");

    const events = await client.GET("/v1/workflow-events", {
      params: { query: { workItemId: workItemId } },
    });
    expect(events.error).toBeUndefined();
    expect(events.data!.at(-1)!.type).toEqual("done_approved");
  });
});
