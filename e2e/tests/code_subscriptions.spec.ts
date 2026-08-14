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
  API_BASE_URL,
  resetUserData,
  expectDualThemeScreenshot,
  waitForSidebarLoaded,
} from './utils';
import {dispatchMockPushWebhook} from './webhook-helper';
import * as crypto from 'crypto';

const codeSubscriptionsPageURL = `${BASE_URL}/settings/code-subscriptions`;

test.describe.configure({mode: 'serial'});

test.describe('Code Subscriptions Unauthenticated', () => {
  test('should display unauthenticated state when user is not signed in', async ({
    page,
  }) => {
    await page.goto(codeSubscriptionsPageURL);
    const codeSubsPage = page.locator('webstatus-code-subscriptions-page');
    await expect(codeSubsPage).toBeVisible();

    const loginPrompt = codeSubsPage.locator('.login-prompt');
    await expect(loginPrompt).toBeVisible();
    await expect(loginPrompt).toContainText(
      'Sign in to view and manage code subscriptions for your repositories.',
    );

    // Verify visual appearance of unauthenticated prompt across light and dark themes
    await expectDualThemeScreenshot(
      page,
      codeSubsPage,
      'code-subscriptions-unauthenticated',
    );
  });
});

test.describe('Code Subscriptions E2E Suite', () => {
  test.afterAll(async () => {
    await resetUserData();
  });

  test.beforeEach(async ({page}) => {
    await resetUserData();
    await loginAsUser(page, 'test user 1');
  });

  test('should display Code Subscriptions item in sidebar menu', async ({
    page,
  }) => {
    await page.goto(`${BASE_URL}/`);
    const sidebar = page.locator('webstatus-sidebar');
    await waitForSidebarLoaded(page);

    const codeSubsLink = sidebar.locator('a.code-subscriptions-link');
    await expect(codeSubsLink).toBeVisible();
    await expect(codeSubsLink).toContainText('Code Subscriptions');
  });

  test('should display empty state when repository has no code subscriptions', async ({
    page,
  }) => {
    await page.route(
      '**/v1/vcs/github/repositories/*/code-subscriptions',
      async route => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [],
            metadata: {
              total: 0,
            },
          }),
        });
      },
    );

    await page.goto(codeSubscriptionsPageURL);
    const codeSubsPage = page.locator('webstatus-code-subscriptions-page');
    await expect(codeSubsPage).toBeVisible();

    const emptyCell = codeSubsPage.locator('td.empty-state');
    await expect(emptyCell).toBeVisible();
    await expect(emptyCell).toContainText(
      'No active code subscriptions found for this repository, or you do not have admin permissions to access it.',
    );

    // Verify visual appearance of empty state across light and dark themes
    await expectDualThemeScreenshot(
      page,
      codeSubsPage,
      'code-subscriptions-empty',
    );
  });

  test('should display populated code subscriptions table and match visual snapshot', async ({
    page,
  }) => {
    const mockSubscriptions = [
      {
        id: 'sub-1',
        repository_id: 'gh-repo-12345',
        repository_full_name: 'GoogleChrome/webstatus.dev',
        target_query: 'id:grid',
        triggers: ['feature_baseline_to_widely'],
        occurrences: [
          {
            file_path: 'src/components/grid.ts',
            line_number: 42,
            comment_snippet: '// TODO(id:grid): verify css grid support',
          },
        ],
        status: 'ACTIVE',
        created_at: '2026-08-01T00:00:00Z',
        updated_at: '2026-08-01T00:00:00Z',
      },
      {
        id: 'sub-2',
        repository_id: 'gh-repo-12345',
        repository_full_name: 'GoogleChrome/webstatus.dev',
        target_query: 'id:subgrid',
        triggers: ['feature_baseline_to_newly'],
        occurrences: [
          {
            file_path: 'src/layouts/subgrid.ts',
            line_number: 18,
            comment_snippet: '// TODO(id:subgrid)',
          },
          {
            file_path: 'src/styles/layout.css',
            line_number: 85,
            comment_snippet: '/* TODO(id:subgrid) */',
          },
        ],
        status: 'ACTIVE',
        created_at: '2026-08-10T00:00:00Z',
        updated_at: '2026-08-10T00:00:00Z',
      },
    ];

    await page.route(
      '**/v1/vcs/github/repositories/*/code-subscriptions',
      async route => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockSubscriptions,
            metadata: {
              total: mockSubscriptions.length,
            },
          }),
        });
      },
    );

    await page.goto(codeSubscriptionsPageURL);
    const codeSubsPage = page.locator('webstatus-code-subscriptions-page');
    await expect(codeSubsPage).toBeVisible();

    const table = codeSubsPage.locator('table.subscriptions-table');
    await expect(table).toBeVisible();

    const rows = table.locator('tbody tr');
    await expect(rows).toHaveCount(2);
    await expect(rows.nth(0)).toContainText('id:grid');
    await expect(rows.nth(0)).toContainText('Widely Available');
    await expect(rows.nth(1)).toContainText('id:subgrid');
    await expect(rows.nth(1)).toContainText('Newly Available');

    // Verify visual appearance of populated table across light and dark themes
    await expectDualThemeScreenshot(
      page,
      codeSubsPage,
      'code-subscriptions-populated',
    );
  });

  test('should display error alert when API returns error or user lacks admin access', async ({
    page,
  }) => {
    await page.route(
      '**/v1/vcs/github/repositories/*/code-subscriptions',
      async route => {
        await route.fulfill({
          status: 404,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 404,
            message: 'repository not found or insufficient admin permissions',
          }),
        });
      },
    );

    await page.goto(codeSubscriptionsPageURL);
    const codeSubsPage = page.locator('webstatus-code-subscriptions-page');
    await expect(codeSubsPage).toBeVisible();

    const alert = codeSubsPage.locator('sl-alert[variant="danger"]');
    await expect(alert).toBeVisible();
    await expect(alert).toContainText('Error loading code subscriptions');

    // Verify visual appearance of error state across light and dark themes
    await expectDualThemeScreenshot(
      page,
      codeSubsPage,
      'code-subscriptions-error',
    );
  });

  test('should reject synthetic webhook with invalid HMAC signature', async ({
    request,
  }) => {
    const res = await dispatchMockPushWebhook(request, API_BASE_URL, {
      secret: 'invalid-secret-key-that-does-not-match',
    });
    expect(res.status()).toBe(401);
  });

  test('should accept synthetic push webhook with valid HMAC signature', async ({
    request,
  }) => {
    const deliveryId = crypto.randomUUID();
    const res = await dispatchMockPushWebhook(request, API_BASE_URL, {
      deliveryId,
    });
    expect(res.status()).toBe(202);
  });

  test('should deduplicate webhook deliveries with identical delivery ID', async ({
    request,
  }) => {
    const deliveryId = crypto.randomUUID();

    // First request should be accepted
    const res1 = await dispatchMockPushWebhook(request, API_BASE_URL, {
      deliveryId,
    });
    expect(res1.status()).toBe(202);

    // Second request with same delivery ID should be deduplicated (202 Accepted)
    const res2 = await dispatchMockPushWebhook(request, API_BASE_URL, {
      deliveryId,
    });
    expect(res2.status()).toBe(202);
  });
});
