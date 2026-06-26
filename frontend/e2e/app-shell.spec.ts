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
  await expect(page.locator(".xterm-rows")).toContainText("seeded terminal output");
  await expect(page.getByRole("button", { name: "Add bookmark for pty_01" })).toBeVisible();
  await expect(page.getByRole("button", { name: /Jump to bookmark Agent handoff/ })).toBeVisible();

  await page.getByRole("button", { name: "Add bookmark for pty_01" }).click();
  await expect(page.getByRole("button", { name: /Jump to bookmark Bookmark @23/ })).toBeVisible();

  await page.getByRole("button", { name: "PTYs" }).click();
  await expect(page.getByRole("button", { name: /Jump to bookmark Agent handoff from PTYs/ })).toBeVisible();

  await page.evaluate(() => window.__WHISK_E2E__.emitCommand("bookmark.previous"));
  await expect(page.locator(".xterm-rows")).toContainText("bookmarked output");

  const calls = await page.evaluate(() => window.__WHISK_E2E__.calls());
  const outputCalls = calls.filter((call) => call.method.endsWith(".Output"));
  const addBookmarkCalls = calls.filter((call) => call.method.endsWith(".AddPTYBookmark"));
  expect(outputCalls.some((call) => (call.args[0] as { fromOffset?: number }).fromOffset === 12)).toBe(true);
  expect(addBookmarkCalls.some((call) => (call.args[0] as { offset?: number }).offset === 23)).toBe(true);
});
