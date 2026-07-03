import { expect, test } from "@playwright/test";

test("updates daemon preferences from status events without polling", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "Settings" }).click();
  await page.getByRole("button", { name: "Daemon" }).click();
  await expect(page.getByText("Running", { exact: true })).toBeVisible();
  await expect(page.getByText(/e2e · e2e/)).toBeVisible();

  const callsBefore = await page.evaluate(() =>
    window.__WHISK_E2E__.calls().filter((call) => call.method.endsWith(".DaemonStatus")).length,
  );

  await page.evaluate(() => {
    window.__WHISK_E2E__.emitDaemonStatus({
      running: false,
      address: "http://127.0.0.1:8877",
      managed: true,
      apiVersion: 0,
      gitSha: "",
      version: "",
      dirty: false,
      error: "connection refused",
      autoRestartEnabled: true,
      restarting: true,
      restartAttempt: 1,
      restartMaxAttempts: 3,
    });
  });

  await expect(page.getByText("Stopped", { exact: true })).toBeVisible();
  await expect(page.getByText("Started by Whisk")).toBeVisible();
  await expect(page.getByText("Auto-restart attempt 1 of 3")).toBeVisible();

  await page.waitForTimeout(150);
  const callsAfter = await page.evaluate(() =>
    window.__WHISK_E2E__.calls().filter((call) => call.method.endsWith(".DaemonStatus")).length,
  );
  expect(callsAfter).toBe(callsBefore);
});
