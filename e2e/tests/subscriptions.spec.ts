/**
 * Copyright 2025 Google LLC
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
import {loginAsUser, BASE_URL, resetUserData} from './utils';

const subscriptionPageURL = `${BASE_URL}/settings/subscriptions`;

test.describe('Subscriptions Page', () => {
  test.beforeAll(async () => {
    await resetUserData();
  });
  test.afterAll(async () => {
    await resetUserData();
  });
  test.beforeEach(async ({page}) => {
    // Log in as a test user. This will setup
    await loginAsUser(page, 'test user 1');
  });

  test('should display existing subscriptions', async ({page}) => {
    // Navigate to the subscriptions page.
    await page.goto(subscriptionPageURL);

    // Wait for the list of subscriptions to be visible.
    await page.waitForSelector('ul');

    // Find the list item for the subscription we created in the fake data.
    const subscriptionListItem = page.locator('li', {
      hasText: 'my first project query',
    });

    // Assert that the subscription details are visible.
    await expect(subscriptionListItem).toBeVisible();
    await expect(subscriptionListItem).toContainText('Channel:');
    await expect(subscriptionListItem).toContainText('Frequency: weekly');
    await expect(
      subscriptionListItem.getByRole('button', {name: 'Edit'}),
    ).toBeVisible();
    await expect(
      subscriptionListItem.getByRole('button', {name: 'Delete'}),
    ).toBeVisible();
  });

  test('should allow creating a new subscription from the saved searches page', async ({
    page,
  }) => {
    // Navigate to the main page.
    await page.goto('/');

    // The user has two saved searches. Click on the second one.
    await page.getByRole('button', {name: 'I like queries'}).click();

    // The subscribe button should now be visible.
    const subscribeButton = page.getByRole('button', {name: 'Subscribe'});
    await expect(subscribeButton).toBeVisible();
    await subscribeButton.click();

    // The dialog should now be open.
    const dialog = page.locator('webstatus-manage-subscriptions-dialog');
    await expect(dialog).toBeVisible();

    // Select the notification channel (the user's email).
    await dialog.getByText('test.user.1@example.com').click();

    // Select a trigger.
    await dialog.getByText('...becomes newly available').click();

    // Select a frequency.
    await dialog.getByText('Monthly updates').click();

    // Save the subscription.
    await dialog.getByRole('button', {name: 'Save'}).click();

    // Assert that the success toast appears.
    await expect(
      page.locator('sl-alert', {hasText: 'Subscription saved!'}),
    ).toBeVisible();

    // Navigate to the subscriptions page to verify the new subscription.
    await page.goto(subscriptionPageURL);
    const newSubscription = page.locator('li', {
      hasText: 'I like queries',
    });
    await expect(newSubscription).toBeVisible();
    await expect(newSubscription).toContainText('Frequency: monthly');
  });

  test('should allow editing an existing subscription', async ({page}) => {
    // Navigate to the subscriptions page.
    await page.goto(subscriptionPageURL);

    // Find the subscription for "my first project query" and click its Edit button.
    const subscriptionListItem = page.locator('li', {
      hasText: 'my first project query',
    });
    await subscriptionListItem.getByRole('button', {name: 'Edit'}).click();

    // The dialog should now be open.
    const dialog = page.locator('webstatus-manage-subscriptions-dialog');
    await expect(dialog).toBeVisible();

    // Change the frequency.
    await dialog.getByText('Monthly updates').click();

    // Save the changes.
    await dialog.getByRole('button', {name: 'Save'}).click();

    // Assert that the success toast appears.
    await expect(
      page.locator('sl-alert', {hasText: 'Subscription saved!'}),
    ).toBeVisible();

    // Assert that the frequency on the page has been updated.
    await expect(subscriptionListItem).toContainText('Frequency: monthly');
  });

  test('should allow deleting an existing subscription', async ({page}) => {
    // Navigate to the subscriptions page.
    await page.goto(subscriptionPageURL);

    // Find the subscription for "my first project query" and click its Delete button.
    const subscriptionListItem = page.locator('li', {
      hasText: 'my first project query',
    });
    await subscriptionListItem.getByRole('button', {name: 'Delete'}).click();

    // The dialog should now be open in delete confirmation mode.
    const dialog = page.locator('webstatus-manage-subscriptions-dialog');
    await expect(dialog).toBeVisible();

    // Confirm the deletion.
    await dialog.getByRole('button', {name: 'Confirm Unsubscribe'}).click();

    // Assert that the success toast appears.
    await expect(
      page.locator('sl-alert', {hasText: 'Subscription deleted!'}),
    ).toBeVisible();

    // Assert that the subscription is no longer in the list.
    await expect(subscriptionListItem).not.toBeVisible();
  });
});
