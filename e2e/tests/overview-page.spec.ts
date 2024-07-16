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
import {gotoOverviewPageUrl} from './utils';
import {fileURLToPath} from 'node:url';

test('matches the screenshot', async ({page}) => {
  await gotoOverviewPageUrl(page, 'http://localhost:5555/');
  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot();
});

test('shows an error that their query is invalid', async ({page}) => {
  await page.goto('http://localhost:5555/?q=available_on%3Achrom');

  const message = page.locator('.message');
  await message.waitFor({state: 'visible'});
  expect(message).toContainText('Invalid query...');

  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot('invalid-query.png');
});

test('shows an unknown error when there is an internal error', async ({
  page,
}) => {
  await page.route('**/v1/features?page_size=25', route =>
    route.fulfill({
      status: 500,
      contentType: 'application/json',
      json: {
        code: 500,
        message: 'uh-oh',
      },
    })
  );
  await page.goto('http://localhost:5555/');

  const message = page.locator('.message');
  await message.waitFor({state: 'visible'});
  expect(message).toContainText('Something went wrong...');

  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot('internal-error.png');
});

// test of hover over a baseline chip to show tooltip
test('shows a tooltip when hovering over a baseline chip', async ({page}) => {
  await gotoOverviewPageUrl(page, 'http://localhost:5555/');

  // Find the tooltip for the first Widely available chip.
  const tooltip = page
    .locator('sl-tooltip')
    .filter({hasText: 'Widely available'})
    .first();
  const baselineText = 'Baseline since 2035-05-06';
  await expect(tooltip.getByText(baselineText)).toBeHidden();
  const widelyAvailableChip = tooltip.locator('span').first();
  await widelyAvailableChip.hover();
  await expect(tooltip.getByText(baselineText)).toBeVisible();
  // Move mouse away
  await page.mouse.move(0, 0);
  await expect(tooltip.getByText(baselineText)).toBeHidden();
});

test('Export to CSV button downloads a file', async ({page}) => {
  await gotoOverviewPageUrl(page, 'http://localhost:5555/');
  const downloadPromise = page.waitForEvent('download');
  const exportButton = page.getByRole('button', {
    name: 'Export to CSV',
  });

  await expect(exportButton).toBeVisible();
  await exportButton.click();
  const download = await downloadPromise;

  const stream = await download.createReadStream();
  const file = (await stream.toArray()).toString();

  expect(file).toMatchSnapshot('webstatus-feature-overview-default.csv');
  expect(download.suggestedFilename()).toBe('webstatus-feature-overview.csv');
});
