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

import {test, expect, request} from '@playwright/test';

test('matches the screenshot for unauthenticated user', async ({page}) => {
  await page.goto('http://localhost:5555/');

  const login = page.locator('webstatus-login');
  await expect(login).toContainText('Log in');

  const header = page.locator('webstatus-header');
  await expect(header).toHaveScreenshot('unauthenticated-header.png');
});

test('can sign in and sign out user', async ({page}) => {
  // Clicking the log in button will create a popup that we need to capture.
  const popupPromise = page.waitForEvent('popup');
  await page.goto('http://localhost:5555/');
  await page.getByText('Log in').click();
  await page.route('*//api.github.com/*', async route => {
    // Mock successful login response
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        email: 'test_email@gmail.com',
        data: {
          login: 'default_user',
        },
      }),
    });
  });
  const popup = await popupPromise;

  await popup.getByText('default_user').waitFor();
  await popup.getByText('default_user').hover(); // Needed for Firefox for some reason.
  await popup.getByText('default_user').click();
  await popup.waitForEvent('close');
  const login = page.locator('webstatus-login');

  const expectedUsername = 'default_user';

  // Should have the githubUsername = 'default_user'
  await expect(login).toContainText(expectedUsername);

  const header = page.locator('webstatus-header');
  await expect(header).toHaveScreenshot('authenticated-header.png');

  // Show the menu
  await login.click();

  const signOutBtn = login.getByText('Sign out');

  await signOutBtn.click();

  await expect(login).toHaveScreenshot('unauthenticated-button.png');
});
