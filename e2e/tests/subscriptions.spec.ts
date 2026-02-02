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

    // Find the list item for the subscription we created in the fake data.
    const subscriptionItem = page.locator('.subscription-item', {
      hasText: 'my first project query',
    });

    // Assert that the subscription details are visible.
    await expect(subscriptionItem).toBeVisible();
    await expect(subscriptionItem).toContainText('test.user.1@example.com');
    await expect(subscriptionItem).toContainText('Weekly');
    await expect(
      subscriptionItem.getByRole('button', {name: 'Edit'}),
    ).toBeVisible();
    await expect(
      subscriptionItem.getByRole('button', {name: 'Delete'}),
    ).toBeVisible();
  });

  test('should allow creating a new subscription from the saved searches page', async ({
    page,
  }) => {
    // Navigate to a saved search page.
    await page.goto(
      `${BASE_URL}/?search_id=a09386fe-65f1-4640-b28d-3cf2f2de69c9`,
    );

    // The subscribe button should now be visible.
    const subscribeButton = page.getByRole('button', {name: 'Subscribe'});
    await expect(subscribeButton).toBeVisible();
    await subscribeButton.click();

    // The dialog should now be open.
    const dialog = page.locator('webstatus-manage-subscriptions-dialog');
    await expect(
      dialog.getByRole('heading', {name: 'Manage notifications'}),
    ).toBeVisible();

    // Select the notification channel (the user's email).
    await dialog.getByText('test.user.1@example.com').click();

    // Select a trigger.
    await dialog
      .locator('sl-checkbox')
      .filter({hasText: '...becomes widely available'})
      .locator('label')
      .click();

    // Select a frequency.
    await dialog.locator('sl-radio', {hasText: 'Monthly'}).click();

    // Save the subscription.
    await dialog.getByRole('button', {name: 'Create Subscription'}).click();

    // Assert that the success toast appears.
    await expect(
      page.locator('sl-alert', {hasText: 'Subscription saved!'}),
    ).toBeVisible();

    // Navigate to the subscriptions page to verify the new subscription.
    await page.goto(subscriptionPageURL);
    const newSubscription = page.locator('.subscription-item', {
      hasText: 'I like queries',
    });
    await expect(newSubscription).toBeVisible();
    await expect(newSubscription).toContainText('Monthly');
  });

  test('should allow editing an existing subscription', async ({page}) => {
    // Navigate to the subscriptions page.
    await page.goto(subscriptionPageURL);

    // Find the subscription for "my first project query" and click its Edit button.
    const subscriptionItem = page.locator('.subscription-item', {
      hasText: 'my first project query',
    });
    await subscriptionItem.getByRole('button', {name: 'Edit'}).click();

    // The dialog should now be open.
    const dialog = page.locator('webstatus-manage-subscriptions-dialog');
    await expect(
      dialog.getByRole('heading', {name: 'Manage notifications'}),
    ).toBeVisible();

    // Change the frequency.
    await dialog.locator('sl-radio', {hasText: 'Monthly'}).click();

    // Save the changes.
    await dialog.getByRole('button', {name: 'Save preferences'}).click();

    // Assert that the success toast appears.
    await expect(
      page.locator('sl-alert', {hasText: 'Subscription saved!'}),
    ).toBeVisible();

    // Assert that the frequency on the page has been updated.
    await expect(subscriptionItem).toContainText('Monthly');
  });

  test('should allow deleting an existing subscription', async ({page}) => {
    // Navigate to the subscriptions page.
    await page.goto(subscriptionPageURL);

    // Find the subscription for "my first project query" and click its Delete button.
    const subscriptionItem = page.locator('.subscription-item', {
      hasText: 'my first project query',
    });
    await subscriptionItem.getByRole('button', {name: 'Delete'}).click();

    // The dialog should now be open in delete confirmation mode.
    const dialog = page.locator('webstatus-manage-subscriptions-dialog');
    await expect(
      dialog.getByRole('heading', {name: 'Manage notifications'}),
    ).toBeVisible();

    // Confirm the deletion.
    const deleteButton = dialog.getByRole('button', {
      name: 'Confirm Unsubscribe',
    });
    await expect(deleteButton).toBeVisible();
    await deleteButton.click();

    // Assert that the success toast appears.
    await expect(
      page.locator('sl-alert', {hasText: 'Subscription deleted!'}),
    ).toBeVisible();

    // Assert that the subscription is no longer in the list.
    await expect(subscriptionItem).not.toBeVisible();
  });

  test('should automatically open the unsubscribe dialog when the unsubscribe query parameter is present', async ({
    page,
  }) => {
    // Navigate to the subscriptions page with the unsubscribe parameter.
    // We use the known subscription ID from the fake data for "my first project query".
    const subId = 'c1aa6418-1229-43a1-9a98-3f3604efe2ae';
    await page.goto(`${subscriptionPageURL}?unsubscribe=${subId}`);

    // The dialog should now be open.
    const dialog = page.locator('webstatus-manage-subscriptions-dialog');
    await expect(
      dialog.getByRole('heading', {name: 'Manage notifications'}),
    ).toBeVisible();

    // It should be in the "Confirm Unsubscribe" state.
    const confirmButton = dialog.getByRole('button', {
      name: 'Confirm Unsubscribe',
    });
    await expect(confirmButton).toBeVisible();

    // Click confirm.
    await confirmButton.click();

    // Assert that the success toast appears.
    await expect(
      page.locator('sl-alert', {hasText: 'Subscription deleted!'}),
    ).toBeVisible();

    // Find the list item for the subscription we deleted.
    const subscriptionItem = page.locator('.subscription-item', {
      hasText: 'my first project query',
    });

    // Assert that the subscription is no longer in the list.
    await expect(subscriptionItem).not.toBeVisible();
  });
});
