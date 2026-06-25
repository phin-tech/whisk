import { expect, test } from "@playwright/test";

test("loads the seeded app shell without browser console errors", async ({ page }) => {
  const browserErrors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") browserErrors.push(message.text());
  });
  page.on("pageerror", (error) => browserErrors.push(error.message));

  await page.goto("/");

  await expect(page.getByRole("button", { name: /Seeded Session/ })).toBeVisible();
  await expect(page.getByText("Whisk E2E")).toBeVisible();
  await expect(page.getByText("No active sessions")).toBeHidden();
  await expect(page.locator("vite-error-overlay")).toHaveCount(0);
  expect(browserErrors).toEqual([]);
});
