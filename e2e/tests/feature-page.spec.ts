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

import {test, expect} from '@playwright/test';
import {setupFakeNow} from './utils';

test.beforeEach(async ({page}) => {
  await setupFakeNow(page);
});

test('matches the screenshot', async ({page}) => {
  await page.goto('http://localhost:5555/features/odit64');

  // Wait for the chart container to exist
  await page.waitForSelector('#feature-wpt-implementation-progress-0-complete');

  // Wait specifically for the "Baseline since" text
  await page.waitForSelector('sl-card.wptScore .avail >> text=Baseline since');

  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot();
});

test('chart width resizes with window', async ({page}) => {
  await page.goto('http://localhost:5555/features/odit64');
  await page.waitForSelector('#feature-wpt-implementation-progress-0-complete');
  await page.waitForTimeout(1000);
  const narrowWidth = 1000;
  const wideWidth = 1200;
  const height = 1000;
  const chartContainer = page.locator(
    '#feature-wpt-implementation-progress-0-complete',
  );

  // Resize to narrow width
  await page.setViewportSize({width: narrowWidth, height});
  await page.waitForTimeout(1000);
  const newChartWidth = await chartContainer.evaluate(el => el.clientWidth);

  // Ensure that the chart is wider than the narrow width
  await page.setViewportSize({width: wideWidth, height});
  await page.waitForTimeout(1000);
  const newChartWidth2 = await chartContainer.evaluate(el => el.clientWidth);
  expect(newChartWidth2).toBeGreaterThan(newChartWidth);

  // And restore to original size
  await page.setViewportSize({width: narrowWidth, height});
  // We may be able to remove the following waitForTimeout after we address:
  // https://github.com/GoogleChrome/webstatus.dev/issues/278
  await page.waitForTimeout(2000);
  const newChartWidth3 = await chartContainer.evaluate(el => el.clientWidth);
  expect(newChartWidth3).toEqual(newChartWidth);

  // Compare screenshot of smaller chart
  await expect(chartContainer).toHaveScreenshot();
});

test('mobile chart displays on click and matches screenshot', async ({
  page,
}) => {
  await page.goto('http://localhost:5555/features/odit64');
  await page.waitForSelector('#feature-wpt-implementation-progress-0-complete');
  const mobileTab = page.locator(
    'sl-tab#feature-wpt-implementation-progress-tab-mobile',
  );

  await mobileTab.click();
  await page.waitForTimeout(2000);
  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot();
});

test('date range changes are preserved in the URL', async ({page}) => {
  await page.goto('http://localhost:5555/features/odit64');
  await page.waitForSelector('#feature-wpt-implementation-progress-1-complete');

  // Get the current default startDate and endDate from the selectors
  // TODO Figure out how to use getByLabel with shoelace and replace page.locator with that.
  const submitBtnSelector = page.locator('sl-button#date-range-picker-btn');
  // Can only detect if the button is enabled by getting the raw <button>
  const submitBtn = submitBtnSelector.locator('button');
  await expect(submitBtn).toBeDisabled();
  const startDateSelector = page.locator('sl-input#start-date');
  const startDateInputElement = startDateSelector.locator('input');
  const startDate = await startDateInputElement.inputValue();
  const endDateSelector = page.locator('sl-input#end-date');
  const endDateInputElement = endDateSelector.locator('input');
  const endDate = await endDateInputElement.inputValue();

  // Change the start date to April 1st, 2020, in yyyy-mm-dd order
  await startDateInputElement.fill('2020-04-01');

  await expect(submitBtn).toBeEnabled();

  // Submit the change
  await submitBtn.click();

  // Check that the URL includes the startDate and endDate
  const url = page.url();
  expect(url).toContain('startDate=2020-04-01');
  expect(url).toContain('endDate=2020-12-01');

  // Refresh the page with that URL.
  await page.goto(url);
  await page.waitForSelector('#feature-wpt-implementation-progress-0-complete');

  // Check that the startDate and endDate are still there.
  const url2 = page.url();
  expect(url2).toContain('startDate=2020-04-01');
  expect(url2).toContain('endDate=2020-12-01');

  // Check that the startDate selector has the right value.
  const startDateSelector2 = page.locator('sl-input#start-date');
  const startDateInputElement2 = startDateSelector2.locator('input');
  const startDateValue2 = await startDateInputElement2.inputValue();
  expect(startDateValue2).toBe('2020-04-01');

  // TODO: Check that the chart has the right start date.

  // Click on the feature breadcrumb.
  const featureCrumb = page.locator('.crumbs >> a:has-text("odit64")');
  await featureCrumb.click();

  // Check that the URL no longer contains the startDate or endDate.
  const url3 = page.url();
  expect(url3).not.toContain('startDate=2020-04-01');
  expect(url3).not.toContain('endDate=2020-12-01');

  // Go to that URL.
  await page.goto(url3);
  await page.waitForSelector('#feature-wpt-implementation-progress-0-complete');

  // Check that the startDate and endDate selectors are reset to the initial default.
  const startDateSelector3 = page.locator('sl-input#start-date');
  const startDateInputElement3 = startDateSelector3.locator('input');
  expect(await startDateInputElement3.inputValue()).toBe(startDate);
  const endDateSelector3 = page.locator('sl-input#end-date');
  const endDateInputElement3 = endDateSelector3.locator('input');
  expect(await endDateInputElement3.inputValue()).toBe(endDate);
});
