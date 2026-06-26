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

test("shows pty bookmarks and jumps terminal replay to their offsets", async ({ page }) => {
  await page.goto("/?e2ePty");

  await expect(page.getByRole("button", { name: /Seeded Session/ })).toBeVisible();
  await expect(page.getByRole("button", { name: /Jump to bookmark Agent handoff/ })).toBeVisible();

  await page.getByRole("button", { name: "PTYs" }).click();
  await expect(page.getByRole("button", { name: /Jump to bookmark Agent handoff from PTYs/ })).toBeVisible();

  await page.getByRole("button", { name: /Jump to bookmark Agent handoff from PTYs/ }).click();

  const outputCalls = await page.evaluate(() =>
    window.__WHISK_E2E__.calls().filter((call) => call.method.endsWith(".Output")),
  );
  expect(outputCalls.some((call) => (call.args[0] as { fromOffset?: number }).fromOffset === 12)).toBe(true);
});
