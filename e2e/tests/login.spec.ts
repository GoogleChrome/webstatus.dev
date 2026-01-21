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

import {test, expect} from '@playwright/test';
import {dismissToast, freezeAnimations, loginAsUser, testUsers} from './utils';

test.describe('Login Component States', () => {
  test('displays spinner and is disabled during profile sync', async ({
    page,
  }) => {
    // Intercept the pingUser request to introduce a delay.
    await page.route('**/v1/users/me/ping', async route => {
      await new Promise(resolve => setTimeout(resolve, 1000)); // Delay to ensure 'syncing' state is capturable
      await route.continue();
    });

    await freezeAnimations(page);
    // Perform login and wait for the 'syncing' state.
    await loginAsUser(page, 'test user 1', {waitFor: 'syncing'});

    // The button should be in a loading state.
    const loginButton = page.getByRole('button', {
      name: testUsers['test user 1'],
    });
    await expect(loginButton.locator('sl-spinner')).toBeVisible();
    await expect(loginButton).toBeDisabled();

    // Take a screenshot for visual regression.
    await page.mouse.move(0, 0); // Move mouse to avoid hover effects.
    await expect(page.locator('webstatus-header')).toHaveScreenshot(
      'login-syncing-state.png',
    );

    // Now, wait for the sync to complete and verify the final state.
    await expect(loginButton.locator('sl-spinner')).toBeHidden();
    await expect(loginButton).not.toBeDisabled();
  });

  test('displays an error icon if profile sync fails', async ({page}) => {
    // Intercept the pingUser request and make it fail.
    await page.route('**/v1/users/me/ping', route => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({message: 'Internal Server Error'}),
      });
    });

    // Perform the login and wait for the 'error' state.
    await loginAsUser(page, 'test user 1', {waitFor: 'error'});

    const loginButton = page.getByRole('button', {
      name: testUsers['test user 1'],
    });

    // Check the button's state.
    await expect(loginButton).toBeVisible();
    await expect(loginButton).not.toBeDisabled();

    // Dismiss the toast.
    await dismissToast(page);

    // Take a screenshot for visual regression.
    await page.mouse.move(0, 0);
    await expect(page.locator('webstatus-header')).toHaveScreenshot(
      'login-error-state.png',
    );
  });

  test('displays the idle state after a successful login', async ({page}) => {
    // Perform a standard successful login using the main helper (waits for 'idle' by default).
    await loginAsUser(page, 'test user 1');

    const loginButton = page.getByRole('button', {
      name: testUsers['test user 1'],
    });
    await expect(loginButton).toBeVisible();
    await expect(loginButton.locator('sl-spinner')).toBeHidden();
    await expect(loginButton).not.toBeDisabled();

    // Take a screenshot for visual regression.
    await page.mouse.move(0, 0);
    await expect(page.locator('webstatus-header')).toHaveScreenshot(
      'login-idle-state.png',
    );
  });
});

test('matches the screenshot for unauthenticated user', async ({page}) => {
  await page.goto('http://localhost:5555/');

  const login = page.locator('webstatus-login');
  await expect(login).toContainText('Log in');

  const header = page.locator('webstatus-header');
  await expect(header).toHaveScreenshot('unauthenticated-header.png');
});

test('can sign in and sign out user', async ({page}) => {
  // Start waiting for the ping request before logging in.
  const pingRequestPromise = page.waitForRequest(
    request =>
      request.url().endsWith('/v1/users/me/ping') &&
      request.method() === 'POST',
  );

  await loginAsUser(page, 'test user 1');
  const login = page.locator('webstatus-login');

  const expectedEmail = 'test.user.1@example.com';

  // Should have the email address
  await expect(login).toContainText(expectedEmail);

  // Wait for the ping request to be made and assert that it happened.
  const pingRequest = await pingRequestPromise;
  expect(pingRequest).toBeTruthy();

  const header = page.locator('webstatus-header');
  await expect(header).toHaveScreenshot('authenticated-header.png');

  // Show the menu
  await login.click();

  const signOutBtn = login.getByText('Sign out');

  await signOutBtn.click();

  await expect(login).toHaveScreenshot('unauthenticated-button.png');
});
