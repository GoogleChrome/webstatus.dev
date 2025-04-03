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

import {test, expect, Request} from '@playwright/test';
import {
  gotoOverviewPageUrl,
  getOverviewPageFeatureCount,
  loginAsUser,
} from './utils';

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
    }),
  );
  await page.goto('http://localhost:5555/');

  const message = page.locator('.message');
  await message.waitFor({state: 'visible'});
  expect(message).toContainText('Something went wrong...');

  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot('internal-error.png');
});

test('hides the Feature column', async ({page}) => {
  await gotoOverviewPageUrl(page, 'http://localhost:5555/');

  // Check that the "Feature" column is visible by default.
  let nameColumn = page.locator('th > a', {hasText: 'Feature'});
  await expect(nameColumn).toBeVisible();

  // Click the Columns button to open the column selector.
  const columnsButton = page.locator('#columns-button');
  await columnsButton.waitFor({state: 'visible'});
  await columnsButton.click();
  const webstatusColumnsDialog = page.locator('webstatus-columns-dialog');
  const columnsDialog = webstatusColumnsDialog.getByRole('dialog');
  await columnsDialog.waitFor({state: 'visible'});

  // Uncheck the "Feature name" checkbox.
  const nameCheckbox = webstatusColumnsDialog.locator(
    'sl-checkbox[value="name"]',
  );
  await nameCheckbox.click();

  // Click the Save button.
  await page.locator('#columns-save-button').click();
  await page.waitForTimeout(500);

  // Make sure the "Feature" column is no longer visible.
  nameColumn = page.locator('th > a', {hasText: 'Feature'});
  await expect(nameColumn).not.toBeVisible();
});

test('shows the Baseline status column with low and high date options', async ({
  page,
}) => {
  await gotoOverviewPageUrl(page, 'http://localhost:5555/');

  // Check that the "Baseline" column is visible by default.
  const baselineStatusColumn = page.locator('th > a', {
    hasText: 'Baseline',
  });
  await expect(baselineStatusColumn).toBeVisible();

  // Click the Columns button to open the column selector.
  const columnsButton = page.locator('#columns-button');
  await columnsButton.waitFor({state: 'visible'});
  await columnsButton.click();
  const webstatusColumnsDialog = page.locator('webstatus-columns-dialog');
  const columnsDialog = webstatusColumnsDialog.getByRole('dialog');
  await columnsDialog.waitFor({state: 'visible'});

  // Check the "Baseline status low date" checkbox.
  const baselineStatusLowDateCheckbox = webstatusColumnsDialog.locator(
    'sl-checkbox[value="baseline_status_low_date"]',
  );
  await baselineStatusLowDateCheckbox.click();
  // Check the "Baseline status high date" checkbox.
  const baselineStatusHighDateCheckbox = webstatusColumnsDialog.locator(
    'sl-checkbox[value="baseline_status_high_date"]',
  );
  await baselineStatusHighDateCheckbox.click();

  // Click the Save button.
  await page.locator('#columns-save-button').click();
  await page.waitForTimeout(500);

  // Check that "Newly available: " text is visible somewhere.
  const baselineStatusLowDateText = page.locator('td', {
    hasText: 'Newly available: ',
  });
  await expect(baselineStatusLowDateText.first()).toBeVisible();
  // Check that "Widely available: " and "Projected Widely available: "
  // text is visible somewhere.
  const baselineStatusHighDateText = page.locator('td', {
    hasText: 'Widely available: ',
  });
  await expect(baselineStatusHighDateText.first()).toBeVisible();
  const baselineStatusProjectedHighDateText = page.locator('td', {
    hasText: 'Projected widely available: ',
  });
  await expect(baselineStatusProjectedHighDateText.first()).toBeVisible();
});

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

test('Export to CSV button fails to request all features and shows toast', async ({
  page,
}) => {
  // Mock the API to return an error when requesting all features.
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

test('Test id search atoms in a query', async ({page}) => {
  await gotoOverviewPageUrl(page, 'http://localhost:5555/');
  const searchbox = page.locator('#inputfield');
  await expect(searchbox).toBeVisible();
  await expect(searchbox).toHaveAttribute('value', '');

  const initialFeatureCount = await getOverviewPageFeatureCount(page);
  expect(initialFeatureCount).toBeGreaterThan(7);

  const sevenIDAtoms =
    'id:Molestiae77 OR id:Ratione74 OR id:Molestias63 OR id:Ut59 OR id:Ad50 OR id:Inventore43 OR id:Rem51';
  await page.keyboard.type('/' + sevenIDAtoms);
  await expect(searchbox).toHaveAttribute('value', sevenIDAtoms);
  await page.locator('#filter-submit-button').click();
  await page.waitForLoadState('networkidle');

  const newFeatureCount = await getOverviewPageFeatureCount(page);
  expect(newFeatureCount).toEqual(7);
});

test('Typing slash focuses on searchbox', async ({page}) => {
  await gotoOverviewPageUrl(page, 'http://localhost:5555/');
  const searchbox = page.locator('#inputfield');
  await expect(searchbox).toBeVisible();
  await expect(searchbox).toHaveAttribute('value', '');
  await page.keyboard.type('abc/def/ghi');
  // Characters before the first slash go to the page.
  // The slash focuses on the searchbox.
  // Later characters, including slashes, go in the searchbox.
  await expect(searchbox).toHaveAttribute('value', 'def/ghi');
});

test('newly logged in user should see no errors (toasts)', async ({page}) => {
  await loginAsUser(page, 'fresh user');
  await gotoOverviewPageUrl(page, 'http://localhost:5555/');
  await expect(page.locator('.toast')).toHaveCount(0);
});

test.describe('saved searches', () => {
  test('unauthenticated user can load a public saved search and navigate pages', async ({
    page,
  }) => {
    let featuresRequests: Request[] = [];
    let savedSearchesRequests: Request[] = [];

    page.on('request', req => {
      if (req.url().startsWith('http://localhost:8080/v1/features')) {
        featuresRequests.push(req);
      }
      if (req.url().startsWith('http://localhost:8080/v1/saved-searches')) {
        savedSearchesRequests.push(req);
      }
    });

    async function verifyFeaturesRequest(expectedQuery: string) {
      expect(
        featuresRequests.length,
        'Should have made one features request',
      ).toBe(1);
      expect(
        new URL(featuresRequests[0].url()).searchParams.get('q'),
        'Features request query should match',
      ).toBe(expectedQuery);
    }

    async function verifySavedSearchesRequest(
      expectedLength: number,
      expectedUrl?: string,
    ) {
      expect(
        savedSearchesRequests.length,
        'Saved searches request length should match',
      ).toBe(expectedLength);
      if (expectedUrl) {
        expect(
          savedSearchesRequests[0].url(),
          'Saved searches request URL should match',
        ).toBe(expectedUrl);
      }
    }

    async function verifyTableRowCount(expectedCount: number) {
      const rowCount = await page.locator('table tbody tr').count();
      expect(rowCount, 'Table row count should match').toBe(expectedCount);
    }

    async function clickNextPage() {
      featuresRequests = [];
      savedSearchesRequests = [];
      await page.getByText('Next').click();
      await page.waitForLoadState('networkidle');
    }

    await test.step('Load saved search', async () => {
      await gotoOverviewPageUrl(
        page,
        'http://localhost:5555/?search_id=a09386fe-65f1-4640-b28d-3cf2f2de69c9',
      );
      const featureCount = await getOverviewPageFeatureCount(page);
      expect(featureCount, 'Feature count should be 74').toEqual(74);

      const searchbox = page.locator('#inputfield');
      await expect(searchbox, 'Search box value should match').toHaveAttribute(
        'value',
        'baseline_status:limited OR available_on:chrome',
      );

      const description = page.locator('#overview-description');
      await expect(
        description,
        'Description should contain text',
      ).toContainText(
        'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed non risus. Suspendisse lectus tortor, dignissim sit amet, adipiscing nec, ultricies sed, dolor. Cras elementum ultrices diam. Maecenas ligula massa, varius a, semper congue, euismod non, mi. Proin porttitor, orci nec nonummy molestie, enim est eleifend mi, non fermentum diam nisl sit amet erat. Duis semper. Duis arcu massa, scelerisque vitae, consequat in, pretium a, enim. Pellentesque congue. Ut in risus volutpat libero pharetra tempor. Cras vestibulum bibendum augue. Praesent egestas leo in pede. Praesent blandit odio eu enim. Pellentesque sed dui ut augue blandit sodales. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Aliquam nibh. Mauris ac mauris sed pede pellentesque fermentum. Maecenas adipiscing ante non diam sodales hendrerit. Ut velit mauris, egestas sed, gravida nec, ornare ut, mi. Aenean ut orci vel massa suscipit pulvinar. Nulla sollicitudin. Fusce varius, ligula non tempus aliquam, nunc turpis ullamcorper nibh, in tempus sapien eros vitae ligula. Pellentesque rhoncus nunc et augue. Integer id felis. Curabitur aliquet pellentesque diam. Integer quis metus vitae elit lobortis egestas. Integer egestas risus ut lectus. Nam viverra, erat vitae porta sodales, nulla diam tincidunt sem, et dictum felis nunc nec ligula. Sed nec lectus. Donec in velit. Curabitur tempus. Sed consequat, leo eget bibendum sodales, augue velit cursus nunc, quis gravida magna mi a libero. Duis vulputate elit eu elit. Donec interdum, metus et hendrerit aliquet, dolor diam sagittis ligula, eget egestas libero turpis vel mi. Nunc nulla. Maecenas vitae neque. Vivamus ultrices luctus nunc. Vivamus cursus, metus quis ullamcorper sodales, lectus lectus tempor enim, vitae gravida nibh purus ut nibh. Duis in augue. Cras nulla. Vivamus laoreet. Curabitur suscipit suscipit tellus.',
      );

      const title = page.locator('#overview-title');
      await expect(title, 'Title should contain text').toContainText(
        'I like queries',
      );

      await verifyFeaturesRequest(
        'baseline_status:limited OR available_on:chrome',
      );
      await verifySavedSearchesRequest(
        1,
        'http://localhost:8080/v1/saved-searches/a09386fe-65f1-4640-b28d-3cf2f2de69c9',
      );
      await verifyTableRowCount(25);
    });

    await test.step('Navigate to next page (1)', async () => {
      await clickNextPage();
      const featureCount = await getOverviewPageFeatureCount(page);
      expect(featureCount, 'Feature count should be 74').toEqual(74);
      await verifyTableRowCount(25);
      await verifyFeaturesRequest(
        'baseline_status:limited OR available_on:chrome',
      );
      await verifySavedSearchesRequest(0);
    });

    await test.step('Navigate to next page (2)', async () => {
      await clickNextPage();
      const featureCount = await getOverviewPageFeatureCount(page);
      expect(featureCount, 'Feature count should be 74').toEqual(74);
      await verifyTableRowCount(24);
      await verifyFeaturesRequest(
        'baseline_status:limited OR available_on:chrome',
      );
      await verifySavedSearchesRequest(0);
    });
  });

  test('Bad search id shows an error', async ({page}) => {
    await gotoOverviewPageUrl(page, 'http://localhost:5555/?search_id=bad-id');

    // Assert toast is visible
    const toast = page.locator('.toast');
    await toast.waitFor({state: 'visible'});
    // TODO: we need to figure out a way to assert toast message.
  });
});
