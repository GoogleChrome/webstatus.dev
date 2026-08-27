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
  expectDualThemeScreenshot,
  waitForSidebarLoaded,
  waitForTabbedChartCompletion,
  setupFakeNow,
} from '../utils/utils.js';
import {setupVisualFixtures} from '../utils/fixture-routes.js';

test.beforeEach(async ({page}) => {
  await setupFakeNow(page);
  await page.setViewportSize({width: 1280, height: 1500});
  await setupVisualFixtures(page);
});

const featureID = 'anchor-positioning';
const discouragedFeatureId = 'discouraged';

test.describe('Feature Page Visual Snapshots', () => {
  test('matches the screenshot', async ({page}) => {
    await page.goto(`http://localhost:5555/features/${featureID}`);

    // Wait for the chart to fully render
    await waitForTabbedChartCompletion(
      page,
      'feature-wpt-implementation-progress',
      0,
    );

    const pageContainer = page.locator('.page-container');
    await waitForSidebarLoaded(page);
    await expectDualThemeScreenshot(page, pageContainer, 'feature-page');
  });

  test('matches the screenshot for a discouraged feature', async ({page}) => {
    await page.goto(`http://localhost:5555/features/${discouragedFeatureId}`);

    // Wait for the specific feature name header to be visible
    await expect(page.locator('#nameAndOffsiteLinks h1')).toBeVisible();

    await waitForTabbedChartCompletion(
      page,
      'feature-wpt-implementation-progress',
      0,
    );

    const pageContainer = page.locator('.page-container');
    await waitForSidebarLoaded(page);
    await expectDualThemeScreenshot(
      page,
      pageContainer,
      'feature-page-discouraged',
    );
  });

  test('mobile chart displays on click and matches screenshot', async ({
    page,
  }) => {
    await page.goto(`http://localhost:5555/features/${featureID}`);
    await waitForTabbedChartCompletion(
      page,
      'feature-wpt-implementation-progress',
      0,
    );
    const mobileTab = page.locator(
      'sl-tab#feature-wpt-implementation-progress-tab-mobile',
    );

    await mobileTab.click();
    await page.waitForTimeout(2000);
    await waitForTabbedChartCompletion(
      page,
      'feature-wpt-implementation-progress',
      1,
    );

    const pageContainer = page.locator('.page-container');
    await waitForSidebarLoaded(page);
    await expectDualThemeScreenshot(page, pageContainer, 'feature-page-mobile');
  });
});
