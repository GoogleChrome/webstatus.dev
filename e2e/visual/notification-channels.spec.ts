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
  expectDualThemeScreenshot,
  waitForSidebarLoaded,
  loginAsUser,
} from '../utils/utils.js';
import {setupVisualFixtures} from '../utils/fixture-routes.js';

test.describe('Notification Channels Page Visual Snapshots', () => {
  test.beforeEach(async ({page}) => {
    await setupVisualFixtures(page, {userRole: 'authenticated'});
    await loginAsUser(page, 'test user 1');
    await page.goto(`${BASE_URL}/settings/notification-channels`);
    await waitForSidebarLoaded(page);
  });

  test('authenticated user sees their email channel and coming soon messages', async ({
    page,
  }) => {
    await expect(page).toHaveURL(`${BASE_URL}/settings/notification-channels`);

    const emailPanel = page.locator('webstatus-notification-email-channels');
    await expect(emailPanel).toBeVisible();
    await expect(emailPanel).toContainText('test.user.1@example.com');
    await expect(emailPanel).toContainText('Enabled');

    const rssPanel = page.locator('webstatus-notification-rss-channels');
    await expect(rssPanel).toBeVisible();

    const webhookPanel = page.locator(
      'webstatus-notification-webhook-channels',
    );
    await expect(webhookPanel).toBeVisible();

    await page.mouse.move(0, 0);

    const pageContainer = page.locator('.page-container');
    await expectDualThemeScreenshot(
      page,
      pageContainer,
      'notification-channels-authenticated',
    );
  });
});
