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

import {test, expect, Page} from '@playwright/test';

async function setupFakeNow(page: Page): Promise<void> {
  // Get fakeNow from UTC to extract the timeZone offset used in the test
  const fakeNowDateTime = "May 1 2020 12:34:56";
  const fakeNowFromUTC = new Date(fakeNowDateTime);
  const offset = fakeNowFromUTC.getTimezoneOffset();
  const offsetSign = offset < 0 ? "-" : "+";
  const offsetHours = `${Math.abs(Math.floor(offset / 60))}`.padStart(2, "0");
  const offsetMinutes = `${Math.abs(offset % 60)}`.padStart(2, "0");
  const offsetText = `${offsetSign}${offsetHours}:${offsetMinutes}`;

  // Get fakeNow from the test timeZone
  const fakeNow = new Date(`${fakeNowDateTime}Z${offsetText}`).valueOf();

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
};

test.beforeEach(async ({ page }) => {
  await setupFakeNow(page);
});

test('matches the screenshot', async ({ page }) => {

  await page.goto('http://localhost:5555/features/a117');

  // Wait for chart to be displayed
  await page.waitForSelector('#feature-support-chart-container');
  await page.waitForTimeout(1000);

  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot();
});
