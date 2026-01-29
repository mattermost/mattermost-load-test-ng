// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type { Page } from "@playwright/test";
import { LoginPage } from "@mattermost/playwright-lib";

import type { Logger } from "../types/log.js";

type ExtraArgs = {
  userId: string;
  password: string;
};

export async function performLogin(
  page: Page,
  log: Logger,
  { userId, password }: ExtraArgs,
): Promise<void> {
  log.info("run--performLogin");

  try {
    const loginPage = new LoginPage(page);
    await loginPage.toBeVisible();

    await loginPage.loginInput.fill(userId);

    await loginPage.passwordInput.waitFor({ state: "visible" });
    await loginPage.passwordInput.fill(password);

    await loginPage.signInButton.waitFor({ state: "visible" });
    await loginPage.signInButton.click();

    log.info("pass--performLogin");
  } catch (error) {
    throw { error, testId: "performLogin" };
  }
}
