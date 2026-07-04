import { expect, test } from "@playwright/test";

test("opens jump palette from native command event and jumps to a work item", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByRole("button", { name: /Seeded Session/ })).toBeVisible();
  await page.evaluate(() => window.__WHISK_E2E__.emitCommand("jumpPalette.open"));

  const search = page.getByLabel("Jump target");
  await expect(search).toBeVisible();
  await search.fill("Polish WorkBoard cards");
  await search.press("Enter");

  await expect(search).toBeHidden();
  await expect(page.getByLabel("Work item title")).toHaveValue("Polish WorkBoard cards");
  await expect(page.getByText("Work").first()).toBeVisible();
});
