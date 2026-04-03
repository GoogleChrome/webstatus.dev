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
  loginAsUser,
  BASE_URL,
  expectDualThemeScreenshot,
  waitForSidebarLoaded,
  resetUserData,
} from './utils';

test('redirects unauthenticated user to home and shows toast', async ({
  page,
}) => {
  await page.goto(`${BASE_URL}/settings/notification-channels`);

  // Expect to be redirected to the home page.
  await expect(page).toHaveURL(`${BASE_URL}/`);
  // FYI: We do not assert the toast because it flashes on the screen due to the redirect.
});

test.describe('Notification Channels Page', () => {
  test.beforeEach(async ({page}) => {
    await resetUserData();
    await loginAsUser(page, 'test user 1');
    await page.goto(`${BASE_URL}/settings/notification-channels`);
    await waitForSidebarLoaded(page);
  });

  test.afterAll(async () => {
    await resetUserData();
  });

  test('authenticated user sees their email channel and coming soon messages', async ({
    page,
  }) => {
    // Expect the URL to be correct.
    await expect(page).toHaveURL(`${BASE_URL}/settings/notification-channels`);

    // Verify Email panel content.
    const emailPanel = page.locator('webstatus-notification-email-channels');
    await expect(emailPanel).toBeVisible();
    await expect(emailPanel).toContainText('test.user.1@example.com');
    await expect(emailPanel).toContainText('Enabled');

    // Verify RSS panel content.
    const rssPanel = page.locator('webstatus-notification-rss-channels');
    await expect(rssPanel).toBeVisible();
    await expect(rssPanel).toContainText('Coming soon');

    // Verify Webhook panel content.
    const webhookPanel = page.locator(
      'webstatus-notification-webhook-channels',
    );
    await expect(webhookPanel).toBeVisible();

    // Move the mouse to a neutral position to avoid hover effects on the screenshot.
    await page.mouse.move(0, 0);

    // Take a screenshot for visual regression.
    const pageContainer = page.locator('.page-container');
    await expectDualThemeScreenshot(
      page,
      pageContainer,
      'notification-channels-authenticated',
    );
  });

  test('authenticated user can create and delete a slack webhook channel', async ({
    page,
  }) => {
    const nonce = Date.now();
    const webhookName = 'PlaywrightTestCreateDeleteTest ' + nonce;
    const webhookUrl =
      'https://hooks.slack.com/services/PLAYWRIGHT/TEST/' + nonce;

    const webhookPanel = page.locator(
      'webstatus-notification-webhook-channels',
    );

    // Don't assert that no webhook channels are configured.
    // There may be some from previous test runs or from manual testing.

    // Click Create button.
    const createButton = webhookPanel.getByRole('button', {
      name: 'Create Webhook channel',
    });
    await expect(createButton).toBeVisible();
    await createButton.click();

    // Fill the dialog.
    const dialog = webhookPanel.locator(
      'webstatus-manage-notification-channel-dialog',
    );
    await expect(
      dialog.getByRole('heading', {name: 'Create Webhook Channel'}),
    ).toBeVisible();

    await dialog.getByRole('textbox', {name: 'Name'}).fill(webhookName);
    await dialog
      .getByRole('textbox', {name: 'Slack Webhook URL'})
      .fill(webhookUrl);

    await dialog.getByRole('button', {name: 'Create', exact: true}).click();

    // Verify it's in the list.
    await expect(dialog.locator('sl-dialog')).not.toBeVisible();
    const channelItem = webhookPanel.locator('.channel-item', {
      hasText: webhookName,
    });
    await expect(channelItem).toBeVisible();

    await channelItem.locator('sl-button[label="Delete"]').click();

    const deleteDialog = webhookPanel.locator('sl-dialog[open]');
    await expect(deleteDialog).toBeVisible();
    await deleteDialog
      .getByRole('button', {name: 'Delete', exact: true})
      .click();

    // Verify it's gone.
    await expect(channelItem).not.toBeVisible();
  });

  test('authenticated user can update a slack webhook channel', async ({
    page,
  }) => {
    // Use a nonce to make sure we don't have any stale data from previous test runs.
    // Avoid using resetUserData() since it's an expensive operation.
    const nonce = Date.now();
    const originalName = 'PlaywrightTestUpdateOriginal ' + nonce;
    const originalUrl =
      'https://hooks.slack.com/services/PLAYWRIGHT/TEST/original-' + nonce;
    const updatedName = 'PlaywrightTestUpdateUpdated ' + nonce;
    const updatedUrl =
      'https://hooks.slack.com/services/PLAYWRIGHT/TEST/updated-' + nonce;

    // Create a channel first.
    const webhookPanel = page.locator(
      'webstatus-notification-webhook-channels',
    );
    await webhookPanel
      .getByRole('button', {name: 'Create Webhook channel'})
      .click();
    const dialog = webhookPanel.locator(
      'webstatus-manage-notification-channel-dialog',
    );
    await expect(
      dialog.getByRole('heading', {name: 'Create Webhook Channel'}),
    ).toBeVisible();
    await dialog.getByRole('textbox', {name: 'Name'}).fill(originalName);
    await dialog
      .getByRole('textbox', {name: 'Slack Webhook URL'})
      .fill(originalUrl);
    await dialog.getByRole('button', {name: 'Create', exact: true}).click();

    // Verify it was created.
    await expect(dialog.locator('sl-dialog')).not.toBeVisible();
    const originalItem = webhookPanel.locator('.channel-item', {
      hasText: originalName,
    });
    await expect(originalItem).toBeVisible();

    await originalItem.locator('sl-button[label="Edit"]').click();

    // Verify current values in dialog.
    await expect(
      dialog.getByRole('heading', {name: 'Edit Webhook Channel'}),
    ).toBeVisible();
    await expect(dialog.getByRole('textbox', {name: 'Name'})).toHaveValue(
      originalName,
    );
    await expect(
      dialog.getByRole('textbox', {name: 'Slack Webhook URL'}),
    ).toHaveValue(originalUrl);

    // Update the values.
    await dialog.getByRole('textbox', {name: 'Name'}).fill(updatedName);
    await dialog
      .getByRole('textbox', {name: 'Slack Webhook URL'})
      .fill(updatedUrl);

    await dialog.getByRole('button', {name: 'Save', exact: true}).click();

    // Verify it was updated.
    await expect(dialog.locator('sl-dialog')).not.toBeVisible();
    const updatedItem = webhookPanel.locator('.channel-item', {
      hasText: updatedName,
    });
    await expect(updatedItem).toBeVisible();
    await expect(originalItem).not.toBeVisible();

    const deleteButton = updatedItem.locator('sl-button[label="Delete"]');
    await expect(deleteButton).toBeVisible();
    await deleteButton.click();

    const deleteDialog = webhookPanel.locator('sl-dialog[open]');
    await expect(deleteDialog).toBeVisible();
    await deleteDialog
      .getByRole('button', {name: 'Delete', exact: true})
      .click();
    await expect(updatedItem).not.toBeVisible();
  });
});
