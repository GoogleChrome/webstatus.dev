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
  gotoOverviewPageUrl,
  waitForSidebarLoaded,
  expectDualThemeScreenshot,
  setupFakeNow,
} from '../utils/utils.js';
import {setupVisualFixtures} from '../utils/fixture-routes.js';

const DEFAULT_PAGE_SIZE = 25;

test.beforeEach(async ({page}) => {
  await setupFakeNow(page);
  await setupVisualFixtures(page);
});

test.describe('Overview Page Visual Snapshots', () => {
  test('matches the screenshot', async ({page}) => {
    await gotoOverviewPageUrl(page, 'http://localhost:5555/');
    const pageContainer = page.locator('.page-container');
    await expectDualThemeScreenshot(page, pageContainer, 'overview-page');
  });

  test('matches the screenshot for mobile columns', async ({page}) => {
    await gotoOverviewPageUrl(
      page,
      'http://localhost:5555/?columns=name,availability_chrome_android,availability_firefox_android,' +
        'availability_safari_ios,stable_chrome_android,stable_firefox_android,stable_safari_ios,' +
        'experimental_chrome_android,experimental_firefox_android,experimental_safari_ios',
    );
    const pageContainer = page.locator('.page-container');
    await expectDualThemeScreenshot(
      page,
      pageContainer,
      'overview-page-mobile',
    );
  });

  test('screenshot for availability sort', async ({page}) => {
    await gotoOverviewPageUrl(
      page,
      'http://localhost:5555/?sort=availability_chrome_asc',
    );
    const pageContainer = page.locator('.page-container');
    await expectDualThemeScreenshot(page, pageContainer, 'overview-page-sort');
  });

  test('screenshot for developer upvotes column', async ({page}) => {
    await gotoOverviewPageUrl(
      page,
      'http://localhost:5555/?columns=name,baseline_status,availability_chrome,availability_firefox,availability_edge,' +
        'availability_safari,chrome_usage,developer_signal_upvotes',
    );
    const pageContainer = page.locator('.page-container');
    await expectDualThemeScreenshot(
      page,
      pageContainer,
      'overview-page-developer-upvotes',
    );
  });

  test('matches screenshot for page 2 pagination', async ({page}) => {
    await gotoOverviewPageUrl(
      page,
      'http://localhost:5555/?page_token=page_2_token',
    );
    const pageContainer = page.locator('.page-container');
    await expectDualThemeScreenshot(
      page,
      pageContainer,
      'overview-page-page-2',
    );
  });

  test('shows an error that their query is invalid', async ({page}) => {
    await page.goto('http://localhost:5555/?q=available_on%3Achrom');
    await waitForSidebarLoaded(page);

    const message = page.locator('.message');
    await message.waitFor({state: 'visible'});
    expect(message).toContainText('Invalid query...');

    const pageContainer = page.locator('.page-container');
    await expectDualThemeScreenshot(page, pageContainer, 'invalid-query');
  });

  test('shows an unknown error when there is an internal error', async ({
    page,
  }) => {
    await page.route('**/v1/features?page_size=' + DEFAULT_PAGE_SIZE, route =>
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
    await waitForSidebarLoaded(page);

    const message = page.locator('.message');
    await message.waitFor({state: 'visible'});
    expect(message).toContainText('Something went wrong...');

    const pageContainer = page.locator('.page-container');
    await expectDualThemeScreenshot(page, pageContainer, 'internal-error');
  });
});
