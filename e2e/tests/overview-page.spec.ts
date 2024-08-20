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

// test('hides the Feature column', async ({page}) => {
//   await gotoOverviewPageUrl(page, 'http://localhost:5555/');

//   // Check that the "Feature" column is visible by default.
//   let nameColumn = page.locator('th > a', {hasText: 'Feature'});
//   await expect(nameColumn).toBeVisible();

//   // Click the Columns button to open the column selector.
//   const columnsButton = page.locator('#columns-button');
//   await columnsButton.waitFor({state: 'visible'});
//   await columnsButton.click();
//   const webstatusColumnsDialog = page.locator('webstatus-columns-dialog');
//   const columnsDialog = webstatusColumnsDialog.getByRole('dialog');
//   await columnsDialog.waitFor({state: 'visible'});

//   // Uncheck the "Feature name" checkbox.
//   const nameCheckbox = webstatusColumnsDialog.locator(
//     'sl-checkbox[value="name"]'
//   );
//   await nameCheckbox.click();

//   // Click the Save button.
//   await page.locator('#columns-save-button').click();
//   await page.waitForTimeout(500);

//   // Make sure the "Feature" column is no longer visible.
//   nameColumn = page.locator('th > a', {hasText: 'Feature'});
//   await expect(nameColumn).not.toBeVisible();
// });

// test('shows the Baseline status column with low and high date options', async ({
//   page,
// }) => {
//   await gotoOverviewPageUrl(page, 'http://localhost:5555/');

//   // Check that the "Baseline" column is visible by default.
//   const baselineStatusColumn = page.locator('th > a', {
//     hasText: 'Baseline',
//   });
//   await expect(baselineStatusColumn).toBeVisible();

//   // Click the Columns button to open the column selector.
//   const columnsButton = page.locator('#columns-button');
//   await columnsButton.waitFor({state: 'visible'});
//   await columnsButton.click();
//   const webstatusColumnsDialog = page.locator('webstatus-columns-dialog');
//   const columnsDialog = webstatusColumnsDialog.getByRole('dialog');
//   await columnsDialog.waitFor({state: 'visible'});

//   // Check the "Baseline status low date" checkbox.
//   const baselineStatusLowDateCheckbox = webstatusColumnsDialog.locator(
//     'sl-checkbox[value="baseline_status_low_date"]'
//   );
//   await baselineStatusLowDateCheckbox.click();
//   // Check the "Baseline status high date" checkbox.
//   const baselineStatusHighDateCheckbox = webstatusColumnsDialog.locator(
//     'sl-checkbox[value="baseline_status_high_date"]'
//   );
//   await baselineStatusHighDateCheckbox.click();

//   // Click the Save button.
//   await page.locator('#columns-save-button').click();
//   await page.waitForTimeout(500);

//   // Check that "Newly available: " text is visible somewhere.
//   const baselineStatusLowDateText = page.locator('td', {
//     hasText: 'Newly available: ',
//   });
//   await expect(baselineStatusLowDateText.first()).toBeVisible();
//   // Check that "Widely available: " and "Projected Widely available: "
//   // text is visible somewhere.
//   const baselineStatusHighDateText = page.locator('td', {
//     hasText: 'Widely available: ',
//   });
//   await expect(baselineStatusHighDateText.first()).toBeVisible();
//   const baselineStatusProjectedHighDateText = page.locator('td', {
//     hasText: 'Projected widely available: ',
//   });
//   await expect(baselineStatusProjectedHighDateText.first()).toBeVisible();
// });

test('Export to CSV button downloads a file with default columns', async ({
  page,
}) => {
  await gotoOverviewPageUrl(page, `http://localhost:5555/`);

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

test('Export to CSV button downloads a file with all columns', async ({
  page,
}) => {
  await gotoOverviewPageUrl(page, `http://localhost:5555/`);

  // Click the Columns button to open the column selector.
  const columnsButton = page.locator('#columns-button');
  await columnsButton.waitFor({state: 'visible'});
  await columnsButton.click();
  const webstatusColumnsDialog = page.locator('webstatus-columns-dialog');
  const columnsDialog = webstatusColumnsDialog.getByRole('dialog');
  await columnsDialog.waitFor({state: 'visible'});
  const checkboxes = webstatusColumnsDialog.locator('sl-checkbox');
  await checkboxes.evaluateAll((checkboxes: HTMLInputElement[]) => {
    checkboxes.forEach(checkbox => {
      // Make sure all the checkboxes are checked.
      console.info(checkbox.name, checkbox.checked);
      if (!checkbox.checked) {
        checkbox.click();
      }
    });
  });
  await page.locator('#columns-save-button').click();
  await page.waitForTimeout(500);

  const downloadPromise = page.waitForEvent('download');
  const exportButton = page.getByRole('button', {
    name: 'Export to CSV',
  });

  await expect(exportButton).toBeVisible();
  await exportButton.click();
  const download = await downloadPromise;

  const stream = await download.createReadStream();
  const file = (await stream.toArray()).toString();

  expect(file).toMatchSnapshot('webstatus-feature-overview-all-columns.csv');
});

test('Export to CSV button fails to download file and shows toast', async ({
  page,
}) => {
  page.on('request', async request => {
    await page.route('**/features*', async route => {
      if (route.request().url().includes('page_size=100')) {
        // allFeaturesFetcher gets features 100 at a time.
        return route.abort();
      } else {
        // Continue with the original request
        route.continue();
      }
    });
  });
  await gotoOverviewPageUrl(page, 'http://localhost:5555/');
  const exportButton = page.getByRole('button', {
    name: 'Export to CSV',
  });

  await expect(exportButton).toBeVisible();
  await exportButton.click();

  // Assert toast is visible
  const toast = page.locator('.toast');
  await toast.waitFor({state: 'visible'});
});
