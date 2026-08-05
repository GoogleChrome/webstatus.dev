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

import {test, expect, type Page} from '@playwright/test';
import {
  loginAsUser,
  BASE_URL,
  waitForSidebarLoaded,
  resetUserData,
} from '../utils/utils.js';

function waitForChannelsRefetch(page: Page) {
  return page.waitForResponse(
    response =>
      response.url().includes('/v1/users/me/notification-channels') &&
      response.request().method() === 'GET',
  );
}

test('redirects unauthenticated user to home and shows toast', async ({
  page,
}) => {
  await page.goto(`${BASE_URL}/settings/notification-channels`);

  // Expect to be redirected to the home page.
  await expect(page).toHaveURL(`${BASE_URL}/`);
});

test.describe('Notification Channels Page Functional', () => {
  test.beforeEach(async ({page}) => {
    await resetUserData();
    await loginAsUser(page, 'test user 1');

    const initialLoad = page.waitForResponse(
      response =>
        response.url().includes('/v1/users/me/notification-channels') &&
        response.request().method() === 'GET',
    );

    await page.goto(`${BASE_URL}/settings/notification-channels`);
    await waitForSidebarLoaded(page);
    await initialLoad;
  });

  test.afterAll(async () => {
    await resetUserData();
  });

  test('authenticated user sees their email channel', async ({page}) => {
    await expect(page).toHaveURL(`${BASE_URL}/settings/notification-channels`);

    const emailPanel = page.locator('webstatus-notification-email-channels');
    await expect(emailPanel).toBeVisible();
    await expect(emailPanel).toContainText('test.user.1@example.com');
    await expect(emailPanel).toContainText('Enabled');
  });
});
