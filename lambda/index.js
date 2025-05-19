const puppeteer = require('puppeteer-core');

// Determine if running locally or in Lambda
const isLocal = process.env.IS_LOCAL === 'true';

// Get the appropriate Chromium configuration
async function getChromium() {
  if (isLocal) {
    return {
      args: [],
      executablePath: async () => {
        return '/usr/bin/google-chrome';
      },
      headless: false // Setting it to false for local testing.
    };
  } else {
    // Use the Lambda layer version
    return require('@sparticuz/chromium');
  }
}

function delay(time) {
  return new Promise(function (resolve) {
    setTimeout(resolve, time)
  });
}

exports.handler = async (event, context) => {
  let browser = null;
  let result = null;

  try {
    // Get the appropriate Chromium configuration
    const chromium = await getChromium();

    // Launch the browser with recommended configuration
    browser = await puppeteer.launch({
      args: chromium.args,
      executablePath: await chromium.executablePath(),
      headless: chromium.headless,
      ignoreHTTPSErrors: true,
    });

    // Create a new page
    const page = await browser.newPage();

    // Extract username and password from event
    const username = event.username || '';
    const password = event.password || '';
    const teamName = event.join_team || '';

    // Navigate to Mattermost with waitUntil option
    await page.goto(event.url, {
      waitUntil: 'load'
    });

    await delay(5000);

    // Wait for and click "View in Browser" button
    // Use the exact selector from the HTML snippet
    const browserButtonSelector = '.get-app__buttons a.btn.btn-tertiary';
    await page.waitForSelector(browserButtonSelector, { timeout: 10000 });
    await page.click(browserButtonSelector);

    // Wait for login page to load
    await delay(5000);
    await page.waitForSelector('#input_loginId', { timeout: 10000 });

    // Enter credentials
    await page.type('#input_loginId', username);
    await page.type('#input_password-input', password);

    // Click login button
    await page.click('#saveSetting');

    // Wait for login to complete
    await page.waitForSelector('#SidebarContainer', { timeout: 10000 });

    // Wait after login
    await delay(event.delay);

    if (event.reload === true) {
      // Reload the page after the delay
      await page.reload({ waitUntil: 'load' });

      // Wait for the page to load after reload
      await page.waitForSelector('#SidebarContainer', { timeout: 10000 });
    }

    if (teamName !== '') {
      await page.click('#sidebarDropdownMenuButton');

      await page.waitForSelector('#joinTeam', { timeout: 5000 });
      await page.click('#joinTeam');

      // Give it some time for team selector page to appear.
      await delay(5000);

      await page.waitForSelector(`#${teamName}`, { timeout: 1000 });
      await page.click(`#${teamName}`);

      // Wait for team to load
      await page.waitForSelector('#SidebarContainer', { timeout: 10000 });
    }

    // Take screenshot for visual debugging
    let screenshot;
    if (event.debug === true) {
      screenshot = await page.screenshot({ encoding: 'base64' });
    }

    result = {
      statusCode: 200,
      body: JSON.stringify({
        message: `Logged in as ${event.username}`,
        screenshot: screenshot
      }),
      headers: {
        'Content-Type': 'application/json'
      }
    };
  } catch (error) {
    return {
      statusCode: 500,
      body: JSON.stringify({
        error: error.message
      }),
      headers: {
        'Content-Type': 'application/json'
      }
    };
  } finally {
    // Close the browser
    if (browser !== null) {
      await browser.close();
    }
  }

  return result;
};
