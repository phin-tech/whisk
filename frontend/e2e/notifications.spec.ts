import { expect, test } from "@playwright/test";

test("renders seeded notifications and clears read status events through the Wails bridge", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "Notifications" }).click();
  await expect(page.getByText("The E2E fake has a notification.")).toBeVisible();
  await expect(page.getByText("Pick a seeded option.")).toBeVisible();

  await page.getByRole("button", { name: "Clear notifications" }).click();

  const readCalls = await page.evaluate(() =>
    window.__WHISK_E2E__.calls().filter((call) => call.method.endsWith(".MarkStatusEventRead")),
  );
  expect(readCalls).toHaveLength(1);
  expect(readCalls[0].args[0]).toMatchObject({ id: "status_01" });
});
