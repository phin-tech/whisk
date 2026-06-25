import { expect, test } from "@playwright/test";

test("opens the new session dialog, uses the fake file picker, and submits through the Wails bridge", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: /New session/i }).click();
  const dialog = page.getByRole("dialog", { name: "New session" });
  await expect(dialog).toBeVisible();

  await dialog.getByRole("button", { name: "Choose directory" }).click();
  await expect(dialog.getByRole("textbox", { name: /Directory/ })).toHaveValue("/tmp/whisk-e2e");

  await dialog.getByRole("textbox", { name: "Name" }).fill("Created from E2E");
  await dialog.getByRole("button", { name: "Create" }).click();
  await expect(dialog).toBeHidden();

  const createSessionCalls = await page.evaluate(() =>
    window.__WHISK_E2E__.calls().filter((call) => call.method.endsWith(".CreateSession")),
  );
  expect(createSessionCalls).toHaveLength(1);
  expect(createSessionCalls[0].args[0]).toMatchObject({
    name: "Created from E2E",
    rootDir: "/tmp/whisk-e2e",
  });
});
