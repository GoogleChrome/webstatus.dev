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

import {test} from '@playwright/test';
import {
  loginAsUser,
  expectDualThemeScreenshot,
  waitForSidebarLoaded,
} from '../utils/utils.js';
import {setupVisualFixtures} from '../utils/fixture-routes.js';

test.beforeEach(async () => {});

test('matches the screenshot', async ({page}) => {
  await setupVisualFixtures(page, {userRole: 'unauthenticated'});
  await page.goto('http://localhost:5555/');

  // The sidebar menu should be shown by default.
  const sidebar = page.locator('webstatus-sidebar');
  await waitForSidebarLoaded(page);
  await expectDualThemeScreenshot(page, sidebar, 'sidebar');
});

test('matches the screenshot for an authenticated user', async ({page}) => {
  await setupVisualFixtures(page, {userRole: 'authenticated'});
  await loginAsUser(page, 'test user 1');
  await page.goto('http://localhost:5555/');

  // The sidebar menu should be shown by default.
  const sidebar = page.locator('webstatus-sidebar');
  await waitForSidebarLoaded(page);
  await expectDualThemeScreenshot(page, sidebar, 'sidebar-authenticated');
});
