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

const DEFAULT_FAKE_NOW = 'Dec 1 2020 12:34:56';

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
    .waitFor({state: 'hidden', timeout: 60000});
}

export async function gotoOverviewPageUrl(page: Page, url: string) {
  await page.goto(url, {timeout: 60000});

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
