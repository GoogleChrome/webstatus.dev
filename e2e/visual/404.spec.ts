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
  BASE_URL,
  expect404PageButtons,
  goTo404Page,
  waitForSidebarLoaded,
  expectDualThemeScreenshot,
} from '../utils/utils.js';
import {setupVisualFixtures} from '../utils/fixture-routes.js';

test.beforeEach(async ({page}) => {
  await setupVisualFixtures(page);
});

test.describe('404 Page Visual Snapshots', () => {
  test('matches the screenshot 404 not found page', async ({page}) => {
    await page.goto(`${BASE_URL}/bad_url`);
    const pageContainer = page.locator('.page-container');
    await waitForSidebarLoaded(page);
    await expectDualThemeScreenshot(
      page,
      pageContainer,
      'not-found-error-page',
    );
  });

  test('matches the screenshot with similar features', async ({page}) => {
    const query = 'g';
    await goTo404Page(page, query);

    await expect(page.locator('.similar-features-container')).toBeVisible();
    await expect404PageButtons(page, {hasSearch: true});

    const pageContainer = page.locator('.page-container');
    await waitForSidebarLoaded(page);
    await expectDualThemeScreenshot(
      page,
      pageContainer,
      'not-found-error-page-similar-results',
    );
  });
});
