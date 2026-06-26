import { expect, test } from "@playwright/test";

test("manages project workflow files and previews migration", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "Projects" }).click();
  await expect(page.getByText("Whisk E2E")).toBeVisible();

  await page.locator("button").filter({ hasText: "plan-execute-review@1" }).last().click();
  await page.getByRole("menuitem", { name: /lean-review@1/ }).click();

  const migrationPlan = page.getByTestId("workflow-migration-plan");
  await expect(migrationPlan.getByText("Pending workflow")).toBeVisible();
  await expect(migrationPlan.getByText("items")).toBeVisible();
  await expect(migrationPlan.getByText("pinned")).toBeVisible();
  await expect(migrationPlan.getByText("incompatible")).toBeVisible();

  await migrationPlan.getByRole("button", { name: "Apply" }).click();
  await expect.poll(async () => {
    const calls = await page.evaluate(() =>
      window.__WHISK_E2E__.calls().filter((call) => call.method.endsWith(".SetProjectWorkflowDefinition")).length,
    );
    return calls;
  }).toBe(1);
  const setWorkflowCalls = await page.evaluate(() =>
    window.__WHISK_E2E__.calls().filter((call) => call.method.endsWith(".SetProjectWorkflowDefinition")),
  );
  expect(setWorkflowCalls[0].args[1]).toMatchObject({ id: "lean-review", version: 1 });

  const fileControls = page.getByTestId("workflow-file-controls");
  await fileControls.getByLabel("Workflow file path").fill("/tmp/whisk-e2e/custom-flow.json");
  await fileControls.getByRole("button", { name: "Validate", exact: true }).click();
  await expect(fileControls.getByText("valid", { exact: true })).toBeVisible();
  await expect(fileControls.getByText("custom-flow@1")).toBeVisible();

  await fileControls.getByRole("button", { name: "Import", exact: true }).click();
  await page.locator("button").filter({ hasText: /plan-execute-review@1|lean-review@1/ }).last().click();
  await page.getByRole("menuitem", { name: /custom-flow@1/ }).click();
  await expect(migrationPlan.getByText("Pending workflow")).toBeVisible();

  await fileControls.getByLabel("Workflow export path").fill("/tmp/whisk-e2e/exported-flow.json");
  await fileControls.getByRole("button", { name: "Export", exact: true }).click();

  page.once("dialog", (dialog) => dialog.accept());
  await fileControls.getByRole("button", { name: "Delete", exact: true }).click();

  const workflowCalls = await page.evaluate(() =>
    window.__WHISK_E2E__.calls().map((call) => call.method),
  );
  expect(workflowCalls.some((method) => method.endsWith(".PlanProjectWorkflowMigration"))).toBe(true);
  expect(workflowCalls.some((method) => method.endsWith(".SetProjectWorkflowDefinition"))).toBe(true);
  expect(workflowCalls.some((method) => method.endsWith(".ValidateWorkflowDefinitionFile"))).toBe(true);
  expect(workflowCalls.some((method) => method.endsWith(".ImportWorkflowDefinitionFile"))).toBe(true);
  expect(workflowCalls.some((method) => method.endsWith(".ExportWorkflowDefinitionFile"))).toBe(true);
  expect(workflowCalls.some((method) => method.endsWith(".DeleteWorkflowDefinition"))).toBe(true);
});
