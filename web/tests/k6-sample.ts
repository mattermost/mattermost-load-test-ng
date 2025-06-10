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
    mixedScenario: {
      executor: "constant-vus",
      exec: "mixedScenario",
      vus: __ENV.VUS || 1,
      duration: __ENV.DURATION || "30s",
      options: {
        browser: {
          type: "chromium",
        },
      },
    },
  },
};

export async function mixedScenario() {
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    await page.goto("https://community.mattermost.com");

    page.waitForNavigation();

    await handlePreferenceCheckbox(page);

    await performLogin(page);

    await goToTeamsChannel(page);

    await postInChannel(page);

    await switchToEachChannel(page, 4000, 5);

    await scrollInTwoChannels(
      page,
      40,
      400,
      1000,
      'sidebarItem_tickets',
      'sidebarItem_developers'
    );

    sleep(2000);
  } finally {
    await page.close();
    await context.close();
  }
}

async function scrollInTwoChannels(
  page: Page,
  scrollCount: number,
  pixelsPerScroll: number,
  delayBetweenScrolls: number,
  firstChannelId: string,
  secondChannelId: string,
): Promise<void> {
  console.log('[k6-browser-log] Started scrolling test in two channels');
  console.log(
    `[k6-browser-log] Configuration: ${scrollCount} scrolls, ${pixelsPerScroll}px per scroll, ${delayBetweenScrolls}ms delay`
  );

  // Scroll in the first channel
  await scrollInChannel(
    page,
    firstChannelId,
    scrollCount,
    pixelsPerScroll,
    delayBetweenScrolls
  );

  // Wait before switching channels
  sleep(2);

  // Scroll in the second channel
  await scrollInChannel(
    page,
    secondChannelId,
    scrollCount,
    pixelsPerScroll,
    delayBetweenScrolls
  );

  console.log('[k6-browser-log] Completed scrolling test in two channels');
}

async function scrollInChannel(
  page: Page,
  channelId: string,
  scrollCount: number,
  scrollStep: number,
  pauseBetweenScrolls: number,
): Promise<void> {
  console.log(`[k6-browser-log] Scrolling in channel: ${channelId}`);

  // Navigate to the specified channel
  await page.evaluate((id) => {
    const element = document.getElementById(id);
    if (element) {
      element.click();
    } else {
      console.error(`Channel with id ${id} not found`);
    }
  }, channelId);

  // Wait for channel content to load
  sleep(3);

  const containerSelector = '.post-list__dynamic';

  for (let i = 0; i < scrollCount; i++) {
    // Scroll up by scrollStep pixels - passing params as a single object
    await page.evaluate((params) => {
      const container = document.querySelector(params.selector) as HTMLElement;
      if (container) {
        // Negative value scrolls UP
        container.scrollBy({top: -params.step, behavior: 'smooth'});
      }
    }, { selector: containerSelector, step: scrollStep });

    // Get current scroll position from the DOM
    const scrollPosition = await page.evaluate((selector) => {
      const container = document.querySelector(selector) as HTMLElement;
      return container ? container.scrollTop : 0;
    }, containerSelector);

    console.log(`[k6-browser-log] Scroll ${i + 1}/${scrollCount}, position: ${scrollPosition}px`);

    // Wait for smooth scrolling to complete
    sleep(pauseBetweenScrolls / 1000);
  }
}

async function switchToEachChannel(page: Page, waitAfterEachSwitch: number = 2000, maxChannelsToSwitch: number = Infinity): Promise<void> {
  console.log('[k6-browser-log] Started switching to each channel');
  console.log(`[k6-browser-log] Configuration: ${waitAfterEachSwitch}ms delay, max ${maxChannelsToSwitch === Infinity ? 'all' : maxChannelsToSwitch} channels`);

  // Wait for sidebar container to appear
  await page.waitForSelector('#sidebar-left');

  // Get all sidebar links within sidebar-left
  const channelLinks = await page.evaluate(() => {
    const sidebar = document.getElementById('sidebar-left');
    if (!sidebar) return [];

    // Find all anchor tags with class SidebarLink directly
    const links = Array.from(sidebar.querySelectorAll('a.SidebarLink'));
    return links
      .map((link) => {
        return {
          id: link.id || 'Unknown Channel id',
          ariaLabel:
            link.getAttribute('aria-label') || 'Unknown Channel aria-label',
        };
      })
      .filter((link) => link.id); // Filter out links without IDs
  });

  for (let i = 0; i < Math.min(channelLinks.length, maxChannelsToSwitch); i++) {
    const channel = channelLinks[i];
    console.log(`[k6-browser-log] Switching to channel: ${channel.ariaLabel} (${channel.id})`);

    // Click on the channel by ID
    await page.evaluate((id) => {
      const element = document.getElementById(id);
      if (element) {
        element.click();
      }
    }, channel.id);

    // Wait for content to load and stabilize
    sleep(waitAfterEachSwitch / 1000); // k6 sleep uses seconds, not milliseconds
  }

  console.log('[k6-browser-log] Finished switching channels');
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

  console.log("[k6-browser-log] Posted message");
}
