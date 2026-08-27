/**
 * Copyright 2026 Google LLC
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
import {
  BASE_URL,
  gotoOverviewPageUrl,
  getOverviewPageFeatureCount,
  waitForOverviewPageLoad,
} from '../utils/utils.js';

test.describe('Synthetic Smoke Probes', () => {
  test('Homepage loads without uncaught errors and displays feature table', async ({
    page,
  }) => {
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    await gotoOverviewPageUrl(page, `${BASE_URL}/`);

    // Verify main app element rendered
    const app = page.locator('webstatus-app');
    await expect(app).toBeVisible();

    // Verify header and navigation
    const header = page.locator('webstatus-header');
    await expect(header).toBeVisible();

    // Verify feature table rendered at least one row
    const rowCount = await getOverviewPageFeatureCount(page);
    expect(rowCount).toBeGreaterThan(0);

    // Verify no fatal console errors
    expect(
      consoleErrors.filter(
        err =>
          !err.includes('favicon.ico') &&
          !err.includes('gtag') &&
          !err.includes('analytics'),
      ),
    ).toEqual([]);
  });

  test('Search filter executes and returns matching features', async ({
    page,
  }) => {
    await gotoOverviewPageUrl(page, `${BASE_URL}/`);

    const searchBox = page.locator('#inputfield');
    await expect(searchBox).toBeVisible();

    // Type query and submit via slash shortcut
    const query = 'baseline_status:widely';
    await page.keyboard.type('/' + query);
    await expect(searchBox).toHaveAttribute('value', query);
    await page.locator('#filter-submit-button').click();
    await waitForOverviewPageLoad(page);

    // Verify URL updated with search query
    expect(page.url()).toContain(`q=${encodeURIComponent(query)}`);

    // Verify rows are displayed for the search query
    const rowCount = await getOverviewPageFeatureCount(page);
    expect(rowCount).toBeGreaterThan(0);
  });

  test('Feature detail page renders metadata and chart panels', async ({
    page,
  }) => {
    await gotoOverviewPageUrl(page, `${BASE_URL}/`);

    // Click on the first feature link in the table
    const firstFeatureLink = page
      .locator('.feature-name-cell a.feature-page-link')
      .first();
    await expect(firstFeatureLink).toBeVisible();
    await firstFeatureLink.click();

    // Verify feature page container rendered
    const featureContainer = page.locator('.page-container');
    await expect(featureContainer).toBeVisible();

    // Verify feature title heading is present
    const titleHeading = page.locator('h1');
    await expect(titleHeading).toBeVisible();
    const titleText = await titleHeading.textContent();
    expect(titleText?.trim().length).toBeGreaterThan(0);

    // Verify baseline status badge is rendered
    const baselineBadge = page.locator('.status-badge');
    await expect(baselineBadge.first()).toBeVisible();
  });

  test('Error page handles non-existent paths gracefully', async ({page}) => {
    await page.goto(`${BASE_URL}/errors-404/page-not-found`);

    const notFoundContainer = page.locator('.page-container');
    await expect(notFoundContainer).toBeVisible();

    // Verify return home button or navigation is present
    const homeLink = page.getByRole('link', {name: /Home|Overview/i});
    await expect(homeLink.first()).toBeVisible();
  });
});
