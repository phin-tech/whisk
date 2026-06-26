import { expect, test } from "@playwright/test";

test("renders seeded work cards and opens the detail modal", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "Work" }).click();

  await expect(page.getByText("Polish WorkBoard cards")).toBeVisible();
  await expect(page.getByText("Validate terminal reconnect")).toBeVisible();
  await expect(page.locator("article").filter({ hasText: "Polish WorkBoard cards" }).getByText("Queued")).toBeVisible();

  await page.getByRole("button", { name: /Polish WorkBoard cards/ }).click();

  const dialog = page.getByRole("dialog", { name: "Work item editor" });
  await expect(dialog).toBeVisible();
  await expect(page.getByLabel("Work item title")).toHaveValue("Polish WorkBoard cards");
  await expect(page.getByText("Ready because")).toBeVisible();

  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
});

test("adds a blocker link from the detail dependency section", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Work" }).click();
  await page.getByRole("button", { name: /Polish WorkBoard cards/ }).click();

  const dialog = page.getByRole("dialog", { name: "Work item editor" });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByText("Ready because")).toBeVisible();
  await expect(dialog.getByText("no blocking dependencies", { exact: true })).toBeVisible();

  await dialog.getByLabel("Blocker work item").click();
  await page.getByRole("option", { name: /Map dependency graph/ }).click();
  await dialog.getByRole("button", { name: "Add blocker" }).click();

  await expect(dialog.getByText("Map dependency graph")).toBeVisible();
  await expect(dialog.getByText("blocked by unfinished dependency")).toBeVisible();

  const addLinkCalls = await page.evaluate(() =>
    window.__WHISK_E2E__.calls().filter((call) => call.method.endsWith(".AddWorkItemLink")),
  );
  expect(addLinkCalls).toHaveLength(1);
  expect(addLinkCalls[0].args[0]).toMatchObject({
    sourceWorkItemId: "wi_ready",
    targetWorkItemId: "wi_dependency",
    type: "blocks",
  });
});

test("opens WorkItemDetail property popovers backed by local menu primitives", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Work" }).click();
  await page.getByRole("button", { name: /Polish WorkBoard cards/ }).click();

  const dialog = page.getByRole("dialog", { name: "Work item editor" });
  const properties = dialog.locator("aside");
  await expect(dialog).toBeVisible();

  await properties.getByRole("button", { name: "More actions" }).click();
  await expect(page.getByRole("menu").getByText("Start Execution")).toBeVisible();
  await page.keyboard.press("Escape");

  await properties.getByRole("button", { name: "Ready" }).click();
  await expect(page.getByRole("menu").getByText("Execution")).toBeVisible();
  await page.keyboard.press("Escape");

  await properties.getByRole("button", { name: "Default agent" }).click();
  await expect(page.getByRole("menu").getByText("Interactive shell")).toBeVisible();
});

test("runs a daemon-provided workflow action from the detail menu", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Work" }).click();
  await page.getByRole("button", { name: /Capture app launch smoke/ }).click();

  const dialog = page.getByRole("dialog", { name: "Work item editor" });
  await expect(dialog).toBeVisible();

  await dialog.locator("aside").getByRole("button", { name: "More actions" }).click();
  await page.getByRole("menuitem", { name: /Start Planning/ }).click();

  await expect(dialog.getByRole("button", { name: "Planning", exact: true })).toBeVisible();
  const calls = await page.evaluate(() =>
    window.__WHISK_E2E__.calls().filter((call) => call.method.endsWith(".StartPlanning")),
  );
  expect(calls).toHaveLength(1);
  expect(calls[0].args[0]).toMatchObject({ workItemId: "wi_backlog" });
});

test("captures WorkBoard and detail screenshots", async ({ page }, testInfo) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Work" }).click();
  await expect(page.getByText("Polish WorkBoard cards")).toBeVisible();

  await testInfo.attach("work-board", {
    body: await page.screenshot({ fullPage: false }),
    contentType: "image/png",
  });

  await page.getByRole("button", { name: /Polish WorkBoard cards/ }).click();
  await expect(page.getByRole("dialog", { name: "Work item editor" })).toBeVisible();

  await testInfo.attach("work-item-detail", {
    body: await page.screenshot({ fullPage: false }),
    contentType: "image/png",
  });
});
