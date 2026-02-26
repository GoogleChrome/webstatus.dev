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
import {loginAsUser, BASE_URL, forceTheme} from './utils';

test.beforeEach(async ({page}) => {
  await forceTheme(page, 'light');
});

test.describe('Notification Channels Page', () => {
  test('redirects unauthenticated user to home and shows toast', async ({
    page,
  }) => {
    await page.goto(`${BASE_URL}/settings/notification-channels`);

    // Expect to be redirected to the home page.
    await expect(page).toHaveURL(BASE_URL);
    // FYI: We do not assert the toast because it flashes on the screen due to the redirect.
  });

  test('authenticated user sees their email channel and coming soon messages', async ({
    page,
  }) => {
    // Log in as a test user
    await loginAsUser(page, 'test user 1');

    // Navigate to the notification channels page
    await page.goto(`${BASE_URL}/settings/notification-channels`);

    // Move the mouse to a neutral position to avoid hover effects on the screenshot
    await page.mouse.move(0, 0);

    // Expect the URL to be correct
    await expect(page).toHaveURL(`${BASE_URL}/settings/notification-channels`);

    // Verify Email panel content
    const emailPanel = page.locator('webstatus-notification-email-channels');
    await expect(emailPanel).toBeVisible();
    await expect(emailPanel).toContainText('test.user.1@example.com');
    await expect(emailPanel).toContainText('Enabled');

    // Verify RSS panel content
    const rssPanel = page.locator('webstatus-notification-rss-channels');
    await expect(rssPanel).toBeVisible();
    await expect(rssPanel).toContainText('Coming soon');

    // Verify Webhook panel content
    const webhookPanel = page.locator(
      'webstatus-notification-webhook-channels',
    );
    await expect(webhookPanel).toBeVisible();
    await expect(webhookPanel).toContainText('Coming soon');

    // Take a screenshot for visual regression
    const pageContainer = page.locator('.page-container');
    await expect(pageContainer).toHaveScreenshot(
      'notification-channels-authenticated.png',
    );
  });

  test('authenticated user sees their email channel in dark mode', async ({
    page,
  }) => {
    // Force dark mode
    await forceTheme(page, 'dark');

    // Log in as a test user
    await loginAsUser(page, 'test user 1');

    // Navigate to the notification channels page
    await page.goto(`${BASE_URL}/settings/notification-channels`);

    // Move the mouse to a neutral position to avoid hover effects on the screenshot
    await page.mouse.move(0, 0);

    // Expect the URL to be correct
    await expect(page).toHaveURL(`${BASE_URL}/settings/notification-channels`);

    // Take a screenshot for visual regression
    const pageContainer = page.locator('.page-container');
    await expect(pageContainer).toHaveScreenshot(
      'notification-channels-authenticated-dark.png',
    );
  });
});
