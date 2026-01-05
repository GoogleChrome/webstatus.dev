// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import {test, expect} from '@playwright/test';
import {resetUserData, loginAsUser, gotoOverviewPageUrl} from './utils.js';
import {
  getLatestEmail,
  triggerBatchJob,
  triggerNonMatchingChange,
  triggerMatchingChange,
  triggerBatchChange,
} from './test-data-util.js';

const TEST_USER_1 = {
  username: 'test user 1',
  email: 'test.user.1@example.com',
};

test.describe('Notifications', () => {
  test.beforeEach(async ({page}) => {
    await loginAsUser(page, TEST_USER_1.username);
    await resetUserData();
  });

  test('Immediate Edit Flow', async ({page}) => {
    await gotoOverviewPageUrl(page, 'http://localhost:5555/');
    await page.getByLabel('Search features').fill('group:css');
    await page.getByLabel('Search features').press('Enter');
    await page.getByRole('button', {name: 'Save Search'}).click();
    await page.getByLabel('Name').fill('Browser Features');
    await page.getByRole('button', {name: 'Save'}).click();
    await page.getByRole('button', {name: 'Subscribe to updates'}).click();
    await page.getByRole('radio', {name: 'Immediately'}).click();
    await page.getByRole('button', {name: 'Save'}).click();
    await expect(page.getByText('Subscription saved!')).toBeVisible();

    // Trigger the notification
    await page.getByRole('button', {name: 'Edit Search'}).click();
    await page.getByLabel('Query').fill('group:html');
    await page.getByRole('button', {name: 'Save'}).click();

    // Verify email
    const email = await test.step('Poll for email', async () => {
      for (let i = 0; i < 10; i++) {
        const email = await getLatestEmail(TEST_USER_1.email);
        if (email) {
          return email;
        }
        await new Promise(resolve => setTimeout(resolve, 1000));
      }
      return null;
    });

    expect(email).not.toBeNull();
    expect(email.Content.Headers.Subject[0]).toContain('Update:');
  });

  test('Batch Schedule Flow', async ({page}) => {
    await gotoOverviewPageUrl(page, 'http://localhost:5555/');
    await page.getByLabel('Search features').fill('group:css');
    await page.getByLabel('Search features').press('Enter');
    await page.getByRole('button', {name: 'Save Search'}).click();
    await page.getByLabel('Name').fill('Browser Features');
    await page.getByRole('button', {name: 'Save'}).click();
    await page.getByRole('button', {name: 'Subscribe to updates'}).click();
    await page.getByRole('radio', {name: 'Weekly updates'}).click();
    await page.getByRole('button', {name: 'Save'}).click();
    await expect(page.getByText('Subscription saved!')).toBeVisible();

    // Backdoor data change
    triggerBatchChange();

    // Trigger the batch job
    await triggerBatchJob('weekly');

    // Verify email
    const email = await test.step('Poll for email', async () => {
      for (let i = 0; i < 10; i++) {
        const email = await getLatestEmail(TEST_USER_1.email);
        if (
          email &&
          email.Content.Headers.Subject[0].includes('Weekly Digest')
        ) {
          return email;
        }
        await new Promise(resolve => setTimeout(resolve, 1000));
      }
      return null;
    });

    expect(email).not.toBeNull();
    expect(email.Content.Headers.Subject[0]).toContain('Weekly Digest');
  });

  test('2-Click Unsubscribe Flow', async ({page}) => {
    // 1. Setup: Run the "Immediate Edit" flow to generate an email.
    await gotoOverviewPageUrl(page, 'http://localhost:5555/');
    await page.getByLabel('Search features').fill('group:css');
    await page.getByLabel('Search features').press('Enter');
    await page.getByRole('button', {name: 'Save Search'}).click();
    await page.getByLabel('Name').fill('Browser Features');
    await page.getByRole('button', {name: 'Save'}).click();
    await page.getByRole('button', {name: 'Subscribe to updates'}).click();
    await page.getByRole('radio', {name: 'Immediately'}).click();
    await page.getByRole('button', {name: 'Save'}).click();
    await expect(page.getByText('Subscription saved!')).toBeVisible();
    await page.getByRole('button', {name: 'Edit Search'}).click();
    await page.getByLabel('Query').fill('group:html');
    await page.getByRole('button', {name: 'Save'}).click();
    const email = await test.step('Poll for email', async () => {
      for (let i = 0; i < 10; i++) {
        const email = await getLatestEmail(TEST_USER_1.email);
        if (email) {
          return email;
        }
        await new Promise(resolve => setTimeout(resolve, 1000));
      }
      return null;
    });
    expect(email).not.toBeNull();

    // 2. Extract Unsubscribe Link
    const unsubscribeLinkMatch = email.Content.Body.match(
      /href="([^"]+action=unsubscribe[^"]+)"/,
    );
    expect(unsubscribeLinkMatch).not.toBeNull();
    const unsubscribeUrl = unsubscribeLinkMatch[1];

    // 3. Action: Navigate to the link
    await page.goto(unsubscribeUrl);

    // 4. Interact: Confirm the unsubscription
    await page.getByRole('button', {name: 'Confirm Unsubscribe'}).click();
    await expect(page.getByText('Subscription deleted!')).toBeVisible();

    // 5. Verify: Go to the subscriptions page and check that the subscription is gone.
    await page.goto('/settings/subscriptions');
    await expect(page.getByText('No subscriptions found.')).toBeVisible();
  });

  test('Noise Filter Flow (Negative Test)', async ({page}) => {
    // 1. Setup
    await gotoOverviewPageUrl(page, 'http://localhost:5555/');
    await page.getByLabel('Search features').fill('group:css');
    await page.getByLabel('Search features').press('Enter');
    await page.getByRole('button', {name: 'Save Search'}).click();
    await page.getByLabel('Name').fill('Browser Features');
    await page.getByRole('button', {name: 'Save'}).click();
    await page.getByRole('button', {name: 'Subscribe to updates'}).click();
    await page.getByRole('radio', {name: 'Weekly updates'}).click();
    await page
      .getByRole('checkbox', {name: '...becomes widely available'})
      .check();
    await page.getByRole('button', {name: 'Save'}).click();
    await expect(page.getByText('Subscription saved!')).toBeVisible();

    // 2. Backdoor Action 1 (Non-matching change)
    triggerNonMatchingChange();

    // 3. Trigger
    await triggerBatchJob('weekly');

    // 4. Verify NO email is received
    await new Promise(resolve => setTimeout(resolve, 5000)); // Wait for a reasonable time
    let email = await getLatestEmail(TEST_USER_1.email);
    expect(email).toBeNull();

    // 5. Backdoor Action 2 (Matching change)
    triggerMatchingChange();

    // 6. Trigger
    await triggerBatchJob('weekly');

    // 7. Verify email IS received
    email = await test.step('Poll for email', async () => {
      for (let i = 0; i < 10; i++) {
        const email = await getLatestEmail(TEST_USER_1.email);
        if (email) {
          return email;
        }
        await new Promise(resolve => setTimeout(resolve, 1000));
      }
      return null;
    });
    expect(email).not.toBeNull();
  });

  test('Idempotency Flow', async ({page}) => {
    // 1. Setup
    await gotoOverviewPageUrl(page, 'http://localhost:5555/');
    await page.getByLabel('Search features').fill('group:css');
    await page.getByLabel('Search features').press('Enter');
    await page.getByRole('button', {name: 'Save Search'}).click();
    await page.getByLabel('Name').fill('Browser Features');
    await page.getByRole('button', {name: 'Save'}).click();
    await page.getByRole('button', {name: 'Subscribe to updates'}).click();
    await page.getByRole('radio', {name: 'Weekly updates'}).click();
    await page.getByRole('button', {name: 'Save'}).click();
    await expect(page.getByText('Subscription saved!')).toBeVisible();

    // Placeholder for backdoor data change
    triggerBatchChange();
    await triggerBatchJob('weekly');
    const firstEmail = await test.step('Poll for first email', async () => {
      for (let i = 0; i < 10; i++) {
        const email = await getLatestEmail(TEST_USER_1.email);
        if (email) {
          return email;
        }
        await new Promise(resolve => setTimeout(resolve, 1000));
      }
      return null;
    });
    expect(firstEmail).not.toBeNull();

    // 2. Action: Trigger again
    await triggerBatchJob('weekly');

    // 3. Verify: No new email
    await new Promise(resolve => setTimeout(resolve, 5000)); // Wait for a reasonable time
    const secondEmail = await getLatestEmail(TEST_USER_1.email);
    expect(secondEmail).not.toBeNull();
    expect(secondEmail.ID).toEqual(firstEmail.ID); // No new email, so latest is the same.
  });
});
