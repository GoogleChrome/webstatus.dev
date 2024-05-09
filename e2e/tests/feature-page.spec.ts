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
  const fakeNowDateTime = 'May 1 2020 12:34:56';
  const fakeNowFromUTC = new Date(fakeNowDateTime);
  const offset = fakeNowFromUTC.getTimezoneOffset();
  const offsetSign = offset < 0 ? '-' : '+';
  const offsetHours = `${Math.abs(Math.floor(offset / 60))}`.padStart(2, '0');
  const offsetMinutes = `${Math.abs(offset % 60)}`.padStart(2, '0');
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
}

test.beforeEach(async ({page}) => {
  await setupFakeNow(page);
});

test('matches the screenshot', async ({page}) => {
  await page.goto('http://localhost:5555/features/a117');

  // Wait for chart to be displayed
  await page.waitForSelector('#feature-support-chart-container');
  await page.waitForTimeout(1000);

  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot();
});

test('date range changes are preserved in the URL', async ({ page }) => {
  await page.goto('http://localhost:5555/features/a117');
  await page.waitForSelector('#feature-support-chart-container');
  await page.waitForTimeout(1000);

  // Change the start date to April 1st, 2020, in yyyy-mm-dd order
  const startDateSelector = page.locator('sl-input#start-date');
  const startDateInputElement = startDateSelector.locator('input');
  await startDateInputElement.fill('2020-04-01');

  // Blur the input to trigger the change event
  await startDateInputElement.blur();

  // Check that the URL includes the startDate and endDate
  const url = page.url();
  expect(url).toContain('startDate=2020-04-01');
  expect(url).toContain('endDate=2020-05-01');

  // Refresh the page with that URL.
  await page.goto(url);
  await page.waitForSelector('#feature-support-chart-container');

  // Check that the startDate and endDate are still there.
  const url2 = page.url();
  expect(url2).toContain('startDate=2020-04-01');
  expect(url2).toContain('endDate=2020-05-01');

  // Check that the startDate selector has the right value.
  const startDateSelector2 = page.locator('sl-input#start-date');
  const startDateInputElement2 = startDateSelector2.locator('input');
  const startDateValue2 = await startDateInputElement2.inputValue()
  expect(startDateValue2).toBe('2020-04-01');

  // TODO: Check that the chart has the right start date.
});
