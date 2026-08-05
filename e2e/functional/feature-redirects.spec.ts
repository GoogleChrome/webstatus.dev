/**
 * Copyright 2024 Google LLC
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
  waitForSidebarLoaded,
  waitForTabbedChartCompletion,
  setupFakeNow,
} from '../utils/utils.js';

test.beforeEach(async ({page}) => {
  await setupFakeNow(page);
  await page.setViewportSize({width: 1280, height: 1500});
});

test('redirects for a moved feature', async ({page, browserName}) => {
  test.skip(browserName === 'webkit', 'Skipping webkit due to flakiness');
  const responsePromise = page.waitForResponse(
    response =>
      response.url().includes('/v1/features/old-feature') &&
      response.request().method() === 'GET',
  );
  await page.goto('http://localhost:5555/features/old-feature');
  await responsePromise;

  // Expect the URL to be updated to the new feature's URL.
  await expect(page).toHaveURL(
    'http://localhost:5555/features/new-feature?redirected_from=old-feature',
  );

  // Expect the title and redirect banner to be correct.
  await expect(page.locator('h1')).toHaveText('New Feature');
  await expect(
    page.locator(
      'sl-alert:has-text("You have been redirected from an old feature ID")',
    ),
  ).toBeVisible();

  // Wait for charts to load to avoid flakiness in the test.
  await waitForTabbedChartCompletion(
    page,
    'feature-wpt-implementation-progress',
    0,
  );
});

test('shows gone page for a split feature', async ({page}) => {
  const responsePromise = page.waitForResponse(
    response =>
      response.url().includes('/v1/features/before-split-feature') &&
      response.request().method() === 'GET',
  );
  await page.goto('http://localhost:5555/features/before-split-feature');
  await responsePromise;

  // Expect to be redirected to the 'feature-gone-split' page.
  await expect(page).toHaveURL(
    'http://localhost:5555/errors-410/feature-gone-split?new_features=after-split-feature-1,after-split-feature-2',
  );

  // Assert that the content of the 410 page is correct.
  await expect(page.locator('.new-results-header')).toContainText(
    'Please see the following new features',
  );
  await expect(
    page.locator('a[href="/features/after-split-feature-1"]'),
  ).toBeVisible();
  await expect(
    page.locator('a[href="/features/after-split-feature-2"]'),
  ).toBeVisible();
});
