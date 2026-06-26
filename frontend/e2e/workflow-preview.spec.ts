import { expect, test } from "@playwright/test";

test("opens the project workflow preview DAG", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "Projects" }).click();
  await expect(page.getByText("Whisk E2E")).toBeVisible();

  await page.getByRole("button", { name: /Workflow\s+plan-execute-review@1/ }).click();

  const dialog = page.getByRole("dialog", { name: "Workflow preview" });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByText("7 stages")).toBeVisible();
  await expect(dialog.getByText("9 actions")).toBeVisible();
  await expect(dialog.getByText("1 gates")).toBeVisible();
  await expect(dialog.locator(".workflow-preview-node", { hasText: "backlog" })).toBeVisible();
  await expect(dialog.locator(".workflow-preview-node", { hasText: "done" })).toBeVisible();
  await expect(dialog.locator(".svelte-flow__edge-label", { hasText: "start_planning" })).toBeVisible();
  await expect(dialog.locator(".svelte-flow__edge-label").first()).toHaveCSS("background-color", "rgb(24, 24, 27)");
  await expect(dialog.locator(".svelte-flow")).toBeVisible();
  await expect(dialog.getByText("Stage config")).toBeVisible();
  await expect(dialog.getByText("Legend")).toBeVisible();
  await expect(dialog.locator("dt", { hasText: "stageId" })).toBeVisible();
  await expect(dialog.locator("dt", { hasText: "incomingActions" })).toBeVisible();

  await dialog.locator(".workflow-preview-node", { hasText: "planning" }).click();
  await expect(dialog.getByText("submit_draft_plan, approve_plan, report_blocked")).toBeVisible();

  await dialog.locator(".workflow-preview-edge").first().click();
  await expect(dialog.getByText("Action config")).toBeVisible();
  await expect(dialog.locator("dt", { hasText: "actionId" })).toBeVisible();
  await expect(dialog.locator("dd").filter({ hasText: /^start_planning$/ })).toBeVisible();

  await dialog.getByRole("button", { name: "review", exact: true }).click();
  await expect(dialog.getByText("Gate config")).toBeVisible();
  await expect(dialog.locator("dt", { hasText: "gateId" })).toBeVisible();
  await expect(dialog.locator("dd").filter({ hasText: /^review$/ }).first()).toBeVisible();
});
