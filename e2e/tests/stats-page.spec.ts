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
import {setupFakeNow} from './utils';

async function waitForAllChartsToLoad(page: Page) {
  // Wait for all charts to finish loading.
  for (const chartLoading of await page.getByText('Loading stats').all())
    await expect(chartLoading).toBeHidden();
}

test.beforeEach(async ({page}) => {
  await setupFakeNow(page);
});

test('matches the screenshot', async ({page}) => {
  await page.goto('http://localhost:5555/stats');

  // Wait for the global feature support chart container to exist.
  await page.waitForSelector('#global-feature-support-chart-container');

  // Wait for the missing one implementatoin chart container to exist.
  await page.waitForSelector('#missing-one-implementation-chart-container');

  await waitForAllChartsToLoad(page);

  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot();
});
