import { browser, Page } from "k6/browser";
import exec from "k6/execution";
import {
  LONG_MESSAGE,
  SHORT_MESSAGE,
  TEAM_NAME,
  TEAM_URL,
  USER_CREDENTIALS,
} from "./utils.ts";
import { sleep } from "k6";

export const options = {
  scenarios: {
    typingScenario: {
      executor: "constant-vus",
      exec: "typingScenario",
      vus: 5,
      duration: "1m",
      options: {
        browser: {
          type: "chromium",
        },
      },
    },
  },
};

export async function typingScenario() {
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    await page.goto("https://community.mattermost.com");

    page.waitForNavigation();

    await handlePreferenceCheckbox(page);

    await performLogin(page);

    await goToTeamsChannel(page);

    await postInChannel(page);

    sleep(2000);
  } finally {
    await page.close();
    await context.close();
  }
}

async function handlePreferenceCheckbox(page: Page) {
  try {
    // Try to find the checkbox with a short timeout
    await page.waitForSelector(
      "label.get-app__preference input.get-app__checkbox",
      { timeout: 2000 }
    );

    // If found, click it
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
  } catch (error) {
    // If checkbox not found, log and skip
    console.log("[k6-browser-log] Preference checkbox not found, skipping...");
  }
}

function getUserCredentials() {
  const vuInObject = exec.vu.idInTest - 1;

  console.log(
    "[k6-browser-log] Logging in with user: ",
    USER_CREDENTIALS[vuInObject].username
  );

  return {
    email: USER_CREDENTIALS[vuInObject].username,
    password: USER_CREDENTIALS[vuInObject].password,
  };
}

async function performLogin(page: Page): Promise<void> {
  const { email, password } = getUserCredentials();

  await page.waitForSelector("#input_loginId");
  await page.type("#input_loginId", email);
  await page.type("#input_password-input", password);
  await page.keyboard.press("Enter");
}

async function goToTeamsChannel(page: Page): Promise<void> {
  if (!page.url().includes(TEAM_NAME)) {
    await page.goto(TEAM_URL);
  }
}

async function postInChannel(page: Page): Promise<void> {
  await page.waitForSelector("#post_textbox");
  await page.type("#post_textbox", LONG_MESSAGE, { delay: 40 });
  await page.keyboard.press("Enter");
}
