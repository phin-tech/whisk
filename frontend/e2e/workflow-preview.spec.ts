import { expect, test } from "@playwright/test";

test("opens the project workflow preview DAG", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "Projects" }).click();
  await expect(page.getByText("Whisk E2E")).toBeVisible();

  await page.getByRole("button", { name: /Workflow\s+plan-execute-review@1/ }).click();

  const dialog = page.getByRole("dialog", { name: "Workflow preview" });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByText("6 stages")).toBeVisible();
  await expect(dialog.getByText("5 actions")).toBeVisible();
  await expect(dialog.getByText("2 gates")).toBeVisible();
  await expect(dialog.locator(".workflow-preview-node", { hasText: "backlog" })).toBeVisible();
  await expect(dialog.locator(".workflow-preview-node", { hasText: "done" })).toBeVisible();
  await expect(dialog.locator(".svelte-flow__edge-label", { hasText: "start_planning" })).toBeVisible();
  await expect(dialog.locator(".svelte-flow__edge-label").first()).toHaveCSS("background-color", "rgb(24, 24, 27)");
  await expect(dialog.locator(".svelte-flow")).toBeVisible();
  await expect(dialog.getByText("Stage config")).toBeVisible();
  await expect(dialog.getByText("Legend")).toBeVisible();
  await expect(dialog.locator("pre")).toContainText('"stageId": "backlog"');

  await dialog.locator(".workflow-preview-node", { hasText: "planning" }).click();
  await expect(dialog.locator("pre")).toContainText('"stageId": "planning"');
  await expect(dialog.locator("pre")).toContainText('"gateId": "plan_approval"');

  await dialog.locator(".workflow-preview-edge").first().click();
  await expect(dialog.getByText("Action config")).toBeVisible();
  await expect(dialog.locator("pre")).toContainText('"actionId": "start_planning"');

  await dialog.getByRole("button", { name: "plan_approval" }).click();
  await expect(dialog.getByText("Gate config")).toBeVisible();
  await expect(dialog.locator("pre")).toContainText('"gateId": "plan_approval"');
});
