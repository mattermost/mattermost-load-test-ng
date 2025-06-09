import { test, expect, Page } from '@playwright/test';
import { LONG_MESSAGE, SHORT_MESSAGE, USER_CREDENTIALS } from "./utils.ts";

test.describe('Typing Scenario', () => {
  test('User posts message in channel', async ({ page }) => {
    await page.goto("http://localhost:8065");

    await handlePreferenceCheckbox(page);

    await performLogin(page);

    await goToTeamsChannel(page);

    await postInChannel(page);

    // Wait for message to be posted
    await page.waitForTimeout(4000);
  });
});

async function handlePreferenceCheckbox(page: Page) {
  await page.waitForSelector("label.get-app__preference input.get-app__checkbox");
  await page.click("label.get-app__preference input.get-app__checkbox");

  await page.waitForSelector("a.btn.btn-tertiary.btn-lg");
  await page.evaluate(() => {
    const buttons = Array.from(
      document.querySelectorAll("a.btn.btn-tertiary.btn-lg")
    );
    const viewButton = buttons.find(
      (button) => button.textContent?.trim() === "View in Browser"
    );
    if (viewButton) {
      (viewButton as HTMLElement).click();
    }
  });
}

function getUserCredentials() {
  const userIndex = 0;

  return {
    email: USER_CREDENTIALS[userIndex].username,
    password: USER_CREDENTIALS[userIndex].password,
  };
}

async function performLogin(page: Page): Promise<void> {
  const { email, password } = getUserCredentials();

  await page.waitForSelector("#input_loginId");
  await page.fill("#input_loginId", email);
  await page.fill("#input_password-input", password);
  await page.keyboard.press("Enter");
}

async function goToTeamsChannel(page: Page): Promise<void> {
  if (
    !page.url().includes("team-cj3nauy79bymtk36kbey4s3rjy/channels/town-square")
  ) {
    await page.goto(
      "http://localhost:8065/team-cj3nauy79bymtk36kbey4s3rjy/channels/town-square"
    );
  }
}

async function postInChannel(page: Page): Promise<void> {
  await page.waitForSelector("#post_textbox");
  await page.fill("#post_textbox", SHORT_MESSAGE);
  await page.keyboard.press("Enter");
}
