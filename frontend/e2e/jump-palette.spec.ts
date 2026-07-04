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

test("persists successful jump targets as empty-query recents", async ({ page }) => {
  await page.goto("/");
  await page.evaluate(() => {
    localStorage.setItem(
      "whisk.clientViewState",
      JSON.stringify({
        version: 1,
        jumpPalette: { recentTargetIds: ["stale:target"] },
      }),
    );
  });
  await page.reload();

  await expect(page.getByRole("button", { name: /Seeded Session/ })).toBeVisible();
  await page.evaluate(() => window.__WHISK_E2E__.emitCommand("jumpPalette.open"));

  const search = page.getByLabel("Jump target");
  await expect(search).toBeVisible();
  await search.fill("Polish WorkBoard cards");
  await search.press("Enter");
  await expect(page.getByLabel("Work item title")).toHaveValue("Polish WorkBoard cards");

  const storedRecents = await page.evaluate(() => {
    const raw = localStorage.getItem("whisk.clientViewState");
    return raw ? JSON.parse(raw).jumpPalette.recentTargetIds : [];
  });
  expect(storedRecents[0]).toBe("work-item:wi_ready");
  expect(storedRecents).toContain("stale:target");

  await page.reload();
  await expect(page.getByRole("button", { name: /Seeded Session/ })).toBeVisible();
  await page.evaluate(() => window.__WHISK_E2E__.emitCommand("jumpPalette.open"));
  await expect(page.getByRole("option", { name: /Polish WorkBoard cards/ })).toBeVisible();

  const optionTexts = await page.getByRole("option").evaluateAll((options) =>
    options.map((option) => option.textContent?.replace(/\s+/g, " ").trim() ?? ""),
  );
  const recentIndex = optionTexts.findIndex((text) => text.includes("Polish WorkBoard cards"));
  const ordinaryIndex = optionTexts.findIndex((text) => text.includes("Capture app launch smoke"));
  expect(recentIndex).toBeGreaterThanOrEqual(0);
  expect(ordinaryIndex).toBeGreaterThanOrEqual(0);
  expect(recentIndex).toBeLessThan(ordinaryIndex);
  expect(optionTexts.some((text) => text.includes("stale:target"))).toBe(false);
});

test("does not persist stale work item run targets as recents", async ({ page }) => {
  await page.goto("/?e2eStaleRun=1");
  await page.evaluate(() => {
    localStorage.setItem(
      "whisk.clientViewState",
      JSON.stringify({
        version: 1,
        jumpPalette: { recentTargetIds: ["work-item:wi_ready"] },
      }),
    );
  });
  await page.reload();

  await expect(page.getByRole("button", { name: /Seeded Session/ })).toBeVisible();
  await page.evaluate(() => window.__WHISK_E2E__.emitCommand("jumpPalette.open"));

  const search = page.getByLabel("Jump target");
  await expect(search).toBeVisible();
  await search.fill("run_stale");
  await expect(page.getByRole("option", { name: /Map dependency graph/ })).toBeVisible();
  await search.press("Enter");
  await expect(search).toBeHidden();

  const storedRecents = await page.evaluate(() => {
    const raw = localStorage.getItem("whisk.clientViewState");
    return raw ? JSON.parse(raw).jumpPalette.recentTargetIds : [];
  });
  expect(storedRecents).toEqual(["work-item:wi_ready"]);

  const staleOutputCalls = await page.evaluate(
    () =>
      window.__WHISK_E2E__
        .calls()
        .filter((call) => call.method.endsWith(".Output") && JSON.stringify(call.args).includes("pty_missing"))
        .length,
  );
  expect(staleOutputCalls).toBe(0);
});
