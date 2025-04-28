/**
 * Copyright 2023 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {Page, expect} from '@playwright/test';
import {execSync} from 'child_process';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const DEFAULT_FAKE_NOW = 'Dec 1 2020 12:34:56';

export const BASE_URL = 'http://localhost:5555';

export async function setupFakeNow(
  page: Page,
  fakeNowDateString = DEFAULT_FAKE_NOW,
): Promise<void> {
  // Get fakeNow from UTC to extract the timeZone offset used in the test
  const fakeNowFromUTC = new Date(fakeNowDateString);
  const offset = fakeNowFromUTC.getTimezoneOffset();
  const offsetSign = offset < 0 ? '-' : '+';
  const offsetHours = `${Math.abs(Math.floor(offset / 60))}`.padStart(2, '0');
  const offsetMinutes = `${Math.abs(offset % 60)}`.padStart(2, '0');
  const offsetText = `${offsetSign}${offsetHours}:${offsetMinutes}`;

  // Get fakeNow from the test timeZone
  const fakeNow = new Date(`${fakeNowDateString}Z${offsetText}`).valueOf();

  // Update the Date accordingly in your test pages
  await page.addInitScript(`{
    // Extend Date constructor to default to fakeNow
    Date = class extends Date {
      constructor(...args) {
        if (args.length === 0) {
          super(${fakeNow});
        } else {
          super(...args);
        }
      }
    }
    // Override Date.now() to start from fakeNow
    const __DateNowOffset = ${fakeNow} - Date.now();
    const __DateNow = Date.now;
    Date.now = () => __DateNow() + __DateNowOffset;
  }`);
}

export async function waitForOverviewPageLoad(page: Page) {
  // Wait for the loading indicator to disappear and be replaced (with timeout):
  await page
    .locator('webstatus-overview-content >> text=Loading features...')
    .waitFor({state: 'hidden', timeout: 15000});
}

export async function gotoOverviewPageUrl(page: Page, url: string) {
  await page.goto(url);

  await waitForOverviewPageLoad(page);
}

export async function getOverviewPageFeatureCount(page: Page): Promise<number> {
  await waitForOverviewPageLoad(page);
  const regex = /(\d+) features/;
  const statsSummary = page.getByText(regex);
  expect(statsSummary).toBeVisible();
  const text = await statsSummary.innerText();
  return parseInt(text.match(regex)![1]);
}

export async function loginAsUser(page: Page, username: string) {
  // Clicking the log in button will create a popup that we need to capture.
  const popupPromise = page.waitForEvent('popup');
  await page.goto('http://localhost:5555/');
  await page.getByText('Log in').click();
  const popup = await popupPromise;

  await popup.getByText(username).waitFor({timeout: 2000});
  await popup.getByText(username).hover(); // Needed for Firefox for some reason.
  await popup.getByText(username).click();
  await popup.waitForEvent('close');
}

export async function goTo404Page(page, query: string): Promise<void> {
  await page.goto(`${BASE_URL}/features/${query}`);
  await expect(page).toHaveURL(
    `${BASE_URL}/errors-404/feature-not-found?q=${query}`,
  );

  const response = await page.context().request.fetch(page.url());
  expect(response.status()).toBe(404);
}

export async function expect404PageButtons(
  page,
  {hasSearch}: {hasSearch: boolean},
) {
  await expect(page.locator('#error-action-home-btn')).toBeVisible();
  await expect(page.locator('#error-action-report')).toBeVisible();

  if (hasSearch) {
    await expect(page.locator('#error-action-search-btn')).toBeVisible();
  } else {
    await expect(page.locator('#error-action-search-btn')).toHaveCount(0);
  }
}

export async function resetUserData(page: Page) {
  const __filename = fileURLToPath(import.meta.url); // get the resolved path to the file
  const __dirname = path.dirname(__filename);
  const projectRootDir = path.resolve(__dirname, '../..');

  try {
    const cmd1 = `make dev_fake_data -o build -o is_local_migration_ready LOAD_FAKE_DATA_FLAGS='-scope=user -reset'`;

    console.log(`Executing command: ${cmd1} in ${projectRootDir}`);
    execSync(cmd1, {cwd: projectRootDir, stdio: 'inherit'});

    console.log('Reset command finished successfully.');

    await page.waitForTimeout(2000);

    const cmd2 = 'make port-forward-manual';
    console.log(`Executing command: ${cmd2} in ${projectRootDir}`);
    execSync(cmd2, {
      cwd: projectRootDir,
      stdio: 'inherit',
    });

    await page.waitForTimeout(2000);
  } catch (error) {
    console.error('Error reset command (make dev_fake_data):', error);
    throw new Error('Reset command finished, halting tests.');
  }
}
