import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    class FakeWebSocket {
      static CONNECTING = 0;
      static OPEN = 1;
      static CLOSING = 2;
      static CLOSED = 3;

      readyState = FakeWebSocket.OPEN;
      onmessage: ((event: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;

      constructor(readonly url: string) {}

      send() {}

      close() {
        this.readyState = FakeWebSocket.CLOSED;
        this.onclose?.();
      }
    }

    window.WebSocket = FakeWebSocket as unknown as typeof WebSocket;
  });
});

test("close-pane confirmation uses the local dialog controls", async ({ page }) => {
  await page.goto("/?e2ePty=1");

  await page.getByRole("button", { name: /Seeded Session/ }).click();

  const closePane = page.getByRole("button", { name: "Close pane pane_01" });
  await expect(closePane).toBeEnabled();

  await closePane.click();
  const dialog = page.getByRole("dialog", { name: "Close Terminal?" });
  await expect(dialog).toBeVisible();
  await expect(dialog).toBeFocused();
  await expect(dialog.getByText("Do not ask again")).toBeVisible();

  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();

  await closePane.click();
  await expect(dialog).toBeVisible();
  const dontAskAgain = dialog.getByRole("checkbox", { name: "Do not ask again" });
  await dontAskAgain.click();
  await expect(dontAskAgain).toBeChecked();
  await dialog.getByRole("button", { name: "Close" }).click();
  await expect(dialog).toBeHidden();

  await page.waitForFunction(() => {
    const calls = window.__WHISK_E2E__.calls();
    return calls.some((call) => call.method.endsWith(".CloseSession"));
  });
  const settings = await page.evaluate(() => localStorage.getItem("whisk.ui.settings"));
  expect(settings).toContain('"closePanePromptDisabled":true');
});
