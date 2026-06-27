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

test("does not expose pty bookmark controls", async ({ page }) => {
  await page.goto("/?e2ePty");

  await expect(page.getByRole("button", { name: /Seeded Session/ })).toBeVisible();
  await expect(page.locator(".xterm-rows")).toContainText("seeded terminal output");
  await expect(page.getByRole("button", { name: /Add bookmark/ })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /Open bookmark/ })).toHaveCount(0);
  await expect(page.locator(".terminal-bookmark-decoration")).toHaveCount(0);

  await page.getByRole("button", { name: "PTYs" }).click();
  await expect(page.getByRole("button", { name: /Jump to bookmark/ })).toHaveCount(0);

  const calls = await page.evaluate(() => window.__WHISK_E2E__.calls());
  const outputCalls = calls.filter((call) => call.method.endsWith(".Output"));
  const bookmarkCalls = calls.filter((call) => call.method.includes("Bookmark"));
  expect(outputCalls.some((call) => (call.args[0] as { fromOffset?: number }).fromOffset === 0)).toBe(true);
  expect(bookmarkCalls).toEqual([]);
});

test("jump to bottom remains available on a long pty", async ({ page }) => {
  await page.goto("/?e2eLongPty");

  await expect(page.getByRole("button", { name: /Seeded Session/ })).toBeVisible();
  await expect(page.locator(".xterm-rows")).toContainText("scrollback line 89");
  await expect(page.locator(".xterm-rows")).not.toContainText("scrollback line 20");

  await page.evaluate(() => window.__WHISK_E2E__.emitCommand("terminal.bottom"));
  await expect(page.locator(".xterm-rows")).toContainText("scrollback line 89");
  await expect(page.locator(".terminal-bookmark-decoration")).toHaveCount(0);
});
