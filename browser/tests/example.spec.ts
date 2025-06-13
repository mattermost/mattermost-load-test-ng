import {test, expect, Page} from '@playwright/test';

test('has title', hasTitle);

test('get started link', async ({page}) => {
  await page.goto('https://playwright.dev/');

  // Click the get started link.
  await page.getByRole('link', {name: 'Get started'}).click();

  // Expects page to have a heading with the name of Installation.
  await expect(page.getByRole('heading', {name: 'Installation'})).toBeVisible();
});

async function hasTitle({page}: {page: Page}) {
  await page.goto('https://playwright.dev/');

  // Expect a title "to contain" a substring.
  await expect(page).toHaveTitle(/Playwright/);
}
